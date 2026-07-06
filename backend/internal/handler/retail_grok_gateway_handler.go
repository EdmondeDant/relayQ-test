package handler

import (
	"errors"
	"net/http"
	"strings"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

type RetailGrokGatewayHandler struct {
	retailGateway *service.RetailGrokGatewayService
}

func NewRetailGrokGatewayHandler(retailGateway *service.RetailGrokGatewayService) *RetailGrokGatewayHandler {
	return &RetailGrokGatewayHandler{retailGateway: retailGateway}
}

func (h *RetailGrokGatewayHandler) ChatCompletions(c *gin.Context) {
	retailKey, ok := servermiddleware.GetRetailGrokKeyFromContext(c)
	if !ok {
		h.writeOpenAIError(c, http.StatusUnauthorized, "authentication_error", "Invalid retail grok key")
		return
	}
	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil || len(body) == 0 || !gjson.ValidBytes(body) {
		h.writeOpenAIError(c, http.StatusBadRequest, "invalid_request_error", "Invalid request body")
		return
	}
	model := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	if !h.retailGateway.IsAllowedChatModel(model) {
		h.writeOpenAIError(c, http.StatusForbidden, "permission_error", "Only Grok chat models are allowed")
		return
	}

	requiredMedia := service.RequiredChatMediaCapabilitiesFromBody(body)
	requiredCaps := []service.OpenAIEndpointCapability{service.OpenAIEndpointCapabilityChatCompletions}
	requiredCaps = append(requiredCaps, requiredMedia...)
	sessionHash := h.retailGateway.OpenAI().GenerateSessionHash(c, body)
	promptCacheKey := h.retailGateway.OpenAI().ExtractSessionID(c, body)

	selection, _, err := h.retailGateway.OpenAI().SelectAccountWithSchedulerForCapabilities(
		c.Request.Context(),
		&retailKey.GroupID,
		"",
		sessionHash,
		model,
		nil,
		service.OpenAIUpstreamTransportAny,
		requiredCaps,
		false,
	)
	if err != nil || selection == nil || selection.Account == nil {
		h.writeOpenAIError(c, http.StatusServiceUnavailable, "api_error", "No available Grok retail account")
		return
	}
	if selection.ReleaseFunc != nil {
		defer selection.ReleaseFunc()
	}

	result, err := h.retailGateway.OpenAI().ForwardAsChatCompletions(
		c.Request.Context(),
		c,
		selection.Account,
		body,
		promptCacheKey,
		"",
	)
	if err != nil {
		h.retailGateway.OpenAI().ReportOpenAIAccountScheduleResult(selection.Account.ID, false, nil)
		logEntry := &service.RetailGrokUsageLog{
			RequestID:       "",
			InboundEndpoint: c.Request.URL.Path,
			Model:           model,
			Status:          "error",
			ErrorMessage:    err.Error(),
		}
		_ = h.retailGateway.Retail().RecordFailure(c.Request.Context(), retailKey, logEntry)
		if !c.Writer.Written() {
			h.writeOpenAIError(c, http.StatusBadGateway, "upstream_error", "Upstream request failed")
		}
		return
	}
	h.retailGateway.OpenAI().ReportOpenAIAccountScheduleResult(selection.Account.ID, true, nil)
	_ = h.retailGateway.Retail().RecordUsage(
		c.Request.Context(),
		retailKey,
		h.retailGateway.BuildUsageLogFromForwardResult(retailKey.ID, c.Request.URL.Path, model, result),
	)
}

func (h *RetailGrokGatewayHandler) Images(c *gin.Context) {
	retailKey, ok := servermiddleware.GetRetailGrokKeyFromContext(c)
	if !ok {
		h.writeOpenAIError(c, http.StatusUnauthorized, "authentication_error", "Invalid retail grok key")
		return
	}
	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil || len(body) == 0 {
		h.writeOpenAIError(c, http.StatusBadRequest, "invalid_request_error", "Invalid request body")
		return
	}
	parsed, err := h.retailGateway.OpenAI().ParseOpenAIImagesRequest(c, body)
	if err != nil {
		h.writeOpenAIError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	if !h.retailGateway.IsAllowedImageModel(parsed.Model) {
		h.writeOpenAIError(c, http.StatusForbidden, "permission_error", "Only Grok image models are allowed")
		return
	}

	sessionHash := h.retailGateway.OpenAI().GenerateExplicitSessionHash(c, body)
	selection, _, err := h.retailGateway.OpenAI().SelectAccountWithSchedulerForImages(
		c.Request.Context(),
		&retailKey.GroupID,
		sessionHash,
		parsed.Model,
		nil,
		parsed.RequiredCapability,
	)
	if err != nil || selection == nil || selection.Account == nil {
		h.writeOpenAIError(c, http.StatusServiceUnavailable, "api_error", "No available Grok retail image account")
		return
	}
	if selection.ReleaseFunc != nil {
		defer selection.ReleaseFunc()
	}

	result, err := h.retailGateway.OpenAI().ForwardImages(c.Request.Context(), c, selection.Account, body, parsed, "")
	if err != nil {
		h.retailGateway.OpenAI().ReportOpenAIAccountScheduleResult(selection.Account.ID, false, nil)
		_ = h.retailGateway.Retail().RecordFailure(c.Request.Context(), retailKey, &service.RetailGrokUsageLog{
			InboundEndpoint: c.Request.URL.Path,
			Model:           parsed.Model,
			Status:          "error",
			ErrorMessage:    err.Error(),
		})
		if !c.Writer.Written() {
			h.writeOpenAIError(c, http.StatusBadGateway, "upstream_error", "Upstream image request failed")
		}
		return
	}
	h.retailGateway.OpenAI().ReportOpenAIAccountScheduleResult(selection.Account.ID, true, nil)
	_ = h.retailGateway.Retail().RecordUsage(
		c.Request.Context(),
		retailKey,
		h.retailGateway.BuildUsageLogFromForwardResult(retailKey.ID, c.Request.URL.Path, parsed.Model, result),
	)
}

func (h *RetailGrokGatewayHandler) Videos(c *gin.Context) {
	retailKey, ok := servermiddleware.GetRetailGrokKeyFromContext(c)
	if !ok {
		h.writeOpenAIError(c, http.StatusUnauthorized, "authentication_error", "Invalid retail grok key")
		return
	}

	if c.Request.Method == http.MethodGet {
		h.handleVideoPoll(c, retailKey)
		return
	}

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil || len(body) == 0 {
		h.writeOpenAIError(c, http.StatusBadRequest, "invalid_request_error", "Invalid request body")
		return
	}
	forwardBody, requestModel, err := service.NormalizeXAIVideoGenerationBodyForHandler(body)
	if err != nil {
		h.writeOpenAIError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	if !h.retailGateway.IsAllowedVideoModel(requestModel) {
		h.writeOpenAIError(c, http.StatusForbidden, "permission_error", "Only Grok video model is allowed")
		return
	}

	sessionHash := h.retailGateway.OpenAI().GenerateExplicitSessionHash(c, forwardBody)
	selection, _, err := h.retailGateway.OpenAI().SelectAccountWithSchedulerForCapability(
		c.Request.Context(),
		&retailKey.GroupID,
		"",
		sessionHash,
		requestModel,
		nil,
		service.OpenAIUpstreamTransportHTTPSSE,
		service.OpenAIEndpointCapabilityChatCompletions,
		false,
	)
	if err != nil || selection == nil || selection.Account == nil {
		h.writeOpenAIError(c, http.StatusServiceUnavailable, "api_error", "No available Grok retail video account")
		return
	}
	if selection.ReleaseFunc != nil {
		defer selection.ReleaseFunc()
	}

	switch {
	case strings.HasSuffix(c.Request.URL.Path, "/videos/edits"):
		err = h.retailGateway.OpenAI().ForwardXAIVideoEdit(c.Request.Context(), c, selection.Account, forwardBody)
	case strings.HasSuffix(c.Request.URL.Path, "/videos/extensions"):
		err = h.retailGateway.OpenAI().ForwardXAIVideoExtension(c.Request.Context(), c, selection.Account, forwardBody)
	default:
		err = h.retailGateway.OpenAI().ForwardXAIVideoGeneration(c.Request.Context(), c, selection.Account, forwardBody)
	}
	if err != nil {
		h.retailGateway.OpenAI().ReportOpenAIAccountScheduleResult(selection.Account.ID, false, nil)
		_ = h.retailGateway.Retail().RecordFailure(c.Request.Context(), retailKey, &service.RetailGrokUsageLog{
			InboundEndpoint: c.Request.URL.Path,
			Model:           requestModel,
			Status:          "error",
			ErrorMessage:    err.Error(),
		})
		if !c.Writer.Written() {
			h.writeOpenAIError(c, http.StatusBadGateway, "upstream_error", "Upstream video request failed")
		}
		return
	}
	h.retailGateway.OpenAI().ReportOpenAIAccountScheduleResult(selection.Account.ID, true, nil)
	_ = h.retailGateway.Retail().RecordUsage(
		c.Request.Context(),
		retailKey,
		h.retailGateway.BuildUsageLogFromForwardResult(retailKey.ID, c.Request.URL.Path, requestModel, nil),
	)
}

func (h *RetailGrokGatewayHandler) Usage(c *gin.Context) {
	retailKey, ok := servermiddleware.GetRetailGrokKeyFromContext(c)
	if !ok {
		response.Unauthorized(c, "Invalid retail grok key")
		return
	}
	summary, err := h.retailGateway.Retail().GetUsageSummary(c.Request.Context(), retailKey.ID, 20)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

func (h *RetailGrokGatewayHandler) handleVideoPoll(c *gin.Context, retailKey *service.RetailGrokKey) {
	requestID := strings.TrimSpace(c.Param("request_id"))
	if requestID == "" {
		h.writeOpenAIError(c, http.StatusBadRequest, "invalid_request_error", "request_id is required")
		return
	}
	account, stickyHit := h.retailGateway.OpenAI().ResolveXAIVideoRequestAccount(c.Request.Context(), &retailKey.GroupID, requestID)
	var releaseFunc func()
	if !stickyHit {
		selection, _, err := h.retailGateway.OpenAI().SelectAccountWithSchedulerForCapability(
			c.Request.Context(),
			&retailKey.GroupID,
			"",
			requestID,
			"grok-imagine-video",
			nil,
			service.OpenAIUpstreamTransportHTTPSSE,
			service.OpenAIEndpointCapabilityChatCompletions,
			false,
		)
		if err != nil || selection == nil || selection.Account == nil {
			h.writeOpenAIError(c, http.StatusServiceUnavailable, "api_error", "No available Grok retail video account")
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
	if account == nil {
		h.writeOpenAIError(c, http.StatusServiceUnavailable, "api_error", "No available Grok retail video account")
		return
	}
	if err := h.retailGateway.OpenAI().ForwardXAIVideoStatus(c.Request.Context(), c, account, requestID); err != nil {
		if !c.Writer.Written() {
			h.writeOpenAIError(c, http.StatusBadGateway, "upstream_error", "Upstream video poll failed")
		}
		return
	}
}

func (h *RetailGrokGatewayHandler) writeOpenAIError(c *gin.Context, status int, code string, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"type":    code,
			"message": message,
		},
	})
}

func isRetailUpstreamFailover(err error) bool {
	var failoverErr *service.UpstreamFailoverError
	return errors.As(err, &failoverErr)
}
