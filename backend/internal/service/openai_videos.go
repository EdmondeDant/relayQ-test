package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"path"
	"strconv"
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
	if strings.EqualFold(strings.TrimSpace(model), "minimax-h3") {
		return true
	}
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

// NormalizeVideoGenerationBodyForHandler accepts either JSON or Infinite Canvas
// multipart/form-data and normalizes it into the OpenAI/xAI video JSON contract.
func NormalizeVideoGenerationBodyForHandler(contentType string, body []byte) ([]byte, string, error) {
	normalized, err := CanvasVideoMultipartToJSON(contentType, body)
	if err != nil {
		return nil, "", err
	}
	return normalizeXAIVideoGenerationBody(normalized)
}

// CanvasVideoMultipartToJSON converts Infinite Canvas video form uploads into the
// shared JSON body used by OpenAI/xAI/Leonardo video handlers.
// If the request is not multipart, the original body is returned unchanged.
func CanvasVideoMultipartToJSON(contentType string, body []byte) ([]byte, error) {
	mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil || !strings.EqualFold(mediaType, "multipart/form-data") {
		return body, nil
	}
	boundary := strings.TrimSpace(params["boundary"])
	if boundary == "" {
		return nil, fmt.Errorf("multipart boundary is required")
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("request body is empty")
	}

	values := map[string]string{}
	var referenceDataURLs []string
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	for {
		part, partErr := reader.NextPart()
		if partErr == io.EOF {
			break
		}
		if partErr != nil {
			return nil, fmt.Errorf("failed to parse multipart body: %w", partErr)
		}
		name := strings.TrimSpace(part.FormName())
		filename := strings.TrimSpace(part.FileName())
		if filename != "" || isCanvasVideoReferenceField(name) {
			data, readErr := io.ReadAll(io.LimitReader(part, 16<<20))
			_ = part.Close()
			if readErr != nil {
				return nil, fmt.Errorf("failed to read multipart file: %w", readErr)
			}
			if len(bytes.TrimSpace(data)) == 0 {
				continue
			}
			if looksLikeTextReference(data) {
				if ref := strings.TrimSpace(string(data)); ref != "" {
					referenceDataURLs = append(referenceDataURLs, ref)
				}
				continue
			}
			mimeType := strings.TrimSpace(part.Header.Get("Content-Type"))
			if mimeType == "" || mimeType == "application/octet-stream" {
				mimeType = inferImageMIMEFromName(filename)
			}
			if mimeType == "" {
				mimeType = "application/octet-stream"
			}
			referenceDataURLs = append(referenceDataURLs, "data:"+mimeType+";base64,"+base64.StdEncoding.EncodeToString(data))
			continue
		}
		value, readErr := io.ReadAll(io.LimitReader(part, 1<<20))
		_ = part.Close()
		if readErr != nil {
			return nil, fmt.Errorf("failed to read multipart field: %w", readErr)
		}
		if name == "" {
			continue
		}
		values[name] = strings.TrimSpace(string(value))
	}

	payload := map[string]any{}
	if model := firstNonEmptyString(values["model"], values["video_model"]); model != "" {
		payload["model"] = model
	}
	if prompt := firstNonEmptyString(values["prompt"], values["input"]); prompt != "" {
		payload["prompt"] = prompt
	}
	if seconds := firstNonEmptyString(values["seconds"], values["duration"]); seconds != "" {
		if n, err := strconv.Atoi(seconds); err == nil {
			payload["seconds"] = n
		} else if f, err := strconv.ParseFloat(seconds, 64); err == nil {
			payload["seconds"] = f
		} else {
			payload["seconds"] = seconds
		}
	}
	if size := strings.TrimSpace(values["size"]); size != "" {
		payload["size"] = size
	}
	if resolution := firstNonEmptyString(values["resolution"], values["resolution_name"], values["vquality"]); resolution != "" {
		payload["resolution"] = resolution
	}
	if aspectRatio := firstNonEmptyString(values["aspect_ratio"], values["aspectRatio"], values["ratio"]); aspectRatio != "" {
		payload["aspect_ratio"] = aspectRatio
	}
	if mode := firstNonEmptyString(values["mode"], values["preset"]); mode != "" && !strings.EqualFold(mode, "normal") {
		payload["mode"] = mode
	}
	if len(referenceDataURLs) == 1 {
		payload["input_reference"] = map[string]string{"image_url": referenceDataURLs[0]}
	} else if len(referenceDataURLs) > 1 {
		refs := make([]map[string]string, 0, len(referenceDataURLs))
		for _, url := range referenceDataURLs {
			refs = append(refs, map[string]string{"url": url})
		}
		payload["reference_images"] = refs
		payload["mode"] = "reference-to-video"
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("multipart body is empty")
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode multipart body: %w", err)
	}
	return encoded, nil
}

