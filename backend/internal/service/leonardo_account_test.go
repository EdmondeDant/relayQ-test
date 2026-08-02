//go:build unit

package service

import (
	"context"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type leonardoAccountRepoStub struct {
	mockAccountRepoForGemini
	account     *Account
	getCalls    int
	createCalls int
	updateCalls int
	bulkCalls   int
	bindCalls   int
	accounts    []*Account
}

func leonardoInt64Ptr(value int64) *int64 { return &value }

type leonardoHTTPSpy struct {
	calls       int
	getCalls    int
	postCalls   int
	proxyURL    string
	accountID   int64
	concurrency int
	request     *http.Request
	response    *http.Response
	err         error
}

func (s *leonardoHTTPSpy) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	s.calls++
	if req.Method == http.MethodGet {
		s.getCalls++
	}
	if req.Method == http.MethodPost {
		s.postCalls++
	}
	s.request = req
	s.proxyURL = proxyURL
	s.accountID = accountID
	s.concurrency = accountConcurrency
	return s.response, s.err
}

func (s *leonardoHTTPSpy) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	s.calls++
	return nil, nil
}

func newLeonardoTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/1/test", nil)
	return c, recorder
}

func (r *leonardoAccountRepoStub) Create(ctx context.Context, account *Account) error {
	r.createCalls++
	r.account = account
	return nil
}

func (r *leonardoAccountRepoStub) GetByID(ctx context.Context, id int64) (*Account, error) {
	r.getCalls++
	return r.account, nil
}

func (r *leonardoAccountRepoStub) Update(ctx context.Context, account *Account) error {
	r.updateCalls++
	r.account = account
	return nil
}

func (r *leonardoAccountRepoStub) GetByIDs(ctx context.Context, ids []int64) ([]*Account, error) {
	return r.accounts, nil
}

func (r *leonardoAccountRepoStub) BulkUpdate(ctx context.Context, ids []int64, updates AccountBulkUpdate) (int64, error) {
	r.bulkCalls++
	return int64(len(ids)), nil
}

func (r *leonardoAccountRepoStub) BindGroups(ctx context.Context, accountID int64, groupIDs []int64) error {
	r.bindCalls++
	return nil
}

func TestLeonardoAccountHelpers(t *testing.T) {
	var nilAccount *Account
	require.False(t, nilAccount.IsLeonardo())
	require.Empty(t, nilAccount.GetLeonardoAPIKey())
	require.Empty(t, nilAccount.GetLeonardoBaseURL())

	account := &Account{
		Platform: PlatformLeonardo,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "  leo-key  ",
		},
	}

	require.True(t, account.IsLeonardo())
	require.Equal(t, "leo-key", account.GetLeonardoAPIKey())
	require.Equal(t, DefaultLeonardoBaseURL, account.GetLeonardoBaseURL())

	account.Credentials["base_url"] = " https://leonardo.example/api/rest/ "
	require.Equal(t, "https://leonardo.example/api/rest", account.GetLeonardoBaseURL())

	account.Type = AccountTypeOAuth
	require.Empty(t, account.GetLeonardoAPIKey())
	require.Empty(t, account.GetLeonardoBaseURL())

	account.Platform = PlatformOpenAI
	account.Type = AccountTypeAPIKey
	require.False(t, account.IsLeonardo())
	require.Empty(t, account.GetLeonardoAPIKey())
	require.Empty(t, account.GetLeonardoBaseURL())
}

