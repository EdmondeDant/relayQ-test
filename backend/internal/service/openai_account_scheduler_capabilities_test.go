//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeRequiredOpenAIEndpointCapabilities(t *testing.T) {
	got := normalizeRequiredOpenAIEndpointCapabilities(
		OpenAIEndpointCapabilityChatCompletions,
		[]OpenAIEndpointCapability{
			OpenAIEndpointCapabilityChatImageInput,
			OpenAIEndpointCapabilityChatCompletions,
			OpenAIEndpointCapabilityChatVideoInput,
			"",
		},
	)
	require.Equal(t, []OpenAIEndpointCapability{
		OpenAIEndpointCapabilityChatCompletions,
		OpenAIEndpointCapabilityChatImageInput,
		OpenAIEndpointCapabilityChatVideoInput,
	}, got)
}

func TestAccountSupportsOpenAIEndpointCapabilitySet(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"openai_capabilities": []any{"chat_completions", "chat_image_input"},
		},
	}

	require.True(t, accountSupportsOpenAIEndpointCapabilitySet(account, []OpenAIEndpointCapability{
		OpenAIEndpointCapabilityChatCompletions,
		OpenAIEndpointCapabilityChatImageInput,
	}, ""))
	require.False(t, accountSupportsOpenAIEndpointCapabilitySet(account, []OpenAIEndpointCapability{
		OpenAIEndpointCapabilityChatCompletions,
		OpenAIEndpointCapabilityChatVideoInput,
	}, ""))
}
