package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSummarizePlaygroundRecordPayloadKeepsOnlyPreviewFields(t *testing.T) {
	raw := json.RawMessage(`{
		"prompt":"hello",
		"text":"world",
		"content":"done",
		"audio_url":"https://example.com/a.mp3",
		"metadata":{"huge":"ignore"},
		"messages":[{"role":"user","content":"drop"}]
	}`)

	request := summarizePlaygroundRecordPayload(raw, false)
	result := summarizePlaygroundRecordPayload(raw, true)

	if string(request) != `{"prompt":"hello","text":"world"}` {
		t.Fatalf("request summary = %s", string(request))
	}
	if !strings.Contains(string(result), `"content":"done"`) || !strings.Contains(string(result), `"audio_url":"https://example.com/a.mp3"`) {
		t.Fatalf("result summary missing expected fields: %s", string(result))
	}
	if strings.Contains(string(result), "metadata") || strings.Contains(string(result), "messages") {
		t.Fatalf("result summary leaked heavy fields: %s", string(result))
	}
}

func TestCompactPlaygroundRecordAssetsKeepsPrimaryAndTextPreview(t *testing.T) {
	assets := []PlaygroundAsset{
		{ID: 1, Kind: "image", StorageKey: "image/a.png"},
		{ID: 2, Kind: "text", Content: "preview"},
		{ID: 3, Kind: "image", StorageKey: "image/b.png"},
	}
	primary := &PlaygroundAsset{ID: 1, Kind: "image", StorageKey: "image/a.png"}

	got := compactPlaygroundRecordAssets(assets, primary)
	if len(got) != 2 {
		t.Fatalf("compact assets len = %d, want 2", len(got))
	}
	if got[0].ID != 1 || got[1].ID != 2 {
		t.Fatalf("compact assets order = %+v", got)
	}
}
