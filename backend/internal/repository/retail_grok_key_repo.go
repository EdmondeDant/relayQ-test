package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type retailGrokKeyRepository struct {
	db *sql.DB
}

func NewRetailGrokKeyRepository(db *sql.DB) service.RetailGrokKeyRepository {
	return &retailGrokKeyRepository{db: db}
}

func (r *retailGrokKeyRepository) Create(ctx context.Context, key *service.RetailGrokKey) error {
	const query = `
INSERT INTO retail_grok_keys (
  key, name, status, group_id, expires_at,
  token_limit_total, token_used_total,
  image_limit_total, image_used_total,
  video_limit_total, video_used_total,
  created_by_admin_id
)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
RETURNING id, created_at, updated_at`

	return r.db.QueryRowContext(
		ctx,
		query,
		key.Key,
		key.Name,
		key.Status,
		key.GroupID,
		key.ExpiresAt,
		key.TokenLimitTotal,
		key.TokenUsedTotal,
		key.ImageLimitTotal,
		key.ImageUsedTotal,
		key.VideoLimitTotal,
		key.VideoUsedTotal,
		key.CreatedByAdminID,
	).Scan(&key.ID, &key.CreatedAt, &key.UpdatedAt)
}

func (r *retailGrokKeyRepository) GetByID(ctx context.Context, id int64) (*service.RetailGrokKey, error) {
	const query = `
SELECT id, key, name, status, group_id, expires_at,
       token_limit_total, token_used_total,
       image_limit_total, image_used_total,
       video_limit_total, video_used_total,
       created_by_admin_id, created_at, updated_at
FROM retail_grok_keys
WHERE id = $1`
	return r.getOne(ctx, query, id)
}

func (r *retailGrokKeyRepository) GetByKey(ctx context.Context, key string) (*service.RetailGrokKey, error) {
	const query = `
SELECT id, key, name, status, group_id, expires_at,
       token_limit_total, token_used_total,
       image_limit_total, image_used_total,
       video_limit_total, video_used_total,
       created_by_admin_id, created_at, updated_at
FROM retail_grok_keys
WHERE key = $1`
	return r.getOne(ctx, query, key)
}

func (r *retailGrokKeyRepository) getOne(ctx context.Context, query string, arg any) (*service.RetailGrokKey, error) {
	var key service.RetailGrokKey
	var expiresAt sql.NullTime
	err := r.db.QueryRowContext(ctx, query, arg).Scan(
		&key.ID,
		&key.Key,
		&key.Name,
		&key.Status,
		&key.GroupID,
		&expiresAt,
		&key.TokenLimitTotal,
		&key.TokenUsedTotal,
		&key.ImageLimitTotal,
		&key.ImageUsedTotal,
		&key.VideoLimitTotal,
		&key.VideoUsedTotal,
		&key.CreatedByAdminID,
		&key.CreatedAt,
		&key.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrRetailGrokKeyNotFound
		}
		return nil, err
	}
	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}
	return &key, nil
}

func (r *retailGrokKeyRepository) List(ctx context.Context, limit int) ([]service.RetailGrokKey, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	const query = `
SELECT id, key, name, status, group_id, expires_at,
       token_limit_total, token_used_total,
       image_limit_total, image_used_total,
       video_limit_total, video_used_total,
       created_by_admin_id, created_at, updated_at
FROM retail_grok_keys
ORDER BY id DESC
LIMIT $1`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := make([]service.RetailGrokKey, 0)
	for rows.Next() {
		var key service.RetailGrokKey
		var expiresAt sql.NullTime
		if err := rows.Scan(
			&key.ID,
			&key.Key,
			&key.Name,
			&key.Status,
			&key.GroupID,
			&expiresAt,
			&key.TokenLimitTotal,
			&key.TokenUsedTotal,
			&key.ImageLimitTotal,
			&key.ImageUsedTotal,
			&key.VideoLimitTotal,
			&key.VideoUsedTotal,
			&key.CreatedByAdminID,
			&key.CreatedAt,
			&key.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			key.ExpiresAt = &expiresAt.Time
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

func (r *retailGrokKeyRepository) IncrementUsage(ctx context.Context, id int64, tokens, images, videos int64) error {
	const query = `
UPDATE retail_grok_keys
SET token_used_total = token_used_total + $2,
    image_used_total = image_used_total + $3,
    video_used_total = video_used_total + $4,
    updated_at = $5
WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id, tokens, images, videos, time.Now())
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrRetailGrokKeyNotFound
	}
	return nil
}
