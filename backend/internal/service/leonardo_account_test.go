//go:build unit

package service

import (
	"context"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type leonardoAccountRepoStub struct {
	mockAccountRepoForGemini
	account     *Account
	createCalls int
	updateCalls int
	bulkCalls   int
	bindCalls   int
	accounts    []*Account
}

type leonardoHTTPSpy struct {
	calls int
}

func (s *leonardoHTTPSpy) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	s.calls++
	return nil, nil
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

func TestAccountTestServiceLeonardoDispatchIsExplicit(t *testing.T) {
	account := &Account{ID: 301, Platform: PlatformLeonardo, Type: AccountTypeAPIKey}
	spy := &leonardoHTTPSpy{}
	svc := &AccountTestService{accountRepo: &leonardoAccountRepoStub{account: account}, httpUpstream: spy}
	c, recorder := newLeonardoTestContext()

	err := svc.TestAccountConnection(c, account.ID, "", "", "")

	require.EqualError(t, err, "Leonardo account test is not available until the Leonardo client is configured")
	require.Contains(t, recorder.Body.String(), "Leonardo account test is not available until the Leonardo client is configured")
	require.Zero(t, spy.calls)
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
