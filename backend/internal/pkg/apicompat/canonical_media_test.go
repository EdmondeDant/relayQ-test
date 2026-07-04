//go:build unit

package apicompat

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanonicalContentPartsFromChat_PreservesModalities(t *testing.T) {
	var parts []ChatContentPart
	require.NoError(t, json.Unmarshal([]byte(`[
		{"type":"text","text":"describe"},
		{"type":"image_url","image_url":"data:image/png;base64,abc123"},
		{"type":"video_url","video_url":"https://example.com/video.mp4"},
		{"type":"audio_url","audio_url":"https://example.com/audio.wav"},
		{"type":"input_audio","input_audio":{"data":"abc123","format":"wav"}}
	]`), &parts))

	got := CanonicalContentPartsFromChat(parts)
	require.Len(t, got, 5)
	require.Equal(t, CanonicalContentText, got[0].Kind)
	require.Equal(t, "describe", got[0].Text)
	require.Equal(t, CanonicalContentImage, got[1].Kind)
	require.Equal(t, "data:image/png;base64,abc123", got[1].URL)
	require.Equal(t, CanonicalContentVideo, got[2].Kind)
	require.Equal(t, "https://example.com/video.mp4", got[2].URL)
	require.Equal(t, CanonicalContentAudio, got[3].Kind)
	require.Equal(t, "https://example.com/audio.wav", got[3].URL)
	require.Equal(t, CanonicalContentAudio, got[4].Kind)
	require.Equal(t, "abc123", got[4].Data)
	require.Equal(t, "wav", got[4].Format)
}

func TestCanonicalKindFromMIMEType(t *testing.T) {
	require.Equal(t, CanonicalContentImage, CanonicalKindFromMIMEType("image/png"))
	require.Equal(t, CanonicalContentVideo, CanonicalKindFromMIMEType("video/mp4"))
	require.Equal(t, CanonicalContentAudio, CanonicalKindFromMIMEType("audio/wav"))
	require.Equal(t, CanonicalContentFile, CanonicalKindFromMIMEType("application/pdf"))
	require.Equal(t, CanonicalContentKind(""), CanonicalKindFromMIMEType(""))
}
