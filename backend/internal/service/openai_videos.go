package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

const (
	openAIVideosGenerationsEndpoint = "/v1/videos/generations"
	openAIVideosEditsEndpoint       = "/v1/videos/edits"
	openAIVideosExtensionsEndpoint  = "/v1/videos/extensions"
	openAIVideosPollEndpoint        = "/v1/videos/{request_id}"

	openAIVideosGenerationsURL = "https://api.x.ai/v1/videos/generations"
	openAIVideosEditsURL       = "https://api.x.ai/v1/videos/edits"
	openAIVideosExtensionsURL  = "https://api.x.ai/v1/videos/extensions"
	openAIVideosPollURLPrefix  = "https://api.x.ai/v1/videos/"
)

type OpenAIVideoInput struct {
	URL    string `json:"url,omitempty"`
	Data   string `json:"data,omitempty"`
	FileID string `json:"file_id,omitempty"`
}

func (i OpenAIVideoInput) isEmpty() bool {
	return strings.TrimSpace(i.URL) == "" &&
		strings.TrimSpace(i.Data) == "" &&
		strings.TrimSpace(i.FileID) == ""
}

type OpenAIVideosRequest struct {
	Method          string
	Endpoint        string
	Model           string
	ExplicitModel   bool
	Prompt          string
	Duration        *int
	AspectRatio     string
	Resolution      string
	Image           *OpenAIVideoInput
	Video           *OpenAIVideoInput
	ReferenceImages []OpenAIVideoInput
	RequestID       string
	Body            []byte
	bodyHash        string
}

func (r *OpenAIVideosRequest) IsGeneration() bool {
	return r != nil && r.Endpoint == openAIVideosGenerationsEndpoint
}

func (r *OpenAIVideosRequest) IsEdit() bool {
	return r != nil && r.Endpoint == openAIVideosEditsEndpoint
}

func (r *OpenAIVideosRequest) IsExtension() bool {
	return r != nil && r.Endpoint == openAIVideosExtensionsEndpoint
}

func (r *OpenAIVideosRequest) IsPoll() bool {
	return r != nil && r.Endpoint == openAIVideosPollEndpoint
}

func (r *OpenAIVideosRequest) StickySessionSeed() string {
	if r == nil {
		return ""
	}
	parts := []string{
		"openai-videos",
		strings.TrimSpace(r.Endpoint),
		strings.TrimSpace(r.Model),
		strings.TrimSpace(r.Prompt),
	}
	if r.Duration != nil && *r.Duration > 0 {
		parts = append(parts, fmt.Sprintf("duration=%d", *r.Duration))
	}
	if v := strings.TrimSpace(r.AspectRatio); v != "" {
		parts = append(parts, "aspect="+v)
	}
	if v := strings.TrimSpace(r.Resolution); v != "" {
		parts = append(parts, "resolution="+v)
	}
	if r.Image != nil && !r.Image.isEmpty() {
		parts = append(parts, "image=1")
	}
	if r.Video != nil && !r.Video.isEmpty() {
		parts = append(parts, "video=1")
	}
	if len(r.ReferenceImages) > 0 {
		parts = append(parts, fmt.Sprintf("refs=%d", len(r.ReferenceImages)))
	}
	if r.bodyHash != "" {
		parts = append(parts, "body="+r.bodyHash)
	}
	return strings.Join(parts, "|")
}

