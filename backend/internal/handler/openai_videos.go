package handler

import (
	"net/http"
	"strings"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// VideoGenerations handles xAI video generation requests through RelayQ.
// POST /v1/videos/generations
func (h *OpenAIGatewayHandler) VideoGenerations(c *gin.Context) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
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
	if err := h.gatewayService.ForwardXAIVideoGeneration(c.Request.Context(), c, selection.Account, forwardBody); err != nil {
		h.errorResponse(c, http.StatusBadGateway, "api_error", err.Error())
		return
	}
}

// VideoStatus handles xAI video polling requests through RelayQ.
// GET /v1/videos/:request_id
func (h *OpenAIGatewayHandler) VideoStatus(c *gin.Context) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	requestID := strings.TrimSpace(c.Param("request_id"))
	if requestID == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "request_id is required")
		return
	}
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
	defer func() {
		if selection.ReleaseFunc != nil {
			selection.ReleaseFunc()
		}
	}()
	if selection.Account.Platform != service.PlatformXAI {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available xAI video accounts")
		return
	}
	if err := h.gatewayService.ForwardXAIVideoStatus(c.Request.Context(), c, selection.Account, requestID); err != nil {
		h.errorResponse(c, http.StatusBadGateway, "api_error", err.Error())
		return
	}
}
