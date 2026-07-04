package service

import (
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
)

// RequiredChatMediaCapabilitiesFromBody inspects an inbound OpenAI-compatible
// chat-completions request and returns the media capabilities required to route
// it without silently dropping modalities. It does not decide whether a model
// supports the capability; scheduler/account selection will use these flags.
func RequiredChatMediaCapabilitiesFromBody(body []byte) []OpenAIEndpointCapability {
	var req apicompat.ChatCompletionsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil
	}

	seen := map[OpenAIEndpointCapability]bool{}
	var out []OpenAIEndpointCapability
	add := func(capability OpenAIEndpointCapability) {
		if capability == "" || seen[capability] {
			return
		}
		seen[capability] = true
		out = append(out, capability)
	}

	for _, msg := range req.Messages {
		var parts []apicompat.ChatContentPart
		if err := json.Unmarshal(msg.Content, &parts); err != nil {
			continue
		}
		for _, part := range apicompat.CanonicalContentPartsFromChat(parts) {
			switch part.Kind {
			case apicompat.CanonicalContentImage:
				add(OpenAIEndpointCapabilityChatImageInput)
			case apicompat.CanonicalContentVideo:
				add(OpenAIEndpointCapabilityChatVideoInput)
			case apicompat.CanonicalContentAudio:
				add(OpenAIEndpointCapabilityChatAudioInput)
			}
		}
	}
	return out
}