func (r *OpenAIVideosRequest) ModerationBody() []byte {
	if r == nil || r.IsPoll() {
		return nil
	}
	payload := map[string]any{}
	if prompt := strings.TrimSpace(r.Prompt); prompt != "" {
		payload["prompt"] = prompt
	}
	images := make([]map[string]string, 0, 1+len(r.ReferenceImages))
	if r.Image != nil {
		if imageURL := firstNonEmptyString(r.Image.URL, r.Image.Data); strings.TrimSpace(imageURL) != "" {
			images = append(images, map[string]string{"image_url": strings.TrimSpace(imageURL)})
		}
	}
	for _, ref := range r.ReferenceImages {
		if imageURL := firstNonEmptyString(ref.URL, ref.Data); strings.TrimSpace(imageURL) != "" {
			images = append(images, map[string]string{"image_url": strings.TrimSpace(imageURL)})
		}
	}
	if len(images) > 0 {
		payload["images"] = images
	}
	if len(payload) == 0 {
		return nil
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	return body
}

func (s *OpenAIGatewayService) ParseOpenAIVideosRequest(c *gin.Context, body []byte) (*OpenAIVideosRequest, error) {
	if c == nil || c.Request == nil {
		return nil, fmt.Errorf("missing request context")
	}
	endpoint, requestID := normalizeOpenAIVideosEndpointPath(c.Request.Method, c.Request.URL.Path)
	if endpoint == "" {
		return nil, fmt.Errorf("unsupported videos endpoint")
	}

	req := &OpenAIVideosRequest{
		Method:    strings.ToUpper(strings.TrimSpace(c.Request.Method)),
		Endpoint:  endpoint,
		RequestID: requestID,
		Body:      body,
	}
	if len(body) > 0 {
		sum := sha256.Sum256(body)
		req.bodyHash = hex.EncodeToString(sum[:8])
	}

	if req.IsPoll() {
		if strings.TrimSpace(req.RequestID) == "" {
			return nil, fmt.Errorf("request_id is required")
		}
		return req, nil
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("request body is empty")
	}
	if !gjson.ValidBytes(body) {
		return nil, fmt.Errorf("failed to parse request body")
	}
	if err := parseOpenAIVideosJSONRequest(body, req); err != nil {
		return nil, err
	}
	applyOpenAIVideosDefaults(req)
	if err := validateOpenAIVideosRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

func parseOpenAIVideosJSONRequest(body []byte, req *OpenAIVideosRequest) error {
	if req == nil {
		return fmt.Errorf("videos request is required")
	}
	if modelResult := gjson.GetBytes(body, "model"); modelResult.Exists() {
		req.Model = strings.TrimSpace(modelResult.String())
		req.ExplicitModel = req.Model != ""
	}
	req.Prompt = strings.TrimSpace(gjson.GetBytes(body, "prompt").String())
	req.AspectRatio = strings.TrimSpace(gjson.GetBytes(body, "aspect_ratio").String())
	req.Resolution = strings.TrimSpace(gjson.GetBytes(body, "resolution").String())

	if durationResult := gjson.GetBytes(body, "duration"); durationResult.Exists() {
		if durationResult.Type != gjson.Number {
			return fmt.Errorf("invalid duration field type")
		}
		duration := int(durationResult.Int())
		if duration <= 0 {
			return fmt.Errorf("duration must be greater than 0")
		}
		req.Duration = &duration
	}

	var err error
	if req.IsGeneration() {
		req.Image, err = parseOpenAIVideoInputField(body, "image", "image_url")
		if err != nil {
			return err
		}
		req.ReferenceImages, err = parseOpenAIVideoInputArrayField(body, "reference_images")
		if err != nil {
			return err
		}
		return nil
	}

	req.Video, err = parseOpenAIVideoInputField(body, "video", "video_url")
	if err != nil {
		return err
	}
	return nil
}

func parseOpenAIVideoInputField(body []byte, objectPath, legacyPath string) (*OpenAIVideoInput, error) {
	if strings.TrimSpace(legacyPath) != "" {
		if legacy := strings.TrimSpace(gjson.GetBytes(body, legacyPath).String()); legacy != "" {
			return &OpenAIVideoInput{URL: legacy}, nil
		}
	}
	return parseOpenAIVideoInput(gjson.GetBytes(body, objectPath), objectPath)
}

func parseOpenAIVideoInputArrayField(body []byte, path string) ([]OpenAIVideoInput, error) {
	result := gjson.GetBytes(body, path)
	if !result.Exists() {
		return nil, nil
	}
	if !result.IsArray() {
		return nil, fmt.Errorf("invalid %s field type", path)
	}
	items := result.Array()
	output := make([]OpenAIVideoInput, 0, len(items))
	for _, item := range items {
		parsed, err := parseOpenAIVideoInput(item, path)
		if err != nil {
			return nil, err
		}
		if parsed == nil || parsed.isEmpty() {
			continue
		}
		output = append(output, *parsed)
	}
	return output, nil
}

func parseOpenAIVideoInput(result gjson.Result, field string) (*OpenAIVideoInput, error) {
	if !result.Exists() {
		return nil, nil
	}
	switch result.Type {
	case gjson.String:
		value := strings.TrimSpace(result.String())
		if value == "" {
			return nil, nil
		}
		return &OpenAIVideoInput{URL: value}, nil
	case gjson.JSON:
		if !result.IsObject() {
			return nil, fmt.Errorf("invalid %s field type", field)
		}
		input := &OpenAIVideoInput{
			URL:    strings.TrimSpace(result.Get("url").String()),
			Data:   strings.TrimSpace(result.Get("data").String()),
			FileID: strings.TrimSpace(result.Get("file_id").String()),
		}
		if input.isEmpty() {
			return nil, fmt.Errorf("%s requires one of url, data, or file_id", field)
		}
		return input, nil
	default:
		return nil, fmt.Errorf("invalid %s field type", field)
	}
}

func applyOpenAIVideosDefaults(req *OpenAIVideosRequest) {
	if req == nil {
		return
	}
	req.Model = strings.TrimSpace(req.Model)
	if req.Model == "" {
		req.Model = "grok-imagine-video"
	}
	req.Prompt = strings.TrimSpace(req.Prompt)
	req.AspectRatio = strings.TrimSpace(req.AspectRatio)
	req.Resolution = strings.TrimSpace(req.Resolution)
	req.RequestID = strings.TrimSpace(req.RequestID)
}

func validateOpenAIVideosRequest(req *OpenAIVideosRequest) error {
	if req == nil {
		return fmt.Errorf("videos request is required")
	}
	if req.IsPoll() {
		if strings.TrimSpace(req.RequestID) == "" {
			return fmt.Errorf("request_id is required")
		}
		return nil
	}
	if err := validateOpenAIVideosModel(req.Model); err != nil {
		return err
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	switch {
	case req.IsGeneration():
		if req.Video != nil && !req.Video.isEmpty() {
			return fmt.Errorf("videos/generations does not accept video input")
		}
		if req.Image != nil && !req.Image.isEmpty() && len(req.ReferenceImages) > 0 {
			return fmt.Errorf("reference_images cannot be combined with image input")
		}
		if len(req.ReferenceImages) > 7 {
			return fmt.Errorf("reference_images supports at most 7 items")
		}
		if len(req.ReferenceImages) > 0 && req.Duration != nil && *req.Duration > 10 {
			return fmt.Errorf("reference_images supports duration up to 10 seconds")
		}
	case req.IsEdit():
		if req.Video == nil || req.Video.isEmpty() {
			return fmt.Errorf("video is required for videos/edits")
		}
		if req.Image != nil && !req.Image.isEmpty() {
			return fmt.Errorf("videos/edits does not accept image input")
		}
		if len(req.ReferenceImages) > 0 {
			return fmt.Errorf("videos/edits does not accept reference_images")
		}
		if req.Duration != nil || req.AspectRatio != "" || req.Resolution != "" {
			return fmt.Errorf("videos/edits does not support duration, aspect_ratio, or resolution")
		}
	case req.IsExtension():
		if req.Video == nil || req.Video.isEmpty() {
			return fmt.Errorf("video is required for videos/extensions")
		}
		if req.Image != nil && !req.Image.isEmpty() {
			return fmt.Errorf("videos/extensions does not accept image input")
		}
		if len(req.ReferenceImages) > 0 {
			return fmt.Errorf("videos/extensions does not accept reference_images")
		}
		if req.AspectRatio != "" || req.Resolution != "" {
			return fmt.Errorf("videos/extensions does not support aspect_ratio or resolution")
		}
	default:
		return fmt.Errorf("unsupported videos endpoint")
	}
	return nil
}

func isOpenAIVideoGenerationModel(model string) bool {
	normalized := strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(normalized, "grok-imagine-video")
}

func validateOpenAIVideosModel(model string) error {
	model = strings.TrimSpace(model)
	if isOpenAIVideoGenerationModel(model) {
		return nil
	}
	if model == "" {
		return fmt.Errorf("videos endpoint requires a video model")
	}
	return fmt.Errorf("videos endpoint requires a video model, got %q", model)
}

func normalizeOpenAIVideosEndpointPath(method string, path string) (string, string) {
	method = strings.ToUpper(strings.TrimSpace(method))
	trimmed := strings.TrimRight(strings.TrimSpace(path), "/")
	switch method {
	case http.MethodPost:
		switch {
		case strings.HasSuffix(trimmed, "/videos/generations"):
			return openAIVideosGenerationsEndpoint, ""
		case strings.HasSuffix(trimmed, "/videos/edits"):
			return openAIVideosEditsEndpoint, ""
		case strings.HasSuffix(trimmed, "/videos/extensions"):
			return openAIVideosExtensionsEndpoint, ""
		default:
			return "", ""
		}
	case http.MethodGet:
		idx := strings.LastIndex(trimmed, "/videos/")
		if idx < 0 {
			return "", ""
		}
		requestID := strings.TrimSpace(trimmed[idx+len("/videos/"):])
		if requestID == "" || strings.Contains(requestID, "/") {
			return "", ""
		}
		return openAIVideosPollEndpoint, requestID
	default:
		return "", ""
	}
}

func defaultVideosRequestModelForAccount(account *Account, parsed *OpenAIVideosRequest, channelMappedModel string) string {
	if mapped := strings.TrimSpace(channelMappedModel); mapped != "" {
		return mapped
	}
	if parsed != nil && parsed.ExplicitModel {
		return strings.TrimSpace(parsed.Model)
	}
	if parsed != nil {
		return strings.TrimSpace(parsed.Model)
	}
	if account != nil && account.Platform == PlatformXAI {
		return "grok-imagine-video"
	}
	return "grok-imagine-video"
}

func (s *OpenAIGatewayService) ForwardVideos(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	parsed *OpenAIVideosRequest,
	channelMappedModel string,
) (*OpenAIForwardResult, error) {
	if parsed == nil {
		return nil, fmt.Errorf("videos request is required")
	}
	if parsed.IsPoll() {
		return s.forwardOpenAIVideoPoll(ctx, c, account, parsed)
	}
	return s.forwardOpenAIVideoSubmit(ctx, c, account, body, parsed, channelMappedModel)
}

func (s *OpenAIGatewayService) forwardOpenAIVideoSubmit(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	parsed *OpenAIVideosRequest,
	channelMappedModel string,
) (*OpenAIForwardResult, error) {
	startTime := time.Now()
	requestModel := defaultVideosRequestModelForAccount(account, parsed, channelMappedModel)
	if err := validateOpenAIVideosModel(requestModel); err != nil {
		return nil, err
	}
	upstreamModel := normalizeOpenAIModelForUpstream(account, resolveOpenAIForwardModel(account, requestModel, channelMappedModel))
	if err := validateOpenAIVideosModel(upstreamModel); err != nil {
		return nil, err
	}

	forwardBody, err := buildOpenAIVideosForwardBody(body, parsed, upstreamModel)
	if err != nil {
		return nil, err
	}

	upstreamCtx, releaseUpstreamCtx := detachUpstreamContext(ctx)
	defer releaseUpstreamCtx()

	token, _, err := s.GetAccessToken(upstreamCtx, account)
	if err != nil {
		return nil, err
	}
	upstreamReq, err := s.buildOpenAIVideosRequest(upstreamCtx, c, account, http.MethodPost, forwardBody, token, parsed.Endpoint, "")
	if err != nil {
		return nil, err
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	upstreamStart := time.Now()
	resp, err := s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: 0,
			UpstreamURL:        safeUpstreamURL(upstreamReq.URL.String()),
			Kind:               "request_error",
			Message:            safeErr,
		})
		return nil, fmt.Errorf("upstream request failed: %s", safeErr)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		respBody := s.readUpstreamErrorBody(resp)
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))

		upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
		upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: resp.StatusCode,
				UpstreamRequestID:  resp.Header.Get("x-request-id"),
				UpstreamURL:        safeUpstreamURL(upstreamReq.URL.String()),
				Kind:               "failover",
				Message:            upstreamMsg,
			})
			s.handleFailoverSideEffects(upstreamCtx, resp, account, upstreamModel)
			return nil, &UpstreamFailoverError{
				StatusCode:             resp.StatusCode,
				ResponseBody:           respBody,
				RetryableOnSameAccount: account.IsPoolMode() && account.IsPoolModeRetryableStatus(resp.StatusCode),
			}
		}
		return nil, s.handleErrorResponsePassthrough(upstreamCtx, resp, c, account, forwardBody)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read upstream body: %w", err)
	}
	requestID := strings.TrimSpace(gjson.GetBytes(respBody, "request_id").String())
	if requestID == "" {
		return nil, fmt.Errorf("upstream video response missing request_id")
	}

	writeOpenAIVideosUpstreamResponse(c, resp, respBody, s.responseHeaderFilter)
	return &OpenAIForwardResult{
		RequestID:       requestID,
		Model:           requestModel,
		UpstreamModel:   upstreamModel,
		Stream:          false,
		ResponseHeaders: resp.Header.Clone(),
		Duration:        time.Since(startTime),
	}, nil
}

