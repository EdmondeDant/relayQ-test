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

type leonardoMediaGetRepoStub struct {
	reads int
}

func (s *leonardoMediaGetRepoStub) GetByPublicID(context.Context, string) (*service.GenerationJob, error) {
	s.reads++
	return nil, service.ErrGenerationJobNotFound
}

func (s *leonardoMediaGetRepoStub) CompareAndSwapPoll(context.Context, string, service.GenerationJobStatus, int, *service.GenerationJob) error {
	return nil
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
	handler := NewLeonardoMediaHandler(service.NewLeonardoMediaCreateService(repo, orchestrator), nil)
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

func TestLeonardoMediaHandlerGetRejectsInvalidPublicID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewLeonardoMediaHandler(nil, service.NewLeonardoMediaGetService(nil, nil))
	groupID := int64(7)
	user := &service.User{ID: 11}
	apiKey := &service.APIKey{ID: 13, UserID: user.ID, User: user, GroupID: &groupID, Group: &service.Group{ID: groupID, Platform: service.PlatformLeonardo}}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/media/generations/not-valid", nil)
	c.Params = gin.Params{{Key: "id", Value: "not-valid"}}
	c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.Get(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestLeonardoMediaHandlerGetShortCircuitsInvalidAuthenticationAndBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const validID = "gen_rq_0123456789abcdef0123456789abcdef"
	tests := []struct {
		name       string
		apiKey     *service.APIKey
		subject    *middleware2.AuthSubject
		wantStatus int
	}{
		{name: "missing api key", subject: &middleware2.AuthSubject{UserID: 11}, wantStatus: http.StatusUnauthorized},
		{name: "invalid api key id", apiKey: func() *service.APIKey { v := leonardoMediaValidAPIKey(); v.ID = 0; return v }(), subject: &middleware2.AuthSubject{UserID: 11}, wantStatus: http.StatusUnauthorized},
		{name: "missing subject", apiKey: leonardoMediaValidAPIKey(), wantStatus: http.StatusUnauthorized},
		{name: "invalid subject", apiKey: leonardoMediaValidAPIKey(), subject: &middleware2.AuthSubject{}, wantStatus: http.StatusUnauthorized},
		{name: "missing user", apiKey: func() *service.APIKey { v := leonardoMediaValidAPIKey(); v.User = nil; return v }(), subject: &middleware2.AuthSubject{UserID: 11}, wantStatus: http.StatusUnauthorized},
		{name: "foreign user", apiKey: leonardoMediaValidAPIKey(), subject: &middleware2.AuthSubject{UserID: 99}, wantStatus: http.StatusUnauthorized},
		{name: "missing group id", apiKey: func() *service.APIKey { v := leonardoMediaValidAPIKey(); v.GroupID = nil; return v }(), subject: &middleware2.AuthSubject{UserID: 11}, wantStatus: http.StatusBadRequest},
		{name: "missing group", apiKey: func() *service.APIKey { v := leonardoMediaValidAPIKey(); v.Group = nil; return v }(), subject: &middleware2.AuthSubject{UserID: 11}, wantStatus: http.StatusBadRequest},
		{name: "foreign group", apiKey: func() *service.APIKey { v := leonardoMediaValidAPIKey(); v.Group.ID = 99; return v }(), subject: &middleware2.AuthSubject{UserID: 11}, wantStatus: http.StatusBadRequest},
		{name: "wrong platform", apiKey: func() *service.APIKey {
			v := leonardoMediaValidAPIKey()
			v.Group.Platform = service.PlatformOpenAI
			return v
		}(), subject: &middleware2.AuthSubject{UserID: 11}, wantStatus: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &leonardoMediaGetRepoStub{}
			handler := NewLeonardoMediaHandler(nil, service.NewLeonardoMediaGetService(repository, &service.LeonardoGenerationPollOrchestrator{}))
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, "/v1/media/generations/"+validID, nil)
			c.Params = gin.Params{{Key: "id", Value: validID}}
			if test.apiKey != nil {
				c.Set(string(middleware2.ContextKeyAPIKey), test.apiKey)
			}
			if test.subject != nil {
				c.Set(string(middleware2.ContextKeyUser), *test.subject)
			}
			handler.Get(c)
			require.Equal(t, test.wantStatus, recorder.Code)
			require.Zero(t, repository.reads)
		})
	}
}

func TestLeonardoMediaHandlerGetRejectsPublicIDMatrixWithoutRepositoryCalls(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, publicID := range []string{"", " ", "gen_x_0123456789abcdef0123456789abcdef", "gen_rq_0123", "gen_rq_0123456789ABCDEF0123456789ABCDEF", "gen_rq_0123456789abcdef0123456789abcdeg"} {
		repository := &leonardoMediaGetRepoStub{}
		handler := NewLeonardoMediaHandler(nil, service.NewLeonardoMediaGetService(repository, &service.LeonardoGenerationPollOrchestrator{}))
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodGet, "/v1/media/generations/test", nil)
		c.Params = gin.Params{{Key: "id", Value: publicID}}
		apiKey := leonardoMediaValidAPIKey()
		c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: apiKey.User.ID})
		handler.Get(c)
		require.Equal(t, http.StatusBadRequest, recorder.Code, "publicID=%q", publicID)
		require.Zero(t, repository.reads)
	}
}

func TestLeonardoMediaHandlerGetRequiresConfiguredServiceWithoutCalls(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/media/generations/gen_rq_0123456789abcdef0123456789abcdef", nil)
	var handler *LeonardoMediaHandler
	handler.Get(c)
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func leonardoMediaValidAPIKey() *service.APIKey {
	groupID := int64(7)
	user := &service.User{ID: 11}
	return &service.APIKey{ID: 13, UserID: user.ID, User: user, GroupID: &groupID, Group: &service.Group{ID: groupID, Platform: service.PlatformLeonardo}}
}