func isCanvasVideoReferenceField(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "input_reference", "input_reference[]", "image", "image[]", "reference_image", "reference_images", "reference_images[]":
		return true
	default:
		return false
	}
}

func looksLikeTextReference(data []byte) bool {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return false
	}
	return strings.HasPrefix(trimmed, "http://") ||
		strings.HasPrefix(trimmed, "https://") ||
		strings.HasPrefix(trimmed, "data:")
}

func inferImageMIMEFromName(filename string) string {
	switch strings.ToLower(path.Ext(strings.TrimSpace(filename))) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return ""
	}
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
	if aspectRatio := firstNonEmptyString(
		gjson.GetBytes(out, "aspect_ratio").String(),
		gjson.GetBytes(out, "aspectRatio").String(),
		gjson.GetBytes(out, "providerOptions.xai.aspectRatio").String(),
		gjson.GetBytes(out, "provider_options.xai.aspectRatio").String(),
	); strings.TrimSpace(aspectRatio) != "" {
		out, _ = sjson.SetBytes(out, "aspect_ratio", strings.TrimSpace(aspectRatio))
	}
	if resolution := firstNonEmptyString(
		gjson.GetBytes(out, "resolution").String(),
		gjson.GetBytes(out, "providerOptions.xai.resolution").String(),
		gjson.GetBytes(out, "provider_options.xai.resolution").String(),
	); strings.TrimSpace(resolution) != "" {
		out, _ = sjson.SetBytes(out, "resolution", normalizeXAIVideoResolution(resolution))
	}
	if size := strings.TrimSpace(gjson.GetBytes(out, "size").String()); size != "" {
		if !gjson.GetBytes(out, "aspect_ratio").Exists() {
			if ratio := inferAspectRatioFromSize(size); ratio != "" {
				out, _ = sjson.SetBytes(out, "aspect_ratio", ratio)
			}
		}
		if !gjson.GetBytes(out, "resolution").Exists() {
			if resolution := inferVideoResolutionFromSize(size); resolution != "" {
				out, _ = sjson.SetBytes(out, "resolution", resolution)
			}
		}
		out, _ = sjson.DeleteBytes(out, "size")
	}

	mode := strings.ToLower(strings.TrimSpace(firstNonEmptyString(
		gjson.GetBytes(out, "mode").String(),
		gjson.GetBytes(out, "providerOptions.xai.mode").String(),
		gjson.GetBytes(out, "provider_options.xai.mode").String(),
	)))

	if ref := strings.TrimSpace(gjson.GetBytes(out, "input_reference.image_url").String()); ref != "" {
		if mode == "reference-to-video" {
			out, _ = sjson.SetBytes(out, "reference_images", []map[string]string{{"url": ref}})
		} else if !gjson.GetBytes(out, "image").Exists() && !gjson.GetBytes(out, "reference_images").Exists() {
			out, _ = sjson.SetBytes(out, "image.url", ref)
		}
		out, _ = sjson.DeleteBytes(out, "input_reference")
	}
	if imageURL := strings.TrimSpace(gjson.GetBytes(out, "image_url").String()); imageURL != "" && !gjson.GetBytes(out, "image").Exists() {
		out, _ = sjson.SetBytes(out, "image.url", imageURL)
		out, _ = sjson.DeleteBytes(out, "image_url")
	}

	if images := gjson.GetBytes(out, "images"); images.Exists() && !gjson.GetBytes(out, "reference_images").Exists() {
		out, _ = sjson.SetRawBytes(out, "reference_images", []byte(images.Raw))
		out, _ = sjson.DeleteBytes(out, "images")
	}
	if refs := gjson.GetBytes(out, "providerOptions.xai.referenceImageUrls"); refs.IsArray() && !gjson.GetBytes(out, "reference_images").Exists() {
		out, _ = sjson.SetRawBytes(out, "reference_images", []byte(refs.Raw))
	}
	if refs := gjson.GetBytes(out, "provider_options.xai.referenceImageUrls"); refs.IsArray() && !gjson.GetBytes(out, "reference_images").Exists() {
		out, _ = sjson.SetRawBytes(out, "reference_images", []byte(refs.Raw))
	}
	out = normalizeXAIVideoReferenceImages(out)
	out, _ = sjson.DeleteBytes(out, "aspectRatio")
	out, _ = sjson.DeleteBytes(out, "providerOptions")
	out, _ = sjson.DeleteBytes(out, "provider_options")
	if isMiniMaxH3Model(requestModel) {
		out, err := normalizeMiniMaxH3VideoBody(out, mode)
		if err != nil {
			return nil, "", err
		}
		return out, requestModel, nil
	}
	return out, requestModel, nil
}