func (s *OpenAIGatewayService) forwardOpenAIVideoPoll(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	parsed *OpenAIVideosRequest,
) (*OpenAIForwardResult, error) {
	startTime := time.Now()

	upstreamCtx, releaseUpstreamCtx := detachUpstreamContext(ctx)
	defer releaseUpstreamCtx()

	token, _, err := s.GetAccessToken(upstreamCtx, account)
	if err != nil {
		return nil, err
	}
	upstreamReq, err := s.buildOpenAIVideosRequest(upstreamCtx, c, account, http.MethodGet, nil, token, openAIVideosPollEndpoint, parsed.RequestID)
	if err != nil {
		return nil, err
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	upstreamStart := time.Now()
	resp, err := s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: 0,
			UpstreamURL:        safeUpstreamURL(upstreamReq.URL.String()),
			Kind:               "request_error",
			Message:            safeErr,
		})
		return nil, fmt.Errorf("upstream request failed: %s", safeErr)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		respBody := s.readUpstreamErrorBody(resp)
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		return nil, s.handleErrorResponsePassthrough(upstreamCtx, resp, c, account, nil)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read upstream body: %w", err)
	}
	writeOpenAIVideosUpstreamResponse(c, resp, respBody, s.responseHeaderFilter)

	return &OpenAIForwardResult{
		RequestID:       parsed.RequestID,
		Model:           strings.TrimSpace(gjson.GetBytes(respBody, "model").String()),
		UpstreamModel:   strings.TrimSpace(gjson.GetBytes(respBody, "model").String()),
		Stream:          false,
		ResponseHeaders: resp.Header.Clone(),
		Duration:        time.Since(startTime),
	}, nil
}

