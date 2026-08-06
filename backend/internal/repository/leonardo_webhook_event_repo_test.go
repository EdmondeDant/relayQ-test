package repository

import (
	"context"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestLeonardoWebhookEventRepositoryClaimsOnlyDueRetryableEvents(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repository := &leonardoWebhookEventRepository{db: db}
	mock.ExpectQuery(regexp.QuoteMeta("next_attempt_at <= NOW()")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "account_id", "event_key", "event_type", "upstream_generation_id", "payload", "attempt_count"}))

	events, err := repository.ClaimPending(context.Background(), 50)

	require.NoError(t, err)
	require.Empty(t, events)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLeonardoWebhookEventRepositoryDeadLettersFifthFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repository := &leonardoWebhookEventRepository{db: db}
	mock.ExpectExec(regexp.QuoteMeta("UPDATE leonardo_webhook_events SET status = $2")).
		WithArgs(int64(7), "dead_letter", 32).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repository.MarkFailed(context.Background(), 7, 5)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
