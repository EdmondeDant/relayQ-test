package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

const leonardoAdapterGenerationID = "123e4567-e89b-42d3-a456-426614174000"

type leonardoAdapterErrorReadCloser struct {
	err error
}

func (r leonardoAdapterErrorReadCloser) Read([]byte) (int, error) {
	return 0, r.err
}

func (leonardoAdapterErrorReadCloser) Close() error {
	return nil
}

type leonardoGenerationUpstreamMock struct {
	response    *http.Response
	err         error
	wrote       bool
	calls       int
	request     *http.Request
	proxyURL    string
	accountID   int64
	concurrency int
}

func (m *leonardoGenerationUpstreamMock) Do(req *http.Request, proxyURL string, accountID int64, concurrency int) (*http.Response, error) {
	m.calls++
	m.request = req
	m.proxyURL = proxyURL
	m.accountID = accountID
	m.concurrency = concurrency
	if m.wrote {
		httptrace.ContextClientTrace(req.Context()).WroteRequest(httptrace.WroteRequestInfo{})
	}
	return m.response, m.err
}

func (m *leonardoGenerationUpstreamMock) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return m.Do(req, proxyURL, accountID, concurrency)
}

func TestLeonardoGenerationAdapterAccountBinding(t *testing.T) {
	proxyID := int64(19)
	account := &Account{
		ID: 41, Platform: PlatformLeonardo, Type: AccountTypeAPIKey, Concurrency: 7,
		Credentials: map[string]any{"api_key": "secret-key", "base_url": "https://leonardo.example/api/rest"},
		ProxyID:     &proxyID, Proxy: &Proxy{Protocol: "http", Host: "proxy.example", Port: 8080},
	}
	upstream := &leonardoGenerationUpstreamMock{response: leonardoGenerationResponse(http.StatusOK, fmt.Sprintf(`{"generationId":%q}`, leonardoAdapterGenerationID))}
	adapter, err := NewLeonardoGenerationAdapter(account, upstream, leonardoGenerationConfig())
	require.NoError(t, err)

	response, err := adapter.CreateGeneration(context.Background(), leonardo.CreateGenerationRequest{Model: "model", Parameters: map[string]any{"prompt": "cat"}})

	require.NoError(t, err)
	require.Equal(t, leonardoAdapterGenerationID, response.GenerationID)
	require.Equal(t, 1, upstream.calls)
	require.Equal(t, http.MethodPost, upstream.request.Method)
	require.Equal(t, "https://leonardo.example/api/rest/v2/generations", upstream.request.URL.String())
	require.Equal(t, "Bearer secret-key", upstream.request.Header.Get("Authorization"))
	require.Equal(t, "http://proxy.example:8080", upstream.proxyURL)
	require.Equal(t, account.ID, upstream.accountID)
	require.Equal(t, account.Concurrency, upstream.concurrency)
}

func TestLeonardoGenerationAdapterUsesProxyOnlyWhenFullyBound(t *testing.T) {
	proxyID := int64(19)
	for _, test := range []struct {
		name    string
		proxyID *int64
		proxy   *Proxy
	}{
		{name: "neither"},
		{name: "id only", proxyID: &proxyID},
		{name: "proxy only", proxy: &Proxy{Protocol: "http", Host: "proxy.example", Port: 8080}},
	} {
		t.Run(test.name, func(t *testing.T) {
			account := leonardoGenerationAccount()
			account.ProxyID = test.proxyID
			account.Proxy = test.proxy
			upstream := &leonardoGenerationUpstreamMock{response: leonardoGenerationResponse(http.StatusOK, fmt.Sprintf(`{"generationId":%q}`, leonardoAdapterGenerationID))}
			adapter, err := NewLeonardoGenerationAdapter(account, upstream, leonardoGenerationConfig())
			require.NoError(t, err)
			_, err = adapter.CreateGeneration(context.Background(), leonardo.CreateGenerationRequest{Model: "model", Parameters: map[string]any{}})
			require.NoError(t, err)
			require.Empty(t, upstream.proxyURL)
		})
	}
}

