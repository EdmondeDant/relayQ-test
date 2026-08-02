package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
)

type LeonardoGenerationAdapter struct {
	client *leonardo.Client
}

type leonardoGenerationRoundTripper struct {
	upstream    HTTPUpstream
	proxyURL    string
	accountID   int64
	concurrency int
}

func NewLeonardoGenerationAdapter(account *Account, upstream HTTPUpstream, cfg *config.Config) (*LeonardoGenerationAdapter, error) {
	if account == nil {
		return nil, errors.New("leonardo account is required")
	}
	if account.Platform != PlatformLeonardo {
		return nil, errors.New("leonardo account platform is required")
	}
	if account.Type != AccountTypeAPIKey {
		return nil, errors.New("leonardo account requires apikey type")
	}
	apiKey := account.GetLeonardoAPIKey()
	if apiKey == "" {
		return nil, errors.New("leonardo API key is required")
	}
	baseURL := strings.TrimSpace(account.GetLeonardoBaseURL())
	if baseURL == "" {
		return nil, errors.New("leonardo base URL is required")
	}
	if upstream == nil {
		return nil, errors.New("leonardo HTTP upstream is required")
	}
	if cfg == nil {
		return nil, errors.New("leonardo config is required")
	}
	var normalizedBaseURL string
	var err error
	if cfg.Security.URLAllowlist.Enabled {
		normalizedBaseURL, err = urlvalidator.ValidateHTTPSURL(baseURL, urlvalidator.ValidationOptions{
			AllowedHosts:     cfg.Security.URLAllowlist.UpstreamHosts,
			RequireAllowlist: true,
			AllowPrivate:     cfg.Security.URLAllowlist.AllowPrivateHosts,
		})
	} else {
		normalizedBaseURL, err = urlvalidator.ValidateURLFormat(baseURL, cfg.Security.URLAllowlist.AllowInsecureHTTP)
	}
	if err != nil {
		return nil, errors.New("leonardo account base URL is invalid")
	}
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	client, err := leonardo.NewClientWithHTTPClient(normalizedBaseURL, apiKey, leonardo.DefaultTimeout, &http.Client{Transport: leonardoGenerationRoundTripper{
		upstream: upstream, proxyURL: proxyURL, accountID: account.ID, concurrency: account.Concurrency,
	}})
	if err != nil {
		return nil, errors.New("leonardo account base URL is invalid")
	}
	return &LeonardoGenerationAdapter{client: client}, nil
}

func (a *LeonardoGenerationAdapter) CreateGeneration(ctx context.Context, request leonardo.CreateGenerationRequest) (*leonardo.CreateGenerationResponse, error) {
	if a == nil || a.client == nil {
		return nil, errors.New("leonardo generation adapter is not configured")
	}
	response, err := a.client.CreateGeneration(ctx, request)
	if errors.Is(err, leonardo.ErrGenerationRequestNotWritten) {
		return nil, fmt.Errorf("%w: %w", ErrLeonardoGenerationRequestNotWritten, err)
	}
	return response, err
}

func (t leonardoGenerationRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.upstream.Do(req, t.proxyURL, t.accountID, t.concurrency)
}
