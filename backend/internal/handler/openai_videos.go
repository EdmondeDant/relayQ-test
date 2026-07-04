package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Videos handles the xAI-compatible videos API.
// POST /v1/videos/generations
// POST /v1/videos/edits
// POST /v1/videos/extensions
// GET  /v1/videos/{request_id}
func (h *OpenAIGatewayHandler) Videos(c *gin.Context) {
	streamStarted := false
	requestStart := time.Now()

	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLogger(
		c,
		"handler.openai_gateway.videos",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}

	var body []byte
	var err error
	if c.Request.Method != http.MethodGet {
		body, err = pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
		if err != nil {
			if maxErr, ok := extractMaxBytesError(err); ok {
				h.errorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
				return
			}
			h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
			return
		}
		if len(body) == 0 {
			h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
			return
		}
	}

	parsed, err := h.gatewayService.ParseOpenAIVideosRequest(c, body)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	requestModel := parsed.Model
	if parsed.IsPoll() {
		requestModel = "grok-imagine-video"
	}
	reqLog = reqLog.With(
		zap.String("model", requestModel),
		zap.String("endpoint", parsed.Endpoint),
		zap.String("method", c.Request.Method),
	)
	setOpsRequestContext(c, requestModel, false)
	setOpsEndpointContext(c, "", int16(service.RequestTypeSync))

	if !parsed.IsPoll() {
		if decision := h.checkContentModeration(c, reqLog, apiKey, subject, service.ContentModerationProtocolOpenAIImages, requestModel, parsed.ModerationBody()); decision != nil && decision.Blocked {
			h.errorResponse(c, contentModerationStatus(decision), contentModerationErrorCode(decision), decision.Message)
			return
		}
	}

	channelMapping, _ := h.gatewayService.ResolveChannelMappingAndRestrict(c.Request.Context(), apiKey.GroupID, requestModel)
	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())

	userReleaseFunc, acquired := h.acquireResponsesUserSlot(c, subject.UserID, subject.Concurrency, false, &streamStarted, reqLog)
	if !acquired {
		return
	}
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription, service.QuotaPlatform(c.Request.Context(), apiKey)); err != nil {
		reqLog.Info("openai.videos.billing_check_failed", zap.Error(err))
		status, code, message, retryAfter := billingErrorDetails(err)
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		h.errorResponse(c, status, code, message)
		return
	}

	if parsed.IsPoll() {
		h.handleOpenAIVideoPoll(c, reqLog, apiKey, parsed)
		return
	}

	sessionHash := h.gatewayService.GenerateExplicitSessionHash(c, body)
	failedAccountIDs := make(map[int64]struct{})
	sameAccountRetryCount := make(map[int64]int)
	var lastFailoverErr *service.UpstreamFailoverError
	switchCount := 0
	maxAccountSwitches := h.maxAccountSwitches
	routingStart := time.Now()

	for {
		selection, scheduleDecision, err := h.gatewayService.SelectAccountWithSchedulerForCapability(
			c.Request.Context(),
			apiKey.GroupID,
			sessionHash,
			"",
			requestModel,
			failedAccountIDs,
			service.OpenAIUpstreamTransportHTTPSSE,
			"",
			false,
		)
		if err != nil {
			reqLog.Warn("openai.videos.account_select_failed", zap.Error(err), zap.Int("excluded_account_count", len(failedAccountIDs)))
			if len(failedAccountIDs) == 0 {
				markOpsRoutingCapacityLimitedIfNoAvailable(c, err)
				h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available compatible accounts")
				return
			}
			if lastFailoverErr != nil {
				h.handleFailoverExhausted(c, lastFailoverErr, false)
			} else {
				h.handleFailoverExhaustedSimple(c, 502, false)
			}
			return
		}
		if selection == nil || selection.Account == nil {
			markOpsRoutingCapacityLimited(c)
			h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available compatible accounts")
			return
		}

		reqLog.Debug("openai.videos.account_schedule_decision",
			zap.String("layer", scheduleDecision.Layer),
			zap.Bool("sticky_session_hit", scheduleDecision.StickySessionHit),
			zap.Int("candidate_count", scheduleDecision.CandidateCount),
			zap.Int("top_k", scheduleDecision.TopK),
			zap.Int64("latency_ms", scheduleDecision.LatencyMs),
			zap.Float64("load_skew", scheduleDecision.LoadSkew),
		)

		account := selection.Account
		sessionHash = ensureOpenAIPoolModeSessionHash(sessionHash, account)
		setOpsSelectedAccount(c, account.ID, account.Platform)

		accountReleaseFunc, accountAcquired := h.acquireResponsesAccountSlot(c, apiKey.GroupID, sessionHash, selection, false, &streamStarted, reqLog)
		if !accountAcquired {
			return
		}

		service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
		forwardStart := time.Now()
		forwardBody := body
		if channelMapping.Mapped {
			forwardBody = h.gatewayService.ReplaceModelInBody(body, channelMapping.MappedModel)
		}
		result, err := func() (*service.OpenAIForwardResult, error) {
			defer func() {
				if accountReleaseFunc != nil {
					accountReleaseFunc()
				}
			}()
			return h.gatewayService.ForwardVideos(c.Request.Context(), c, account, forwardBody, parsed, channelMapping.MappedModel)
		}()
		forwardDurationMs := time.Since(forwardStart).Milliseconds()
		upstreamLatencyMs, _ := getContextInt64(c, service.OpsUpstreamLatencyMsKey)
		responseLatencyMs := forwardDurationMs
		if upstreamLatencyMs > 0 && forwardDurationMs > upstreamLatencyMs {
			responseLatencyMs = forwardDurationMs - upstreamLatencyMs
		}
		service.SetOpsLatencyMs(c, service.OpsResponseLatencyMsKey, responseLatencyMs)

		if err != nil {
			var failoverErr *service.UpstreamFailoverError
			if errors.As(err, &failoverErr) {
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
				if failoverErr.RetryableOnSameAccount {
					retryLimit := account.GetPoolModeRetryCount()
					if sameAccountRetryCount[account.ID] < retryLimit {
						sameAccountRetryCount[account.ID]++
						reqLog.Warn("openai.videos.pool_mode_same_account_retry",
							zap.Int64("account_id", account.ID),
							zap.Int("upstream_status", failoverErr.StatusCode),
							zap.Int("retry_limit", retryLimit),
							zap.Int("retry_count", sameAccountRetryCount[account.ID]),
						)
						select {
						case <-c.Request.Context().Done():
							return
						case <-time.After(sameAccountRetryDelay):
						}
						continue
					}
				}
				h.gatewayService.RecordOpenAIAccountSwitch()
				failedAccountIDs[account.ID] = struct{}{}
				lastFailoverErr = failoverErr
				if switchCount >= maxAccountSwitches {
					h.handleFailoverExhausted(c, failoverErr, false)
					return
				}
				switchCount++
				reqLog.Warn("openai.videos.upstream_failover_switching",
					zap.Int64("account_id", account.ID),
					zap.Int("upstream_status", failoverErr.StatusCode),
					zap.Int("switch_count", switchCount),
					zap.Int("max_switches", maxAccountSwitches),
				)
				continue
			}

			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
			reqLog.Error("openai.videos.forward_failed", zap.Int64("account_id", account.ID), zap.Error(err))
			if !c.Writer.Written() {
				h.errorResponse(c, http.StatusBadGateway, "upstream_error", "Upstream request failed")
			}
			return
		}

		h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)
		if result != nil && result.RequestID != "" && apiKey.GroupID != nil {
			h.gatewayService.BindVideoRequestAccount(context.Background(), *apiKey.GroupID, result.RequestID, account.ID)
		}
		reqLog.Debug("openai.videos.request_completed", zap.Int64("account_id", account.ID), zap.Int("switch_count", switchCount))
		return
	}
}