func TestLeonardoGenerationAdapterValidation(t *testing.T) {
	valid := leonardoGenerationAccount()
	tests := []struct {
		name     string
		account  *Account
		upstream HTTPUpstream
		cfg      *config.Config
		want     string
	}{
		{name: "nil account", upstream: &leonardoGenerationUpstreamMock{}, cfg: leonardoGenerationConfig(), want: "account is required"},
		{name: "wrong platform", account: &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}, upstream: &leonardoGenerationUpstreamMock{}, cfg: leonardoGenerationConfig(), want: "platform is required"},
		{name: "wrong type", account: &Account{Platform: PlatformLeonardo, Type: AccountTypeOAuth}, upstream: &leonardoGenerationUpstreamMock{}, cfg: leonardoGenerationConfig(), want: "apikey type"},
		{name: "missing key", account: &Account{Platform: PlatformLeonardo, Type: AccountTypeAPIKey}, upstream: &leonardoGenerationUpstreamMock{}, cfg: leonardoGenerationConfig(), want: "API key is required"},
		{name: "invalid base URL", account: &Account{Platform: PlatformLeonardo, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "secret", "base_url": "://invalid"}}, upstream: &leonardoGenerationUpstreamMock{}, cfg: leonardoGenerationConfig(), want: "base URL is invalid"},
		{name: "insecure base URL", account: &Account{Platform: PlatformLeonardo, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "secret", "base_url": "http://leonardo.example/api/rest"}}, upstream: &leonardoGenerationUpstreamMock{}, cfg: leonardoGenerationConfig(), want: "base URL is invalid"},
		{name: "missing upstream", account: valid, cfg: leonardoGenerationConfig(), want: "HTTP upstream is required"},
		{name: "missing config", account: valid, upstream: &leonardoGenerationUpstreamMock{}, want: "config is required"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			adapter, err := NewLeonardoGenerationAdapter(test.account, test.upstream, test.cfg)
			require.Nil(t, adapter)
			require.ErrorContains(t, err, test.want)
			require.NotContains(t, err.Error(), "secret")
		})
	}
}

func TestLeonardoGenerationAdapterMapsOnlyNotWrittenSentinel(t *testing.T) {
	for _, test := range []struct {
		name         string
		wrote        bool
		status       int
		body         string
		transportErr error
		wantMapped   bool
		wantUnknown  bool
	}{
		{name: "not written", transportErr: errors.New("dial secret-key"), wantMapped: true},
		{name: "written transport", wrote: true, transportErr: errors.New("reset secret-key"), wantUnknown: true},
		{name: "redirect", status: http.StatusTemporaryRedirect, body: `{"error":"redirect"}`, wantUnknown: true},
		{name: "non 2xx", status: http.StatusBadGateway, body: `{"error":"failed"}`, wantUnknown: true},
		{name: "decode", status: http.StatusOK, body: `{`, wantUnknown: true},
		{name: "invalid id", status: http.StatusOK, body: `{"generationId":"bad"}`, wantUnknown: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			upstream := &leonardoGenerationUpstreamMock{wrote: test.wrote, err: test.transportErr}
			if test.transportErr == nil {
				upstream.response = leonardoGenerationResponse(test.status, test.body)
			}
			adapter, err := NewLeonardoGenerationAdapter(leonardoGenerationAccount(), upstream, leonardoGenerationConfig())
			require.NoError(t, err)
			_, err = adapter.CreateGeneration(context.Background(), leonardo.CreateGenerationRequest{Model: "model", Parameters: map[string]any{}})
			require.Error(t, err)
			require.Equal(t, test.wantMapped, errors.Is(err, ErrLeonardoGenerationRequestNotWritten))
			require.Equal(t, 1, upstream.calls)
			require.NotContains(t, err.Error(), "secret-key")
			if test.wantUnknown {
				var apiErr *leonardo.LeonardoError
				require.ErrorAs(t, err, &apiErr)
				require.Equal(t, leonardo.SubmissionUnknown, apiErr.SubmissionStatus)
				require.Equal(t, leonardo.SideEffectUnknown, apiErr.SideEffectStatus)
				require.False(t, apiErr.SafeToRetry)
			}
			if test.wantMapped {
				require.ErrorIs(t, err, leonardo.ErrGenerationRequestNotWritten)
			}
		})
	}
}

func TestLeonardoGenerationAdapterReadFailureIsUnknown(t *testing.T) {
	upstream := &leonardoGenerationUpstreamMock{response: &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: leonardoAdapterErrorReadCloser{err: errors.New("read secret-key")}}}
	adapter, err := NewLeonardoGenerationAdapter(leonardoGenerationAccount(), upstream, leonardoGenerationConfig())
	require.NoError(t, err)
	_, err = adapter.CreateGeneration(context.Background(), leonardo.CreateGenerationRequest{Model: "model", Parameters: map[string]any{}})
	var apiErr *leonardo.LeonardoError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, leonardo.SubmissionUnknown, apiErr.SubmissionStatus)
	require.False(t, errors.Is(err, ErrLeonardoGenerationRequestNotWritten))
	require.False(t, apiErr.SafeToRetry)
	require.NotContains(t, err.Error(), "secret-key")
}

func leonardoGenerationAccount() *Account {
	return &Account{ID: 41, Platform: PlatformLeonardo, Type: AccountTypeAPIKey, Concurrency: 7, Credentials: map[string]any{"api_key": "secret-key", "base_url": "https://leonardo.example/api/rest"}}
}

func leonardoGenerationConfig() *config.Config {
	return &config.Config{}
}

func leonardoGenerationResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}