func buildOpenAIVideosForwardBody(body []byte, parsed *OpenAIVideosRequest, upstreamModel string) ([]byte, error) {
	if parsed == nil {
		return nil, fmt.Errorf("videos request is required")
	}
	payload := map[string]any{}
	if gjson.ValidBytes(body) {
		var raw map[string]any
		if err := json.Unmarshal(body, &raw); err == nil {
			payload = raw
		}
	}
	payload["model"] = upstreamModel
	payload["prompt"] = parsed.Prompt
	delete(payload, "image_url")
	delete(payload, "video_url")

	if parsed.IsGeneration() {
		if parsed.Duration != nil && *parsed.Duration > 0 {
			payload["duration"] = *parsed.Duration
		}
		if parsed.AspectRatio != "" {
			payload["aspect_ratio"] = parsed.AspectRatio
		}
		if parsed.Resolution != "" {
			payload["resolution"] = parsed.Resolution
		}
		if parsed.Image != nil && !parsed.Image.isEmpty() {
			payload["image"] = parsed.Image
		}
		if len(parsed.ReferenceImages) > 0 {
			payload["reference_images"] = parsed.ReferenceImages
		}
	} else {
		if parsed.Video != nil && !parsed.Video.isEmpty() {
			payload["video"] = parsed.Video
		}
		if parsed.IsExtension() && parsed.Duration != nil && *parsed.Duration > 0 {
			payload["duration"] = *parsed.Duration
		}
	}

	forwardBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return forwardBody, nil
}

