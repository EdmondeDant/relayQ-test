package service

import (
	"encoding/json"
	"fmt"
	"strings"
)

type playgroundJobMediaRef struct {
	URL      string `json:"url"`
	Role     string `json:"role,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type playgroundJobMedia struct {
	Images         []playgroundJobMediaRef `json:"images"`
	Mask           *playgroundJobMediaRef  `json:"mask,omitempty"`
	Audio          *playgroundJobMediaRef  `json:"audio,omitempty"`
	ReferenceAudio *playgroundJobMediaRef  `json:"reference_audio,omitempty"`
	InputReference *playgroundJobMediaRef  `json:"input_reference,omitempty"`
}

type playgroundJobBatchItem struct {
	Prompt   string                 `json:"prompt"`
	Media    playgroundJobMedia     `json:"media"`
	Metadata map[string]any         `json:"metadata"`
}

type playgroundJobBatch struct {
	Items []playgroundJobBatchItem `json:"items"`
}

type playgroundJobPayload struct {
	Title              string                 `json:"title"`
	AssetKind          string                 `json:"asset_kind"`
	Prompt             string                 `json:"prompt"`
	Text               string                 `json:"text"`
	Messages           []map[string]any       `json:"messages"`
	Size               string                 `json:"size"`
	Quality            string                 `json:"quality"`
	Style              string                 `json:"style"`
	Background         string                 `json:"background"`
	SourceLanguage     string                 `json:"source_language"`
	TargetLanguage     string                 `json:"target_language"`
	Filename           string                 `json:"filename"`
	Language           string                 `json:"language"`
	AsrLanguage        string                 `json:"asr_language"`
	OutputMode         string                 `json:"output_mode"`
	Mode               string                 `json:"mode"`
	VoicePreset        string                 `json:"voice_preset"`
	VoiceDescription   string                 `json:"voice_description"`
	Persona            string                 `json:"persona"`
	Duration           int                    `json:"duration"`
	AspectRatio        string                 `json:"aspect_ratio"`
	Resolution         string                 `json:"resolution"`
	WatermarkText      string                 `json:"watermark_text"`
	WatermarkPosition  string                 `json:"watermark_position"`
	WatermarkStyle     string                 `json:"watermark_style"`
	Media              playgroundJobMedia     `json:"media"`
	Batch              *playgroundJobBatch    `json:"batch"`
	Metadata           map[string]any         `json:"metadata"`
}

func parsePlaygroundJobPayload(kind string, raw json.RawMessage) (*playgroundJobPayload, error) {
	var payload playgroundJobPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("%w: invalid job payload", ErrPlaygroundInvalidInput)
	}
	payload.normalize(kind)
	return &payload, nil
}

func (p *playgroundJobPayload) normalize(kind string) {
	p.Title = strings.TrimSpace(p.Title)
	p.AssetKind = strings.TrimSpace(p.AssetKind)
	p.Prompt = strings.TrimSpace(p.Prompt)
	p.Text = strings.TrimSpace(p.Text)
	p.Size = strings.TrimSpace(p.Size)
	p.Quality = strings.TrimSpace(p.Quality)
	p.Style = strings.TrimSpace(p.Style)
	p.Background = strings.TrimSpace(p.Background)
	p.SourceLanguage = strings.TrimSpace(p.SourceLanguage)
	p.TargetLanguage = strings.TrimSpace(p.TargetLanguage)
	p.Filename = strings.TrimSpace(p.Filename)
	p.Language = strings.TrimSpace(p.Language)
	p.AsrLanguage = strings.TrimSpace(p.AsrLanguage)
	p.OutputMode = strings.TrimSpace(p.OutputMode)
	p.Mode = strings.TrimSpace(p.Mode)
	p.VoicePreset = strings.TrimSpace(p.VoicePreset)
	p.VoiceDescription = strings.TrimSpace(p.VoiceDescription)
	p.Persona = strings.TrimSpace(p.Persona)
	p.AspectRatio = strings.TrimSpace(p.AspectRatio)
	p.Resolution = strings.TrimSpace(p.Resolution)
	p.WatermarkText = strings.TrimSpace(p.WatermarkText)
	p.WatermarkPosition = strings.TrimSpace(p.WatermarkPosition)
	p.WatermarkStyle = strings.TrimSpace(p.WatermarkStyle)
	p.Media.normalize()
	if p.Batch != nil {
		for index := range p.Batch.Items {
			p.Batch.Items[index].Prompt = strings.TrimSpace(p.Batch.Items[index].Prompt)
			p.Batch.Items[index].Media.normalize()
			if p.Batch.Items[index].Metadata == nil {
				p.Batch.Items[index].Metadata = map[string]any{}
			}
		}
	}
	if p.Metadata == nil {
		p.Metadata = map[string]any{}
	}
	if p.AssetKind == "" {
		p.AssetKind = defaultPlaygroundAssetKind(kind)
	}
	if p.Title == "" {
		p.Title = defaultPlaygroundAssetTitle(kind)
	}
}

func (m *playgroundJobMedia) normalize() {
	for index := range m.Images {
		m.Images[index].URL = strings.TrimSpace(m.Images[index].URL)
		m.Images[index].Role = strings.TrimSpace(m.Images[index].Role)
		m.Images[index].Filename = strings.TrimSpace(m.Images[index].Filename)
	}
	if m.Mask != nil {
		m.Mask.URL = strings.TrimSpace(m.Mask.URL)
		m.Mask.Role = strings.TrimSpace(m.Mask.Role)
		m.Mask.Filename = strings.TrimSpace(m.Mask.Filename)
	}
	if m.Audio != nil {
		m.Audio.URL = strings.TrimSpace(m.Audio.URL)
		m.Audio.Role = strings.TrimSpace(m.Audio.Role)
		m.Audio.Filename = strings.TrimSpace(m.Audio.Filename)
	}
	if m.ReferenceAudio != nil {
		m.ReferenceAudio.URL = strings.TrimSpace(m.ReferenceAudio.URL)
		m.ReferenceAudio.Role = strings.TrimSpace(m.ReferenceAudio.Role)
		m.ReferenceAudio.Filename = strings.TrimSpace(m.ReferenceAudio.Filename)
	}
	if m.InputReference != nil {
		m.InputReference.URL = strings.TrimSpace(m.InputReference.URL)
		m.InputReference.Role = strings.TrimSpace(m.InputReference.Role)
		m.InputReference.Filename = strings.TrimSpace(m.InputReference.Filename)
	}
}

func defaultPlaygroundAssetKind(kind string) string {
	switch kind {
	case "chat", "copywriting", "audio-transcribe":
		return "text"
	case "audio-generate", "audio-voice-design", "audio-voice-clone":
		return "audio"
	case "video":
		return "video"
	default:
		return "image"
	}
}

func defaultPlaygroundAssetTitle(kind string) string {
	switch kind {
	case "copywriting":
		return "商品文案"
	case "image-translate":
		return "图片翻译"
	case "chat":
		return "对话助手"
	case "batch-main":
		return "批量商品主图"
	case "batch-clone":
		return "参考图批量克隆"
	case "watermark":
		return "水印处理"
	case "image":
		return "AI 生图"
	case "edit":
		return "图片编辑"
	case "video":
		return "AI 视频"
	case "audio-transcribe":
		return "语音转写"
	case "audio-generate":
		return "AI 配音"
	case "audio-voice-design":
		return "音色设计"
	case "audio-voice-clone":
		return "声音克隆"
	default:
		return kind
	}
}

func isPlaygroundGrokImageModel(model string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "grok-imagine-image")
}

func isPlaygroundGptImageModel(model string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "gpt-image-")
}

func (p playgroundJobPayload) buildImageRequest(kind, model string) (string, json.RawMessage, error) {
	switch kind {
	case "image":
		if p.Prompt == "" {
			return "", nil, fmt.Errorf("%w: prompt is required", ErrPlaygroundInvalidInput)
		}
		body := map[string]any{
			"model":  model,
			"prompt": p.Prompt,
			"n":      1,
		}
		applyPlaygroundImageOptions(body, model, p, false)
		return "/v1/images/generations", mustJSON(body), nil
	case "edit", "image-translate", "watermark":
		body, err := buildPlaygroundImageEditBody(model, p.Prompt, p.Media, p)
		if err != nil {
			return "", nil, err
		}
		return "/v1/images/edits", mustJSON(body), nil
	default:
		return "", nil, fmt.Errorf("%w: unsupported image kind %s", ErrPlaygroundInvalidInput, kind)
	}
}

func (p playgroundJobPayload) buildBatchImageRequests(model string) (string, []json.RawMessage, error) {
	if p.Batch == nil || len(p.Batch.Items) == 0 {
		return "", nil, fmt.Errorf("%w: batch items are required", ErrPlaygroundInvalidInput)
	}
	bodies := make([]json.RawMessage, 0, len(p.Batch.Items))
	for _, item := range p.Batch.Items {
		body, err := buildPlaygroundImageEditBody(model, playgroundFirstNonEmpty(item.Prompt, p.Prompt), item.Media, p)
		if err != nil {
			return "", nil, err
		}
		bodies = append(bodies, mustJSON(body))
	}
	return "/v1/images/edits", bodies, nil
}

func buildPlaygroundImageEditBody(model, prompt string, media playgroundJobMedia, options playgroundJobPayload) (map[string]any, error) {
	if strings.TrimSpace(prompt) == "" {
		return nil, fmt.Errorf("%w: prompt is required", ErrPlaygroundInvalidInput)
	}
	images := make([]map[string]any, 0, len(media.Images))
	for _, item := range media.Images {
		if item.URL == "" {
			continue
		}
		images = append(images, map[string]any{"image_url": item.URL})
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("%w: at least one input image is required", ErrPlaygroundInvalidInput)
	}
	body := map[string]any{
		"model":  model,
		"prompt": prompt,
		"images": images,
	}
	if media.Mask != nil && media.Mask.URL != "" {
		body["mask"] = map[string]any{"image_url": media.Mask.URL}
	}
	applyPlaygroundImageOptions(body, model, options, true)
	return body, nil
}

func applyPlaygroundImageOptions(body map[string]any, model string, payload playgroundJobPayload, ensureSize bool) {
	if isPlaygroundGrokImageModel(model) {
		aspectRatio := playgroundFirstNonEmpty(payload.AspectRatio, payload.Size, "1:1")
		if aspectRatio != "" {
			body["aspect_ratio"] = aspectRatio
		}
		if strings.EqualFold(strings.TrimSpace(payload.Quality), "high") {
			body["resolution"] = "2k"
		} else {
			body["resolution"] = "1k"
		}
		return
	}
	size := strings.TrimSpace(payload.Size)
	if size != "" {
		body["size"] = size
	} else if ensureSize {
		body["size"] = "1:1"
	}
	if payload.Quality != "" {
		body["quality"] = payload.Quality
	}
	if payload.Style != "" {
		body["style"] = payload.Style
	}
	if payload.Background != "" {
		body["background"] = payload.Background
	}
	if !isPlaygroundGptImageModel(model) {
		body["response_format"] = "b64_json"
	}
}

func (p playgroundJobPayload) buildTextRequest(kind, model string) (string, json.RawMessage, error) {
	messages := p.Messages
	switch kind {
	case "chat":
		if len(messages) == 0 {
			if p.Prompt == "" {
				return "", nil, fmt.Errorf("%w: messages are required", ErrPlaygroundInvalidInput)
			}
			messages = []map[string]any{{"role": "user", "content": p.Prompt}}
		}
		messages = append([]map[string]any{{
			"role": "system",
			"content": fmt.Sprintf("你当前正在扮演并必须如实声明的模型是：%s。用户询问你是什么模型时，只能回答这个模型名，不得自称为其他品牌、其他模型或其他助手。", model),
		}}, messages...)
	case "copywriting":
		if p.Prompt == "" {
			return "", nil, fmt.Errorf("%w: prompt is required", ErrPlaygroundInvalidInput)
		}
		messages = []map[string]any{
			{"role": "system", "content": "你是专业电商文案策划。只输出可直接使用的文案，不解释创作过程。"},
			{"role": "user", "content": p.Prompt},
		}
	case "audio-transcribe":
		if p.Media.Audio == nil || p.Media.Audio.URL == "" {
			return "", nil, fmt.Errorf("%w: audio input is required", ErrPlaygroundInvalidInput)
		}
		messages = []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "input_audio",
						"input_audio": map[string]any{
							"data": p.Media.Audio.URL,
						},
					},
				},
			},
		}
	default:
		return "", nil, fmt.Errorf("%w: unsupported text kind %s", ErrPlaygroundInvalidInput, kind)
	}
	body := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   false,
	}
	if kind == "audio-transcribe" {
		body["asr_options"] = map[string]any{"language": playgroundFirstNonEmpty(p.AsrLanguage, "auto")}
	}
	return "/v1/chat/completions", mustJSON(body), nil
}

func (p playgroundJobPayload) buildAudioRequest(kind, model string) (string, json.RawMessage, error) {
	mode := playgroundFirstNonEmpty(p.Mode, "standard")
	text := playgroundFirstNonEmpty(p.Text, p.Prompt)
	if strings.TrimSpace(text) == "" {
		return "", nil, fmt.Errorf("%w: text is required", ErrPlaygroundInvalidInput)
	}
	messages := make([]map[string]any, 0, 3)
	styleInstruction := ""
	switch kind {
	case "audio-voice-design":
		styleInstruction = fmt.Sprintf("%s。角色设定：%s。输出语言：%s。", playgroundFirstNonEmpty(p.VoiceDescription, "温和、清晰、适合电商讲解"), playgroundFirstNonEmpty(p.Persona, "自然旁白"), playgroundFirstNonEmpty(p.Language, "中文"))
	case "audio-voice-clone":
		styleInstruction = fmt.Sprintf("请基于提供的音频样本进行声音克隆，保持自然稳定的发音。输出语言：%s。风格：%s。", playgroundFirstNonEmpty(p.Language, "中文"), playgroundFirstNonEmpty(p.Style, "自然讲述"))
	default:
		styleInstruction = fmt.Sprintf("请用%s风格朗读，输出语言：%s。", playgroundFirstNonEmpty(p.Style, "自然讲述"), playgroundFirstNonEmpty(p.Language, "中文"))
	}
	if strings.TrimSpace(styleInstruction) != "" {
		messages = append(messages, map[string]any{"role": "user", "content": styleInstruction})
	}
	if kind == "audio-voice-clone" {
		if p.Media.ReferenceAudio == nil || p.Media.ReferenceAudio.URL == "" {
			return "", nil, fmt.Errorf("%w: reference audio is required", ErrPlaygroundInvalidInput)
		}
		messages = append(messages, map[string]any{
			"role": "user",
			"content": []map[string]any{
				{
					"type": "audio_url",
					"audio_url": map[string]any{
						"url": p.Media.ReferenceAudio.URL,
					},
				},
			},
		})
	}
	messages = append(messages, map[string]any{"role": "assistant", "content": text})
	body := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   false,
	}
	if kind == "audio-voice-design" || strings.EqualFold(mode, "voicedesign") {
		body["audio"] = map[string]any{
			"format":                "wav",
			"optimize_text_preview": true,
		}
	} else {
		body["audio"] = map[string]any{
			"format": "wav",
			"voice":  playgroundFirstNonEmpty(p.VoicePreset, playgroundMediaURL(p.Media.ReferenceAudio), "mimo_default"),
		}
	}
	return "/v1/chat/completions", mustJSON(body), nil
}

func (p playgroundJobPayload) buildVideoRequest(model string) (string, json.RawMessage, error) {
	if p.Prompt == "" {
		return "", nil, fmt.Errorf("%w: prompt is required", ErrPlaygroundInvalidInput)
	}
	body := map[string]any{
		"model":        model,
		"prompt":       p.Prompt,
		"duration":     p.Duration,
		"aspect_ratio": playgroundFirstNonEmpty(p.AspectRatio, "16:9"),
		"resolution":   playgroundFirstNonEmpty(p.Resolution, "720p"),
	}
	if p.Media.InputReference != nil && p.Media.InputReference.URL != "" {
		body["input_reference"] = map[string]any{"image_url": p.Media.InputReference.URL}
	}
	return "/v1/videos/generations", mustJSON(body), nil
}

func playgroundMediaURL(ref *playgroundJobMediaRef) string {
	if ref == nil {
		return ""
	}
	return strings.TrimSpace(ref.URL)
}
