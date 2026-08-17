package handler

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type canvasModel struct {
	ID        string
	Kind      string
	Protocol  string
	Endpoints []string
}

func canvasModelFor(id string) canvasModel {
	lower := strings.ToLower(id)
	if strings.Contains(lower, "video") || strings.Contains(lower, "sora") {
		return canvasModel{ID: id, Kind: "video", Protocol: "openai-async", Endpoints: []string{"/v1/videos/generations"}}
	}
	if strings.Contains(lower, "leonardo") {
		return canvasModel{ID: id, Kind: "image", Protocol: "relayq-media", Endpoints: []string{"/v1/media/generations"}}
	}
	return canvasModel{ID: id, Kind: "image", Protocol: "openai", Endpoints: []string{"/v1/images/generations"}}
}

type CanvasHandler struct {
	apiKeyService *service.APIKeyService
	userService   *service.UserService
}

func NewCanvasHandler(apiKeyService *service.APIKeyService, userService *service.UserService, canvasRouting *service.CanvasRoutingService, channelService *service.ChannelService) *CanvasHandler {
	canvasRouting.SetChannelService(channelService)
	return &CanvasHandler{apiKeyService: apiKeyService, userService: userService}
}

func (h *CanvasHandler) Bootstrap(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	groups, err := h.apiKeyService.GetAvailableGroups(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if len(groups) == 0 {
		c.JSON(http.StatusConflict, gin.H{"code": "CANVAS_GROUP_UNAVAILABLE", "message": "No available group for canvas"})
		return
	}
	apiKey, err := h.apiKeyService.GetOrCreateCanvasAPIKey(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	user, err := h.userService.GetByID(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	models, err := h.apiKeyService.GetCanvasCatalog(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	c.Header("Cache-Control", "no-store")
	response.Success(c, gin.H{
		"base_url":       "/v1",
		"media_base_url": "/v1",
		"api_key":        apiKey.Key,
		"api_key_id":     apiKey.ID,
		"api_key_name":   apiKey.Name,
		"client_app":     apiKey.ClientApp,
		"user":           gin.H{"id": user.ID, "username": user.Username, "balance": user.Balance},
		"models":         models,
		"dashboard_url":  "/dashboard",
		"usage_url":      "/usage",
	})
}
