package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

type leonardoWebhookAccountRepository struct {
	AccountRepository
	account *Account
}

func (r *leonardoWebhookAccountRepository) GetByID(context.Context, int64) (*Account, error) {
	return r.account, nil
}

type leonardoWebhookEventRepositoryFake struct {
	created bool
	event   *LeonardoWebhookEvent
}

func (r *leonardoWebhookEventRepositoryFake) CreatePending(_ context.Context, event *LeonardoWebhookEvent) (bool, error) {
	r.event = event
	return r.created, nil
}

func (r *leonardoWebhookEventRepositoryFake) ClaimPending(context.Context, int) ([]*LeonardoWebhookEvent, error) {
	return nil, nil
}

func (r *leonardoWebhookEventRepositoryFake) MarkProcessed(context.Context, int64) error {
	return nil
}

func (r *leonardoWebhookEventRepositoryFake) MarkFailed(context.Context, int64, int) error {
	return nil
}

func TestLeonardoWebhookServiceAuthenticatesRedactsAndDeduplicates(t *testing.T) {
	accounts := &leonardoWebhookAccountRepository{account: &Account{ID: 7, Platform: PlatformLeonardo, Extra: map[string]any{"webhook_route_token": "route-secret", "webhook_secret": "bearer-secret"}}}
	events := &leonardoWebhookEventRepositoryFake{created: true}
	webhook := NewLeonardoWebhookService(accounts, events)
	body := []byte(`{"type":"image_generation.complete","authorization":"Bearer leaked","data":{"object":{"id":"generation-1","api_key":"leaked"}}}`)

	replayed, err := webhook.Receive(context.Background(), 7, "route-secret", "Bearer bearer-secret", body)

	require.NoError(t, err)
	require.False(t, replayed)
	require.Equal(t, "image_generation.complete", events.event.EventType)
	require.Equal(t, "generation-1", events.event.UpstreamGenerationID)
	var stored map[string]any
	require.NoError(t, json.Unmarshal(events.event.Payload, &stored))
	require.Equal(t, "[REDACTED]", stored["authorization"])
	require.Equal(t, "[REDACTED]", stored["data"].(map[string]any)["object"].(map[string]any)["api_key"])

	events.created = false
	replayed, err = webhook.Receive(context.Background(), 7, "route-secret", "Bearer bearer-secret", body)
	require.NoError(t, err)
	require.True(t, replayed)
}

func TestLeonardoWebhookServiceRejectsInvalidSecrets(t *testing.T) {
	webhook := NewLeonardoWebhookService(&leonardoWebhookAccountRepository{account: &Account{ID: 7, Platform: PlatformLeonardo, Extra: map[string]any{"webhook_route_token": "route-secret", "webhook_secret": "bearer-secret"}}}, &leonardoWebhookEventRepositoryFake{})
	body := []byte(`{"type":"image_generation.complete"}`)

	for _, input := range [][2]string{{"wrong", "Bearer bearer-secret"}, {"route-secret", "Bearer wrong"}, {"route-secret", "Basic bearer-secret"}} {
		_, err := webhook.Receive(context.Background(), 7, input[0], input[1], body)
		require.ErrorIs(t, err, ErrLeonardoWebhookUnauthorized)
	}
}
