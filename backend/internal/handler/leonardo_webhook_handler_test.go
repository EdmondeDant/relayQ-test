package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type leonardoWebhookHandlerAccountRepository struct {
	service.AccountRepository
}

func (leonardoWebhookHandlerAccountRepository) GetByID(context.Context, int64) (*service.Account, error) {
	return &service.Account{ID: 7, Platform: service.PlatformLeonardo, Extra: map[string]any{"webhook_route_token": "route-secret", "webhook_secret": "bearer-secret"}}, nil
}

type leonardoWebhookHandlerEventRepository struct {
	created bool
}

func (r *leonardoWebhookHandlerEventRepository) CreatePending(context.Context, *service.LeonardoWebhookEvent) (bool, error) {
	return r.created, nil
}

func (r *leonardoWebhookHandlerEventRepository) ClaimPending(context.Context, int) ([]*service.LeonardoWebhookEvent, error) {
	return nil, nil
}

func (r *leonardoWebhookHandlerEventRepository) MarkProcessed(context.Context, int64) error {
	return nil
}

func (r *leonardoWebhookHandlerEventRepository) MarkFailed(context.Context, int64, int) error {
	return nil
}

func TestLeonardoWebhookHandlerSecurityAndBodyLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	events := &leonardoWebhookHandlerEventRepository{created: true}
	handler := NewLeonardoWebhookHandler(service.NewLeonardoWebhookService(leonardoWebhookHandlerAccountRepository{}, events))

	request := func(routeToken, authorization string, body []byte) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Params = gin.Params{{Key: "account_id", Value: "7"}, {Key: "route_token", Value: routeToken}}
		c.Request = httptest.NewRequest(http.MethodPost, "/internal/webhooks/leonardo/7/"+routeToken, bytes.NewReader(body))
		c.Request.Header.Set("Authorization", authorization)
		handler.Handle(c)
		return recorder
	}

	require.Equal(t, http.StatusUnauthorized, request("wrong", "Bearer bearer-secret", []byte(`{"type":"complete"}`)).Code)
	require.Equal(t, http.StatusRequestEntityTooLarge, request("route-secret", "Bearer bearer-secret", bytes.Repeat([]byte("x"), leonardoWebhookMaxBodyBytes+1)).Code)
	require.Equal(t, http.StatusAccepted, request("route-secret", "Bearer bearer-secret", []byte(`{"type":"complete"}`)).Code)
	events.created = false
	replay := request("route-secret", "Bearer bearer-secret", []byte(`{"type":"complete"}`))
	require.Equal(t, http.StatusAccepted, replay.Code)
	require.Equal(t, "true", replay.Header().Get("X-Idempotency-Replayed"))
}
