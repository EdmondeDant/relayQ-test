package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
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

type UpdatePlaygroundTaskInput struct {
	Status        string
	RequestID     string
	ResultPayload json.RawMessage
	ErrorMessage  string
}

type SubmitPlaygroundJobInput struct {
	Kind            string
	Model           string
	APIKey          string
	InternalBaseURL string
	RequestPayload  json.RawMessage
}

type CreatePlaygroundAssetInput struct {
	TaskID          *int64
	Kind            string
	Title           string
	Content         string
	URL             string
	InternalBaseURL string
	StorageKey      string
	ContentType     string
	ByteSize        *int64
	Metadata        json.RawMessage
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
	UpdateTask(context.Context, int64, int64, UpdatePlaygroundTaskInput) (*PlaygroundTask, error)
	ListTasks(context.Context, int64, pagination.PaginationParams, string) ([]PlaygroundTask, int64, error)
	ListRecords(context.Context, int64, pagination.PaginationParams, string) ([]PlaygroundRecord, int64, error)
	GetTask(context.Context, int64, int64) (*PlaygroundTask, error)
	CancelTask(context.Context, int64, int64) error
	DeleteTask(context.Context, int64, int64) error
	CreateAsset(context.Context, int64, CreatePlaygroundAssetInput) (*PlaygroundAsset, error)
	ListAssets(context.Context, int64, pagination.PaginationParams, string) ([]PlaygroundAsset, int64, error)
	ListAssetsByTaskIDs(context.Context, int64, []int64) ([]PlaygroundAsset, error)
	GetAsset(context.Context, int64, int64) (*PlaygroundAsset, error)
	GetAssetByStorageKey(context.Context, int64, string) (*PlaygroundAsset, error)
	GetAssetByStorageKeyAnyUser(context.Context, string) (*PlaygroundAsset, error)
	DeleteAsset(context.Context, int64, int64) error
	EnforceUserTaskLimit(context.Context, int64, int) error
	DeleteExpired(context.Context) (int64, int64, error)
}

type PlaygroundService struct {
	repo          PlaygroundRepository
	storage       *PlaygroundAssetStorage
	stopCh        chan struct{}
	mu            sync.Mutex
	running       map[int64]context.CancelFunc
	billingService *BillingService
	resolver      *ModelPricingResolver
	usageRepo     UsageLogRepository
	usageBillingRepo UsageBillingRepository
	userRepo      UserRepository
	userSubRepo   UserSubscriptionRepository
	accountRepo   AccountRepository
	apiKeyRepo    APIKeyRepository
	apiKeyService *APIKeyService
	billingCacheService *BillingCacheService
	deferredService *DeferredService
	cfg           *config.Config
}

