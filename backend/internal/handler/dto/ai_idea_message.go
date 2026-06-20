package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type IdeaMessage struct {
	ID          int64      `json:"id"`
	AuthorID    int64      `json:"author_id"`
	AuthorName  string     `json:"author_name"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	AdminReply  *string    `json:"admin_reply,omitempty"`
	AdminReplyAt *time.Time `json:"admin_reply_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	IsMine      bool       `json:"is_mine"`
	CanDelete   bool       `json:"can_delete"`
	CanReply    bool       `json:"can_reply"`
}

func IdeaMessageFromService(message *service.IdeaMessageView) *IdeaMessage {
	if message == nil {
		return nil
	}
	return &IdeaMessage{
		ID:           message.ID,
		AuthorID:     message.AuthorID,
		AuthorName:   message.AuthorName,
		Title:        message.Title,
		Content:      message.Content,
		AdminReply:   message.AdminReply,
		AdminReplyAt: message.AdminReplyAt,
		CreatedAt:    message.CreatedAt,
		UpdatedAt:    message.UpdatedAt,
		IsMine:       message.IsMine,
		CanDelete:    message.CanDelete,
		CanReply:     message.CanReply,
	}
}
