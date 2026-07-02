package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func isXAIOAuthAccount(account *Account) bool {
	return account != nil && account.Platform == PlatformXAI && account.Type == AccountTypeOAuth
}

func isXAIBadCredentials(body []byte) bool {
	text := strings.ToLower(string(body))
	return strings.Contains(text, "bad-credentials") || strings.Contains(text, "access token could not be validated")
}

func (s *OpenAIGatewayService) forceRefreshXAIOAuthAccount(ctx context.Context, account *Account) (string, error) {
	if s == nil || s.xaiOAuthService == nil || !isXAIOAuthAccount(account) {
		if account == nil {
			return "", nil
		}
		return account.GetOpenAIAccessToken(), nil
	}

	tokenInfo, err := s.xaiOAuthService.RefreshAccountToken(ctx, account)
	if err != nil {
		return "", err
	}

	newCredentials := s.xaiOAuthService.BuildAccountCredentials(tokenInfo)
	for k, v := range account.Credentials {
		if _, exists := newCredentials[k]; !exists {
			newCredentials[k] = v
		}
	}
	account.Credentials = newCredentials

	if s.accountRepo != nil {
		if updateErr := s.accountRepo.Update(ctx, account); updateErr != nil {
			return tokenInfo.AccessToken, nil
		}
	}

	return account.GetOpenAIAccessToken(), nil
}

func (s *OpenAIGatewayService) buildXAIOAuthChatCompletionsRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	token string,
	isStream bool,
) (*http.Request, error) {
	if !isXAIOAuthAccount(account) {
		return nil, fmt.Errorf("account %d is not an xai oauth account", account.ID)
	}

	baseURL := account.GetOpenAIBaseURL()
	if baseURL == "" {
		baseURL = "https://api.x.ai"
	}
	validatedURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid xai base_url: %w", err)
	}

	targetURL := buildOpenAIChatCompletionsURL(validatedURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	if isStream {
		req.Header.Set("Accept", "text/event-stream")
	} else {
		req.Header.Set("Accept", "application/json")
	}

	if c != nil && c.Request != nil {
		for key, values := range c.Request.Header {
			lowerKey := strings.ToLower(key)
			if openaiCCRawAllowedHeaders[lowerKey] {
				for _, v := range values {
					req.Header.Add(key, v)
				}
			}
		}
	}
	if customUA := account.GetOpenAIUserAgent(); customUA != "" {
		req.Header.Set("user-agent", customUA)
	}

	return req, nil
}

func (s *OpenAIGatewayService) doXAIOAuthChatCompletionsRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	isStream bool,
	token string,
) (*http.Response, error) {
	upstreamCtx, releaseUpstreamCtx := detachUpstreamContext(ctx)
	req, err := s.buildXAIOAuthChatCompletionsRequest(upstreamCtx, c, account, body, token, isStream)
	releaseUpstreamCtx()
	if err != nil {
		return nil, err
	}

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	return s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
}

func (s *OpenAIGatewayService) maybeRetryXAIOAuthChatCompletionsRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	isStream bool,
	resp *http.Response,
) (*http.Response, error) {
	if !isXAIOAuthAccount(account) || resp == nil || resp.StatusCode < http.StatusBadRequest {
		return resp, nil
	}

	respBody := s.readUpstreamErrorBody(resp)
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(respBody))

	shouldRetry := resp.StatusCode == http.StatusUnauthorized ||
		(resp.StatusCode == http.StatusForbidden && isXAIBadCredentials(respBody))
	if !shouldRetry {
		return resp, nil
	}

	refreshedToken, err := s.forceRefreshXAIOAuthAccount(ctx, account)
	if err != nil || strings.TrimSpace(refreshedToken) == "" {
		return resp, nil
	}

	return s.doXAIOAuthChatCompletionsRequest(ctx, c, account, body, isStream, refreshedToken)
}
