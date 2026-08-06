package repository

import (
	"context"
	"database/sql"
	"errors"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const leonardoWebhookEventBatchSize = 50

type leonardoWebhookEventRepository struct {
	db *sql.DB
}

func (r *leonardoWebhookEventRepository) ClaimPending(ctx context.Context, limit int) ([]*service.LeonardoWebhookEvent, error) {
	if r == nil || r.db == nil {
		return nil, service.ErrLeonardoWebhookNotConfigured
	}
	if limit <= 0 || limit > leonardoWebhookEventBatchSize {
		limit = leonardoWebhookEventBatchSize
	}
	rows, err := r.db.QueryContext(ctx, `
		UPDATE leonardo_webhook_events SET status = 'processing', updated_at = NOW()
		WHERE id IN (
			SELECT id FROM leonardo_webhook_events
			WHERE (status IN ('pending', 'failed') AND next_attempt_at <= NOW()) OR (status = 'processing' AND updated_at <= NOW() - INTERVAL '30 seconds')
			ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT $1
		)
		RETURNING id, account_id, event_key, event_type, COALESCE(upstream_generation_id, ''), payload, attempt_count
	`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var events []*service.LeonardoWebhookEvent
	for rows.Next() {
		event := &service.LeonardoWebhookEvent{}
		if err = rows.Scan(&event.ID, &event.AccountID, &event.EventKey, &event.EventType, &event.UpstreamGenerationID, &event.Payload, &event.AttemptCount); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *leonardoWebhookEventRepository) MarkProcessed(ctx context.Context, id int64) error {
	return r.mark(ctx, id, "processed")
}

func (r *leonardoWebhookEventRepository) MarkFailed(ctx context.Context, id int64, attemptCount int) error {
	status := "failed"
	if attemptCount >= 5 {
		status = "dead_letter"
	}
	delaySeconds := 1 << min(attemptCount, 6)
	result, err := r.db.ExecContext(ctx, `UPDATE leonardo_webhook_events SET status = $2, attempt_count = attempt_count + 1, next_attempt_at = NOW() + ($3 * INTERVAL '1 second'), updated_at = NOW() WHERE id = $1 AND status = 'processing'`, id, status, delaySeconds)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return service.ErrGenerationJobConflict
	}
	return nil
}

func (r *leonardoWebhookEventRepository) mark(ctx context.Context, id int64, status string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE leonardo_webhook_events SET status = $2, updated_at = NOW() WHERE id = $1 AND status = 'processing'`, id, status)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return service.ErrGenerationJobConflict
	}
	return nil
}

func NewLeonardoWebhookEventRepository(_ *dbent.Client, db *sql.DB) service.LeonardoWebhookEventRepository {
	return &leonardoWebhookEventRepository{db: db}
}

func (r *leonardoWebhookEventRepository) CreatePending(ctx context.Context, event *service.LeonardoWebhookEvent) (bool, error) {
	if r == nil || r.db == nil || event == nil {
		return false, service.ErrLeonardoWebhookNotConfigured
	}
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO leonardo_webhook_events (account_id, event_key, event_type, upstream_generation_id, payload, status)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, 'pending')
		ON CONFLICT (account_id, event_key) DO NOTHING
		RETURNING id
	`, event.AccountID, event.EventKey, event.EventType, event.UpstreamGenerationID, event.Payload).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}
