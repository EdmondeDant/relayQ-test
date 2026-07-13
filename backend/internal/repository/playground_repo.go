package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type playgroundRepository struct{ db *sql.DB }

func NewPlaygroundRepository(db *sql.DB) service.PlaygroundRepository {
	return &playgroundRepository{db: db}
}

func (r *playgroundRepository) CreateTask(ctx context.Context, userID int64, input service.CreatePlaygroundTaskInput) (*service.PlaygroundTask, error) {
	// 时间戳在 Go 侧决定，避免 PG 对同一参数既写 varchar 又参与 CASE 的类型推断冲突
	setStarted := input.Status == "running" || input.Status == "submitted" || input.Status == "succeeded" || input.Status == "failed"
	setCompleted := input.Status == "succeeded" || input.Status == "failed" || input.Status == "canceled"
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO playground_tasks (
			user_id, kind, status, model, request_id, request_payload, result_payload, error_message, started_at, completed_at
		) VALUES (
			$1, $2, $3, $4, NULLIF($5, ''), $6, $7, NULLIF($8, ''),
			CASE WHEN $9 THEN NOW() ELSE NULL END,
			CASE WHEN $10 THEN NOW() ELSE NULL END
		)
		RETURNING id, user_id, kind, status, model, COALESCE(request_id, ''), request_payload, result_payload,
			COALESCE(error_message, ''), started_at, completed_at, created_at, updated_at, expires_at
	`, userID, input.Kind, input.Status, input.Model, input.RequestID, input.RequestPayload, input.ResultPayload, input.ErrorMessage, setStarted, setCompleted)
	return scanPlaygroundTask(row)
}

func (r *playgroundRepository) ListTasks(ctx context.Context, userID int64, params pagination.PaginationParams, kind string) ([]service.PlaygroundTask, int64, error) {
	where, args := "user_id=$1 AND expires_at>NOW()", []any{userID}
	if kind != "" {
		where += " AND kind=$2"
		args = append(args, kind)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM playground_tasks WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, params.Limit(), params.Offset())
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`SELECT id,user_id,kind,status,model,COALESCE(request_id,''),request_payload,result_payload,COALESCE(error_message,''),started_at,completed_at,created_at,updated_at,expires_at FROM playground_tasks WHERE %s ORDER BY created_at DESC,id DESC LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]service.PlaygroundTask, 0)
	for rows.Next() {
		item, err := scanPlaygroundTask(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (r *playgroundRepository) GetTask(ctx context.Context, userID, id int64) (*service.PlaygroundTask, error) {
	return scanPlaygroundTask(r.db.QueryRowContext(ctx, `SELECT id,user_id,kind,status,model,COALESCE(request_id,''),request_payload,result_payload,COALESCE(error_message,''),started_at,completed_at,created_at,updated_at,expires_at FROM playground_tasks WHERE id=$1 AND user_id=$2 AND expires_at>NOW()`, id, userID))
}

func (r *playgroundRepository) CancelTask(ctx context.Context, userID, id int64) error {
	result, err := r.db.ExecContext(ctx, `UPDATE playground_tasks SET status='canceled',completed_at=NOW(),updated_at=NOW() WHERE id=$1 AND user_id=$2 AND status IN ('pending','running','submitted')`, id, userID)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count > 0 {
		return nil
	}
	var exists bool
	if err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM playground_tasks WHERE id=$1 AND user_id=$2)`, id, userID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return service.ErrPlaygroundNotFound
	}
	return service.ErrPlaygroundInvalidState
}

func (r *playgroundRepository) DeleteTask(ctx context.Context, userID, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM playground_assets WHERE task_id=$1 AND user_id=$2`, id, userID); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM playground_tasks WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return service.ErrPlaygroundNotFound
	}
	return tx.Commit()
}

