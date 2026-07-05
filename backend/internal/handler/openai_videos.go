package handler

import (
	"net/http"
	"strings"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// VideoGenerations handles xAI/OpenAI-compatible video generation requests through RelayQ.
// POST /v1/videos/generations or POST /v1/videos
func (h *OpenAIGatewayHandler) VideoGenerations(c *gin.Context) {
	h.forwardXAIVideoSubmit(c, "generation")
}

// VideoEdits handles xAI/OpenAI-compatible video edit requests through RelayQ.
// POST /v1/videos/edits
func (h *OpenAIGatewayHandler) VideoEdits(c *gin.Context) {
	h.forwardXAIVideoSubmit(c, "edit")
}

// VideoExtensions handles xAI-compatible video extension requests through RelayQ.
// POST /v1/videos/extensions
func (h *OpenAIGatewayHandler) VideoExtensions(c *gin.Context) {
	h.forwardXAIVideoSubmit(c, "extension")
}

func (h *OpenAIGatewayHandler) forwardXAIVideoSubmit(c *gin.Context, mode string) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	if h.gatewayService == nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "Gateway service unavailable")
		return
	}
	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	forwardBody, requestModel, err := service.NormalizeXAIVideoGenerationBodyForHandler(body)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	setOpsRequestContext(c, requestModel, false)
	setOpsEndpointContext(c, "", int16(service.RequestTypeFromLegacy(false, false)))

	selection, _, err := h.gatewayService.SelectAccountWithSchedulerForCapability(
		c.Request.Context(),
		apiKey.GroupID,
		"",
		h.gatewayService.GenerateExplicitSessionHash(c, forwardBody),
		requestModel,
		nil,
		service.OpenAIUpstreamTransportHTTPSSE,
		service.OpenAIEndpointCapabilityChatCompletions,
		false,
	)
	if err != nil || selection == nil || selection.Account == nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available xAI video accounts")
		return
	}
	defer func() {
		if selection.ReleaseFunc != nil {
			selection.ReleaseFunc()
		}
	}()
	if selection.Account.Platform != service.PlatformXAI {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available xAI video accounts")
		return
	}

	switch mode {
	case "edit":
		err = h.gatewayService.ForwardXAIVideoEdit(c.Request.Context(), c, selection.Account, forwardBody)
	case "extension":
		err = h.gatewayService.ForwardXAIVideoExtension(c.Request.Context(), c, selection.Account, forwardBody)
	default:
		err = h.gatewayService.ForwardXAIVideoGeneration(c.Request.Context(), c, selection.Account, forwardBody)
	}
	if err != nil {
		h.errorResponse(c, http.StatusBadGateway, "api_error", err.Error())
		return
	}
}

// VideoStatus handles xAI video polling requests through RelayQ.
// GET /v1/videos/:request_id
func (h *OpenAIGatewayHandler) VideoStatus(c *gin.Context) {
	h.forwardXAIVideoLookup(c, false)
}

// VideoContent handles OpenAI/Sora-compatible video content downloads.
// GET /v1/videos/:request_id/content
func (h *OpenAIGatewayHandler) VideoContent(c *gin.Context) {
	h.forwardXAIVideoLookup(c, true)
}

func (h *OpenAIGatewayHandler) forwardXAIVideoLookup(c *gin.Context, content bool) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	if h.gatewayService == nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "Gateway service unavailable")
		return
	}
	requestID := strings.TrimSpace(c.Param("request_id"))
	if requestID == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "request_id is required")
		return
	}
	account, stickyHit := h.gatewayService.ResolveXAIVideoRequestAccount(c.Request.Context(), apiKey.GroupID, requestID)
	var releaseFunc func()
	if !stickyHit {
		selection, _, err := h.gatewayService.SelectAccountWithSchedulerForCapability(
			c.Request.Context(),
			apiKey.GroupID,
			"",
			requestID,
			"grok-imagine-video",
			nil,
			service.OpenAIUpstreamTransportHTTPSSE,
			service.OpenAIEndpointCapabilityChatCompletions,
			false,
		)
		if err != nil || selection == nil || selection.Account == nil {
			h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available xAI video accounts")
			return
		}
		account = selection.Account
		releaseFunc = selection.ReleaseFunc
	}
	defer func() {
		if releaseFunc != nil {
			releaseFunc()
		}
	}()
	if account == nil || account.Platform != service.PlatformXAI {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available xAI video accounts")
		return
	}
	var err error
	if content {
		err = h.gatewayService.ForwardXAIVideoContent(c.Request.Context(), c, account, requestID)
	} else {
		err = h.gatewayService.ForwardXAIVideoStatus(c.Request.Context(), c, account, requestID)
	}
	if err != nil {
		h.errorResponse(c, http.StatusBadGateway, "api_error", err.Error())
		return
	}
}
