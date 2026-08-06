package admin

import (
	"context"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type LeonardoManualReviewHandler struct {
	service *service.LeonardoManualReviewService
}

func NewLeonardoManualReviewHandler(service *service.LeonardoManualReviewService) *LeonardoManualReviewHandler {
	return &LeonardoManualReviewHandler{service: service}
}

func (h *LeonardoManualReviewHandler) Get(c *gin.Context) {
	job, err := h.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.JSON(http.StatusOK, manualReviewResponse(job))
}

func (h *LeonardoManualReviewHandler) AttachUpstreamID(c *gin.Context) {
	var request struct {
		UpstreamGenerationID string `json:"upstream_generation_id" binding:"required"`
		Reason               string `json:"reason" binding:"required"`
	}
	if c.ShouldBindJSON(&request) != nil || strings.TrimSpace(request.Reason) == "" {
		response.ErrorFrom(c, service.ErrLeonardoManualReviewInvalid)
		return
	}
	executeAdminIdempotentJSON(c, "leonardo_manual_review_upstream_id", request, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		job, err := h.service.AttachUpstreamID(ctx, c.Param("id"), request.UpstreamGenerationID)
		return manualReviewResponse(job), err
	})
}

func (h *LeonardoManualReviewHandler) Refund(c *gin.Context) {
	var request struct {
		Reason string `json:"reason" binding:"required"`
	}
	if c.ShouldBindJSON(&request) != nil || strings.TrimSpace(request.Reason) == "" {
		response.ErrorFrom(c, service.ErrLeonardoManualReviewInvalid)
		return
	}
	executeAdminIdempotentJSON(c, "leonardo_manual_review_refund", request, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		job, err := h.service.Refund(ctx, c.Param("id"), request.Reason)
		return manualReviewResponse(job), err
	})
}

func manualReviewResponse(job *service.GenerationJob) any {
	if job == nil {
		return nil
	}
	return gin.H{"id": job.PublicID, "status": job.Status, "billing_status": job.BillingStatus, "upstream_generation_id": job.UpstreamGenerationID, "created_at": job.CreatedAt}
}