func (r *playgroundRepository) ListAssetsByTaskIDs(ctx context.Context, userID int64, taskIDs []int64) ([]service.PlaygroundAsset, error) {
	if len(taskIDs) == 0 {
		return []service.PlaygroundAsset{}, nil
	}
	// 构建 IN 查询
	placeholders := make([]string, len(taskIDs))
	args := make([]any, 0, len(taskIDs)+1)
	args = append(args, userID)
	for i, id := range taskIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}
	query := fmt.Sprintf(`SELECT id,user_id,task_id,kind,title,COALESCE(content,''),COALESCE(url,''),COALESCE(storage_key,''),COALESCE(content_type,''),byte_size,metadata,created_at,updated_at,expires_at FROM playground_assets WHERE user_id=$1 AND expires_at>NOW() AND task_id IN (%s) ORDER BY created_at DESC,id DESC`, strings.Join(placeholders, ","))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]service.PlaygroundAsset, 0)
	for rows.Next() {
		item, err := scanPlaygroundAsset(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *playgroundRepository) EnforceUserTaskLimit(ctx context.Context, userID int64, limit int) error {
	if limit < 1 {
		return nil
	}
	// 仅删除超出最近 N 条的记录；用 OFFSET 避免 keep 为空时误删
	_, err := r.db.ExecContext(ctx, `
		WITH outdated AS (
			SELECT id
			FROM playground_tasks
			WHERE user_id = $1
			ORDER BY created_at DESC, id DESC
			OFFSET $2
		),
		deleted_assets AS (
			DELETE FROM playground_assets
			WHERE task_id IN (SELECT id FROM outdated)
			RETURNING 1
		)
		DELETE FROM playground_tasks
		WHERE id IN (SELECT id FROM outdated)
	`, userID, limit)
	return err
}

func (r *playgroundRepository) CreateAsset(ctx context.Context, userID int64, input service.CreatePlaygroundAssetInput) (*service.PlaygroundAsset, error) {
	row := r.db.QueryRowContext(ctx, `INSERT INTO playground_assets (user_id,task_id,kind,title,content,url,storage_key,content_type,byte_size,metadata)
		SELECT $1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),NULLIF($7,''),NULLIF($8,''),$9,$10
		WHERE $2::BIGINT IS NULL OR EXISTS (SELECT 1 FROM playground_tasks WHERE id=$2 AND user_id=$1)
		RETURNING id,user_id,task_id,kind,title,COALESCE(content,''),COALESCE(url,''),COALESCE(storage_key,''),COALESCE(content_type,''),byte_size,metadata,created_at,updated_at,expires_at`, userID, input.TaskID, input.Kind, input.Title, input.Content, input.URL, input.StorageKey, input.ContentType, input.ByteSize, input.Metadata)
	return scanPlaygroundAsset(row)
}

func (r *playgroundRepository) ListAssets(ctx context.Context, userID int64, params pagination.PaginationParams, kind string) ([]service.PlaygroundAsset, int64, error) {
	where, args := "user_id=$1 AND expires_at>NOW()", []any{userID}
	if kind != "" {
		where += " AND kind=$2"
		args = append(args, kind)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM playground_assets WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, params.Limit(), params.Offset())
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`SELECT id,user_id,task_id,kind,title,COALESCE(content,''),COALESCE(url,''),COALESCE(storage_key,''),COALESCE(content_type,''),byte_size,metadata,created_at,updated_at,expires_at FROM playground_assets WHERE %s ORDER BY created_at DESC,id DESC LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]service.PlaygroundAsset, 0)
	for rows.Next() {
		item, err := scanPlaygroundAsset(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (r *playgroundRepository) GetAsset(ctx context.Context, userID, id int64) (*service.PlaygroundAsset, error) {
	return scanPlaygroundAsset(r.db.QueryRowContext(ctx, `SELECT id,user_id,task_id,kind,title,COALESCE(content,''),COALESCE(url,''),COALESCE(storage_key,''),COALESCE(content_type,''),byte_size,metadata,created_at,updated_at,expires_at FROM playground_assets WHERE id=$1 AND user_id=$2 AND expires_at>NOW()`, id, userID))
}

func (r *playgroundRepository) DeleteExpired(ctx context.Context) (int64, int64, error) {
	assetResult, err := r.db.ExecContext(ctx, `DELETE FROM playground_assets WHERE expires_at<=NOW()`)
	if err != nil {
		return 0, 0, err
	}
	taskResult, err := r.db.ExecContext(ctx, `DELETE FROM playground_tasks WHERE expires_at<=NOW()`)
	if err != nil {
		return 0, 0, err
	}
	assets, _ := assetResult.RowsAffected()
	tasks, _ := taskResult.RowsAffected()
	return tasks, assets, nil
}

func (r *playgroundRepository) DeleteAsset(ctx context.Context, userID, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM playground_assets WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return service.ErrPlaygroundNotFound
	}
	return nil
}

type rowScanner interface{ Scan(...any) error }

func scanPlaygroundTask(row rowScanner) (*service.PlaygroundTask, error) {
	var item service.PlaygroundTask
	var requestPayload, resultPayload []byte
	err := row.Scan(&item.ID, &item.UserID, &item.Kind, &item.Status, &item.Model, &item.RequestID, &requestPayload, &resultPayload, &item.ErrorMessage, &item.StartedAt, &item.CompletedAt, &item.CreatedAt, &item.UpdatedAt, &item.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrPlaygroundNotFound
	}
	if err != nil {
		return nil, err
	}
	item.RequestPayload = json.RawMessage(requestPayload)
	item.ResultPayload = json.RawMessage(resultPayload)
	return &item, nil
}

func scanPlaygroundAsset(row rowScanner) (*service.PlaygroundAsset, error) {
	var item service.PlaygroundAsset
	var metadata []byte
	err := row.Scan(&item.ID, &item.UserID, &item.TaskID, &item.Kind, &item.Title, &item.Content, &item.URL, &item.StorageKey, &item.ContentType, &item.ByteSize, &metadata, &item.CreatedAt, &item.UpdatedAt, &item.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrPlaygroundNotFound
	}
	if err != nil {
		return nil, err
	}
	item.Metadata = json.RawMessage(metadata)
	return &item, nil
}
