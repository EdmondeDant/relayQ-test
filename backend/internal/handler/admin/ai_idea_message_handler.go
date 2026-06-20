package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
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

type ReplyIdeaMessageRequest struct {
	AdminReply string `json:"admin_reply" binding:"required"`
}

func (h *IdeaMessageHandler) Reply(c *gin.Context) {
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

	var req ReplyIdeaMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	item, err := h.ideaMessageService.Reply(c.Request.Context(), messageID, &service.ReplyIdeaMessageInput{
		ActorID: subject.UserID,
		Reply:   req.AdminReply,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.IdeaMessageFromService(item))
}