func TestAccountTestServiceLeonardoListsModelsOnce(t *testing.T) {
	account := &Account{
		ID: 301, Platform: PlatformLeonardo, Type: AccountTypeAPIKey, Concurrency: 7,
		Credentials: map[string]any{"api_key": "secret-key", "base_url": DefaultLeonardoBaseURL},
		ProxyID:     leonardoInt64Ptr(1),
		Proxy:       &Proxy{Protocol: "http", Host: "proxy.example", Port: 8080},
	}
	repo := &leonardoAccountRepoStub{account: account}
	spy := &leonardoHTTPSpy{response: leonardoHTTPResponse(http.StatusOK, `{"productionApiAvailableModels":[{"id":"model-id","name":"Model Name"}]}`)}
	svc := &AccountTestService{accountRepo: repo, httpUpstream: spy, cfg: leonardoTestConfig()}
	c, recorder := newLeonardoTestContext()

	err := svc.TestAccountConnection(c, account.ID, "", "", "")

	require.NoError(t, err)
	require.Contains(t, recorder.Body.String(), "Leonardo model list returned 1 models")
	require.Contains(t, recorder.Body.String(), `"success":true`)
	require.Equal(t, 1, repo.getCalls)
	require.Equal(t, 1, spy.calls)
	require.Equal(t, 1, spy.getCalls)
	require.Zero(t, spy.postCalls)
	require.Equal(t, http.MethodGet, spy.request.Method)
	require.Equal(t, "/api/rest/v2/models", spy.request.URL.Path)
	require.Equal(t, "Bearer secret-key", spy.request.Header.Get("Authorization"))
	require.Equal(t, "http://proxy.example:8080", spy.proxyURL)
	require.Equal(t, account.ID, spy.accountID)
	require.Equal(t, account.Concurrency, spy.concurrency)
}

func leonardoHTTPResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func leonardoTestConfig() *config.Config {
	return &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false, AllowInsecureHTTP: true}}}
}

func TestAccountTestServiceLeonardoFailures(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		baseURL   string
		upstream  HTTPUpstream
		wantError string
		wantCode  string
		secret    string
		wantCalls int
	}{
		{name: "empty key", wantError: "Leonardo API key is required"},
		{name: "nil upstream", apiKey: "secret-key", wantError: "Leonardo HTTP upstream is not configured"},
		{name: "invalid base url", apiKey: "secret-key", baseURL: "://invalid", upstream: &leonardoHTTPSpy{}, wantError: "Leonardo base URL is invalid"},
		{name: "empty models", apiKey: "secret-key", upstream: &leonardoHTTPSpy{response: leonardoHTTPResponse(http.StatusOK, `{"productionApiAvailableModels":[]}`)}, wantError: "Leonardo model list is empty", wantCalls: 1},
		{name: "upstream error", apiKey: "secret-key", upstream: &leonardoHTTPSpy{err: errors.New("dial failed secret-key authorization=secret-key")}, wantError: "Leonardo model list failed: leonardo: list models:", secret: "secret-key", wantCalls: 1},
		{name: "rate limited", apiKey: "secret-key", upstream: &leonardoHTTPSpy{response: leonardoHTTPResponse(http.StatusTooManyRequests, `{"error":"rejected secret-key","code":"rate-limit"}`)}, wantError: "HTTP 429", wantCode: "code=rate-limit", secret: "secret-key", wantCalls: 1},
		{name: "server error", apiKey: "secret-key", upstream: &leonardoHTTPSpy{response: leonardoHTTPResponse(http.StatusBadGateway, `{"error":"failed secret-key","code":"upstream-failure"}`)}, wantError: "HTTP 502", wantCode: "code=upstream-failure", secret: "secret-key", wantCalls: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			credentials := map[string]any{"api_key": tt.apiKey}
			if tt.baseURL != "" {
				credentials["base_url"] = tt.baseURL
			}
			account := &Account{ID: 303, Platform: PlatformLeonardo, Type: AccountTypeAPIKey, Credentials: credentials}
			repo := &leonardoAccountRepoStub{account: account}
			svc := &AccountTestService{accountRepo: repo, httpUpstream: tt.upstream, cfg: leonardoTestConfig()}
			c, recorder := newLeonardoTestContext()

			err := svc.TestAccountConnection(c, account.ID, "", "", "")

			result := recorder.Body.String()
			require.ErrorContains(t, err, tt.wantError)
			require.Contains(t, result, tt.wantError)
			if tt.wantCode != "" {
				require.ErrorContains(t, err, tt.wantCode)
				require.Contains(t, result, tt.wantCode)
			}
			if tt.secret != "" {
				require.NotContains(t, result, tt.secret)
			}
			require.Equal(t, 1, repo.getCalls)
			if spy, ok := tt.upstream.(*leonardoHTTPSpy); ok {
				require.Equal(t, tt.wantCalls, spy.calls)
				require.Equal(t, tt.wantCalls, spy.getCalls)
				require.Zero(t, spy.postCalls)
			}
		})
	}
}

