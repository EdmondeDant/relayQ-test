//go:build unit

package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountTestService_XAIAPIKeyUsesAPIKeyInsteadOfAccessToken(t *testing.T) {
	ctx, recorder := newTestContext()

	resp := newJSONResponse(http.StatusOK, "")
	resp.Body = http.NoBody

	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &AccountTestService{httpUpstream: upstream}
	account := &Account{
		ID:          101,
		Platform:    PlatformXAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "xai-test-key",
			"base_url": "https://api.muskapi.cc/v1",
		},
	}

	err := svc.testXAIAccountConnection(ctx, account, "grok-4.5", "hi")
	require.NoError(t, err)
	require.Len(t, upstream.requests, 1)
	require.Equal(t, "Bearer xai-test-key", upstream.requests[0].Header.Get("Authorization"))
	require.Contains(t, upstream.requests[0].URL.String(), "/v1/chat/completions")
	require.NotContains(t, recorder.Body.String(), "No access token available")
}

func TestAccountTestService_XAIAPIKeyWithoutKeyReturnsAPIKeyError(t *testing.T) {
	ctx, recorder := newTestContext()

	svc := &AccountTestService{}
	account := &Account{
		ID:          102,
		Platform:    PlatformXAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{},
	}

	err := svc.testXAIAccountConnection(ctx, account, "grok-4.5", "hi")
	require.Error(t, err)
	require.Contains(t, recorder.Body.String(), "No API key available")
	require.NotContains(t, recorder.Body.String(), "No access token available")
}