func isMiniMaxH3Model(model string) bool {
	return strings.EqualFold(strings.TrimSpace(model), "minimax-h3") ||
		strings.EqualFold(strings.TrimSpace(model), "MiniMax-H3")
}

// normalizeMiniMaxH3VideoBody maps Canvas/OpenAI/xAI image fields into MiniMax H3
// multimodal content[] while keeping gateway-compatible image/reference_images.
// Official H3: text-only | first/last frame | reference_image(s). Frames and
// reference roles are mutually exclusive.
func normalizeMiniMaxH3VideoBody(body []byte, mode string) ([]byte, error) {
	prompt := strings.TrimSpace(firstNonEmptyString(
		gjson.GetBytes(body, "prompt").String(),
		gjson.GetBytes(body, "input").String(),
		extractMiniMaxTextFromContent(body),
	))
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	firstFrame := ""
	lastFrame := ""
	referenceURLs := make([]string, 0, 8)
	seen := map[string]struct{}{}
	addRef := func(url string) {
		url = strings.TrimSpace(url)
		if url == "" {
			return
		}
		if _, ok := seen[url]; ok {
			return
		}
		seen[url] = struct{}{}
		referenceURLs = append(referenceURLs, url)
	}

	// Prefer an already-provided MiniMax content[] payload when present.
	if content := gjson.GetBytes(body, "content"); content.IsArray() {
		for _, item := range content.Array() {
			switch strings.ToLower(strings.TrimSpace(item.Get("type").String())) {
			case "image_url":
				url := strings.TrimSpace(firstNonEmptyString(
					item.Get("image_url.url").String(),
					item.Get("image_url").String(),
					item.Get("url").String(),
				))
				role := strings.ToLower(strings.TrimSpace(item.Get("role").String()))
				switch role {
				case "first_frame", "":
					if firstFrame == "" {
						firstFrame = url
					} else {
						addRef(url)
					}
				case "last_frame":
					lastFrame = url
				case "reference_image":
					addRef(url)
				default:
					addRef(url)
				}
			}
		}
	}

	if firstFrame == "" {
		firstFrame = strings.TrimSpace(firstNonEmptyString(
			gjson.GetBytes(body, "image.url").String(),
			gjson.GetBytes(body, "image_url").String(),
			gjson.GetBytes(body, "input_reference.image_url").String(),
		))
	}
	if refs := gjson.GetBytes(body, "reference_images"); refs.IsArray() {
		for _, item := range refs.Array() {
			addRef(firstNonEmptyString(item.Get("url").String(), item.Get("image_url").String(), item.String()))
		}
	}
	if mode == "reference-to-video" && firstFrame != "" {
		addRef(firstFrame)
		firstFrame = ""
		lastFrame = ""
	}
	// Multiple images from Canvas mean reference-to-video, not first+last frame.
	if len(referenceURLs) > 0 && firstFrame != "" && lastFrame == "" {
		addRef(firstFrame)
		firstFrame = ""
	}
	if firstFrame != "" && lastFrame != "" && len(referenceURLs) > 0 {
		return nil, fmt.Errorf("minimax-h3 cannot mix first/last frame with reference images")
	}
	if len(referenceURLs) > 9 {
		referenceURLs = referenceURLs[:9]
	}
	// Official H3 multi-reference requires prompt-side Image N binding.
	// Without it the gateway often ignores identity and only weakly picks props.
	if len(referenceURLs) > 0 {
		prompt = ensureMiniMaxH3ReferencePrompt(prompt, len(referenceURLs))
	}

	content := make([]map[string]any, 0, 2+len(referenceURLs))
	content = append(content, map[string]any{"type": "text", "text": prompt})
	if len(referenceURLs) > 0 {
		for _, url := range referenceURLs {
			content = append(content, map[string]any{
				"type":      "image_url",
				"role":      "reference_image",
				"image_url": map[string]string{"url": url},
			})
		}
	} else {
		if firstFrame != "" {
			content = append(content, map[string]any{
				"type":      "image_url",
				"role":      "first_frame",
				"image_url": map[string]string{"url": firstFrame},
			})
		}
		if lastFrame != "" {
			content = append(content, map[string]any{
				"type":      "image_url",
				"role":      "last_frame",
				"image_url": map[string]string{"url": lastFrame},
			})
		}
	}

	// Official H3 only needs content[]. Do NOT also attach image/reference_images
// with the same base64 payload — that doubles request size and can stall the
// private gateway for many minutes before it even returns a task id.
	out := body
	out, _ = sjson.SetBytes(out, "model", "minimax-h3")
	out, _ = sjson.SetBytes(out, "prompt", prompt)
	out, _ = sjson.SetBytes(out, "content", content)

	duration := int(gjson.GetBytes(out, "duration").Int())
	if duration <= 0 {
		duration = 5
	}
	if duration < 4 {
		duration = 4
	}
	if duration > 15 {
		duration = 15
	}
	out, _ = sjson.SetBytes(out, "duration", duration)

	// Private gateway currently accepts 480p for this model; keep official-ish
	// values when callers already send 768P/2K.
	resolution := normalizeMiniMaxH3Resolution(gjson.GetBytes(out, "resolution").String())
	out, _ = sjson.SetBytes(out, "resolution", resolution)

	ratio := firstNonEmptyString(
		gjson.GetBytes(out, "ratio").String(),
		gjson.GetBytes(out, "aspect_ratio").String(),
	)
	ratio = strings.TrimSpace(ratio)
	if len(referenceURLs) > 0 {
		if ratio == "" {
			ratio = "adaptive"
		}
		out, _ = sjson.SetBytes(out, "ratio", ratio)
		if ratio != "adaptive" {
			out, _ = sjson.SetBytes(out, "aspect_ratio", ratio)
		} else {
			out, _ = sjson.DeleteBytes(out, "aspect_ratio")
		}
	} else if firstFrame != "" || lastFrame != "" {
		// Official H3 image-to-video always uses adaptive ratio from the frame.
		out, _ = sjson.SetBytes(out, "ratio", "adaptive")
		out, _ = sjson.DeleteBytes(out, "aspect_ratio")
	} else {
		if ratio == "" || ratio == "adaptive" {
			ratio = "16:9"
		}
		out, _ = sjson.SetBytes(out, "ratio", ratio)
		out, _ = sjson.SetBytes(out, "aspect_ratio", ratio)
	}

	// Strip all legacy/alias image carriers so base64 is only in content[].
	for _, key := range []string{
		"image", "image_url", "images", "reference_images", "input_reference",
		"mode", "seconds", "size",
	} {
		out, _ = sjson.DeleteBytes(out, key)
	}
	return out, nil
}

