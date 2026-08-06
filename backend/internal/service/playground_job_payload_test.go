package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildTextRequestChatInjectsSelectedModelIdentity(t *testing.T) {
	payload := playgroundJobPayload{
		Prompt: "你是什么模型？",
	}

	_, body, err := payload.buildTextRequest("chat", "grok-4.5")
	if err != nil {
		t.Fatalf("buildTextRequest() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal body error = %v", err)
	}
	messages, _ := decoded["messages"].([]any)
	if len(messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(messages))
	}
	system, _ := messages[0].(map[string]any)
	if got := system["role"]; got != "system" {
		t.Fatalf("system role = %v, want system", got)
	}
	content, _ := system["content"].(string)
	if content == "" || !containsAll(content, "grok-4.5", "不得自称为其他品牌") {
		t.Fatalf("unexpected system content = %q", content)
	}
}

func TestBuildImageRequestAdaptsLeonardoOpenAIContract(t *testing.T) {
	payload := playgroundJobPayload{Prompt: "老牛吃嫩草", Size: "1:1", Quality: "high", Style: "natural", Background: "opaque", Metadata: map[string]any{"platform": "leonardo"}}
	endpoint, body, err := payload.buildImageRequest("image", "nano-banana-2-lite")
	if err != nil {
		t.Fatalf("buildImageRequest() error = %v", err)
	}
	if endpoint != "/v1/images/generations" {
		t.Fatalf("endpoint = %q", endpoint)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal body error = %v", err)
	}
	if decoded["size"] != "1024x1024" || decoded["quality"] != "low" || decoded["response_format"] != "url" {
		t.Fatalf("unexpected Leonardo request = %#v", decoded)
	}
	if _, ok := decoded["style"]; ok {
		t.Fatalf("Leonardo request contains style")
	}
	if _, ok := decoded["background"]; ok {
		t.Fatalf("Leonardo request contains background")
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
