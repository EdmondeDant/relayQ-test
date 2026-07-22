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

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