func extractMiniMaxTextFromContent(body []byte) string {
	content := gjson.GetBytes(body, "content")
	if !content.IsArray() {
		return ""
	}
	for _, item := range content.Array() {
		if strings.EqualFold(strings.TrimSpace(item.Get("type").String()), "text") {
			if text := strings.TrimSpace(item.Get("text").String()); text != "" {
				return text
			}
		}
	}
	return ""
}

func ensureMiniMaxH3ReferencePrompt(prompt string, imageCount int) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" || imageCount <= 0 {
		return prompt
	}
	lower := strings.ToLower(prompt)
	if strings.Contains(lower, "image 1") || strings.Contains(lower, "image1") ||
		strings.Contains(prompt, "图片1") || strings.Contains(prompt, "图1") {
		return prompt
	}
	labels := make([]string, 0, imageCount)
	for i := 1; i <= imageCount; i++ {
		labels = append(labels, fmt.Sprintf("Image %d", i))
	}
	binding := strings.Join(labels, " and ")
	// Keep both English Image N markers (official) and a short Chinese instruction
	// so Canvas natural-language prompts still bind to uploaded references.
	return fmt.Sprintf(
		"Using %s as visual references. Keep the person's face, hairstyle, clothing and body identity consistent with the human reference image, and keep the product appearance consistent with the product reference image. %s",
		binding,
		prompt,
	)
}

