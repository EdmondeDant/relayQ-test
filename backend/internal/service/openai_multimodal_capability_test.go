//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequiredChatMediaCapabilitiesFromBody(t *testing.T) {
	body := []byte(`{
		"model":"grok-4.3",
		"messages":[{"role":"user","content":[
			{"type":"text","text":"describe"},
			{"type":"image_url","image_url":"data:image/png;base64,abc123"},
			{"type":"video_url","video_url":"https://example.com/video.mp4"},
			{"type":"input_audio","input_audio":{"data":"abc123","format":"wav"}}
		]}]
	}`)

	got := RequiredChatMediaCapabilitiesFromBody(body)
	require.Equal(t, []OpenAIEndpointCapability{
		OpenAIEndpointCapabilityChatImageInput,
		OpenAIEndpointCapabilityChatVideoInput,
		OpenAIEndpointCapabilityChatAudioInput,
	}, got)
}

func TestRequiredChatMediaCapabilitiesFromBody_TextOnly(t *testing.T) {
	body := []byte(`{"model":"gpt-5.5","messages":[{"role":"user","content":"hello"}]}`)
	require.Empty(t, RequiredChatMediaCapabilitiesFromBody(body))
}
