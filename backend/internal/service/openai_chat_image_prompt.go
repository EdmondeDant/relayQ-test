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