func normalizeMiniMaxH3Resolution(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	switch v {
	case "", "480", "480p", "sd":
		// Current private gateway path is fixed around 480p for this model.
		return "480p"
	case "768", "768p":
		return "768P"
	case "2k", "1080", "1080p", "fhd", "fullhd":
		return "2K"
	case "720", "720p", "hd":
		// Gateway currently maps non-official mid tiers down to 480p.
		return "480p"
	default:
		if strings.EqualFold(strings.TrimSpace(value), "768P") {
			return "768P"
		}
		if strings.EqualFold(strings.TrimSpace(value), "2K") {
			return "2K"
		}
		return "480p"
	}
}

func normalizeXAIVideoResolution(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	switch v {
	case "480", "480p", "sd":
		return "480p"
	case "720", "720p", "hd":
		return "720p"
	case "1080", "1080p", "fhd", "fullhd":
		return "1080p"
	default:
		return strings.TrimSpace(value)
	}
}

func inferVideoResolutionFromSize(size string) string {
	width, height := parseDimensionPair(size)
	if width <= 0 || height <= 0 {
		return ""
	}
	short := width
	if height < short {
		short = height
	}
	if short >= 1000 {
		return "1080p"
	}
	if short >= 700 {
		return "720p"
	}
	return "480p"
}

func inferAspectRatioFromSize(size string) string {
	width, height := parseDimensionPair(size)
	if width <= 0 || height <= 0 {
		return ""
	}
	g := gcdInt(width, height)
	if g <= 0 {
		return ""
	}
	return fmt.Sprintf("%d:%d", width/g, height/g)
}

func parseDimensionPair(size string) (int, int) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(size)), "x")
	if len(parts) != 2 {
		return 0, 0
	}
	w, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	return w, h
}

func gcdInt(a, b int) int {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func normalizeXAIVideoReferenceImages(body []byte) []byte {
	refs := gjson.GetBytes(body, "reference_images")
	if !refs.IsArray() {
		return body
	}
	normalized := make([]map[string]string, 0, len(refs.Array()))
	for _, item := range refs.Array() {
		url := strings.TrimSpace(firstNonEmptyString(item.Get("url").String(), item.Get("image_url").String(), item.String()))
		if url != "" {
			normalized = append(normalized, map[string]string{"url": url})
		}
	}
	body, _ = sjson.SetBytes(body, "reference_images", normalized)
	return body
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
	return buildOpenAIEndpointURL(validatedURL, suffix), nil
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
	if account == nil || (account.Platform != PlatformOpenAI && !isXAIOAuthAccount(account)) {
		return fmt.Errorf("account is not an OpenAI-compatible video account")
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
	if account == nil || (account.Platform != PlatformOpenAI && !isXAIOAuthAccount(account)) {
		return fmt.Errorf("account is not an OpenAI-compatible video account")
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
	if account == nil || (account.Platform != PlatformOpenAI && !isXAIOAuthAccount(account)) {
		return fmt.Errorf("account is not an OpenAI-compatible video account")
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return fmt.Errorf("request_id is required")
	}
	targetURL, err := s.buildXAIVideoURL(account, "/v1/videos/"+requestID)
	if err != nil {
		return err
	}
	if account.Platform == PlatformOpenAI {
		targetURL += "/content"
		return s.forwardXAIVideoRequest(ctx, c, account, http.MethodGet, targetURL, nil)
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
	if account.Platform == PlatformOpenAI {
		token = account.GetOpenAIApiKey()
	}
	if strings.TrimSpace(token) == "" && account.Platform == PlatformXAI {
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
	if account.Platform == PlatformXAI && (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) && isXAIBadCredentials(respBody) {
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
		gjson.GetBytes(body, "job_id").String(),
		gjson.GetBytes(body, "id").String(),
		gjson.GetBytes(body, "data.request_id").String(),
		gjson.GetBytes(body, "data.job_id").String(),
		gjson.GetBytes(body, "data.id").String(),
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
	if IsCanvasAPIKey(apiKey) && s.canvasResourceRoutes != nil {
		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := s.canvasResourceRoutes.Upsert(persistCtx, &CanvasResourceRoute{
			APIKeyID: apiKey.ID, UserID: apiKey.UserID, ResourceID: requestID,
			GroupID: *apiKey.GroupID, Platform: account.Platform, EndpointFamily: "videos",
			ExpiresAt: time.Now().Add(xaiVideoRequestAccountTTL),
		}); err != nil {
			slog.Error("failed to persist canvas video route", "request_id", requestID, "api_key_id", apiKey.ID, "error", err)
		}
	}
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
	if err != nil || account == nil || (account.Platform != PlatformXAI && account.Platform != PlatformOpenAI) {
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