func TestAccountTestServiceUnknownPlatformDoesNotFallBackToClaude(t *testing.T) {
	account := &Account{ID: 302, Platform: "unknown", Type: AccountTypeAPIKey}
	spy := &leonardoHTTPSpy{}
	svc := &AccountTestService{accountRepo: &leonardoAccountRepoStub{account: account}, httpUpstream: spy}
	c, recorder := newLeonardoTestContext()

	err := svc.TestAccountConnection(c, account.ID, "", "", "")

	require.EqualError(t, err, "Unsupported account platform: unknown")
	require.Contains(t, recorder.Body.String(), "Unsupported account platform: unknown")
	require.Zero(t, spy.calls)
}

func TestCreateLeonardoAccountValidation(t *testing.T) {
	tests := []struct {
		name        string
		accountType string
		credentials map[string]any
		wantError   string
		wantBaseURL string
	}{
		{name: "valid", accountType: AccountTypeAPIKey, credentials: map[string]any{"api_key": "leo-key"}, wantBaseURL: DefaultLeonardoBaseURL},
		{name: "normalized url", accountType: AccountTypeAPIKey, credentials: map[string]any{"api_key": "leo-key", "base_url": " https://leonardo.example/api/rest/ "}, wantBaseURL: "https://leonardo.example/api/rest"},
		{name: "wrong type", accountType: AccountTypeOAuth, credentials: map[string]any{"api_key": "leo-key"}, wantError: "Leonardo accounts require apikey type"},
		{name: "missing key", accountType: AccountTypeAPIKey, credentials: map[string]any{}, wantError: "Leonardo API key is required"},
		{name: "insecure base url", accountType: AccountTypeAPIKey, credentials: map[string]any{"api_key": "leo-key", "base_url": "http://cloud.leonardo.ai/api/rest"}, wantError: "invalid Leonardo base_url"},
		{name: "private base url", accountType: AccountTypeAPIKey, credentials: map[string]any{"api_key": "leo-key", "base_url": "https://127.0.0.1/api/rest"}, wantError: "invalid Leonardo base_url"},
		{name: "localhost", accountType: AccountTypeAPIKey, credentials: map[string]any{"api_key": "leo-key", "base_url": "https://localhost/api/rest"}, wantError: "invalid Leonardo base_url"},
		{name: "private network", accountType: AccountTypeAPIKey, credentials: map[string]any{"api_key": "leo-key", "base_url": "https://192.168.1.1/api/rest"}, wantError: "invalid Leonardo base_url"},
		{name: "pool mode", accountType: AccountTypeAPIKey, credentials: map[string]any{"api_key": "leo-key", "pool_mode": true}, wantError: "Leonardo accounts do not support retry or pool credentials: pool_mode"},
		{name: "retry count", accountType: AccountTypeAPIKey, credentials: map[string]any{"api_key": "leo-key", "pool_mode_retry_count": 2}, wantError: "Leonardo accounts do not support retry or pool credentials: pool_mode_retry_count"},
		{name: "retry status codes", accountType: AccountTypeAPIKey, credentials: map[string]any{"api_key": "leo-key", "pool_mode_retry_status_codes": []int{429}}, wantError: "Leonardo accounts do not support retry or pool credentials: pool_mode_retry_status_codes"},
		{name: "custom errors enabled", accountType: AccountTypeAPIKey, credentials: map[string]any{"api_key": "leo-key", "custom_error_codes_enabled": true}, wantError: "Leonardo accounts do not support retry or pool credentials: custom_error_codes_enabled"},
		{name: "custom errors", accountType: AccountTypeAPIKey, credentials: map[string]any{"api_key": "leo-key", "custom_error_codes": []int{500}}, wantError: "Leonardo accounts do not support retry or pool credentials: custom_error_codes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := maps.Clone(tt.credentials)
			repo := &leonardoAccountRepoStub{}
			svc := &adminServiceImpl{accountRepo: repo}
			_, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
				Name:                 "Leonardo",
				Platform:             PlatformLeonardo,
				Type:                 tt.accountType,
				Credentials:          tt.credentials,
				SkipDefaultGroupBind: true,
			})
			if tt.wantError == "" {
				require.NoError(t, err)
				require.Equal(t, 1, repo.createCalls)
				require.Equal(t, tt.wantBaseURL, repo.account.Credentials["base_url"])
				require.Equal(t, original, tt.credentials)
				repo.account.Credentials["api_key"] = "stored-key"
				require.Equal(t, original, tt.credentials)
				return
			}
			require.ErrorContains(t, err, tt.wantError)
			require.Zero(t, repo.createCalls)
			require.Equal(t, original, tt.credentials)
		})
	}
}

