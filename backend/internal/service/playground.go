package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var ErrPlaygroundNotFound = errors.New("playground resource not found")
var ErrPlaygroundInvalidState = errors.New("playground task cannot be changed in its current state")
var ErrPlaygroundInvalidInput = errors.New("invalid playground input")

type PlaygroundTask struct {
	ID             int64           `json:"id"`
	UserID         int64           `json:"-"`
	Kind           string          `json:"kind"`
	Status         string          `json:"status"`
	Model          string          `json:"model"`
	RequestID      string          `json:"request_id,omitempty"`
	RequestPayload json.RawMessage `json:"request_payload"`
	ResultPayload  json.RawMessage `json:"result_payload"`
	ErrorMessage   string          `json:"error_message,omitempty"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	ExpiresAt      time.Time       `json:"expires_at"`
}

type PlaygroundAsset struct {
	ID          int64           `json:"id"`
	UserID      int64           `json:"-"`
	TaskID      *int64          `json:"task_id,omitempty"`
	Kind        string          `json:"kind"`
	Title       string          `json:"title"`
	Content     string          `json:"content,omitempty"`
	URL         string          `json:"url,omitempty"`
	StorageKey  string          `json:"storage_key,omitempty"`
	ContentType string          `json:"content_type,omitempty"`
	ByteSize    *int64          `json:"byte_size,omitempty"`
	Metadata    json.RawMessage `json:"metadata"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	ExpiresAt   time.Time       `json:"expires_at"`
}

type CreatePlaygroundTaskInput struct {
	Kind           string
	Status         string
	Model          string
	RequestID      string
	RequestPayload json.RawMessage
	ResultPayload  json.RawMessage
	ErrorMessage   string
}

type CreatePlaygroundAssetInput struct {
	TaskID      *int64
	Kind        string
	Title       string
	Content     string
	URL         string
	StorageKey  string
	ContentType string
	ByteSize    *int64
	Metadata    json.RawMessage
}

