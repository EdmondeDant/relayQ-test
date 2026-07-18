//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type ideaMessageRepoStub struct {
	items  map[int64]*IdeaMessage
	nextID int64
}

func (s *ideaMessageRepoStub) Create(_ context.Context, message *IdeaMessage) error {
	if s.items == nil {
		s.items = make(map[int64]*IdeaMessage)
	}
	s.nextID++
	if s.nextID == 1 {
		s.nextID = 1
	}
	now := time.Now()
	cloned := *message
	cloned.ID = s.nextID
	cloned.CreatedAt = now
	cloned.UpdatedAt = now
	s.items[cloned.ID] = &cloned
	*message = cloned
	return nil
}

func (s *ideaMessageRepoStub) GetByID(_ context.Context, id int64) (*IdeaMessage, error) {
	item, ok := s.items[id]
	if !ok || item.DeletedAt != nil {
		return nil, ErrIdeaMessageNotFound
	}
	cloned := *item
	return &cloned, nil
}

func (s *ideaMessageRepoStub) Update(_ context.Context, message *IdeaMessage) error {
	if _, ok := s.items[message.ID]; !ok {
		return ErrIdeaMessageNotFound
	}
	cloned := *message
	cloned.UpdatedAt = time.Now()
	s.items[message.ID] = &cloned
	*message = cloned
	return nil
}

func (s *ideaMessageRepoStub) List(_ context.Context, params pagination.PaginationParams, filters IdeaMessageListFilters) ([]IdeaMessage, *pagination.PaginationResult, error) {
	all := make([]IdeaMessage, 0, len(s.items))
	for _, item := range s.items {
		if item.DeletedAt != nil || item.Status != IdeaMessageStatusActive {
			continue
		}
		if filters.MineOnly && item.AuthorID != filters.AuthorID {
			continue
		}
		all = append(all, *item)
	}

	for i := 0; i < len(all)-1; i++ {
		for j := i + 1; j < len(all); j++ {
			if all[i].CreatedAt.Before(all[j].CreatedAt) {
				all[i], all[j] = all[j], all[i]
			}
		}
	}

	total := int64(len(all))
	pageItems := paginateIdeaMessages(all, params)
	return pageItems, &pagination.PaginationResult{
		Total:    total,
		Page:     params.Page,
		PageSize: params.Limit(),
		Pages:    int((total + int64(params.Limit()) - 1) / int64(params.Limit())),
	}, nil
}

func (s *ideaMessageRepoStub) DeleteExpiredByAuthor(_ context.Context, authorID int64, olderThan time.Time) (int, error) {
	affected := 0
	now := time.Now()
	for _, item := range s.items {
		if item.AuthorID != authorID || item.Status != IdeaMessageStatusActive || item.DeletedAt != nil {
			continue
		}
		if item.CreatedAt.Before(olderThan) {
			affected++
			item.Status = IdeaMessageStatusUserDeleted
			item.DeletedAt = &now
		}
	}
	return affected, nil
}

func (s *ideaMessageRepoStub) DeleteOldestExcessByAuthor(_ context.Context, authorID int64, keep int) (int, error) {
	active := make([]*IdeaMessage, 0)
	for _, item := range s.items {
		if item.AuthorID == authorID && item.Status == IdeaMessageStatusActive && item.DeletedAt == nil {
			active = append(active, item)
		}
	}
	for i := 0; i < len(active)-1; i++ {
		for j := i + 1; j < len(active); j++ {
			if active[i].CreatedAt.Before(active[j].CreatedAt) {
				active[i], active[j] = active[j], active[i]
			}
		}
	}
	if len(active) <= keep {
		return 0, nil
	}
	now := time.Now()
	affected := 0
	for _, item := range active[keep:] {
		affected++
		item.Status = IdeaMessageStatusUserDeleted
		item.DeletedAt = &now
	}
	return affected, nil
}

func paginateIdeaMessages(items []IdeaMessage, params pagination.PaginationParams) []IdeaMessage {
	offset := params.Offset()
	if offset >= len(items) {
		return []IdeaMessage{}
	}
	limit := params.Limit()
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}
type ideaMessageUserRepoStub struct {
	users map[int64]*User
}


