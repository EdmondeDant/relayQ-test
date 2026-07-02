package repository

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/imroc/req/v3"
)

func NewXAIOAuthClient() service.XAIOAuthClient { return &xaiOAuthClient{tokenURL: xai.TokenURL} }

type xaiOAuthClient struct{ tokenURL string }

func (c *xaiOAuthClient) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI, proxyURL, clientID string) (*xai.TokenResponse, error) {
	client, err := createXAIReqClient(proxyURL)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "XAI_OAUTH_CLIENT_INIT_FAILED", "create HTTP client: %v", err)
	}
	if redirectURI == "" {
		redirectURI = xai.DefaultRedirectURI
	}
	if strings.TrimSpace(clientID) == "" {
		clientID = xai.ClientID
	}
	formData := url.Values{}
	formData.Set("grant_type", "authorization_code")
	formData.Set("client_id", clientID)
	formData.Set("code", code)
	formData.Set("redirect_uri", redirectURI)
	formData.Set("code_verifier", codeVerifier)
	var tokenResp xai.TokenResponse
	resp, err := client.R().SetContext(ctx).SetHeader("User-Agent", "Mozilla/5.0").SetFormDataFromValues(formData).SetSuccessResult(&tokenResp).Post(c.tokenURL)
	if err != nil {
		if shouldReturnXAINoProxyHint(ctx, proxyURL, err) {
			return nil, newXAINoProxyHintError(err)
		}
		return nil, infraerrors.Newf(http.StatusBadGateway, "XAI_OAUTH_REQUEST_FAILED", "request failed: %v", err)
	}
	if !resp.IsSuccessState() {
		return nil, infraerrors.Newf(http.StatusBadGateway, "XAI_OAUTH_TOKEN_EXCHANGE_FAILED", "token exchange failed: status %d, body: %s", resp.StatusCode, resp.String())
	}
	return &tokenResp, nil
}

func (c *xaiOAuthClient) RefreshToken(ctx context.Context, refreshToken, proxyURL string) (*xai.TokenResponse, error) {
	return c.RefreshTokenWithClientID(ctx, refreshToken, proxyURL, "")
}
func (c *xaiOAuthClient) RefreshTokenWithClientID(ctx context.Context, refreshToken, proxyURL, clientID string) (*xai.TokenResponse, error) {
	client, err := createXAIReqClient(proxyURL)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "XAI_OAUTH_CLIENT_INIT_FAILED", "create HTTP client: %v", err)
	}
	if strings.TrimSpace(clientID) == "" {
		clientID = xai.ClientID
	}
	formData := url.Values{}
	formData.Set("grant_type", "refresh_token")
	formData.Set("refresh_token", refreshToken)
	formData.Set("client_id", clientID)
	formData.Set("scope", xai.RefreshScopes)
	var tokenResp xai.TokenResponse
	resp, err := client.R().SetContext(ctx).SetHeader("User-Agent", "Mozilla/5.0").SetFormDataFromValues(formData).SetSuccessResult(&tokenResp).Post(c.tokenURL)
	if err != nil {
		if shouldReturnXAINoProxyHint(ctx, proxyURL, err) {
			return nil, newXAINoProxyHintError(err)
		}
		return nil, infraerrors.Newf(http.StatusBadGateway, "XAI_OAUTH_REQUEST_FAILED", "request failed: %v", err)
	}
	if !resp.IsSuccessState() {
		return nil, infraerrors.Newf(http.StatusBadGateway, "XAI_OAUTH_TOKEN_REFRESH_FAILED", "token refresh failed: status %d, body: %s", resp.StatusCode, resp.String())
	}
	return &tokenResp, nil
}

func createXAIReqClient(proxyURL string) (*req.Client, error) {
	return getSharedReqClient(reqClientOptions{ProxyURL: proxyURL, Timeout: 120 * time.Second})
}
func shouldReturnXAINoProxyHint(ctx context.Context, proxyURL string, err error) bool {
	if strings.TrimSpace(proxyURL) != "" || err == nil {
		return false
	}
	if ctx != nil && ctx.Err() != nil {
		return false
	}
	return !errors.Is(err, context.Canceled)
}
func newXAINoProxyHintError(cause error) error {
	return infraerrors.New(http.StatusBadGateway, "XAI_OAUTH_PROXY_REQUIRED", "XAI/Grok OAuth request failed: no proxy is configured and this server could not reach xAI directly. Select a proxy that can access xAI, then retry; if the authorization code has expired, regenerate the authorization URL.").WithCause(cause)
}
