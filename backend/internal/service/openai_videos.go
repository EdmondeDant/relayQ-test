package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	xaiVideosGenerationsEndpoint = "/v1/videos/generations"
	xaiVideosEditsEndpoint       = "/v1/videos/edits"
	xaiVideosExtensionsEndpoint  = "/v1/videos/extensions"
	xaiVideoDefaultModel         = "grok-imagine-video"
	xaiVideoRequestAccountTTL    = 6 * time.Hour
)

func IsXAIVideoModel(model string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "grok-imagine-video")
}

func IsOpenAICompatibleVideoModel(model string) bool {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "sora-2", "sora-2-pro":
		return true
	default:
		return IsXAIVideoModel(model)
	}
}

func NormalizeXAIVideoModel(model string) string {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "sora-2", "sora-2-pro":
		return xaiVideoDefaultModel
	default:
		return strings.TrimSpace(model)
	}
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
	if !IsOpenAICompatibleVideoModel(model) {
		return nil, "", fmt.Errorf("videos endpoint requires an xAI-compatible video model, got %q", model)
	}
	requestModel := NormalizeXAIVideoModel(model)

	out := body
	if requestModel != model {
		if next, err := sjson.SetBytes(out, "model", requestModel); err == nil {
			out = next
		}
	}
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
	// OpenAI/Sora-compatible clients (for example OpenClaw's openai video provider)
	// send image-to-video references as input_reference.image_url. xAI expects
	// reference_images, so normalize at RelayQ's edge.
	if ref := strings.TrimSpace(gjson.GetBytes(out, "input_reference.image_url").String()); ref != "" {
		if !gjson.GetBytes(out, "reference_images").Exists() {
			if next, err := sjson.SetBytes(out, "reference_images", []map[string]string{{"image_url": ref}}); err == nil {
				out = next
			}
		}
		if next, err := sjson.DeleteBytes(out, "input_reference"); err == nil {
			out = next
		}
	}
	return out, requestModel, nil
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
	return s.forwardXAIVideoSubmit(ctx, c, account, body, xaiVideosGenerationsEndpoint)
}

func (s *OpenAIGatewayService) ForwardXAIVideoEdit(ctx context.Context, c *gin.Context, account *Account, body []byte) error {
	return s.forwardXAIVideoSubmit(ctx, c, account, body, xaiVideosEditsEndpoint)
}

func (s *OpenAIGatewayService) ForwardXAIVideoExtension(ctx context.Context, c *gin.Context, account *Account, body []byte) error {
	return s.forwardXAIVideoSubmit(ctx, c, account, body, xaiVideosExtensionsEndpoint)
}

func (s *OpenAIGatewayService) forwardXAIVideoSubmit(ctx context.Context, c *gin.Context, account *Account, body []byte, endpoint string) error {
	if !isXAIOAuthAccount(account) {
		return fmt.Errorf("account is not an xai oauth account")
	}
	forwardBody, _, err := normalizeXAIVideoGenerationBody(body)
	if err != nil {
		return err
	}
	targetURL, err := s.buildXAIVideoURL(account, endpoint)
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

func (s *OpenAIGatewayService) ForwardXAIVideoContent(ctx context.Context, c *gin.Context, account *Account, requestID string) error {
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
	status, _, body, err := s.fetchXAIVideoResponse(ctx, account, http.MethodGet, targetURL, nil)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		c.Data(status, "application/json", body)
		return nil
	}
	videoURL := firstNonEmptyString(
		gjson.GetBytes(body, "video.url").String(),
		gjson.GetBytes(body, "output.video.url").String(),
		gjson.GetBytes(body, "output_url").String(),
		gjson.GetBytes(body, "url").String(),
	)
	videoURL = strings.TrimSpace(videoURL)
	if videoURL == "" {
		return fmt.Errorf("video content is not ready or no video url was returned")
	}
	c.Redirect(http.StatusFound, videoURL)
	return nil
}

func (s *OpenAIGatewayService) forwardXAIVideoRequest(ctx context.Context, c *gin.Context, account *Account, method, targetURL string, body []byte) error {
	status, headers, respBody, err := s.fetchXAIVideoResponse(ctx, account, method, targetURL, body)
	if err != nil {
		return err
	}
	if method == http.MethodPost && status >= 200 && status < 300 {
		s.maybeBindXAIVideoRequestAccount(ctx, c, account, respBody)
	}
	contentType := headers.Get("Content-Type")
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/json"
	}
	c.Data(status, contentType, respBody)
	return nil
}

func (s *OpenAIGatewayService) fetchXAIVideoResponse(ctx context.Context, account *Account, method, targetURL string, body []byte) (int, http.Header, []byte, error) {
	token := account.GetOpenAIAccessToken()
	if strings.TrimSpace(token) == "" {
		refreshedToken, err := s.forceRefreshXAIOAuthAccount(ctx, account)
		if err != nil {
			return 0, nil, nil, err
		}
		token = refreshedToken
	}
	resp, err := s.doXAIVideoRequest(ctx, account, method, targetURL, body, token)
	if err != nil {
		return 0, nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return 0, nil, nil, readErr
	}
	if (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) && isXAIBadCredentials(respBody) {
		refreshedToken, refreshErr := s.forceRefreshXAIOAuthAccount(ctx, account)
		if refreshErr == nil && strings.TrimSpace(refreshedToken) != "" {
			resp, err = s.doXAIVideoRequest(ctx, account, method, targetURL, body, refreshedToken)
			if err != nil {
				return 0, nil, nil, err
			}
			defer func() { _ = resp.Body.Close() }()
			respBody, readErr = io.ReadAll(resp.Body)
			if readErr != nil {
				return 0, nil, nil, readErr
			}
		}
	}
	return resp.StatusCode, resp.Header, respBody, nil
}

func (s *OpenAIGatewayService) maybeBindXAIVideoRequestAccount(ctx context.Context, c *gin.Context, account *Account, body []byte) {
	if s == nil || account == nil || len(body) == 0 {
		return
	}
	requestID := firstNonEmptyString(
		gjson.GetBytes(body, "request_id").String(),
		gjson.GetBytes(body, "id").String(),
	)
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return
	}
	apiKey := getAPIKeyFromContext(c)
	if apiKey == nil || apiKey.GroupID == nil {
		return
	}
	store := s.getOpenAIWSStateStore()
	if store == nil {
		return
	}
	_ = store.BindVideoRequestAccount(ctx, *apiKey.GroupID, requestID, account.ID, xaiVideoRequestAccountTTL)
}

func (s *OpenAIGatewayService) ResolveXAIVideoRequestAccount(ctx context.Context, groupID *int64, requestID string) (*Account, bool) {
	if s == nil || groupID == nil || strings.TrimSpace(requestID) == "" {
		return nil, false
	}
	store := s.getOpenAIWSStateStore()
	if store == nil || s.accountRepo == nil {
		return nil, false
	}
	accountID, err := store.GetVideoRequestAccount(ctx, *groupID, requestID)
	if err != nil || accountID <= 0 {
		return nil, false
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil || account == nil || account.Platform != PlatformXAI {
		return nil, false
	}
	return account, true
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