func (s *OpenAIGatewayService) buildOpenAIVideosRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	method string,
	body []byte,
	token string,
	endpoint string,
	requestID string,
) (*http.Request, error) {
	targetURL := buildOpenAIVideosURL("", endpoint, requestID)
	baseURL := account.GetOpenAIBaseURL()
	if baseURL != "" {
		validatedURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		targetURL = buildOpenAIVideosURL(validatedURL, endpoint, requestID)
	}

	var reader *bytes.Reader
	if len(body) == 0 {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, targetURL, reader)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	req.Header.Set("Authorization", "Bearer "+token)
	for key, values := range c.Request.Header {
		if !openaiPassthroughAllowedHeaders[strings.ToLower(key)] {
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.Header.Set("Accept", "application/json")
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	if customUA := account.GetOpenAIUserAgent(); customUA != "" {
		req.Header.Set("User-Agent", customUA)
	}
	return req, nil
}

func buildOpenAIVideosURL(base string, endpoint string, requestID string) string {
	if strings.TrimSpace(base) == "" {
		switch endpoint {
		case openAIVideosGenerationsEndpoint:
			return openAIVideosGenerationsURL
		case openAIVideosEditsEndpoint:
			return openAIVideosEditsURL
		case openAIVideosExtensionsEndpoint:
			return openAIVideosExtensionsURL
		case openAIVideosPollEndpoint:
			return openAIVideosPollURLPrefix + strings.TrimSpace(requestID)
		default:
			return openAIVideosGenerationsURL
		}
	}
	if endpoint == openAIVideosPollEndpoint {
		return buildOpenAIEndpointURL(base, "/v1/videos/"+strings.TrimSpace(requestID))
	}
	return buildOpenAIEndpointURL(base, endpoint)
}

func writeOpenAIVideosUpstreamResponse(c *gin.Context, resp *http.Response, body []byte, filter *responseheaders.CompiledHeaderFilter) {
	if c == nil || resp == nil || c.Writer.Written() {
		return
	}
	if resp.Header != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, filter)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		c.Writer.Header().Set("Content-Type", ct)
	} else {
		c.Writer.Header().Set("Content-Type", "application/json")
	}
	c.Writer.WriteHeader(resp.StatusCode)
	_, _ = c.Writer.Write(body)
}