func TestUpdateLeonardoAccountValidation(t *testing.T) {
	original := &Account{
		ID:       303,
		Platform: PlatformLeonardo,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "existing-key",
		},
	}
	repo := &leonardoAccountRepoStub{account: original}
	svc := &adminServiceImpl{accountRepo: repo}

	_, err := svc.UpdateAccount(context.Background(), repo.account.ID, &UpdateAccountInput{
		Credentials: map[string]any{"base_url": "https://127.0.0.1/api/rest"},
	})

	require.ErrorContains(t, err, "invalid Leonardo base_url")
	require.Zero(t, repo.updateCalls)
	require.Equal(t, map[string]any{"api_key": "existing-key"}, original.Credentials)
}

func TestUpdateLeonardoAccountRotatesAPIKey(t *testing.T) {
	original := &Account{
		ID: 305, Platform: PlatformLeonardo, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "old-key", "base_url": DefaultLeonardoBaseURL},
	}
	repo := &leonardoAccountRepoStub{account: original}
	svc := &adminServiceImpl{accountRepo: repo}
	incoming := map[string]any{"api_key": "new-key"}

	_, err := svc.UpdateAccount(context.Background(), repo.account.ID, &UpdateAccountInput{Credentials: incoming})

	require.NoError(t, err)
	require.Equal(t, "new-key", repo.account.Credentials["api_key"])
	require.Equal(t, DefaultLeonardoBaseURL, repo.account.Credentials["base_url"])
	require.Equal(t, map[string]any{"api_key": "old-key", "base_url": DefaultLeonardoBaseURL}, original.Credentials)
	require.Equal(t, map[string]any{"api_key": "new-key"}, incoming)
}

func TestUpdateLeonardoAccountRejectsUnsupportedCredentials(t *testing.T) {
	for _, key := range []string{"pool_mode", "pool_mode_retry_count", "pool_mode_retry_status_codes", "custom_error_codes_enabled", "custom_error_codes"} {
		t.Run(key, func(t *testing.T) {
			original := &Account{
				ID: 306, Platform: PlatformLeonardo, Type: AccountTypeAPIKey,
				Credentials: map[string]any{"api_key": "existing-key", "base_url": DefaultLeonardoBaseURL},
			}
			repo := &leonardoAccountRepoStub{account: original}
			svc := &adminServiceImpl{accountRepo: repo}
			incoming := map[string]any{key: true}

			_, err := svc.UpdateAccount(context.Background(), original.ID, &UpdateAccountInput{Credentials: incoming})

			require.EqualError(t, err, "Leonardo accounts do not support retry or pool credentials: "+key)
			require.Zero(t, repo.updateCalls)
			require.Equal(t, map[string]any{"api_key": "existing-key", "base_url": DefaultLeonardoBaseURL}, original.Credentials)
			require.Equal(t, map[string]any{key: true}, incoming)
		})
	}
}

