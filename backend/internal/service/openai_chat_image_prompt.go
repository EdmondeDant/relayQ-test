package service

import (
	"strings"

	"github.com/tidwall/gjson"
)

func ExtractOpenAIChatImagePrompt(body []byte) string {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return ""
	}
	messages := gjson.GetBytes(body, "messages")
	if !messages.IsArray() {
		return ""
	}
	arr := messages.Array()
	for i := len(arr) - 1; i >= 0; i-- {
		msg := arr[i]
		if strings.TrimSpace(msg.Get("role").String()) != "user" {
			continue
		}
		parts := extractOpenAIChatContentText(msg.Get("content"))
		if strings.TrimSpace(parts) != "" {
			return strings.TrimSpace(parts)
		}
	}
	return ""
}

func ExtractOpenAIChatImageOptions(body []byte) map[string]any {
	options := map[string]any{}
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return options
	}
	getString := func(paths ...string) string {
		for _, path := range paths {
			if value := strings.TrimSpace(gjson.GetBytes(body, path).String()); value != "" {
				return value
			}
		}
		return ""
	}
	if aspectRatio := getString("aspect_ratio", "aspectRatio", "image_options.aspect_ratio", "image_options.aspectRatio", "providerOptions.xai.aspectRatio", "provider_options.xai.aspectRatio"); aspectRatio != "" {
		options["aspect_ratio"] = aspectRatio
	}
	if resolution := normalizeXAIImageResolution(getString("resolution", "image_options.resolution", "providerOptions.xai.resolution", "provider_options.xai.resolution")); resolution != "" {
		options["resolution"] = resolution
	}
	if quality := getString("quality", "image_options.quality", "providerOptions.xai.quality", "provider_options.xai.quality"); quality != "" {
		options["quality"] = quality
	}
	if user := getString("user"); user != "" {
		options["user"] = user
	}
	if n := gjson.GetBytes(body, "n"); n.Exists() && n.Type == gjson.Number {
		v := int(n.Int())
		if v > 0 {
			if v > 10 {
				v = 10
			}
			options["n"] = v
		}
	}
	return options
}

func extractOpenAIChatContentText(content gjson.Result) string {
	if content.Type == gjson.String {
		return strings.TrimSpace(content.String())
	}
	if !content.IsArray() {
		return ""
	}
	var parts []string
	for _, item := range content.Array() {
		typ := strings.TrimSpace(item.Get("type").String())
		switch typ {
		case "text", "input_text":
			if text := strings.TrimSpace(item.Get("text").String()); text != "" {
				parts = append(parts, text)
			}
		default:
			if text := strings.TrimSpace(item.Get("text").String()); text != "" && typ == "" {
				parts = append(parts, text)
			}
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}
