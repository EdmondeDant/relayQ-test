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

func TestBuildImageRequestAdaptsFluxKleinSize(t *testing.T) {
	tests := map[string]string{"1:1": "1024x1024", "16:9": "1024x576", "9:16": "576x1024", "3:2": "1024x682", "2:3": "682x1024"}
	for ratio, want := range tests {
		payload := playgroundJobPayload{Prompt: "product photo", Size: ratio, Quality: "high", Style: "natural", Background: "opaque", Metadata: map[string]any{"platform": PlatformOpenAI}}
		_, body, err := payload.buildImageRequest("image", "flux-2-klein-9b-kv")
		if err != nil {
			t.Fatalf("buildImageRequest(%s) error = %v", ratio, err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatalf("unmarshal body error = %v", err)
		}
		if decoded["size"] != want {
			t.Fatalf("size for %s = %v, want %s", ratio, decoded["size"], want)
		}
		for _, key := range []string{"quality", "style", "background"} {
			if _, ok := decoded[key]; ok {
				t.Fatalf("FLUX request contains %s", key)
			}
		}
	}
}

func TestBuildVideoRequestAdaptsLeonardoOpenAIContract(t *testing.T) {
	tests := []struct {
		model, resolution, ratio string
		duration, seconds        float64
		size                     string
	}{
		{"motion_2.0-fast", "720p", "9:16", 0, 0, "720x1152"},
		{"seedance-1.0-pro-fast", "720p", "1:1", 6, 6, "960x960"},
		{"seedance-1.0-pro", "1080p", "16:9", 10, 10, "1920x1088"},
		{"wan-2.7", "1080p", "9:16", 10, 10, "1080x1920"},
	}
	for _, test := range tests {
		payload := playgroundJobPayload{Prompt: "video prompt", Duration: int(test.duration), Resolution: test.resolution, AspectRatio: test.ratio, Metadata: map[string]any{"platform": "leonardo"}}
		_, body, err := payload.buildVideoRequest(test.model)
		if err != nil {
			t.Fatalf("buildVideoRequest() error = %v", err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatalf("unmarshal body error = %v", err)
		}
		if decoded["seconds"] != test.seconds || decoded["size"] != test.size {
			t.Fatalf("unexpected Leonardo request = %#v", decoded)
		}
		if _, ok := decoded["duration"]; ok {
			t.Fatalf("Leonardo request contains duration")
		}
	}
}

func TestBuildVideoRequestPassesSupportedLeonardoFirstFrame(t *testing.T) {
	payload := playgroundJobPayload{
		Prompt: "video prompt", Duration: 4, Resolution: "480p", AspectRatio: "16:9",
		Metadata: map[string]any{"platform": "leonardo"},
		Media:    playgroundJobMedia{InputReference: &playgroundJobMediaRef{URL: "data:image/png;base64,AA=="}},
	}
	_, body, err := payload.buildVideoRequest("seedance-1.0-pro-fast")
	if err != nil {
		t.Fatalf("buildVideoRequest() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal body error = %v", err)
	}
	reference, _ := decoded["input_reference"].(map[string]any)
	if reference["image_url"] != "data:image/png;base64,AA==" {
		t.Fatalf("unexpected first frame = %#v", decoded)
	}
}

func TestBuildVideoRequestPassesMotionFirstFrame(t *testing.T) {
	payload := playgroundJobPayload{
		Prompt: "video prompt", Duration: 0, Resolution: "480p", AspectRatio: "16:9",
		Metadata: map[string]any{"platform": "leonardo"},
		Media:    playgroundJobMedia{InputReference: &playgroundJobMediaRef{URL: "data:image/png;base64,AA=="}},
	}
	if _, body, err := payload.buildVideoRequest("motion_2.0-fast"); err != nil || !strings.Contains(string(body), "input_reference") {
		t.Fatalf("buildVideoRequest() body = %s, err = %v", body, err)
	}
}

func TestNormalizePlaygroundProgress(t *testing.T) {
	tests := []struct {
		value any
		want  int
	}{{0.5, 50}, {"50%", 50}, {120, 100}, {-1, 0}}
	for _, test := range tests {
		if got := normalizePlaygroundProgress(test.value); got != test.want {
			t.Fatalf("normalizePlaygroundProgress(%v) = %d, want %d", test.value, got, test.want)
		}
	}
}

func TestBuildVideoRequestAdaptsMiniMaxH3(t *testing.T) {
	payload := playgroundJobPayload{
		Prompt: "city night", Duration: 10, Resolution: "1080p", AspectRatio: "16:9",
		Metadata: map[string]any{"platform": PlatformOpenAI},
		Media:    playgroundJobMedia{InputReference: &playgroundJobMediaRef{URL: "https://cdn.example/start.png"}},
	}
	_, body, err := payload.buildVideoRequest("minimax-h3")
	if err != nil {
		t.Fatalf("buildVideoRequest() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal body error = %v", err)
	}
	if decoded["model"] != "minimax-h3" || decoded["prompt"] != "city night" {
		t.Fatalf("unexpected MiniMax request = %#v", decoded)
	}
	if decoded["duration"] != float64(10) || decoded["resolution"] != "2K" || decoded["ratio"] != "adaptive" {
		t.Fatalf("unexpected MiniMax request = %#v", decoded)
	}
	if decoded["image"] != nil {
		t.Fatalf("MiniMax request contains image: %#v", decoded)
	}
	if decoded["reference_images"] != nil {
		t.Fatalf("MiniMax request contains reference_images: %#v", decoded)
	}
	content, _ := decoded["content"].([]any)
	if len(content) < 2 {
		t.Fatalf("unexpected MiniMax content = %#v", decoded)
	}

	payload = playgroundJobPayload{
		Prompt: "city night", Duration: 10, Resolution: "1080p", AspectRatio: "16:9",
		Metadata: map[string]any{"platform": PlatformOpenAI},
	}
	_, body, err = payload.buildVideoRequest("minimax-h3")
	if err != nil {
		t.Fatalf("buildVideoRequest() error = %v", err)
	}
	decoded = map[string]any{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal body error = %v", err)
	}
	if decoded["model"] != "minimax-h3" || decoded["prompt"] != "city night" {
		t.Fatalf("unexpected MiniMax request = %#v", decoded)
	}
	if decoded["duration"] != float64(10) || decoded["resolution"] != "2K" || decoded["ratio"] != "16:9" {
		t.Fatalf("unexpected MiniMax request = %#v", decoded)
	}
	if _, ok := decoded["image"]; ok {
		t.Fatalf("MiniMax request contains image without first frame: %#v", decoded)
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
