package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type leonardoMediaAccountRepoStub struct {
	service.AccountRepository
	calls int
}

func (s *leonardoMediaAccountRepoStub) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]service.Account, error) {
	s.calls++
	return nil, nil
}

func TestLeonardoMediaHandlerStrictJSON(t *testing.T) {
	decoder := json.NewDecoder(strings.NewReader(`{"model":"flux","modality":"image","prompt":"cat","parameters":{"width":1,"height":1,"quantity":1},"customer_quote_usd":"0.01"}`))
	decoder.DisallowUnknownFields()
	var request leonardoMediaCreateHTTPRequest
	require.NoError(t, decoder.Decode(&request))
	require.NoError(t, ensureLeonardoMediaJSONEOF(decoder))

	decoder = json.NewDecoder(strings.NewReader(`{} {}`))
	require.NoError(t, decoder.Decode(&request))
	require.Error(t, ensureLeonardoMediaJSONEOF(decoder))
}

func TestLeonardoMediaHandlerRequiresIdempotencyKeyInObserveOnlyMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &leonardoMediaAccountRepoStub{}
	orchestrator := service.NewLeonardoImageCreateOrchestrator(nil, nil, nil, nil, nil)
	handler := NewLeonardoMediaHandler(service.NewLeonardoMediaCreateService(repo, orchestrator))
	cfg := service.DefaultIdempotencyConfig()
	require.True(t, cfg.ObserveOnly)
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(nil, cfg))
	t.Cleanup(func() { service.SetDefaultIdempotencyCoordinator(nil) })

	groupID := int64(7)
	user := &service.User{ID: 11}
	apiKey := &service.APIKey{ID: 13, UserID: user.ID, User: user, GroupID: &groupID, Group: &service.Group{ID: groupID, Platform: service.PlatformLeonardo}}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/media/generations", strings.NewReader(`{"model":"flux-schnell","modality":"image","prompt":"cat","public":false,"parameters":{"width":896,"height":896,"quantity":1},"customer_quote_usd":"0.005"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.Create(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, 0, repo.calls)
}
