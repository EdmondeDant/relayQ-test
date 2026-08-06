//go:build integration

package routes

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

type leonardoRoutesFixture struct {
	userID    int64
	accountID int64
	apiKeyID  int64
	apiKey    string
}

type leonardoRoutesClock struct{}

func (leonardoRoutesClock) Now() time.Time {
	return time.Now().UTC().Add(time.Hour)
}

func TestLeonardoMediaCreateOfflineIntegration(t *testing.T) {
	ctx := context.Background()
	client, sqlDB := startLeonardoRoutesPostgres(t, ctx)

	var upstreamPosts atomic.Int32
	var upstreamGets atomic.Int32
	safe := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer upstream-secret", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			upstreamPosts.Add(1)
			require.Equal(t, "/v2/generations", r.URL.Path)
			_, _ = w.Write([]byte(`{"generationId":"11111111-1111-4111-8111-111111111111","cost":{"amount":0.003,"unit":"USD"}}`))
		case http.MethodGet:
			upstreamGets.Add(1)
			require.Equal(t, "/v1/generations/11111111-1111-4111-8111-111111111111", r.URL.Path)
			_, _ = w.Write([]byte(`{"generations_by_pk":{"id":"11111111-1111-4111-8111-111111111111","status":"COMPLETE","generated_images":[{"id":"image-1","url":"https://cdn.example/image.png","nsfw":false}]}}`))
		default:
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(upstream.Close)

	fixture := createLeonardoRoutesFixture(t, ctx, client, upstream.URL)
	cfg := &config.Config{
		RunMode:  config.RunModeSimple,
		Gateway:  config.GatewayConfig{MaxBodySize: 1 << 20},
		Leonardo: config.LeonardoConfig{ProviderEnabled: true, MediaEnabled: true},
		Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{
			AllowPrivateHosts: true,
			AllowInsecureHTTP: true,
		}},
	}
	apiKeys := repository.NewAPIKeyRepository(client, sqlDB)
	users := repository.NewUserRepository(client, sqlDB)
	groups := repository.NewGroupRepository(client, sqlDB)
	accounts := repository.NewAccountRepository(client, sqlDB, nil)
	jobs := repository.NewGenerationJobRepository(client, sqlDB)
	funds := repository.NewLeonardoImageFundsRepository(client, sqlDB)
	upstreamHTTP := repository.NewHTTPUpstream(cfg)
	apiKeyService := service.NewAPIKeyService(apiKeys, users, groups, nil, nil, nil, cfg)
	apiKeyAuth := servermiddleware.NewAPIKeyAuthMiddleware(apiKeyService, nil, cfg)
	quotes := service.NewLeonardoImageQuoteGuard(service.NewLeonardoImagePriceResolver(), service.NewLeonardoImageQuoteUserBalanceReader(users))
	clients := service.NewLeonardoImageAccountAdapterFactory(upstreamHTTP, cfg)
	orchestrator := service.NewLeonardoImageCreateOrchestrator(quotes, funds, accounts, clients, jobs)
	createService := service.NewLeonardoMediaCreateService(accounts, orchestrator)
	pollOrchestrator := service.NewLeonardoGenerationPollOrchestrator(jobs.(service.LeonardoGenerationPollRepository), accounts, upstreamHTTP, cfg, leonardoRoutesClock{}, funds)
	getService := service.NewLeonardoMediaGetService(jobs.(service.GenerationJobPollRepository), pollOrchestrator)
	leonardoHandler := handler.NewLeonardoMediaHandler(createService, getService)
	idempotencyConfig := service.DefaultIdempotencyConfig()
	idempotencyConfig.ObserveOnly = false
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(repository.NewIdempotencyRepository(client, sqlDB), idempotencyConfig))
	t.Cleanup(func() { service.SetDefaultIdempotencyCoordinator(nil) })

	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterGatewayRoutes(router, &handler.Handlers{
		Gateway:       &handler.GatewayHandler{},
		OpenAIGateway: &handler.OpenAIGatewayHandler{},
		LeonardoMedia: leonardoHandler,
	}, apiKeyAuth, apiKeyService, nil, nil, nil, cfg)

	body := `{"model":"flux-schnell","modality":"image","prompt":"draw a cat","public":false,"parameters":{"width":896,"height":896,"quantity":1}}`
	first := performLeonardoCreateRequest(router, fixture.apiKey, "offline-create-1", body)
	require.Equal(t, http.StatusOK, first.Code, first.Body.String())
	var envelope struct {
		response.Response
		Data service.LeonardoMediaCreateResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &envelope))
	require.Equal(t, "media.generation", envelope.Data.Object)
	require.Equal(t, service.PlatformLeonardo, envelope.Data.Provider)
	require.Equal(t, string(service.GenerationJobStatusQueued), envelope.Data.Status)
	require.Equal(t, int32(1), upstreamPosts.Load())

	replay := performLeonardoCreateRequest(router, fixture.apiKey, "offline-create-1", body)
	require.Equal(t, http.StatusOK, replay.Code, replay.Body.String())
	require.Equal(t, "true", replay.Header().Get("X-Idempotency-Replayed"))
	require.Equal(t, first.Body.String(), replay.Body.String())
	require.Equal(t, int32(1), upstreamPosts.Load())

	get := performLeonardoGetRequest(router, fixture.apiKey, envelope.Data.ID)
	require.Equal(t, http.StatusOK, get.Code, get.Body.String())
	var getResult service.LeonardoMediaGetResult
	require.NoError(t, json.Unmarshal(get.Body.Bytes(), &getResult))
	require.Equal(t, string(service.GenerationJobStatusSucceeded), getResult.Status)
	require.Len(t, getResult.Data, 1)
	require.Equal(t, "image-1", getResult.Data[0].ID)
	require.Equal(t, "https://cdn.example/image.png", getResult.Data[0].URL)
	require.Equal(t, safe, getResult.Data[0].NSFW)
	require.Equal(t, int32(1), upstreamGets.Load())

	terminal := performLeonardoGetRequest(router, fixture.apiKey, envelope.Data.ID)
	require.Equal(t, http.StatusOK, terminal.Code, terminal.Body.String())
	require.Equal(t, get.Body.String(), terminal.Body.String())
	require.Equal(t, int32(1), upstreamGets.Load())

	stored, err := jobs.GetByPublicID(ctx, envelope.Data.ID)
	require.NoError(t, err)
	require.Equal(t, fixture.userID, stored.UserID)
	require.Equal(t, fixture.apiKeyID, stored.APIKeyID)
	require.Equal(t, fixture.accountID, stored.AccountID)
	require.Equal(t, service.GenerationJobBillingStatusSettled, stored.BillingStatus)
	require.Equal(t, service.GenerationJobStatusSucceeded, stored.Status)
	require.NotNil(t, stored.UpstreamGenerationID)
	require.Equal(t, "11111111-1111-4111-8111-111111111111", *stored.UpstreamGenerationID)
	require.NotNil(t, stored.SubmittedAt)

	var balance string
	require.NoError(t, sqlDB.QueryRowContext(ctx, `SELECT balance::text FROM users WHERE id = $1`, fixture.userID).Scan(&balance))
	require.Equal(t, "0.97870000", balance)
	var reservations, jobsCount int
	require.NoError(t, sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM leonardo_image_funds_reservations WHERE user_id = $1 AND public_id = $2 AND status = 'settled'`, fixture.userID, envelope.Data.ID).Scan(&reservations))
	require.NoError(t, sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM generation_jobs WHERE public_id = $1`, envelope.Data.ID).Scan(&jobsCount))
	require.Equal(t, 1, reservations)
	require.Equal(t, 1, jobsCount)
}