func TestBulkCredentialUpdateRejectsLeonardoBatch(t *testing.T) {
	repo := &leonardoAccountRepoStub{accounts: []*Account{
		{ID: 1, Platform: PlatformOpenAI},
		{ID: 2, Platform: PlatformLeonardo},
	}}
	svc := &adminServiceImpl{accountRepo: repo}
	credentials := map[string]any{"api_key": "replacement"}
	groupIDs := []int64{1}

	_, err := svc.BulkUpdateAccounts(context.Background(), &BulkUpdateAccountsInput{AccountIDs: []int64{1, 2}, Credentials: credentials, GroupIDs: &groupIDs})

	require.EqualError(t, err, "Leonardo credentials must be updated individually")
	require.Zero(t, repo.bulkCalls)
	require.Zero(t, repo.bindCalls)
	require.Equal(t, map[string]any{"api_key": "replacement"}, credentials)
}

func TestUpdateLeonardoAccountRejectsInvalidTypeWithoutMutation(t *testing.T) {
	original := &Account{
		ID: 307, Platform: PlatformLeonardo, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "existing-key", "base_url": DefaultLeonardoBaseURL},
	}
	repo := &leonardoAccountRepoStub{account: original}
	svc := &adminServiceImpl{accountRepo: repo}
	incoming := map[string]any{"api_key": "new-key"}

	_, err := svc.UpdateAccount(context.Background(), original.ID, &UpdateAccountInput{Type: AccountTypeOAuth, Credentials: incoming})

	require.EqualError(t, err, "Leonardo accounts require apikey type")
	require.Zero(t, repo.updateCalls)
	require.Equal(t, AccountTypeAPIKey, original.Type)
	require.Equal(t, map[string]any{"api_key": "existing-key", "base_url": DefaultLeonardoBaseURL}, original.Credentials)
	require.Equal(t, map[string]any{"api_key": "new-key"}, incoming)
}

func TestBulkNonCredentialUpdateAllowsLeonardo(t *testing.T) {
	repo := &leonardoAccountRepoStub{accounts: []*Account{{ID: 2, Platform: PlatformLeonardo}}}
	svc := &adminServiceImpl{accountRepo: repo}
	schedulable := true

	result, err := svc.BulkUpdateAccounts(context.Background(), &BulkUpdateAccountsInput{AccountIDs: []int64{2}, Schedulable: &schedulable})

	require.NoError(t, err)
	require.Equal(t, 1, result.Success)
	require.Equal(t, 1, repo.bulkCalls)
}

func TestUpdateLeonardoAccountNormalizesBaseURLAndPreservesBlankAPIKey(t *testing.T) {
	original := &Account{
		ID:       304,
		Platform: PlatformLeonardo,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "existing-key",
			"base_url": DefaultLeonardoBaseURL,
		},
	}
	repo := &leonardoAccountRepoStub{account: original}
	svc := &adminServiceImpl{accountRepo: repo}
	incoming := map[string]any{
		"api_key":  "   ",
		"base_url": " https://leonardo.example/api/rest/ ",
	}

	_, err := svc.UpdateAccount(context.Background(), repo.account.ID, &UpdateAccountInput{
		Credentials: incoming,
	})

	require.NoError(t, err)
	require.Equal(t, 1, repo.updateCalls)
	require.Equal(t, "existing-key", repo.account.Credentials["api_key"])
	require.Equal(t, "https://leonardo.example/api/rest", repo.account.Credentials["base_url"])
	require.Equal(t, map[string]any{"api_key": "existing-key", "base_url": DefaultLeonardoBaseURL}, original.Credentials)
	require.Equal(t, map[string]any{"api_key": "   ", "base_url": " https://leonardo.example/api/rest/ "}, incoming)
}
