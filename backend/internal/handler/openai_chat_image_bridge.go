package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *OpenAIGatewayHandler) maybeHandleChatCompletionsImageBridge(
	c *gin.Context,
	apiKey *service.APIKey,
	subscription *service.UserSubscription,
	reqLog *zap.Logger,
	body []byte,
	reqModel string,
	reqStream bool,
	channelMapping service.ChannelMappingResult,
	streamStarted *bool,
) bool {
	if !service.IsOpenAIImageGenerationModel(reqModel) {
		return false
	}
	if !service.GroupAllowsImageGeneration(apiKey.Group) {
		h.handleStreamingAwareError(c, http.StatusForbidden, "permission_error", service.ImageGenerationPermissionMessage(), *streamStarted)
		return true
	}
	prompt := service.ExtractOpenAIChatImagePrompt(body)
	if prompt == "" {
		h.handleStreamingAwareError(c, http.StatusBadRequest, "invalid_request_error", "image prompt is required in the last user message", *streamStarted)
		return true
	}
	imageReleaseFunc, imageAcquired := h.acquireImageGenerationSlot(c, *streamStarted)
	if !imageAcquired {
		return true
	}
	if imageReleaseFunc != nil {
		defer imageReleaseFunc()
	}

	sessionHash := h.gatewayService.GenerateExplicitSessionHash(c, body)
	selectModel := reqModel
	if channelMapping.Mapped {
		selectModel = channelMapping.MappedModel
	}
	selection, scheduleDecision, err := h.gatewayService.SelectAccountWithSchedulerForImages(
		c.Request.Context(),
		apiKey.GroupID,
		sessionHash,
		selectModel,
		nil,
		service.OpenAIImagesCapabilityNative,
	)
	if err != nil || selection == nil || selection.Account == nil {
		markOpsRoutingCapacityLimitedIfNoAvailable(c, err)
		h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available compatible image accounts", *streamStarted)
		return true
	}
	account := selection.Account
	setOpsSelectedAccount(c, account.ID, account.Platform)
	if reqLog != nil {
		reqLog.Debug("openai_chat_completions.image_bridge_account_schedule_decision",
			zap.String("layer", scheduleDecision.Layer),
			zap.Bool("sticky_session_hit", scheduleDecision.StickySessionHit),
			zap.Int("candidate_count", scheduleDecision.CandidateCount),
			zap.Int("top_k", scheduleDecision.TopK),
			zap.Int64("latency_ms", scheduleDecision.LatencyMs),
			zap.Float64("load_skew", scheduleDecision.LoadSkew),
		)
	}

	accountReleaseFunc, acquired := h.acquireResponsesAccountSlot(c, apiKey.GroupID, sessionHash, selection, reqStream, streamStarted, reqLog)
	if !acquired {
		return true
	}
	if accountReleaseFunc != nil {
		defer accountReleaseFunc()
	}

	service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, 0)
	forwardStart := time.Now()
	options := service.ExtractOpenAIChatImageOptions(body)
	result, err := h.gatewayService.ForwardChatCompletionsImageBridge(c.Request.Context(), c, account, prompt, selectModel, reqStream, options)
	service.SetOpsLatencyMs(c, service.OpsResponseLatencyMsKey, time.Since(forwardStart).Milliseconds())
	if err != nil {
		h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
		h.ensureForwardErrorResponse(c, *streamStarted)
		if reqLog != nil {
			reqLog.Warn("openai_chat_completions.image_bridge_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		}
		return true
	}
	*streamStarted = reqStream
	if result != nil && result.FirstTokenMs != nil {
		service.SetOpsLatencyMs(c, service.OpsTimeToFirstTokenMsKey, int64(*result.FirstTokenMs))
	}
	h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)

	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetClientIP(c)
	requestPayloadHash := service.HashUsageRequestPayload(body)
	inboundEndpoint := GetInboundEndpoint(c)
	upstreamEndpoint := GetUpstreamEndpoint(c, account.Platform)
	upstreamModel := ""
	if result != nil {
		upstreamModel = result.UpstreamModel
	}
	h.submitMandatoryUsageRecordTask(c.Request.Context(), func(ctx context.Context) {
		if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
			Result:             result,
			APIKey:             apiKey,
			User:               apiKey.User,
			Account:            account,
			Subscription:       subscription,
			InboundEndpoint:    inboundEndpoint,
			UpstreamEndpoint:   upstreamEndpoint,
			UserAgent:          userAgent,
			IPAddress:          clientIP,
			RequestPayloadHash: requestPayloadHash,
			APIKeyService:      h.apiKeyService,
			ChannelUsageFields: channelMapping.ToUsageFields(reqModel, upstreamModel),
		}); err != nil {
			logger.L().With(
				zap.String("component", "handler.openai_gateway.chat_image_bridge"),
				zap.Any("group_id", apiKey.GroupID),
				zap.String("model", reqModel),
				zap.Int64("account_id", account.ID),
			).Error("openai_chat_completions.image_bridge_record_usage_failed", zap.Error(err))
		}
	})
	return true
}
