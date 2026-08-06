package leonardo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
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

func (c *Client) CreateInitImageUpload(ctx context.Context, extension string) (*InitImageUpload, error) {
	extension = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(extension), "."))
	if extension != "jpg" && extension != "jpeg" && extension != "png" && extension != "webp" {
		return nil, errors.New("leonardo: unsupported init image extension")
	}
	body, _ := json.Marshal(map[string]string{"extension": extension})
	req, err := c.newRequest(ctx, http.MethodPost, "/v1/init-image", bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("leonardo: build init image request")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.New(c.sanitize("leonardo: create init image upload: " + err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()
	responseBody, err := readBody(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, c.responseError(resp, responseBody)
	}
	var decoded initImageUploadResponse
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return nil, errors.New("leonardo: decode init image response")
	}
	fields, err := decodeInitImageFields(decoded.Upload.Fields)
	if err != nil || strings.TrimSpace(decoded.Upload.ID) == "" || strings.TrimSpace(decoded.Upload.URL) == "" || len(fields) == 0 {
		return nil, errors.New("leonardo: invalid init image response")
	}
	return &InitImageUpload{ID: decoded.Upload.ID, URL: decoded.Upload.URL, Key: decoded.Upload.Key, Fields: fields}, nil
}

func (c *Client) UploadInitImage(ctx context.Context, upload *InitImageUpload, filename string, file io.Reader) error {
	if upload == nil || strings.TrimSpace(upload.URL) == "" || len(upload.Fields) == 0 || file == nil {
		return errors.New("leonardo: invalid init image upload")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range upload.Fields {
		if err := writer.WriteField(key, value); err != nil {
			return errors.New("leonardo: build init image upload")
		}
	}
	part, err := writer.CreateFormFile("file", strings.TrimSpace(filename))
	if err != nil {
		return errors.New("leonardo: build init image upload")
	}
	if written, err := io.Copy(part, file); err != nil || written == 0 {
		return errors.New("leonardo: read init image file")
	}
	if err := writer.Close(); err != nil {
		return errors.New("leonardo: build init image upload")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, upload.URL, &body)
	if err != nil {
		return errors.New("leonardo: build init image upload request")
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.New(c.sanitize("leonardo: upload init image: " + err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		_, _ = readBody(resp.Body)
		return fmt.Errorf("leonardo: init image upload returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func decodeInitImageFields(raw json.RawMessage) (map[string]string, error) {
	var fields map[string]string
	if err := json.Unmarshal(raw, &fields); err == nil {
		return fields, nil
	}
	var encoded string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(encoded), &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func (c *Client) CreateGeneration(ctx context.Context, request CreateGenerationRequest) (*CreateGenerationResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("leonardo: encode generation request: %w", err)
	}
	return c.CreateGenerationRaw(ctx, body)
}

func (c *Client) CreateGenerationRaw(ctx context.Context, body []byte) (*CreateGenerationResponse, error) {
	if len(body) == 0 || !json.Valid(body) {
		return nil, &LeonardoError{Class: GenerationErrorClassRequestNotWritten, Message: "leonardo: invalid generation request", SafeToRetry: true, cause: ErrGenerationRequestNotWritten}
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
			return nil, submissionUnknownError(GenerationErrorClassTransportAfterWrite, 0, message, "", "", "", nil, false, false)
		}
		return nil, &LeonardoError{Class: GenerationErrorClassRequestNotWritten, Message: message, SafeToRetry: true, cause: ErrGenerationRequestNotWritten}
	}
	defer func() { _ = resp.Body.Close() }()
	responseBody, err := readGenerationBody(resp.Body)
	if err != nil {
		class := GenerationErrorClassResponseReadFailed
		if errors.Is(err, ErrResponseTooLarge) {
			class = GenerationErrorClassResponseTooLarge
		}
		return nil, submissionUnknownError(class, resp.StatusCode, c.sanitize(err.Error()), "", sanitizeHeader(c.sanitize(resp.Header.Get("X-Request-ID"))), "", responseBody, true, false)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		apiErr := c.responseError(resp, responseBody)
		apiErr.Class = GenerationErrorClassUpstreamNon2xx
		apiErr.BodySHA256 = bodySHA256(responseBody)
		apiErr.BodySize = int64(len(responseBody))
		apiErr.RetryableRead = false
		apiErr.SubmissionStatus = SubmissionUnknown
		apiErr.SideEffectStatus = SideEffectUnknown
		return nil, apiErr
	}
	var decoded struct {
		GenerationID    string          `json:"generationId"`
		Cost            json.RawMessage `json:"cost"`
		APICreditCost   json.RawMessage `json:"apiCreditCost"`
		Generate        json.RawMessage `json:"generate"`
		SDGenerationJob json.RawMessage `json:"sdGenerationJob"`
	}
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return nil, submissionUnknownError(GenerationErrorClassResponseDecodeFailed, resp.StatusCode, c.sanitize("leonardo: decode generation response: "+err.Error()), "", sanitizeHeader(c.sanitize(resp.Header.Get("X-Request-ID"))), "", responseBody, false, true)
	}
	result := &CreateGenerationResponse{GenerationID: decoded.GenerationID}
	cost := decoded.Cost
	apiCreditCost := decoded.APICreditCost
	if result.GenerationID == "" && len(decoded.Generate) > 0 {
		result.GenerationID, cost, apiCreditCost = decodeGenerationEnvelope(decoded.Generate)
	}
	if result.GenerationID == "" && len(decoded.SDGenerationJob) > 0 {
		result.GenerationID, cost, apiCreditCost = decodeGenerationEnvelope(decoded.SDGenerationJob)
	}
	if result.GenerationID == "" {
		return nil, submissionUnknownError(GenerationErrorClassGenerationIDMissing, resp.StatusCode, "leonardo: generation response has missing generationId", "", sanitizeHeader(c.sanitize(resp.Header.Get("X-Request-ID"))), "", responseBody, false, true)
	}
	if !validUUID(result.GenerationID) {
		return nil, submissionUnknownError(GenerationErrorClassGenerationIDInvalid, resp.StatusCode, "leonardo: generation response has invalid generationId", "", sanitizeHeader(c.sanitize(resp.Header.Get("X-Request-ID"))), "", responseBody, false, true)
	}
	var generationCost GenerationCost
	if json.Unmarshal(cost, &generationCost) == nil {
		result.Cost = &generationCost
	}
	var credits float64
	if json.Unmarshal(apiCreditCost, &credits) == nil {
		result.APICreditCost = &credits
	}
	return result, nil
}

func decodeGenerationEnvelope(raw json.RawMessage) (string, json.RawMessage, json.RawMessage) {
	var envelope struct {
		GenerationID  string          `json:"generationId"`
		ID            string          `json:"id"`
		Cost          json.RawMessage `json:"cost"`
		APICreditCost json.RawMessage `json:"apiCreditCost"`
	}
	if json.Unmarshal(raw, &envelope) != nil {
		return "", nil, nil
	}
	if envelope.GenerationID != "" {
		return envelope.GenerationID, envelope.Cost, envelope.APICreditCost
	}
	return envelope.ID, envelope.Cost, envelope.APICreditCost
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
		if errors.Is(err, ErrResponseTooLarge) {
			return nil, err
		}
		return nil, errors.New(c.sanitize(err.Error()))
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, c.responseError(resp, body)
	}
	var decoded generationResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, errors.New(c.sanitize("leonardo: decode generation status response: " + err.Error()))
	}
	switch decoded.Generation.Status {
	case "PENDING", "COMPLETE", "FAILED":
	default:
		return nil, errors.New("leonardo: unsupported generation status")
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

func readGenerationBody(reader io.Reader) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, MaxResponseBodySize+1))
	if err != nil {
		return body, fmt.Errorf("leonardo: read response: %w", err)
	}
	if len(body) > MaxResponseBodySize {
		return body, ErrResponseTooLarge
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

func submissionUnknownError(class string, statusCode int, message, code, requestID, path string, body []byte, truncated, complete bool) *LeonardoError {
	err := &LeonardoError{
		Class:            class,
		StatusCode:       statusCode,
		Code:             code,
		Message:          message,
		Path:             path,
		RequestID:        requestID,
		SubmissionStatus: SubmissionUnknown,
		SideEffectStatus: SideEffectUnknown,
		BodyTruncated:    truncated,
	}
	if body != nil {
		err.BodySize = int64(len(body))
		if complete {
			err.BodySHA256 = bodySHA256(body)
		}
	}
	return err
}

func bodySHA256(body []byte) string {
	hash := sha256.Sum256(body)
	return fmt.Sprintf("%x", hash)
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
