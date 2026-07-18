package repository

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/ideamessage"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"

	entsql "entgo.io/ent/dialect/sql"
)

type ideaMessageRepository struct {
	client *dbent.Client
}

func NewIdeaMessageRepository(client *dbent.Client) service.IdeaMessageRepository {
	return &ideaMessageRepository{client: client}
}

func (r *ideaMessageRepository) Create(ctx context.Context, message *service.IdeaMessage) error {
	client := clientFromContext(ctx, r.client)
	builder := client.IdeaMessage.Create().
		SetAuthorID(message.AuthorID).
		SetAuthorName(message.AuthorName).
		SetTitle(message.Title).
		SetContent(message.Content).
		SetStatus(message.Status)

	if message.AdminReply != nil {
		builder.SetAdminReply(*message.AdminReply)
	}
	if message.AdminReplyBy != nil {
		builder.SetAdminReplyBy(*message.AdminReplyBy)
	}
	if message.AdminReplyAt != nil {
		builder.SetAdminReplyAt(*message.AdminReplyAt)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return err
	}
	applyIdeaMessageEntityToService(message, created)
	return nil
}

func (r *ideaMessageRepository) GetByID(ctx context.Context, id int64) (*service.IdeaMessage, error) {
	item, err := r.client.IdeaMessage.Query().
		Where(ideamessage.IDEQ(id)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrIdeaMessageNotFound, nil)
	}
	return ideaMessageEntityToService(item), nil
}

func (r *ideaMessageRepository) Update(ctx context.Context, message *service.IdeaMessage) error {
	client := clientFromContext(ctx, r.client)
	builder := client.IdeaMessage.UpdateOneID(message.ID).
		SetAuthorID(message.AuthorID).
		SetAuthorName(message.AuthorName).
		SetTitle(message.Title).
		SetContent(message.Content).
		SetStatus(message.Status)

	if message.AdminReply != nil {
		builder.SetAdminReply(*message.AdminReply)
	} else {
		builder.ClearAdminReply()
	}
	if message.AdminReplyBy != nil {
		builder.SetAdminReplyBy(*message.AdminReplyBy)
	} else {
		builder.ClearAdminReplyBy()
	}
	if message.AdminReplyAt != nil {
		builder.SetAdminReplyAt(*message.AdminReplyAt)
	} else {
		builder.ClearAdminReplyAt()
	}
	if message.DeletedAt != nil {
		builder.SetDeletedAt(*message.DeletedAt)
	} else {
		builder.ClearDeletedAt()
	}

	updated, err := builder.Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrIdeaMessageNotFound, nil)
	}
	applyIdeaMessageEntityToService(message, updated)
	return nil
}

func (r *ideaMessageRepository) List(
	ctx context.Context,
	params pagination.PaginationParams,
	filters service.IdeaMessageListFilters,
) ([]service.IdeaMessage, *pagination.PaginationResult, error) {
	client := clientFromContext(ctx, r.client)
	q := client.IdeaMessage.Query().
		Where(ideamessage.StatusEQ(service.IdeaMessageStatusActive))

	if filters.MineOnly && filters.AuthorID > 0 {
		q = q.Where(ideamessage.AuthorIDEQ(filters.AuthorID))
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	itemsQuery := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(
			dbent.Desc(ideamessage.FieldCreatedAt),
			dbent.Desc(ideamessage.FieldID),
		)

	items, err := itemsQuery.All(ctx)
	if err != nil {
		return nil, nil, err
	}

	out := make([]service.IdeaMessage, 0, len(items))
	for i := range items {
		if mapped := ideaMessageEntityToService(items[i]); mapped != nil {
			out = append(out, *mapped)
		}
	}
	return out, paginationResultFromTotal(int64(total), params), nil
}

func (r *ideaMessageRepository) DeleteExpiredByAuthor(ctx context.Context, authorID int64, olderThan time.Time) (int, error) {
	client := clientFromContext(ctx, r.client)
	affected, err := client.IdeaMessage.Update().
		Where(
			ideamessage.AuthorIDEQ(authorID),
			ideamessage.StatusEQ(service.IdeaMessageStatusActive),
			ideamessage.CreatedAtLT(olderThan),
		).
		SetStatus(service.IdeaMessageStatusUserDeleted).
		SetDeletedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return 0, err
	}
	return affected, nil
}

func (r *ideaMessageRepository) DeleteOldestExcessByAuthor(ctx context.Context, authorID int64, keep int) (int, error) {
	if keep <= 0 {
		keep = 0
	}
	client := clientFromContext(ctx, r.client)
	items, err := client.IdeaMessage.Query().
		Where(
			ideamessage.AuthorIDEQ(authorID),
			ideamessage.StatusEQ(service.IdeaMessageStatusActive),
		).
		Order(
			dbent.Desc(ideamessage.FieldCreatedAt),
			dbent.Desc(ideamessage.FieldID),
		).
		All(ctx)
	if err != nil {
		return 0, err
	}
	if len(items) <= keep {
		return 0, nil
	}
	excessIDs := make([]int64, 0, len(items)-keep)
	for _, item := range items[keep:] {
		excessIDs = append(excessIDs, item.ID)
	}
	affected, err := client.IdeaMessage.Update().
		Where(ideamessage.IDIn(excessIDs...)).
		SetStatus(service.IdeaMessageStatusUserDeleted).
		SetDeletedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return 0, err
	}
	return affected, nil
}

func applyIdeaMessageEntityToService(dst *service.IdeaMessage, src *dbent.IdeaMessage) {
	if dst == nil || src == nil {
		return
	}
	*dst = *ideaMessageEntityToService(src)
}

func ideaMessageEntityToService(item *dbent.IdeaMessage) *service.IdeaMessage {
	if item == nil {
		return nil
	}
	return &service.IdeaMessage{
		ID:           item.ID,
		AuthorID:     item.AuthorID,
		AuthorName:   item.AuthorName,
		Title:        item.Title,
		Content:      item.Content,
		AdminReply:   item.AdminReply,
		AdminReplyBy: item.AdminReplyBy,
		AdminReplyAt: item.AdminReplyAt,
		Status:       item.Status,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
		DeletedAt:    item.DeletedAt,
	}
}

func ideaMessageListOrder(_ pagination.PaginationParams) []func(*entsql.Selector) {
	return []func(*entsql.Selector){
		dbent.Desc(ideamessage.FieldCreatedAt),
		dbent.Desc(ideamessage.FieldID),
	}
}
