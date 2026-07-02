package xai

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	ClientIDEnv     = "XAI_OAUTH_CLIENT_ID"
	DefaultClientID = "b1a00492-073a-47ea-816f-4c329264a828"

	AuthorizeURLEnv = "XAI_OAUTH_AUTHORIZE_URL"
	TokenURLEnv     = "XAI_OAUTH_TOKEN_URL"

	DefaultAuthorizeURL = "https://auth.x.ai/oauth2/authorize"
	DefaultTokenURL     = "https://auth.x.ai/oauth2/token"
	DefaultRedirectURI  = "http://127.0.0.1:1455/callback"
	DefaultScopes       = "openid profile email offline_access grok-cli:access api:access"
	RefreshScopes       = "openid profile email offline_access grok-cli:access api:access"

	SessionTTL = 30 * time.Minute
)

var (
	ClientID     = firstNonEmpty(os.Getenv(ClientIDEnv), DefaultClientID)
	AuthorizeURL = firstNonEmpty(os.Getenv(AuthorizeURLEnv), DefaultAuthorizeURL)
	TokenURL     = firstNonEmpty(os.Getenv(TokenURLEnv), DefaultTokenURL)
)

type OAuthSession struct {
	State        string    `json:"state"`
	CodeVerifier string    `json:"code_verifier"`
	ClientID     string    `json:"client_id,omitempty"`
	ProxyURL     string    `json:"proxy_url,omitempty"`
	RedirectURI  string    `json:"redirect_uri"`
	CreatedAt    time.Time `json:"created_at"`
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*OAuthSession
	stopOnce sync.Once
	stopCh   chan struct{}
}

func NewSessionStore() *SessionStore {
	store := &SessionStore{sessions: make(map[string]*OAuthSession), stopCh: make(chan struct{})}
	go store.cleanup()
	return store
}

func (s *SessionStore) Set(sessionID string, session *OAuthSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionID] = session
}

func (s *SessionStore) Get(sessionID string) (*OAuthSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[sessionID]
	if !ok || time.Since(session.CreatedAt) > SessionTTL {
		return nil, false
	}
	return session, true
}

func (s *SessionStore) Delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}
func (s *SessionStore) Stop() { s.stopOnce.Do(func() { close(s.stopCh) }) }
func (s *SessionStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.Lock()
			for id, session := range s.sessions {
				if time.Since(session.CreatedAt) > SessionTTL {
					delete(s.sessions, id)
				}
			}
			s.mu.Unlock()
		}
	}
}

func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return b, err
}
func GenerateState() (string, error) {
	b, err := GenerateRandomBytes(32)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func GenerateSessionID() (string, error) {
	b, err := GenerateRandomBytes(16)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func GenerateCodeVerifier() (string, error) {
	b, err := GenerateRandomBytes(32)
	if err != nil {
		return "", err
	}
	return base64URLEncode(b), nil
}
func GenerateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64URLEncode(h[:])
}
func base64URLEncode(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

func BuildAuthorizationURL(state, codeChallenge, redirectURI string) string {
	if redirectURI == "" {
		redirectURI = DefaultRedirectURI
	}
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", ClientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", DefaultScopes)
	params.Set("state", state)
	params.Set("nonce", state)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")
	return fmt.Sprintf("%s?%s", AuthorizeURL, params.Encode())
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

type IDTokenClaims struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Iss           string `json:"iss"`
	Aud           any    `json:"aud"`
	Exp           int64  `json:"exp"`
	Iat           int64  `json:"iat"`
}

type UserInfo struct {
	UserID string
	Email  string
	Name   string
}

func ParseIDToken(token string) (*IDTokenClaims, error) { return DecodeIDToken(token) }
func DecodeIDToken(token string) (*IDTokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid id token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims IDTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	return &claims, nil
}

func (c *IDTokenClaims) GetUserInfo() *UserInfo {
	if c == nil {
		return nil
	}
	return &UserInfo{UserID: c.Sub, Email: c.Email, Name: c.Name}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
