package leonardo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
	"github.com/google/uuid"
)

const (
	DefaultBaseURL      = "https://cloud.leonardo.ai/api/rest"
	DefaultTimeout      = 30 * time.Second
	MaxResponseBodySize = 1 << 20
	SubmissionUnknown   = "submission_unknown"
	SideEffectUnknown   = "side_effect_unknown"
)

var ErrResponseTooLarge = errors.New("leonardo response body is too large")

type Client struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string, timeout time.Duration) (*Client, error) {
	return NewClientWithHTTPClient(baseURL, apiKey, timeout, nil)
}

func NewClientWithHTTPClient(baseURL, apiKey string, timeout time.Duration, httpClient *http.Client) (*Client, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("leonardo: invalid base URL")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, errors.New("leonardo: API key is required")
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if httpClient == nil {
		httpClient = &http.Client{}
	} else {
		clone := *httpClient
		httpClient = &clone
	}
	httpClient.Timeout = timeout
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &Client{baseURL: parsed, apiKey: apiKey, httpClient: httpClient}, nil
}

func (c *Client) ListModels(ctx context.Context) ([]Model, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/v2/models", nil)
	if err != nil {
		return nil, fmt.Errorf("leonardo: build models request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.New(c.sanitize("leonardo: list models: " + err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readBody(resp.Body)
	if err != nil {
		if !errors.Is(err, ErrResponseTooLarge) {
			return nil, errors.New(c.sanitize(err.Error()))
		}
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, c.responseError(resp, body)
	}
	var decoded modelsResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("leonardo: decode models response: %w", err)
	}
	return decoded.Models, nil
}

func (c *Client) CreateGeneration(ctx context.Context, request CreateGenerationRequest) (*CreateGenerationResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("leonardo: encode generation request: %w", err)
	}
	req, err := c.newRequest(ctx, http.MethodPost, "/v2/generations", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("leonardo: build generation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	var wroteRequest atomic.Bool
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{
		WroteRequest: func(httptrace.WroteRequestInfo) { wroteRequest.Store(true) },
	}))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		message := c.sanitize("leonardo: create generation: " + err.Error())
		if wroteRequest.Load() {
			return nil, submissionUnknownError(0, message, "", "", "")
		}
		return nil, &LeonardoError{Message: message, SafeToRetry: true, cause: ErrGenerationRequestNotWritten}
	}
	defer func() { _ = resp.Body.Close() }()
	responseBody, err := readBody(resp.Body)
	if err != nil {
		return nil, submissionUnknownError(resp.StatusCode, c.sanitize(err.Error()), "", sanitizeHeader(c.sanitize(resp.Header.Get("X-Request-ID"))), "")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		apiErr := c.responseError(resp, responseBody)
		apiErr.RetryableRead = false
		apiErr.SubmissionStatus = SubmissionUnknown
		apiErr.SideEffectStatus = SideEffectUnknown
		return nil, apiErr
	}
	var decoded CreateGenerationResponse
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return nil, submissionUnknownError(resp.StatusCode, c.sanitize("leonardo: decode generation response: "+err.Error()), "", sanitizeHeader(c.sanitize(resp.Header.Get("X-Request-ID"))), "")
	}
	if !validUUID(decoded.GenerationID) {
		return nil, submissionUnknownError(resp.StatusCode, "leonardo: generation response has invalid generationId", "", sanitizeHeader(c.sanitize(resp.Header.Get("X-Request-ID"))), "")
	}
	return &decoded, nil
}

func (c *Client) GetGeneration(ctx context.Context, id string) (*Generation, error) {
	if !validUUID(id) {
		return nil, errors.New("leonardo: invalid generation ID")
	}
	req, err := c.newRequest(ctx, http.MethodGet, "/v1/generations/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("leonardo: build generation status request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.New(c.sanitize("leonardo: get generation: " + err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readBody(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, c.responseError(resp, body)
	}
	var decoded generationResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("leonardo: decode generation status response: %w", err)
	}
	switch decoded.Generation.Status {
	case "PENDING", "COMPLETE", "FAILED":
	default:
		return nil, fmt.Errorf("leonardo: unsupported generation status %q", decoded.Generation.Status)
	}
	return &decoded.Generation, nil
}

func validUUID(value string) bool {
	parsed, err := uuid.Parse(value)
	return err == nil && parsed.String() == value
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	u := *c.baseURL
	u.Path = strings.TrimRight(u.Path, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	return req, nil
}

func readBody(reader io.Reader) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, MaxResponseBodySize+1))
	if err != nil {
		return nil, fmt.Errorf("leonardo: read response: %w", err)
	}
	if len(body) > MaxResponseBodySize {
		return nil, ErrResponseTooLarge
	}
	return body, nil
}

func (c *Client) responseError(resp *http.Response, body []byte) *LeonardoError {
	var decoded errorResponse
	_ = json.Unmarshal(body, &decoded)
	message := decoded.Error
	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}
	return &LeonardoError{
		StatusCode:    resp.StatusCode,
		Code:          c.sanitize(decoded.Code),
		Message:       c.sanitize(message),
		Path:          c.sanitize(decoded.Path),
		RequestID:     sanitizeHeader(c.sanitize(resp.Header.Get("X-Request-ID"))),
		RetryAfter:    parseRetryAfter(resp.Header.Get("Retry-After"), time.Now()),
		RetryableRead: resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError,
	}
}

func submissionUnknownError(statusCode int, message, code, requestID, path string) *LeonardoError {
	return &LeonardoError{
		StatusCode:       statusCode,
		Code:             code,
		Message:          message,
		Path:             path,
		RequestID:        requestID,
		SubmissionStatus: SubmissionUnknown,
		SideEffectStatus: SideEffectUnknown,
	}
}

func (c *Client) sanitize(value string) string {
	value = strings.ReplaceAll(value, c.apiKey, "***")
	return logredact.RedactText(value, "api_key", "authorization", "cookie", "signature", "x-api-key")
}

func sanitizeHeader(value string) string {
	value = strings.TrimSpace(value)
	for _, r := range value {
		if r < 0x21 || r > 0x7e {
			return ""
		}
	}
	if len(value) > 256 {
		return ""
	}
	return value
}

func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds >= 0 && seconds <= int64((time.Duration(1<<63-1)/time.Second)) {
		return time.Duration(seconds) * time.Second
	}
	when, err := http.ParseTime(value)
	if err != nil || !when.After(now) {
		return 0
	}
	return when.Sub(now)
}
