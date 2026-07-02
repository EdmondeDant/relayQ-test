package service

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
)

type XAIOAuthClient interface {
	ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI, proxyURL, clientID string) (*xai.TokenResponse, error)
	RefreshToken(ctx context.Context, refreshToken, proxyURL string) (*xai.TokenResponse, error)
	RefreshTokenWithClientID(ctx context.Context, refreshToken, proxyURL string, clientID string) (*xai.TokenResponse, error)
}

type XAIOAuthService struct {
	sessionStore *xai.SessionStore
	proxyRepo    ProxyRepository
	oauthClient  XAIOAuthClient
}

func NewXAIOAuthService(proxyRepo ProxyRepository, oauthClient XAIOAuthClient) *XAIOAuthService {
	return &XAIOAuthService{sessionStore: xai.NewSessionStore(), proxyRepo: proxyRepo, oauthClient: oauthClient}
}

type XAIAuthURLResult struct {
	AuthURL      string `json:"auth_url"`
	SessionID    string `json:"session_id"`
	CodeVerifier string `json:"code_verifier"`
}

func (s *XAIOAuthService) GenerateAuthURL(ctx context.Context, proxyID *int64, redirectURI string) (*XAIAuthURLResult, error) {
	state, err := xai.GenerateState()
	if err != nil {
		return nil, infraerrors.Newf(http.StatusInternalServerError, "XAI_OAUTH_STATE_FAILED", "failed to generate state: %v", err)
	}
	codeVerifier, err := xai.GenerateCodeVerifier()
	if err != nil {
		return nil, infraerrors.Newf(http.StatusInternalServerError, "XAI_OAUTH_VERIFIER_FAILED", "failed to generate code verifier: %v", err)
	}
	sessionID, err := xai.GenerateSessionID()
	if err != nil {
		return nil, infraerrors.Newf(http.StatusInternalServerError, "XAI_OAUTH_SESSION_FAILED", "failed to generate session ID: %v", err)
	}
	var proxyURL string
	if proxyID != nil {
		proxy, err := s.proxyRepo.GetByID(ctx, *proxyID)
		if err != nil {
			return nil, infraerrors.Newf(http.StatusBadRequest, "XAI_OAUTH_PROXY_NOT_FOUND", "proxy not found: %v", err)
		}
		if proxy != nil {
			proxyURL = proxy.URL()
		}
	}
	if redirectURI == "" {
		redirectURI = xai.DefaultRedirectURI
	}
	s.sessionStore.Set(sessionID, &xai.OAuthSession{State: state, CodeVerifier: codeVerifier, ClientID: xai.ClientID, RedirectURI: redirectURI, ProxyURL: proxyURL, CreatedAt: time.Now()})
	return &XAIAuthURLResult{AuthURL: xai.BuildAuthorizationURL(state, xai.GenerateCodeChallenge(codeVerifier), redirectURI), SessionID: sessionID, CodeVerifier: codeVerifier}, nil
}

type XAIExchangeCodeInput struct {
	SessionID   string
	Code        string
	State       string
	RedirectURI string
	ProxyID     *int64
}

type XAITokenInfo struct {
	AccessToken        string `json:"access_token"`
	RefreshToken       string `json:"refresh_token"`
	IDToken            string `json:"id_token,omitempty"`
	TokenType          string `json:"token_type,omitempty"`
	ExpiresIn          int64  `json:"expires_in"`
	ExpiresAt          int64  `json:"expires_at"`
	ClientID           string `json:"client_id,omitempty"`
	Email              string `json:"email,omitempty"`
	Name               string `json:"name,omitempty"`
	XAIUserID          string `json:"xai_user_id,omitempty"`
	SubscriptionPlan   string `json:"subscription_plan,omitempty"`
	SubscriptionStatus string `json:"subscription_status,omitempty"`
}

func (s *XAIOAuthService) ExchangeCode(ctx context.Context, input *XAIExchangeCodeInput) (*XAITokenInfo, error) {
	session, ok := s.sessionStore.Get(input.SessionID)
	if !ok {
		return nil, infraerrors.New(http.StatusBadRequest, "XAI_OAUTH_SESSION_NOT_FOUND", "session not found or expired")
	}
	if input.State == "" {
		return nil, infraerrors.New(http.StatusBadRequest, "XAI_OAUTH_STATE_REQUIRED", "oauth state is required")
	}
	if subtle.ConstantTimeCompare([]byte(input.State), []byte(session.State)) != 1 {
		return nil, infraerrors.New(http.StatusBadRequest, "XAI_OAUTH_INVALID_STATE", "invalid oauth state")
	}
	proxyURL := session.ProxyURL
	if input.ProxyID != nil {
		proxy, err := s.proxyRepo.GetByID(ctx, *input.ProxyID)
		if err != nil {
			return nil, infraerrors.Newf(http.StatusBadRequest, "XAI_OAUTH_PROXY_NOT_FOUND", "proxy not found: %v", err)
		}
		if proxy != nil {
			proxyURL = proxy.URL()
		}
	}
	redirectURI := session.RedirectURI
	if input.RedirectURI != "" {
		redirectURI = input.RedirectURI
	}
	clientID := strings.TrimSpace(session.ClientID)
	if clientID == "" {
		clientID = xai.ClientID
	}
	tokenResp, err := s.oauthClient.ExchangeCode(ctx, input.Code, session.CodeVerifier, redirectURI, proxyURL, clientID)
	if err != nil {
		return nil, err
	}
	s.sessionStore.Delete(input.SessionID)
	return s.buildTokenInfo(tokenResp, clientID), nil
}

