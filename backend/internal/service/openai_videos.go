package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	xaiVideosGenerationsEndpoint = "/v1/videos/generations"
)

func IsXAIVideoModel(model string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "grok-imagine-video")
}

func NormalizeXAIVideoGenerationBodyForHandler(body []byte) ([]byte, string, error) {
	return normalizeXAIVideoGenerationBody(body)
}

func normalizeXAIVideoGenerationBody(body []byte) ([]byte, string, error) {
	if len(body) == 0 {
		return nil, "", fmt.Errorf("request body is empty")
	}
	if !gjson.ValidBytes(body) {
		return nil, "", fmt.Errorf("failed to parse request body")
	}
	model := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	if model == "" {
		return nil, "", fmt.Errorf("model is required")
	}
	if !IsXAIVideoModel(model) {
		return nil, "", fmt.Errorf("videos endpoint requires an xAI video model, got %q", model)
	}

	out := body
	// xAI currently treats seconds as an alias of duration; sending both returns a duplicate-field error.
	if !gjson.GetBytes(out, "duration").Exists() {
		if seconds := gjson.GetBytes(out, "seconds"); seconds.Exists() {
			if next, err := sjson.SetBytes(out, "duration", seconds.Value()); err == nil {
				out = next
			}
		}
	}
	if gjson.GetBytes(out, "seconds").Exists() {
		if next, err := sjson.DeleteBytes(out, "seconds"); err == nil {
			out = next
		}
	}
	// Upstream deprecated images in favor of reference_images. Accept both at the RelayQ edge.
	if images := gjson.GetBytes(out, "images"); images.Exists() {
		if !gjson.GetBytes(out, "reference_images").Exists() {
			if next, err := sjson.SetRawBytes(out, "reference_images", []byte(images.Raw)); err == nil {
				out = next
			}
		}
		if next, err := sjson.DeleteBytes(out, "images"); err == nil {
			out = next
		}
	}
	return out, model, nil
}

func (s *OpenAIGatewayService) buildXAIVideoURL(account *Account, suffix string) (string, error) {
	baseURL := ""
	if account != nil {
		baseURL = account.GetOpenAIBaseURL()
	}
	if baseURL == "" {
		baseURL = "https://api.x.ai"
	}
	validatedURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid xai base_url: %w", err)
	}
	return strings.TrimRight(validatedURL, "/") + suffix, nil
}

func (s *OpenAIGatewayService) ForwardXAIVideoGeneration(ctx context.Context, c *gin.Context, account *Account, body []byte) error {
	if !isXAIOAuthAccount(account) {
		return fmt.Errorf("account is not an xai oauth account")
	}
	forwardBody, _, err := normalizeXAIVideoGenerationBody(body)
	if err != nil {
		return err
	}
	targetURL, err := s.buildXAIVideoURL(account, xaiVideosGenerationsEndpoint)
	if err != nil {
		return err
	}
	return s.forwardXAIVideoRequest(ctx, c, account, http.MethodPost, targetURL, forwardBody)
}

func (s *OpenAIGatewayService) ForwardXAIVideoStatus(ctx context.Context, c *gin.Context, account *Account, requestID string) error {
	if !isXAIOAuthAccount(account) {
		return fmt.Errorf("account is not an xai oauth account")
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return fmt.Errorf("request_id is required")
	}
	targetURL, err := s.buildXAIVideoURL(account, "/v1/videos/"+requestID)
	if err != nil {
		return err
	}
	return s.forwardXAIVideoRequest(ctx, c, account, http.MethodGet, targetURL, nil)
}

func (s *OpenAIGatewayService) forwardXAIVideoRequest(ctx context.Context, c *gin.Context, account *Account, method, targetURL string, body []byte) error {
	token := account.GetOpenAIAccessToken()
	if strings.TrimSpace(token) == "" {
		refreshedToken, err := s.forceRefreshXAIOAuthAccount(ctx, account)
		if err != nil {
			return err
		}
		token = refreshedToken
	}
	resp, err := s.doXAIVideoRequest(ctx, account, method, targetURL, body, token)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return readErr
	}
	if (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) && isXAIBadCredentials(respBody) {
		refreshedToken, refreshErr := s.forceRefreshXAIOAuthAccount(ctx, account)
		if refreshErr == nil && strings.TrimSpace(refreshedToken) != "" {
			resp, err = s.doXAIVideoRequest(ctx, account, method, targetURL, body, refreshedToken)
			if err != nil {
				return err
			}
			defer func() { _ = resp.Body.Close() }()
			respBody, readErr = io.ReadAll(resp.Body)
			if readErr != nil {
				return readErr
			}
		}
	}
	contentType := resp.Header.Get("Content-Type")
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/json"
	}
	c.Data(resp.StatusCode, contentType, respBody)
	return nil
}

func (s *OpenAIGatewayService) doXAIVideoRequest(ctx context.Context, account *Account, method, targetURL string, body []byte, token string) (*http.Response, error) {
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, targetURL, reader)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	return s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
}
