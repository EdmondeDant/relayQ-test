package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	IdeaMessageStatusActive      = "active"
	IdeaMessageStatusUserDeleted = "user_deleted"
	IdeaMessageStatusAdminDelete = "admin_deleted"
)

var (
	ErrIdeaMessageNotFound = infraerrors.NotFound("IDEA_MESSAGE_NOT_FOUND", "idea message not found")
	ErrIdeaMessageNilInput = infraerrors.BadRequest("IDEA_MESSAGE_INPUT_REQUIRED", "idea message input is required")
	ErrIdeaMessageTitleInvalid = infraerrors.BadRequest(
		"IDEA_MESSAGE_TITLE_INVALID",
		"title must be between 1 and 120 characters",
	)
	ErrIdeaMessageContentInvalid = infraerrors.BadRequest(
		"IDEA_MESSAGE_CONTENT_INVALID",
		"content must be between 1 and 2000 characters",
	)
	ErrIdeaMessageReplyInvalid = infraerrors.BadRequest(
		"IDEA_MESSAGE_REPLY_INVALID",
		"reply must be between 1 and 1000 characters",
	)
	ErrIdeaMessageDeleteForbidden = infraerrors.Forbidden(
		"IDEA_MESSAGE_DELETE_FORBIDDEN",
		"you can only delete your own idea messages",
	)
	ErrIdeaMessageReplyForbidden = infraerrors.Forbidden(
		"IDEA_MESSAGE_REPLY_FORBIDDEN",
		"only administrators can reply to idea messages",
	)
)

type IdeaMessage struct {
	ID           int64
	AuthorID     int64
	AuthorName   string
	Title        string
	Content      string
	AdminReply   *string
	AdminReplyBy *int64
	AdminReplyAt *time.Time
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

type IdeaMessageView struct {
	IdeaMessage
	IsMine    bool
	CanDelete bool
	CanReply  bool
}

type IdeaMessageListFilters struct {
	MineOnly bool
	AuthorID int64
}

type IdeaMessageRepository interface {
	Create(ctx context.Context, message *IdeaMessage) error
	GetByID(ctx context.Context, id int64) (*IdeaMessage, error)
	Update(ctx context.Context, message *IdeaMessage) error
	List(ctx context.Context, params pagination.PaginationParams, filters IdeaMessageListFilters) ([]IdeaMessage, *pagination.PaginationResult, error)
}