func (s *XAIOAuthService) RefreshToken(ctx context.Context, refreshToken, proxyURL string) (*XAITokenInfo, error) {
	return s.RefreshTokenWithClientID(ctx, refreshToken, proxyURL, "")
}
func (s *XAIOAuthService) RefreshTokenWithClientID(ctx context.Context, refreshToken, proxyURL, clientID string) (*XAITokenInfo, error) {
	tokenResp, err := s.oauthClient.RefreshTokenWithClientID(ctx, refreshToken, proxyURL, clientID)
	if err != nil {
		return nil, err
	}
	return s.buildTokenInfo(tokenResp, clientID), nil
}

func (s *XAIOAuthService) RefreshAccountToken(ctx context.Context, account *Account) (*XAITokenInfo, error) {
	if account.Platform != PlatformXAI {
		return nil, infraerrors.New(http.StatusBadRequest, "XAI_OAUTH_INVALID_ACCOUNT", "account is not an XAI account")
	}
	if account.Type != AccountTypeOAuth {
		return nil, infraerrors.New(http.StatusBadRequest, "XAI_OAUTH_INVALID_ACCOUNT_TYPE", "account is not an OAuth account")
	}
	var proxyURL string
	if account.ProxyID != nil {
		if proxy, err := s.proxyRepo.GetByID(ctx, *account.ProxyID); err == nil && proxy != nil {
			proxyURL = proxy.URL()
		}
	}
	refreshToken := account.GetCredential("refresh_token")
	if refreshToken == "" {
		return nil, infraerrors.New(http.StatusBadRequest, "XAI_OAUTH_NO_REFRESH_TOKEN", "no refresh token available")
	}
	return s.RefreshTokenWithClientID(ctx, refreshToken, proxyURL, account.GetCredential("client_id"))
}

func (s *XAIOAuthService) BuildAccountCredentials(tokenInfo *XAITokenInfo) map[string]any {
	creds := map[string]any{"access_token": tokenInfo.AccessToken, "expires_at": time.Unix(tokenInfo.ExpiresAt, 0).Format(time.RFC3339)}
	if strings.TrimSpace(tokenInfo.RefreshToken) != "" {
		creds["refresh_token"] = tokenInfo.RefreshToken
	}
	if tokenInfo.IDToken != "" {
		creds["id_token"] = tokenInfo.IDToken
	}
	if tokenInfo.TokenType != "" {
		creds["token_type"] = tokenInfo.TokenType
	}
	if tokenInfo.Email != "" {
		creds["email"] = tokenInfo.Email
	}
	if tokenInfo.Name != "" {
		creds["name"] = tokenInfo.Name
	}
	if tokenInfo.XAIUserID != "" {
		creds["xai_user_id"] = tokenInfo.XAIUserID
	}
	if tokenInfo.SubscriptionPlan != "" {
		creds["subscription_plan"] = tokenInfo.SubscriptionPlan
	}
	if tokenInfo.SubscriptionStatus != "" {
		creds["subscription_status"] = tokenInfo.SubscriptionStatus
	}
	if strings.TrimSpace(tokenInfo.ClientID) != "" {
		creds["client_id"] = strings.TrimSpace(tokenInfo.ClientID)
	}
	return creds
}

func (s *XAIOAuthService) Stop() { s.sessionStore.Stop() }

func (s *XAIOAuthService) buildTokenInfo(tokenResp *xai.TokenResponse, clientID string) *XAITokenInfo {
	expiresIn := tokenResp.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	info := &XAITokenInfo{AccessToken: tokenResp.AccessToken, RefreshToken: tokenResp.RefreshToken, IDToken: tokenResp.IDToken, TokenType: tokenResp.TokenType, ExpiresIn: expiresIn, ExpiresAt: time.Now().Unix() + expiresIn, ClientID: strings.TrimSpace(clientID)}
	if info.ClientID == "" {
		info.ClientID = xai.ClientID
	}
	if tokenResp.IDToken != "" {
		claims, err := xai.ParseIDToken(tokenResp.IDToken)
		if err != nil {
			slog.Warn("xai_oauth_id_token_parse_failed", "error", err)
		} else if userInfo := claims.GetUserInfo(); userInfo != nil {
			info.Email = userInfo.Email
			info.Name = userInfo.Name
			info.XAIUserID = userInfo.UserID
		}
	}
	return info
}