func NewPlaygroundService(repo PlaygroundRepository, billingService *BillingService, resolver *ModelPricingResolver, usageRepo UsageLogRepository, usageBillingRepo UsageBillingRepository, userRepo UserRepository, userSubRepo UserSubscriptionRepository, accountRepo AccountRepository, apiKeyRepo APIKeyRepository, apiKeyService *APIKeyService, billingCacheService *BillingCacheService, deferredService *DeferredService, cfg *config.Config) *PlaygroundService {
	s := &PlaygroundService{
		repo:    repo,
		storage: NewPlaygroundAssetStorage(),
		stopCh:  make(chan struct{}),
		running: make(map[int64]context.CancelFunc),
		billingService: billingService,
		resolver: resolver,
		usageRepo: usageRepo,
		usageBillingRepo: usageBillingRepo,
		userRepo: userRepo,
		userSubRepo: userSubRepo,
		accountRepo: accountRepo,
		apiKeyRepo: apiKeyRepo,
		apiKeyService: apiKeyService,
		billingCacheService: billingCacheService,
		deferredService: deferredService,
		cfg: cfg,
	}
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

func (s *PlaygroundService) UpdateTask(ctx context.Context, userID, id int64, input UpdatePlaygroundTaskInput) (*PlaygroundTask, error) {
	input.Status = strings.TrimSpace(input.Status)
	input.RequestID = strings.TrimSpace(input.RequestID)
	if input.Status == "" {
		return nil, fmt.Errorf("%w: task status is required", ErrPlaygroundInvalidInput)
	}
	if !validPlaygroundStatus(input.Status) {
		return nil, fmt.Errorf("%w: invalid task status", ErrPlaygroundInvalidInput)
	}
	if len(input.ResultPayload) == 0 {
		input.ResultPayload = json.RawMessage(`{}`)
	}
	return s.repo.UpdateTask(ctx, userID, id, input)
}

func (s *PlaygroundService) ListTasks(ctx context.Context, userID int64, params pagination.PaginationParams, kind string) ([]PlaygroundTask, int64, error) {
	return s.repo.ListTasks(ctx, userID, params, strings.TrimSpace(kind))
}

func (s *PlaygroundService) GetTask(ctx context.Context, userID, id int64) (*PlaygroundTask, error) {
	return s.repo.GetTask(ctx, userID, id)
}

func (s *PlaygroundService) CancelTask(ctx context.Context, userID, id int64) error {
	s.mu.Lock()
	cancel := s.running[id]
	delete(s.running, id)
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
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
	persisted, err := s.storage.Persist(ctx, userID, input)
	if err != nil {
		return nil, err
	}
	item, err := s.repo.CreateAsset(ctx, userID, persisted)
	if err != nil {
		return nil, err
	}
	_ = s.repo.EnforceUserTaskLimit(ctx, userID, PlaygroundRecordLimit)
	return item, nil
}

func (s *PlaygroundService) ListAssets(ctx context.Context, userID int64, params pagination.PaginationParams, kind string) ([]PlaygroundAsset, int64, error) {
	return s.repo.ListAssets(ctx, userID, params, strings.TrimSpace(kind))
}

func (s *PlaygroundService) GetAsset(ctx context.Context, userID, id int64) (*PlaygroundAsset, error) {
	return s.repo.GetAsset(ctx, userID, id)
}

func (s *PlaygroundService) GetAssetByStorageKey(ctx context.Context, userID int64, storageKey string) (*PlaygroundAsset, error) {
	return s.repo.GetAssetByStorageKey(ctx, userID, strings.TrimSpace(storageKey))
}

func (s *PlaygroundService) GetAssetByStorageKeyAnyUser(ctx context.Context, storageKey string) (*PlaygroundAsset, error) {
	return s.repo.GetAssetByStorageKeyAnyUser(ctx, strings.TrimSpace(storageKey))
}

func (s *PlaygroundService) DeleteAsset(ctx context.Context, userID, id int64) error {
	asset, err := s.repo.GetAsset(ctx, userID, id)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteAsset(ctx, userID, id); err != nil {
		return err
	}
	return s.storage.Remove(asset.StorageKey)
}

func (s *PlaygroundService) ListRecords(ctx context.Context, userID int64, params pagination.PaginationParams, kind string) ([]PlaygroundRecord, int64, error) {
	if params.PageSize <= 0 || params.PageSize > PlaygroundRecordLimit {
		params.PageSize = PlaygroundRecordLimit
	}
	items, total, err := s.repo.ListRecords(ctx, userID, params, strings.TrimSpace(kind))
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		sanitizePlaygroundRecordAssets(&items[i])
	}
	return items, total, nil
}

func sanitizePlaygroundRecordAssets(record *PlaygroundRecord) {
	if record == nil {
		return
	}
	for i := range record.Assets {
		sanitizePlaygroundAsset(&record.Assets[i])
	}
	if record.PrimaryAsset != nil {
		sanitizePlaygroundAsset(record.PrimaryAsset)
	}
}

func sanitizePlaygroundAsset(asset *PlaygroundAsset) {
	if asset == nil {
		return
	}
	kind := strings.TrimSpace(strings.ToLower(asset.Kind))
	if kind == "text" {
		if len(asset.Content) > 32*1024 {
			asset.Content = asset.Content[:32*1024]
		}
		return
	}
	if strings.TrimSpace(asset.URL) == "" && strings.TrimSpace(asset.StorageKey) != "" {
		asset.URL = buildPlaygroundAssetURL(asset.StorageKey)
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(asset.Content)), "data:") || len(asset.Content) > 256 {
		asset.Content = ""
	}
}

func (s *PlaygroundService) DeleteRecord(ctx context.Context, userID, id int64) error {
	records, _, err := s.repo.ListRecords(ctx, userID, pagination.PaginationParams{Page: 1, PageSize: PlaygroundRecordLimit}, "")
	if err == nil {
		for _, record := range records {
			if record.ID != id {
				continue
			}
			for _, asset := range record.Assets {
				_ = s.storage.Remove(asset.StorageKey)
			}
			break
		}
	}
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
