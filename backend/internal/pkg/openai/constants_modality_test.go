package openai

import "testing"

func TestModelModalityRecognizesRelayQOpenAIMediaModels(t *testing.T) {
	tests := map[string]string{
		"GPT Image-2":               "image",
		"Nano Banana":               "image",
		"Nano Banana 2":             "image",
		"Nano Banana Pro":           "image",
		"Seedream 4.5":              "image",
		"seedance-2.0":              "video",
		"seedance-2.0-fast":         "video",
		"seedance-2.0-mini":         "video",
		"kling-3.0":                 "video",
		"kling-video-o-3":           "video",
		"wan-2.7":                   "video",
		"mimo-v2.5-tts":             "audio",
		"mimo-v2.5-tts-voiceclone":  "audio",
		"mimo-v2.5-tts-voicedesign": "audio",
		"mimo-v2-tts":               "audio",
	}
	for model, want := range tests {
		if got := ModelModality(model); got != want {
			t.Fatalf("ModelModality(%q) = %q, want %q", model, got, want)
		}
	}
	if got := ModelModality("mimo-v2.5-asr"); got == "audio" {
		t.Fatalf("ModelModality(%q) = %q, must not be audio", "mimo-v2.5-asr", got)
	}
}