func (h *OpenAIGatewayHandler) handleOpenAIVideoPoll(c *gin.Context, reqLog *zap.Logger, apiKey *service.APIKey, parsed *service.OpenAIVideosRequest) {
	if apiKey.GroupID == nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "group_id is required for video polling")
		return
	}
	account, err := h.gatewayService.GetBoundVideoRequestAccount(c.Request.Context(), *apiKey.GroupID, parsed.RequestID)
	if err != nil {
		reqLog.Warn("openai.videos.poll_account_lookup_failed", zap.String("request_id", parsed.RequestID), zap.Error(err))
		h.errorResponse(c, http.StatusNotFound, "not_found_error", "Video request not found")
		return
	}

	setOpsSelectedAccount(c, account.ID, account.Platform)
	result, err := h.gatewayService.ForwardVideos(c.Request.Context(), c, account, nil, parsed, "")
	if err != nil {
		reqLog.Error("openai.videos.poll_forward_failed", zap.Int64("account_id", account.ID), zap.String("request_id", parsed.RequestID), zap.Error(err))
		if !c.Writer.Written() {
			h.errorResponse(c, http.StatusBadGateway, "upstream_error", "Upstream request failed")
		}
		return
	}
	h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)
	if result != nil && result.RequestID != "" {
		reqLog.Debug("openai.videos.poll_completed", zap.Int64("account_id", account.ID), zap.String("request_id", result.RequestID))
	}
}
