package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
)

func RegisterLeonardoWebhookRoutes(r *gin.Engine, webhook *handler.LeonardoWebhookHandler) {
	r.POST("/internal/webhooks/leonardo/:account_id/:route_token", webhook.Handle)
}
