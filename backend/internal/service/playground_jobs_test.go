package service

import "testing"

func TestExtractAudioResultSupportsMessageAudioBase64(t *testing.T) {
	payload := map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"content": "",
					"audio": map[string]any{
						"data":   "UklGRg==",
						"format": "wav",
					},
				},
			},
		},
	}

	url, dataURL, contentType := extractAudioResult(payload)
	if url != "" {
		t.Fatalf("extractAudioResult() url = %q, want empty", url)
	}
	if want := "data:audio/wav;base64,UklGRg=="; dataURL != want {
		t.Fatalf("extractAudioResult() dataURL = %q, want %q", dataURL, want)
	}
	if want := "audio/wav"; contentType != want {
		t.Fatalf("extractAudioResult() contentType = %q, want %q", contentType, want)
	}
}

func TestExtractAudioResultSupportsOutputAudioURLPart(t *testing.T) {
	payload := map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"content": []any{
						map[string]any{
							"type": "output_audio",
							"output_audio": map[string]any{
								"url":    "https://example.com/audio.ogg",
								"format": "ogg",
							},
						},
					},
				},
			},
		},
	}

	url, dataURL, contentType := extractAudioResult(payload)
	if want := "https://example.com/audio.ogg"; url != want {
		t.Fatalf("extractAudioResult() url = %q, want %q", url, want)
	}
	if dataURL != "" {
		t.Fatalf("extractAudioResult() dataURL = %q, want empty", dataURL)
	}
	if want := "audio/ogg"; contentType != want {
		t.Fatalf("extractAudioResult() contentType = %q, want %q", contentType, want)
	}
}
