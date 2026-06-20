package handler

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type IdeaMessageHandler struct {
	ideaMessageService *service.IdeaMessageService
}

func NewIdeaMessageHandler(ideaMessageService *service.IdeaMessageService) *IdeaMessageHandler {
	return &IdeaMessageHandler{
		ideaMessageService: ideaMessageService,
	}
}

type CreateIdeaMessageRequest struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}

func (h *IdeaMessageHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	role, _ := middleware2.GetUserRoleFromContext(c)
	page, pageSize := response.ParsePagination(c)
	if pageSize > 30 {
		pageSize = 30
	}

	items, result, err := h.ideaMessageService.List(
		c.Request.Context(),
		subject.UserID,
		role == service.RoleAdmin,
		pagination.PaginationParams{
			Page:      page,
			PageSize:  pageSize,
			SortBy:    "created_at",
			SortOrder: pagination.SortOrderDesc,
		},
		service.IdeaMessageListFilters{
			MineOnly: parseIdeaMessageBoolQuery(c.Query("mine_only")),
			AuthorID: subject.UserID,
		},
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.IdeaMessage, 0, len(items))
	for i := range items {
		out = append(out, *dto.IdeaMessageFromService(&items[i]))
	}
	response.Paginated(c, out, result.Total, page, pageSize)
}

func (h *IdeaMessageHandler) Create(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	var req CreateIdeaMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	item, err := h.ideaMessageService.Create(c.Request.Context(), &service.CreateIdeaMessageInput{
		AuthorID: subject.UserID,
		Title:    req.Title,
		Content:  req.Content,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, dto.IdeaMessageFromService(item))
}

func (h *IdeaMessageHandler) Delete(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	messageID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || messageID <= 0 {
		response.BadRequest(c, "Invalid idea message ID")
		return
	}

	role, _ := middleware2.GetUserRoleFromContext(c)
	if err := h.ideaMessageService.Delete(c.Request.Context(), subject.UserID, role == service.RoleAdmin, messageID); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Idea message deleted successfully"})
}

func parseIdeaMessageBoolQuery(v string) bool {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