// PlaygroundRecord 统一创作记录：任务 + 关联产物
type PlaygroundRecord struct {
	ID             int64             `json:"id"`
	Kind           string            `json:"kind"`
	Status         string            `json:"status"`
	Model          string            `json:"model"`
	RequestID      string            `json:"request_id,omitempty"`
	RequestPayload json.RawMessage   `json:"request_payload"`
	ResultPayload  json.RawMessage   `json:"result_payload"`
	ErrorMessage   string            `json:"error_message,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	ExpiresAt      time.Time         `json:"expires_at"`
	Assets         []PlaygroundAsset `json:"assets"`
	PrimaryAsset   *PlaygroundAsset  `json:"primary_asset,omitempty"`
}

const PlaygroundRecordLimit = 10

type PlaygroundRepository interface {
	CreateTask(context.Context, int64, CreatePlaygroundTaskInput) (*PlaygroundTask, error)
	ListTasks(context.Context, int64, pagination.PaginationParams, string) ([]PlaygroundTask, int64, error)
	GetTask(context.Context, int64, int64) (*PlaygroundTask, error)
	CancelTask(context.Context, int64, int64) error
	DeleteTask(context.Context, int64, int64) error
	CreateAsset(context.Context, int64, CreatePlaygroundAssetInput) (*PlaygroundAsset, error)
	ListAssets(context.Context, int64, pagination.PaginationParams, string) ([]PlaygroundAsset, int64, error)
	ListAssetsByTaskIDs(context.Context, int64, []int64) ([]PlaygroundAsset, error)
	GetAsset(context.Context, int64, int64) (*PlaygroundAsset, error)
	DeleteAsset(context.Context, int64, int64) error
	EnforceUserTaskLimit(context.Context, int64, int) error
	DeleteExpired(context.Context) (int64, int64, error)
}

type PlaygroundService struct {
	repo   PlaygroundRepository
	stopCh chan struct{}
}

func NewPlaygroundService(repo PlaygroundRepository) *PlaygroundService {
	s := &PlaygroundService{repo: repo, stopCh: make(chan struct{})}
	s.Start()
	return s
}

func (s *PlaygroundService) Start() {
	go func() {
		s.cleanup(context.Background())
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.cleanup(context.Background())
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *PlaygroundService) Stop() { close(s.stopCh) }

func (s *PlaygroundService) cleanup(ctx context.Context) { _, _, _ = s.repo.DeleteExpired(ctx) }

func (s *PlaygroundService) CreateTask(ctx context.Context, userID int64, input CreatePlaygroundTaskInput) (*PlaygroundTask, error) {
	input.Kind = strings.TrimSpace(input.Kind)
	input.Model = strings.TrimSpace(input.Model)
	input.RequestID = strings.TrimSpace(input.RequestID)
	if input.Kind == "" {
		return nil, fmt.Errorf("%w: task kind is required", ErrPlaygroundInvalidInput)
	}
	if input.Status == "" {
		input.Status = "succeeded"
	}
	if !validPlaygroundStatus(input.Status) {
		return nil, fmt.Errorf("%w: invalid task status", ErrPlaygroundInvalidInput)
	}
	if len(input.RequestPayload) == 0 {
		input.RequestPayload = json.RawMessage(`{}`)
	}
	if len(input.ResultPayload) == 0 {
		input.ResultPayload = json.RawMessage(`{}`)
	}
	item, err := s.repo.CreateTask(ctx, userID, input)
	if err != nil {
		return nil, err
	}
	_ = s.repo.EnforceUserTaskLimit(ctx, userID, PlaygroundRecordLimit)
	return item, nil
}

func (s *PlaygroundService) ListTasks(ctx context.Context, userID int64, params pagination.PaginationParams, kind string) ([]PlaygroundTask, int64, error) {
	return s.repo.ListTasks(ctx, userID, params, strings.TrimSpace(kind))
}

func (s *PlaygroundService) GetTask(ctx context.Context, userID, id int64) (*PlaygroundTask, error) {
	return s.repo.GetTask(ctx, userID, id)
}

func (s *PlaygroundService) CancelTask(ctx context.Context, userID, id int64) error {
	return s.repo.CancelTask(ctx, userID, id)
}

func (s *PlaygroundService) CreateAsset(ctx context.Context, userID int64, input CreatePlaygroundAssetInput) (*PlaygroundAsset, error) {
	input.Kind = strings.TrimSpace(input.Kind)
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)
	input.URL = strings.TrimSpace(input.URL)
	input.StorageKey = strings.TrimSpace(input.StorageKey)
	if input.Kind == "" || (input.Content == "" && input.URL == "" && input.StorageKey == "") {
		return nil, fmt.Errorf("%w: asset kind and content are required", ErrPlaygroundInvalidInput)
	}
	if len(input.Metadata) == 0 {
		input.Metadata = json.RawMessage(`{}`)
	}
	item, err := s.repo.CreateAsset(ctx, userID, input)
	if err != nil {
		return nil, err
	}
	// 资产写入后也兜底执行一次 10 条保留策略
	_ = s.repo.EnforceUserTaskLimit(ctx, userID, PlaygroundRecordLimit)
	return item, nil
}

func (s *PlaygroundService) ListAssets(ctx context.Context, userID int64, params pagination.PaginationParams, kind string) ([]PlaygroundAsset, int64, error) {
	return s.repo.ListAssets(ctx, userID, params, strings.TrimSpace(kind))
}

func (s *PlaygroundService) GetAsset(ctx context.Context, userID, id int64) (*PlaygroundAsset, error) {
	return s.repo.GetAsset(ctx, userID, id)
}

func (s *PlaygroundService) DeleteAsset(ctx context.Context, userID, id int64) error {
	return s.repo.DeleteAsset(ctx, userID, id)
}

func (s *PlaygroundService) ListRecords(ctx context.Context, userID int64, params pagination.PaginationParams, kind string) ([]PlaygroundRecord, int64, error) {
	if params.PageSize <= 0 || params.PageSize > PlaygroundRecordLimit {
		params.PageSize = PlaygroundRecordLimit
	}
	tasks, total, err := s.repo.ListTasks(ctx, userID, params, strings.TrimSpace(kind))
	if err != nil {
		return nil, 0, err
	}
	if len(tasks) == 0 {
		return []PlaygroundRecord{}, total, nil
	}
	ids := make([]int64, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.ID)
	}
	assets, err := s.repo.ListAssetsByTaskIDs(ctx, userID, ids)
	if err != nil {
		return nil, 0, err
	}
	assetsByTask := map[int64][]PlaygroundAsset{}
	for _, asset := range assets {
		if asset.TaskID == nil {
			continue
		}
		assetsByTask[*asset.TaskID] = append(assetsByTask[*asset.TaskID], asset)
	}
	records := make([]PlaygroundRecord, 0, len(tasks))
	for _, task := range tasks {
		taskAssets := assetsByTask[task.ID]
		if taskAssets == nil {
			taskAssets = []PlaygroundAsset{}
		}
		record := PlaygroundRecord{
			ID:             task.ID,
			Kind:           task.Kind,
			Status:         task.Status,
			Model:          task.Model,
			RequestID:      task.RequestID,
			RequestPayload: task.RequestPayload,
			ResultPayload:  task.ResultPayload,
			ErrorMessage:   task.ErrorMessage,
			CreatedAt:      task.CreatedAt,
			UpdatedAt:      task.UpdatedAt,
			ExpiresAt:      task.ExpiresAt,
			Assets:         taskAssets,
			PrimaryAsset:   pickPrimaryAsset(taskAssets),
		}
		records = append(records, record)
	}
	return records, total, nil
}

func (s *PlaygroundService) DeleteRecord(ctx context.Context, userID, id int64) error {
	return s.repo.DeleteTask(ctx, userID, id)
}

func pickPrimaryAsset(assets []PlaygroundAsset) *PlaygroundAsset {
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

func validPlaygroundStatus(status string) bool {
	switch status {
	case "pending", "running", "submitted", "succeeded", "failed", "canceled":
		return true
	default:
		return false
	}
}
