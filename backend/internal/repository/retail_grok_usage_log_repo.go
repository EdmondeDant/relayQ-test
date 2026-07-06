package repository

import (
	"context"
	"database/sql"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type retailGrokUsageLogRepository struct {
	db *sql.DB
}

func NewRetailGrokUsageLogRepository(db *sql.DB) service.RetailGrokUsageLogRepository {
	return &retailGrokUsageLogRepository{db: db}
}

func (r *retailGrokUsageLogRepository) Create(ctx context.Context, log *service.RetailGrokUsageLog) error {
	const query = `
INSERT INTO retail_grok_usage_logs (
  retail_grok_key_id, request_id, inbound_endpoint, model,
  input_tokens, output_tokens, total_tokens, image_count, video_count,
  status, error_message, upstream_model, upstream_request_id, completed_at
)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
RETURNING id, created_at`

	return r.db.QueryRowContext(
		ctx,
		query,
		log.RetailGrokKeyID,
		log.RequestID,
		log.InboundEndpoint,
		log.Model,
		log.InputTokens,
		log.OutputTokens,
		log.TotalTokens,
		log.ImageCount,
		log.VideoCount,
		log.Status,
		log.ErrorMessage,
		log.UpstreamModel,
		log.UpstreamRequestID,
		log.CompletedAt,
	).Scan(&log.ID, &log.CreatedAt)
}

func (r *retailGrokUsageLogRepository) ListByKeyID(ctx context.Context, keyID int64, limit int) ([]service.RetailGrokUsageLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	const query = `
SELECT id, retail_grok_key_id, request_id, inbound_endpoint, model,
       input_tokens, output_tokens, total_tokens, image_count, video_count,
       status, error_message, created_at, upstream_model, upstream_request_id, completed_at
FROM retail_grok_usage_logs
WHERE retail_grok_key_id = $1
ORDER BY id DESC
LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, keyID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]service.RetailGrokUsageLog, 0)
	for rows.Next() {
		var log service.RetailGrokUsageLog
		var completedAt sql.NullTime
		if err := rows.Scan(
			&log.ID,
			&log.RetailGrokKeyID,
			&log.RequestID,
			&log.InboundEndpoint,
			&log.Model,
			&log.InputTokens,
			&log.OutputTokens,
			&log.TotalTokens,
			&log.ImageCount,
			&log.VideoCount,
			&log.Status,
			&log.ErrorMessage,
			&log.CreatedAt,
			&log.UpstreamModel,
			&log.UpstreamRequestID,
			&completedAt,
		); err != nil {
			return nil, err
		}
		if completedAt.Valid {
			log.CompletedAt = &completedAt.Time
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}
