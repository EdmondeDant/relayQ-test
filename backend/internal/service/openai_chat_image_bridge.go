package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// ForwardChatCompletionsImageBridge lets OpenAI-compatible chat clients call an
// image-only model (for example grok-imagine-image) through /v1/chat/completions.
// It extracts the prompt before this call and writes a standard Chat Completions
// response whose content is a markdown image URL/data URL.
func (s *OpenAIGatewayService) ForwardChatCompletionsImageBridge(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	prompt string,
	model string,
	stream bool,
	options map[string]any,
) (*OpenAIForwardResult, error) {
	startTime := time.Now()
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, fmt.Errorf("image prompt is required")
	}
	requestModel := strings.TrimSpace(model)
	if requestModel == "" {
		requestModel = "grok-imagine-image-quality"
	}
	n := 1
	if v, ok := options["n"].(int); ok && v > 0 {
		n = v
	}
	parsed := &OpenAIImagesRequest{
		Endpoint:           openAIImagesGenerationsEndpoint,
		Model:              requestModel,
		ExplicitModel:      true,
		Prompt:             prompt,
		N:                  n,
		ResponseFormat:     "url",
		RequiredCapability: OpenAIImagesCapabilityNative,
	}
	if aspectRatio, ok := options["aspect_ratio"].(string); ok {
		parsed.AspectRatio = aspectRatio
	}
	if resolution, ok := options["resolution"].(string); ok {
		parsed.Resolution = resolution
	}
	if quality, ok := options["quality"].(string); ok {
		parsed.Quality = quality
	}
	applyOpenAIImagesDefaults(parsed)
	parsed.SizeTier = normalizeOpenAIImageSizeTier(parsed.Size)

	upstreamModel := account.GetMappedModel(requestModel)
	if err := validateOpenAIImagesModel(upstreamModel); err != nil {
		return nil, err
	}
	payload := map[string]any{
		"model":           upstreamModel,
		"prompt":          prompt,
		"n":               n,
		"response_format": "b64_json",
	}
	for _, key := range []string{"aspect_ratio", "resolution", "quality", "user"} {
		if value, ok := options[key]; ok && value != nil {
			payload[key] = value
		}
	}
	imagesBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	upstreamCtx, releaseUpstreamCtx := detachStreamUpstreamContext(ctx, stream)
	defer releaseUpstreamCtx()

	token, _, err := s.GetAccessToken(upstreamCtx, account)
	if err != nil {
		return nil, err
	}
	upstreamReq, err := s.buildOpenAIImagesRequest(upstreamCtx, c, account, imagesBody, "application/json", token, openAIImagesGenerationsEndpoint)
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
		return nil, fmt.Errorf("upstream request failed: %s", sanitizeUpstreamErrorMessage(err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return s.handleErrorResponse(upstreamCtx, resp, c, account, imagesBody)
	}
	body = convertOpenAIImagesB64JSONToDataURL(body)
	var imageURLs []string
	for _, item := range gjson.GetBytes(body, "data").Array() {
		imageURL := strings.TrimSpace(item.Get("url").String())
		if imageURL == "" {
			if b64 := cleanOpenAIImageBase64ForDataURL(item.Get("b64_json").String()); b64 != "" {
				mimeType := strings.TrimSpace(item.Get("mime_type").String())
				if mimeType == "" {
					mimeType = detectOpenAIImageBase64MimeType(b64)
				}
				imageURL = fmt.Sprintf("data:%s;base64,%s", mimeType, b64)
			}
		}
		if imageURL != "" {
			imageURLs = append(imageURLs, imageURL)
		}
	}
	if len(imageURLs) == 0 {
		return nil, fmt.Errorf("No images returned from upstream")
	}
	var contentParts []string
	for _, imageURL := range imageURLs {
		contentParts = append(contentParts, fmt.Sprintf("![generated image](%s)", imageURL))
	}
	content := strings.Join(contentParts, "\n")
	if stream {
		writeChatCompletionsImageBridgeStream(c, requestModel, content)
	} else {
		writeChatCompletionsImageBridgeJSON(c, requestModel, content)
	}
	return &OpenAIForwardResult{
		RequestID:       resp.Header.Get("x-request-id"),
		Model:           requestModel,
		UpstreamModel:   upstreamModel,
		Stream:          stream,
		ResponseHeaders: resp.Header.Clone(),
		Duration:        time.Since(startTime),
		ImageCount:      len(imageURLs),
		ImageSize:       parsed.SizeTier,
		ImageInputSize:  parsed.Size,
	}, nil
}

func writeChatCompletionsImageBridgeJSON(c *gin.Context, model string, content string) {
	now := time.Now().Unix()
	c.JSON(http.StatusOK, gin.H{
		"id":      fmt.Sprintf("chatcmpl-image-%d", now),
		"object":  "chat.completion",
		"created": now,
		"model":   model,
		"choices": []gin.H{{
			"index": 0,
			"message": gin.H{
				"role":    "assistant",
				"content": content,
			},
			"finish_reason": "stop",
		}},
		"usage": gin.H{"prompt_tokens": 0, "completion_tokens": 0, "total_tokens": 0},
	})
}

func writeChatCompletionsImageBridgeStream(c *gin.Context, model string, content string) {
	now := time.Now().Unix()
	id := fmt.Sprintf("chatcmpl-image-%d", now)
	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	write := func(payload any) {
		b, _ := json.Marshal(payload)
		_, _ = c.Writer.Write([]byte("data: "))
		_, _ = c.Writer.Write(b)
		_, _ = c.Writer.Write([]byte("\n\n"))
		if f, ok := c.Writer.(http.Flusher); ok {
			f.Flush()
		}
	}
	write(gin.H{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": now,
		"model":   model,
		"choices": []gin.H{{"index": 0, "delta": gin.H{"role": "assistant", "content": content}, "finish_reason": nil}},
	})
	write(gin.H{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": now,
		"model":   model,
		"choices": []gin.H{{"index": 0, "delta": gin.H{}, "finish_reason": "stop"}},
	})
	_, _ = c.Writer.Write([]byte("data: [DONE]\n\n"))
	if f, ok := c.Writer.(http.Flusher); ok {
		f.Flush()
	}
}