func (s *OpenAIGatewayService) BindVideoRequestAccount(ctx context.Context, groupID int64, requestID string, accountID int64) {
	if s == nil || groupID <= 0 || accountID <= 0 || strings.TrimSpace(requestID) == "" {
		return
	}
	store := s.getOpenAIWSStateStore()
	if store == nil {
		return
	}
	if err := store.BindVideoRequestAccount(ctx, groupID, requestID, accountID, s.openAIWSResponseStickyTTL()); err != nil {
		logger.L().With(
			zap.Int64("group_id", groupID),
			zap.Int64("account_id", accountID),
			zap.String("video_request_id", requestID),
		).Warn("openai_videos.bind_request_account_failed", zap.Error(err))
	}
}

func (s *OpenAIGatewayService) GetBoundVideoRequestAccount(ctx context.Context, groupID int64, requestID string) (*Account, error) {
	if s == nil || s.accountRepo == nil {
		return nil, fmt.Errorf("account repository is not configured")
	}
	store := s.getOpenAIWSStateStore()
	if store == nil {
		return nil, fmt.Errorf("video request state store is not configured")
	}
	accountID, err := store.GetVideoRequestAccount(ctx, groupID, requestID)
	if err != nil {
		return nil, fmt.Errorf("lookup video request account: %w", err)
	}
	if accountID <= 0 {
		return nil, fmt.Errorf("video request account binding not found")
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		_ = store.DeleteVideoRequestAccount(ctx, groupID, requestID)
		return nil, err
	}
	if account == nil {
		_ = store.DeleteVideoRequestAccount(ctx, groupID, requestID)
		return nil, fmt.Errorf("account %d not found", accountID)
	}
	if !accountBelongsToOpenAIGroup(account, groupID) {
		_ = store.DeleteVideoRequestAccount(ctx, groupID, requestID)
		return nil, fmt.Errorf("account %d no longer belongs to group %d", account.ID, groupID)
	}
	return account, nil
}

func accountBelongsToOpenAIGroup(account *Account, groupID int64) bool {
	if account == nil || groupID <= 0 {
		return false
	}
	for _, id := range account.GroupIDs {
		if id == groupID {
			return true
		}
	}
	return false
}
