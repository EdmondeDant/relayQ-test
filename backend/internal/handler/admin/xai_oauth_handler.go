package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type XAIOAuthHandler struct {
	xaiOAuthService *service.XAIOAuthService
	adminService    service.AdminService
}

func NewXAIOAuthHandler(xaiOAuthService *service.XAIOAuthService, adminService service.AdminService) *XAIOAuthHandler {
	return &XAIOAuthHandler{xaiOAuthService: xaiOAuthService, adminService: adminService}
}

type XAIGenerateAuthURLRequest struct {
	ProxyID     *int64 `json:"proxy_id"`
	RedirectURI string `json:"redirect_uri"`
}

func (h *XAIOAuthHandler) GenerateAuthURL(c *gin.Context) {
	var req XAIGenerateAuthURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = XAIGenerateAuthURLRequest{}
	}
	result, err := h.xaiOAuthService.GenerateAuthURL(c.Request.Context(), req.ProxyID, req.RedirectURI)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

type XAIExchangeCodeRequest struct {
	SessionID   string `json:"session_id" binding:"required"`
	Code        string `json:"code" binding:"required"`
	State       string `json:"state" binding:"required"`
	RedirectURI string `json:"redirect_uri"`
	ProxyID     *int64 `json:"proxy_id"`
}

func (h *XAIOAuthHandler) ExchangeCode(c *gin.Context) {
	var req XAIExchangeCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	tokenInfo, err := h.xaiOAuthService.ExchangeCode(c.Request.Context(), &service.XAIExchangeCodeInput{SessionID: req.SessionID, Code: req.Code, State: req.State, RedirectURI: req.RedirectURI, ProxyID: req.ProxyID})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tokenInfo)
}

type XAIRefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
	RT           string `json:"rt"`
	ClientID     string `json:"client_id"`
	ProxyID      *int64 `json:"proxy_id"`
}

func (h *XAIOAuthHandler) RefreshToken(c *gin.Context) {
	var req XAIRefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	refreshToken := strings.TrimSpace(req.RefreshToken)
	if refreshToken == "" {
		refreshToken = strings.TrimSpace(req.RT)
	}
	if refreshToken == "" {
		response.BadRequest(c, "refresh_token is required")
		return
	}
	var proxyURL string
	if req.ProxyID != nil {
		if proxy, err := h.adminService.GetProxy(c.Request.Context(), *req.ProxyID); err == nil && proxy != nil {
			proxyURL = proxy.URL()
		}
	}
	clientID := strings.TrimSpace(req.ClientID)
	if clientID == "" {
		clientID = xai.ClientID
	}
	tokenInfo, err := h.xaiOAuthService.RefreshTokenWithClientID(c.Request.Context(), refreshToken, proxyURL, clientID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tokenInfo)
}

func (h *XAIOAuthHandler) RefreshAccountToken(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	account, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if account.Platform != service.PlatformXAI {
		response.BadRequest(c, "Account platform does not match OAuth endpoint")
		return
	}
	if !account.IsOAuth() {
		response.BadRequest(c, "Cannot refresh non-OAuth account credentials")
		return
	}
	tokenInfo, err := h.xaiOAuthService.RefreshAccountToken(c.Request.Context(), account)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	newCredentials := h.xaiOAuthService.BuildAccountCredentials(tokenInfo)
	for k, v := range account.Credentials {
		if _, exists := newCredentials[k]; !exists {
			newCredentials[k] = v
		}
	}
	updatedAccount, err := h.adminService.UpdateAccount(c.Request.Context(), accountID, &service.UpdateAccountInput{Credentials: newCredentials})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.AccountFromService(updatedAccount))
}

func (h *XAIOAuthHandler) CreateAccountFromOAuth(c *gin.Context) {
	var req struct {
		SessionID   string  `json:"session_id" binding:"required"`
		Code        string  `json:"code" binding:"required"`
		State       string  `json:"state" binding:"required"`
		RedirectURI string  `json:"redirect_uri"`
		ProxyID     *int64  `json:"proxy_id"`
		Name        string  `json:"name"`
		Concurrency int     `json:"concurrency"`
		Priority    int     `json:"priority"`
		GroupIDs    []int64 `json:"group_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	tokenInfo, err := h.xaiOAuthService.ExchangeCode(c.Request.Context(), &service.XAIExchangeCodeInput{SessionID: req.SessionID, Code: req.Code, State: req.State, RedirectURI: req.RedirectURI, ProxyID: req.ProxyID})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	credentials := h.xaiOAuthService.BuildAccountCredentials(tokenInfo)
	name := req.Name
	if name == "" && tokenInfo.Email != "" {
		name = tokenInfo.Email
	}
	if name == "" {
		name = "Grok OAuth Account"
	}
	account, err := h.adminService.CreateAccount(c.Request.Context(), &service.CreateAccountInput{Name: name, Platform: service.PlatformXAI, Type: service.AccountTypeOAuth, Credentials: credentials, ProxyID: req.ProxyID, Concurrency: req.Concurrency, Priority: req.Priority, GroupIDs: req.GroupIDs})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.AccountFromService(account))
}