func (s *ideaMessageUserRepoStub) Create(context.Context, *User) error { panic("unexpected Create call") }
func (s *ideaMessageUserRepoStub) GetByID(_ context.Context, id int64) (*User, error) {
	user, ok := s.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	cloned := *user
	return &cloned, nil
}
func (s *ideaMessageUserRepoStub) GetByIDIncludeDeleted(ctx context.Context, id int64) (*User, error) {
	return s.GetByID(ctx, id)
}
func (s *ideaMessageUserRepoStub) GetByEmail(context.Context, string) (*User, error) {
	panic("unexpected GetByEmail call")
}
func (s *ideaMessageUserRepoStub) GetFirstAdmin(context.Context) (*User, error) {
	panic("unexpected GetFirstAdmin call")
}
func (s *ideaMessageUserRepoStub) Update(context.Context, *User) error { panic("unexpected Update call") }
func (s *ideaMessageUserRepoStub) Delete(context.Context, int64) error { panic("unexpected Delete call") }
func (s *ideaMessageUserRepoStub) GetUserAvatar(context.Context, int64) (*UserAvatar, error) {
	panic("unexpected GetUserAvatar call")
}
func (s *ideaMessageUserRepoStub) UpsertUserAvatar(context.Context, int64, UpsertUserAvatarInput) (*UserAvatar, error) {
	panic("unexpected UpsertUserAvatar call")
}
func (s *ideaMessageUserRepoStub) DeleteUserAvatar(context.Context, int64) error {
	panic("unexpected DeleteUserAvatar call")
}
func (s *ideaMessageUserRepoStub) List(context.Context, pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}
func (s *ideaMessageUserRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, UserListFilters) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}
func (s *ideaMessageUserRepoStub) GetLatestUsedAtByUserIDs(context.Context, []int64) (map[int64]*time.Time, error) {
	panic("unexpected GetLatestUsedAtByUserIDs call")
}
func (s *ideaMessageUserRepoStub) GetLatestUsedAtByUserID(context.Context, int64) (*time.Time, error) {
	panic("unexpected GetLatestUsedAtByUserID call")
}
func (s *ideaMessageUserRepoStub) UpdateUserLastActiveAt(context.Context, int64, time.Time) error {
	panic("unexpected UpdateUserLastActiveAt call")
}
func (s *ideaMessageUserRepoStub) UpdateBalance(context.Context, int64, float64) error {
	panic("unexpected UpdateBalance call")
}
func (s *ideaMessageUserRepoStub) DeductBalance(context.Context, int64, float64) error {
	panic("unexpected DeductBalance call")
}
func (s *ideaMessageUserRepoStub) UpdateConcurrency(context.Context, int64, int) error {
	panic("unexpected UpdateConcurrency call")
}
func (s *ideaMessageUserRepoStub) BatchSetConcurrency(context.Context, []int64, int) (int, error) {
	panic("unexpected BatchSetConcurrency call")
}
func (s *ideaMessageUserRepoStub) BatchAddConcurrency(context.Context, []int64, int) (int, error) {
	panic("unexpected BatchAddConcurrency call")
}
func (s *ideaMessageUserRepoStub) ExistsByEmail(context.Context, string) (bool, error) {
	panic("unexpected ExistsByEmail call")
}
func (s *ideaMessageUserRepoStub) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	panic("unexpected RemoveGroupFromAllowedGroups call")
}
func (s *ideaMessageUserRepoStub) AddGroupToAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected AddGroupToAllowedGroups call")
}
func (s *ideaMessageUserRepoStub) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected RemoveGroupFromUserAllowedGroups call")
}
func (s *ideaMessageUserRepoStub) ListUserAuthIdentities(context.Context, int64) ([]UserAuthIdentityRecord, error) {
	panic("unexpected ListUserAuthIdentities call")
}
func (s *ideaMessageUserRepoStub) UnbindUserAuthProvider(context.Context, int64, string) error {
	panic("unexpected UnbindUserAuthProvider call")
}
func (s *ideaMessageUserRepoStub) UpdateTotpSecret(context.Context, int64, *string) error {
	panic("unexpected UpdateTotpSecret call")
}
func (s *ideaMessageUserRepoStub) EnableTotp(context.Context, int64) error {
	panic("unexpected EnableTotp call")
}
func (s *ideaMessageUserRepoStub) DisableTotp(context.Context, int64) error {
	panic("unexpected DisableTotp call")
}

