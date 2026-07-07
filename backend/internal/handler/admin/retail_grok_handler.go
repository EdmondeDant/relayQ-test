package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type RetailGrokHandler struct {
	retailService *service.RetailGrokService
}

func NewRetailGrokHandler(retailService *service.RetailGrokService) *RetailGrokHandler {
	return &RetailGrokHandler{retailService: retailService}
}

func (h *RetailGrokHandler) BatchGenerate(c *gin.Context) {
	var req service.RetailGrokBatchGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Admin authentication required")
		return
	}
	result, err := h.retailService.BatchGenerate(c.Request.Context(), subject.UserID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *RetailGrokHandler) ListKeys(c *gin.Context) {
	limit := 100
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	keys, err := h.retailService.ListKeys(c.Request.Context(), limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": keys})
}

func (h *RetailGrokHandler) GetUsage(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid retail grok key id")
		return
	}
	summary, err := h.retailService.GetUsageSummary(c.Request.Context(), id, 50)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

func (h *RetailGrokHandler) DeleteKey(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid retail grok key id")
		return
	}
	if err := h.retailService.DeleteKey(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
