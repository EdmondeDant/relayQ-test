package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	ideaMessageRetentionDays = 2
	ideaMessageMaxPerAuthor  = 10
)

type IdeaMessageService struct {
	repo     IdeaMessageRepository
	userRepo UserRepository
}

func NewIdeaMessageService(repo IdeaMessageRepository, userRepo UserRepository) *IdeaMessageService {
	return &IdeaMessageService{
		repo:     repo,
		userRepo: userRepo,
	}
}

type CreateIdeaMessageInput struct {
	AuthorID int64
	Title    string
	Content  string
}

type ReplyIdeaMessageInput struct {
	ActorID int64
	Reply   string
}

func (s *IdeaMessageService) List(
	ctx context.Context,
	actorID int64,
	actorIsAdmin bool,
	params pagination.PaginationParams,
	filters IdeaMessageListFilters,
) ([]IdeaMessageView, *pagination.PaginationResult, error) {
	items, result, err := s.repo.List(ctx, params, filters)
	if err != nil {
		return nil, nil, fmt.Errorf("list idea messages: %w", err)
	}

	out := make([]IdeaMessageView, 0, len(items))
	for i := range items {
		out = append(out, buildIdeaMessageView(items[i], actorID, actorIsAdmin))
	}
	return out, result, nil
}

func (s *IdeaMessageService) Create(ctx context.Context, input *CreateIdeaMessageInput) (*IdeaMessageView, error) {
	if input == nil {
		return nil, ErrIdeaMessageNilInput
	}
	if input.AuthorID <= 0 {
		return nil, ErrIdeaMessageNilInput
	}

	title := strings.TrimSpace(input.Title)
	content := strings.TrimSpace(input.Content)
	if title == "" || len([]rune(title)) > 120 {
		return nil, ErrIdeaMessageTitleInvalid
	}
	if content == "" || len([]rune(content)) > 2000 {
		return nil, ErrIdeaMessageContentInvalid
	}

	author, err := s.userRepo.GetByID(ctx, input.AuthorID)
	if err != nil {
		return nil, fmt.Errorf("get message author: %w", err)
	}

	message := &IdeaMessage{
		AuthorID:   input.AuthorID,
		AuthorName: buildIdeaMessageAuthorName(author),
		Title:      title,
		Content:    content,
		Status:     IdeaMessageStatusActive,
	}

	if err := s.repo.Create(ctx, message); err != nil {
		return nil, fmt.Errorf("create idea message: %w", err)
	}
	if _, err := s.repo.DeleteExpiredByAuthor(ctx, input.AuthorID, time.Now().Add(-ideaMessageRetentionDays*24*time.Hour)); err != nil {
		return nil, fmt.Errorf("cleanup expired idea messages: %w", err)
	}
	if _, err := s.repo.DeleteOldestExcessByAuthor(ctx, input.AuthorID, ideaMessageMaxPerAuthor); err != nil {
		return nil, fmt.Errorf("trim idea messages: %w", err)
	}

	view := buildIdeaMessageView(*message, input.AuthorID, author.Role == RoleAdmin)
	return &view, nil
}

func (s *IdeaMessageService) Delete(ctx context.Context, actorID int64, actorIsAdmin bool, id int64) error {
	message, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if !actorIsAdmin && message.AuthorID != actorID {
		return ErrIdeaMessageDeleteForbidden
	}

	now := time.Now()
	message.DeletedAt = &now
	if actorIsAdmin {
		message.Status = IdeaMessageStatusAdminDelete
	} else {
		message.Status = IdeaMessageStatusUserDeleted
	}

	if err := s.repo.Update(ctx, message); err != nil {
		return fmt.Errorf("delete idea message: %w", err)
	}
	return nil
}

func (s *IdeaMessageService) Reply(ctx context.Context, id int64, input *ReplyIdeaMessageInput) (*IdeaMessageView, error) {
	if input == nil {
		return nil, ErrIdeaMessageNilInput
	}
	if input.ActorID <= 0 {
		return nil, ErrIdeaMessageReplyForbidden
	}

	reply := strings.TrimSpace(input.Reply)
	if reply == "" || len([]rune(reply)) > 1000 {
		return nil, ErrIdeaMessageReplyInvalid
	}

	actor, err := s.userRepo.GetByID(ctx, input.ActorID)
	if err != nil {
		return nil, fmt.Errorf("get reply actor: %w", err)
	}
	if actor.Role != RoleAdmin {
		return nil, ErrIdeaMessageReplyForbidden
	}

	message, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	message.AdminReply = &reply
	message.AdminReplyBy = &input.ActorID
	message.AdminReplyAt = &now

	if err := s.repo.Update(ctx, message); err != nil {
		return nil, fmt.Errorf("reply idea message: %w", err)
	}

	view := buildIdeaMessageView(*message, input.ActorID, true)
	return &view, nil
}

func buildIdeaMessageView(message IdeaMessage, actorID int64, actorIsAdmin bool) IdeaMessageView {
	isMine := actorID > 0 && message.AuthorID == actorID
	return IdeaMessageView{
		IdeaMessage: message,
		IsMine:      isMine,
		CanDelete:   actorIsAdmin || isMine,
		CanReply:    actorIsAdmin,
	}
}

func buildIdeaMessageAuthorName(user *User) string {
	if user == nil {
		return "匿名用户"
	}
	if name := strings.TrimSpace(user.Username); name != "" {
		return name
	}
	if email := strings.TrimSpace(user.Email); email != "" {
		return email
	}
	return fmt.Sprintf("用户#%d", user.ID)
}