func TestIdeaMessageServiceCreatePrunesExpiredAndExcessMessages(t *testing.T) {
	now := time.Now()
	repo := &ideaMessageRepoStub{
		items: map[int64]*IdeaMessage{
			1: {ID: 1, AuthorID: 7, AuthorName: "maker", Title: "old", Content: "expired", Status: IdeaMessageStatusActive, CreatedAt: now.Add(-49 * time.Hour), UpdatedAt: now.Add(-49 * time.Hour)},
			2: {ID: 2, AuthorID: 7, AuthorName: "maker", Title: "2", Content: "c2", Status: IdeaMessageStatusActive, CreatedAt: now.Add(-10 * time.Hour), UpdatedAt: now.Add(-10 * time.Hour)},
			3: {ID: 3, AuthorID: 7, AuthorName: "maker", Title: "3", Content: "c3", Status: IdeaMessageStatusActive, CreatedAt: now.Add(-9 * time.Hour), UpdatedAt: now.Add(-9 * time.Hour)},
			4: {ID: 4, AuthorID: 7, AuthorName: "maker", Title: "4", Content: "c4", Status: IdeaMessageStatusActive, CreatedAt: now.Add(-8 * time.Hour), UpdatedAt: now.Add(-8 * time.Hour)},
			5: {ID: 5, AuthorID: 7, AuthorName: "maker", Title: "5", Content: "c5", Status: IdeaMessageStatusActive, CreatedAt: now.Add(-7 * time.Hour), UpdatedAt: now.Add(-7 * time.Hour)},
			6: {ID: 6, AuthorID: 7, AuthorName: "maker", Title: "6", Content: "c6", Status: IdeaMessageStatusActive, CreatedAt: now.Add(-6 * time.Hour), UpdatedAt: now.Add(-6 * time.Hour)},
			7: {ID: 7, AuthorID: 7, AuthorName: "maker", Title: "7", Content: "c7", Status: IdeaMessageStatusActive, CreatedAt: now.Add(-5 * time.Hour), UpdatedAt: now.Add(-5 * time.Hour)},
			8: {ID: 8, AuthorID: 7, AuthorName: "maker", Title: "8", Content: "c8", Status: IdeaMessageStatusActive, CreatedAt: now.Add(-4 * time.Hour), UpdatedAt: now.Add(-4 * time.Hour)},
			9: {ID: 9, AuthorID: 7, AuthorName: "maker", Title: "9", Content: "c9", Status: IdeaMessageStatusActive, CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now.Add(-3 * time.Hour)},
			10: {ID: 10, AuthorID: 7, AuthorName: "maker", Title: "10", Content: "c10", Status: IdeaMessageStatusActive, CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
			11: {ID: 11, AuthorID: 7, AuthorName: "maker", Title: "11", Content: "c11", Status: IdeaMessageStatusActive, CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
		},
		nextID: 11,
	}
	userRepo := &ideaMessageUserRepoStub{users: map[int64]*User{7: {ID: 7, Role: RoleUser, Username: "maker"}}}
	svc := NewIdeaMessageService(repo, userRepo)

	_, err := svc.Create(context.Background(), &CreateIdeaMessageInput{AuthorID: 7, Title: "new", Content: "new content"})
	require.NoError(t, err)

	activeCount := 0
	for _, item := range repo.items {
		if item.AuthorID == 7 && item.Status == IdeaMessageStatusActive && item.DeletedAt == nil {
			activeCount++
		}
	}
	require.Equal(t, 10, activeCount)
	require.Equal(t, IdeaMessageStatusUserDeleted, repo.items[1].Status)
	require.NotNil(t, repo.items[1].DeletedAt)
	require.Equal(t, IdeaMessageStatusUserDeleted, repo.items[2].Status)
	require.NotNil(t, repo.items[2].DeletedAt)
}

func TestIdeaMessageServiceDeletePermissions(t *testing.T) {
	repo := &ideaMessageRepoStub{
		items: map[int64]*IdeaMessage{
			1: {
				ID:         1,
				AuthorID:   7,
				AuthorName: "maker",
				Title:      "title",
				Content:    "content",
				Status:     IdeaMessageStatusActive,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
		},
		nextID: 1,
	}
	svc := NewIdeaMessageService(repo, &ideaMessageUserRepoStub{})

	err := svc.Delete(context.Background(), 8, false, 1)
	require.ErrorIs(t, err, ErrIdeaMessageDeleteForbidden)

	err = svc.Delete(context.Background(), 7, false, 1)
	require.NoError(t, err)
	require.Equal(t, IdeaMessageStatusUserDeleted, repo.items[1].Status)
	require.NotNil(t, repo.items[1].DeletedAt)
}

func TestIdeaMessageServiceReplyAdminOnly(t *testing.T) {
	repo := &ideaMessageRepoStub{
		items: map[int64]*IdeaMessage{
			1: {
				ID:         1,
				AuthorID:   7,
				AuthorName: "maker",
				Title:      "title",
				Content:    "content",
				Status:     IdeaMessageStatusActive,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
		},
		nextID: 1,
	}
	userRepo := &ideaMessageUserRepoStub{
		users: map[int64]*User{
			1: {ID: 1, Role: RoleAdmin, Username: "admin"},
			2: {ID: 2, Role: RoleUser, Username: "user"},
		},
	}
	svc := NewIdeaMessageService(repo, userRepo)

	_, err := svc.Reply(context.Background(), 1, &ReplyIdeaMessageInput{
		ActorID: 2,
		Reply:   "普通用户不能回",
	})
	require.ErrorIs(t, err, ErrIdeaMessageReplyForbidden)

	view, err := svc.Reply(context.Background(), 1, &ReplyIdeaMessageInput{
		ActorID: 1,
		Reply:   "这个方向值得先做 MVP。",
	})
	require.NoError(t, err)
	require.NotNil(t, view.AdminReply)
	require.Equal(t, "这个方向值得先做 MVP。", *view.AdminReply)
	require.True(t, view.CanReply)
	require.NotNil(t, repo.items[1].AdminReplyAt)
}

func TestIdeaMessageServiceListMineOnlyAndPagination(t *testing.T) {
	now := time.Now()
	repo := &ideaMessageRepoStub{
		items: map[int64]*IdeaMessage{
			1: {ID: 1, AuthorID: 9, AuthorName: "me", Title: "A", Content: "1", Status: IdeaMessageStatusActive, CreatedAt: now.Add(-2 * time.Minute), UpdatedAt: now.Add(-2 * time.Minute)},
			2: {ID: 2, AuthorID: 10, AuthorName: "other", Title: "B", Content: "2", Status: IdeaMessageStatusActive, CreatedAt: now.Add(-1 * time.Minute), UpdatedAt: now.Add(-1 * time.Minute)},
			3: {ID: 3, AuthorID: 9, AuthorName: "me", Title: "C", Content: "3", Status: IdeaMessageStatusActive, CreatedAt: now, UpdatedAt: now},
		},
		nextID: 3,
	}
	svc := NewIdeaMessageService(repo, &ideaMessageUserRepoStub{})

	items, result, err := svc.List(context.Background(), 9, false, pagination.PaginationParams{
		Page:     1,
		PageSize: 1,
	}, IdeaMessageListFilters{
		MineOnly: true,
		AuthorID: 9,
	})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(3), items[0].ID)
	require.Equal(t, int64(2), result.Total)
	require.Equal(t, 2, result.Pages)
	require.True(t, items[0].IsMine)
	require.True(t, items[0].CanDelete)
	require.False(t, items[0].CanReply)
}

func TestIdeaMessageServiceCreateSnapshotsAuthorName(t *testing.T) {
	repo := &ideaMessageRepoStub{}
	userRepo := &ideaMessageUserRepoStub{
		users: map[int64]*User{
			11: {
				ID:       11,
				Role:     RoleUser,
				Username: "",
				Email:    "founder@example.com",
			},
		},
	}
	svc := NewIdeaMessageService(repo, userRepo)

	view, err := svc.Create(context.Background(), &CreateIdeaMessageInput{
		AuthorID: 11,
		Title:    "  想做一个 AI 需求验证小站  ",
		Content:  "  先帮独立开发者验证 landing page 文案是否能收集到需求。  ",
	})
	require.NoError(t, err)
	require.Equal(t, "想做一个 AI 需求验证小站", view.Title)
	require.Equal(t, "先帮独立开发者验证 landing page 文案是否能收集到需求。", view.Content)
	require.Equal(t, "founder@example.com", view.AuthorName)
	require.True(t, view.IsMine)
}