func performLeonardoGetRequest(router http.Handler, apiKey, publicID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/v1/media/generations/"+publicID, nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func performLeonardoCreateRequest(router http.Handler, apiKey, idempotencyKey, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/media/generations", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func startLeonardoRoutesPostgres(t *testing.T, ctx context.Context) (*dbent.Client, *sql.DB) {
	t.Helper()
	container, err := tcpostgres.Run(ctx, "postgres:18.1-alpine3.23", tcpostgres.WithDatabase("sub2api_routes_test"), tcpostgres.WithUsername("postgres"), tcpostgres.WithPassword("postgres"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable", "TimeZone=UTC")
	require.NoError(t, err)
	sqlDB, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	require.Eventually(t, func() bool { return sqlDB.PingContext(ctx) == nil }, 30*time.Second, 250*time.Millisecond)
	require.NoError(t, repository.ApplyMigrations(ctx, sqlDB))
	driver := entsql.OpenDB(dialect.Postgres, sqlDB)
	client := dbent.NewClient(dbent.Driver(driver))
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	return client, sqlDB
}

func createLeonardoRoutesFixture(t *testing.T, ctx context.Context, client *dbent.Client, upstreamURL string) leonardoRoutesFixture {
	t.Helper()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	user := client.User.Create().SetEmail("leonardo-routes-" + suffix + "@example.test").SetPasswordHash("integration-not-used").SetBalance(1).SetStatus(service.StatusActive).SaveX(ctx)
	group := client.Group.Create().SetName("leonardo-routes-" + suffix).SetPlatform(service.PlatformLeonardo).SetStatus(service.StatusActive).SaveX(ctx)
	account := client.Account.Create().SetName("leonardo-upstream-" + suffix).SetPlatform(service.PlatformLeonardo).SetType(service.AccountTypeAPIKey).SetCredentials(map[string]any{"api_key": "upstream-secret", "base_url": upstreamURL}).SetStatus(service.StatusActive).SetSchedulable(true).AddGroups(group).SaveX(ctx)
	key := "sk-leonardo-routes-" + suffix
	apiKey := client.APIKey.Create().SetUserID(user.ID).SetKey(key).SetName("leonardo-routes").SetGroupID(group.ID).SetStatus(service.StatusActive).SaveX(ctx)
	return leonardoRoutesFixture{userID: user.ID, accountID: account.ID, apiKeyID: apiKey.ID, apiKey: key}
}
