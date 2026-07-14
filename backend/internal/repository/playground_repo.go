package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

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

func (r *playgroundRepository) ListRecords(ctx context.Context, userID int64, params pagination.PaginationParams, kind string) ([]service.PlaygroundRecord, int64, error) {
	where, args := "t.user_id=$1 AND t.expires_at>NOW()", []any{userID}
	if kind != "" {
		where += " AND t.kind=$2"
		args = append(args, kind)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM playground_tasks t WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, params.Limit(), params.Offset())
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			t.id,
			t.kind,
			t.status,
			t.model,
			COALESCE(t.request_id, ''),
			t.request_payload,
			t.result_payload,
			COALESCE(t.error_message, ''),
			t.created_at,
			t.updated_at,
			t.expires_at,
			COALESCE(
				json_agg(
					json_build_object(
						'id', a.id,
						'task_id', a.task_id,
						'kind', a.kind,
						'title', a.title,
						-- 列表禁止返回媒体大 content（历史 base64 可到数 MB）；text 保留短文本
						'content', CASE
							WHEN a.kind = 'text' THEN LEFT(COALESCE(a.content, ''), 32768)
							ELSE ''
						END,
						'url', CASE
							WHEN COALESCE(a.url, '') <> '' THEN a.url
							WHEN COALESCE(a.storage_key, '') <> '' THEN '/api/v1/playground/assets/content/' || a.storage_key
							ELSE ''
						END,
						'storage_key', COALESCE(a.storage_key, ''),
						'content_type', COALESCE(a.content_type, ''),
						'byte_size', a.byte_size,
						'metadata', a.metadata,
						'created_at', a.created_at,
						'updated_at', a.updated_at,
						'expires_at', a.expires_at
					)
					ORDER BY a.created_at DESC, a.id DESC
				) FILTER (WHERE a.id IS NOT NULL),
				'[]'::json
			)
		FROM playground_tasks t
		LEFT JOIN playground_assets a ON a.task_id = t.id AND a.user_id = t.user_id AND a.expires_at > NOW()
		WHERE %s
		GROUP BY t.id
		ORDER BY t.created_at DESC, t.id DESC
		LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]service.PlaygroundRecord, 0)
	for rows.Next() {
		var item service.PlaygroundRecord
		var requestPayload, resultPayload, assetsJSON []byte
		if err := rows.Scan(
			&item.ID,
			&item.Kind,
			&item.Status,
			&item.Model,
			&item.RequestID,
			&requestPayload,
			&resultPayload,
			&item.ErrorMessage,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.ExpiresAt,
			&assetsJSON,
		); err != nil {
			return nil, 0, err
		}
		item.RequestPayload = json.RawMessage(requestPayload)
		item.ResultPayload = json.RawMessage(resultPayload)
		if len(assetsJSON) == 0 {
			assetsJSON = []byte("[]")
		}
		if err := decodePlaygroundRecordAssets(assetsJSON, &item.Assets); err != nil {
			return nil, 0, err
		}
		item.PrimaryAsset = pickPrimaryPlaygroundAsset(item.Assets)
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func decodePlaygroundRecordAssets(raw []byte, dest *[]service.PlaygroundAsset) error {
	var payloads []struct {
		ID          int64           `json:"id"`
		TaskID      *int64          `json:"task_id"`
		Kind        string          `json:"kind"`
		Title       string          `json:"title"`
		Content     string          `json:"content"`
		URL         string          `json:"url"`
		StorageKey  string          `json:"storage_key"`
		ContentType string          `json:"content_type"`
		ByteSize    *int64          `json:"byte_size"`
		Metadata    json.RawMessage `json:"metadata"`
		CreatedAt   string          `json:"created_at"`
		UpdatedAt   string          `json:"updated_at"`
		ExpiresAt   string          `json:"expires_at"`
	}
	if err := json.Unmarshal(raw, &payloads); err != nil {
		return err
	}
	items := make([]service.PlaygroundAsset, 0, len(payloads))
	for _, payload := range payloads {
		createdAt, err := parsePlaygroundTime(payload.CreatedAt)
		if err != nil {
			return err
		}
		updatedAt, err := parsePlaygroundTime(payload.UpdatedAt)
		if err != nil {
			return err
		}
		expiresAt, err := parsePlaygroundTime(payload.ExpiresAt)
		if err != nil {
			return err
		}
		items = append(items, service.PlaygroundAsset{
			ID:          payload.ID,
			TaskID:      payload.TaskID,
			Kind:        payload.Kind,
			Title:       payload.Title,
			Content:     payload.Content,
			URL:         payload.URL,
			StorageKey:  payload.StorageKey,
			ContentType: payload.ContentType,
			ByteSize:    payload.ByteSize,
			Metadata:    payload.Metadata,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
			ExpiresAt:   expiresAt,
		})
	}
	*dest = items
	return nil
}

func parsePlaygroundTime(raw string) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return parsed, nil
	}
	return time.Parse("2006-01-02T15:04:05.999999999Z07:00", raw)
}

func pickPrimaryPlaygroundAsset(assets []service.PlaygroundAsset) *service.PlaygroundAsset {
	if len(assets) == 0 {
		return nil
	}
	priority := map[string]int{"image": 1, "video": 1, "audio": 2, "text": 3}
	best := 0
	bestScore := 99
	for i, asset := range assets {
		score, ok := priority[asset.Kind]
		if !ok {
			score = 50
		}
		if score < bestScore {
			best = i
			bestScore = score
		}
	}
	item := assets[best]
	return &item
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
