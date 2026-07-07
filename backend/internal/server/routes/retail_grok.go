package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRetailGrokRoutes(r *gin.Engine, h *handler.Handlers, auth middleware.RetailGrokKeyAuthMiddleware) {
	retail := r.Group("/retail/v1")
	retail.Use(gin.HandlerFunc(auth))
	{
		retail.GET("/models", h.RetailGrok.Models)
		retail.POST("/chat/completions", h.RetailGrok.ChatCompletions)
		retail.POST("/images/generations", h.RetailGrok.Images)
		retail.POST("/images/edits", h.RetailGrok.Images)
		retail.POST("/videos/generations", h.RetailGrok.Videos)
		retail.POST("/videos/edits", h.RetailGrok.Videos)
		retail.POST("/videos/extensions", h.RetailGrok.Videos)
		retail.GET("/videos/:request_id", h.RetailGrok.Videos)
		retail.GET("/usage", h.RetailGrok.Usage)
	}

	apiRetail := r.Group("/api/v1/retail-grok")
	apiRetail.Use(gin.HandlerFunc(auth))
	{
		apiRetail.GET("/usage", h.RetailGrok.Usage)
	}
}
