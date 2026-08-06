package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const leonardoWebhookMaxBodyBytes = 1 << 20

type LeonardoWebhookHandler struct {
	service *service.LeonardoWebhookService
}

func NewLeonardoWebhookHandler(webhookService *service.LeonardoWebhookService) *LeonardoWebhookHandler {
	return &LeonardoWebhookHandler{service: webhookService}
}

func (h *LeonardoWebhookHandler) Handle(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("account_id"), 10, 64)
	if err != nil || accountID <= 0 || h == nil || h.service == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	reader := http.MaxBytesReader(c.Writer, c.Request.Body, leonardoWebhookMaxBodyBytes)
	body, err := io.ReadAll(reader)
	if err != nil {
		c.AbortWithStatus(http.StatusRequestEntityTooLarge)
		return
	}
	replayed, err := h.service.Receive(c.Request.Context(), accountID, c.Param("route_token"), c.GetHeader("Authorization"), body)
	if errors.Is(err, service.ErrLeonardoWebhookUnauthorized) {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	if errors.Is(err, service.ErrLeonardoWebhookInvalid) {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	if replayed {
		c.Header("X-Idempotency-Replayed", "true")
	}
	c.AbortWithStatus(http.StatusAccepted)
}
