package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestCanvasResourceRouteRepositoryOwnershipAndExpiry(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := NewCanvasResourceRouteRepository(db)
	expiresAt := time.Now().Add(time.Hour)

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO canvas_resource_routes")).
		WithArgs(int64(2), int64(1), "video-1", int64(3), "xai", "grok-imagine-video", "videos", expiresAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	require.NoError(t, repo.Upsert(context.Background(), &service.CanvasResourceRoute{
		APIKeyID: 2, UserID: 1, ResourceID: "video-1", GroupID: 3, Platform: "xai", Model: "grok-imagine-video", EndpointFamily: "videos", ExpiresAt: expiresAt,
	}))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT api_key_id, user_id, resource_id, group_id, platform, model, endpoint_family, expires_at")).
		WithArgs(int64(1), int64(2), "video-1").
		WillReturnRows(sqlmock.NewRows([]string{"api_key_id", "user_id", "resource_id", "group_id", "platform", "model", "endpoint_family", "expires_at"}).
			AddRow(2, 1, "video-1", 3, "xai", "grok-imagine-video", "videos", expiresAt))
	route, err := repo.GetActive(context.Background(), 1, 2, "video-1")
	require.NoError(t, err)
	require.Equal(t, int64(3), route.GroupID)
	require.NoError(t, mock.ExpectationsWereMet())
}
