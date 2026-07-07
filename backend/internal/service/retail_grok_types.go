package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	RetailGrokKeyStatusActive   = "active"
	RetailGrokKeyStatusDisabled = "disabled"
)

var (
	ErrRetailGrokKeyNotFound      = infraerrors.NotFound("RETAIL_GROK_KEY_NOT_FOUND", "retail grok key not found")
	ErrRetailGrokKeyDisabled      = infraerrors.Forbidden("RETAIL_GROK_KEY_DISABLED", "retail grok key is disabled")
	ErrRetailGrokKeyExpired       = infraerrors.Forbidden("RETAIL_GROK_KEY_EXPIRED", "retail grok key has expired")
	ErrRetailGrokTokenExhausted   = infraerrors.TooManyRequests("RETAIL_GROK_TOKEN_EXHAUSTED", "retail grok token quota exhausted")
	ErrRetailGrokImageExhausted   = infraerrors.TooManyRequests("RETAIL_GROK_IMAGE_EXHAUSTED", "retail grok image quota exhausted")
	ErrRetailGrokVideoExhausted   = infraerrors.TooManyRequests("RETAIL_GROK_VIDEO_EXHAUSTED", "retail grok video quota exhausted")
	ErrRetailGrokInvalidScope     = infraerrors.Forbidden("RETAIL_GROK_INVALID_SCOPE", "retail grok key cannot access this endpoint")
	ErrRetailGrokGroupRequired    = infraerrors.BadRequest("RETAIL_GROK_GROUP_REQUIRED", "group_id is required")
	ErrRetailGrokInvalidBatchSize = infraerrors.BadRequest("RETAIL_GROK_INVALID_BATCH_SIZE", "count must be between 1 and 200")
)

type RetailGrokKey struct {
	ID               int64      `json:"id"`
	Key              string     `json:"key"`
	Name             string     `json:"name"`
	Status           string     `json:"status"`
	GroupID          int64      `json:"group_id"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	TokenLimitTotal  int64      `json:"token_limit_total"`
	TokenUsedTotal   int64      `json:"token_used_total"`
	ImageLimitTotal  int64      `json:"image_limit_total"`
	ImageUsedTotal   int64      `json:"image_used_total"`
	VideoLimitTotal  int64      `json:"video_limit_total"`
	VideoUsedTotal   int64      `json:"video_used_total"`
	CreatedByAdminID int64      `json:"created_by_admin_id"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (k *RetailGrokKey) IsExpired(now time.Time) bool {
	if k == nil || k.ExpiresAt == nil {
		return false
	}
	return now.After(*k.ExpiresAt)
}

func (k *RetailGrokKey) IsEnabled() bool {
	return k != nil && k.Status == RetailGrokKeyStatusActive
}

func (k *RetailGrokKey) TokenRemaining() int64 {
	if k == nil || k.TokenLimitTotal <= 0 {
		return 0
	}
	remaining := k.TokenLimitTotal - k.TokenUsedTotal
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (k *RetailGrokKey) ImageRemaining() int64 {
	if k == nil || k.ImageLimitTotal <= 0 {
		return 0
	}
	remaining := k.ImageLimitTotal - k.ImageUsedTotal
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (k *RetailGrokKey) VideoRemaining() int64 {
	if k == nil || k.VideoLimitTotal <= 0 {
		return 0
	}
	remaining := k.VideoLimitTotal - k.VideoUsedTotal
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (k *RetailGrokKey) ValidateAvailability(now time.Time, allowUsage bool) error {
	if k == nil {
		return ErrRetailGrokKeyNotFound
	}
	if !k.IsEnabled() {
		return ErrRetailGrokKeyDisabled
	}
	if k.IsExpired(now) {
		return ErrRetailGrokKeyExpired
	}
	if allowUsage {
		return nil
	}
	if k.TokenLimitTotal > 0 && k.TokenUsedTotal >= k.TokenLimitTotal {
		return ErrRetailGrokTokenExhausted
	}
	if k.ImageLimitTotal > 0 && k.ImageUsedTotal >= k.ImageLimitTotal {
		return ErrRetailGrokImageExhausted
	}
	if k.VideoLimitTotal > 0 && k.VideoUsedTotal >= k.VideoLimitTotal {
		return ErrRetailGrokVideoExhausted
	}
	return nil
}

type RetailGrokUsageLog struct {
	ID                int64      `json:"id"`
	RetailGrokKeyID   int64      `json:"retail_grok_key_id"`
	RequestID         string     `json:"request_id"`
	InboundEndpoint   string     `json:"inbound_endpoint"`
	Model             string     `json:"model"`
	InputTokens       int64      `json:"input_tokens"`
	OutputTokens      int64      `json:"output_tokens"`
	TotalTokens       int64      `json:"total_tokens"`
	ImageCount        int64      `json:"image_count"`
	VideoCount        int64      `json:"video_count"`
	Status            string     `json:"status"`
	ErrorMessage      string     `json:"error_message,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpstreamModel     string     `json:"upstream_model,omitempty"`
	UpstreamRequestID string     `json:"upstream_request_id,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
}

type RetailGrokUsageSummary struct {
	Key        *RetailGrokKey       `json:"key"`
	RecentLogs []RetailGrokUsageLog `json:"recent_logs"`
}

type RetailGrokBatchGenerateRequest struct {
	GroupID         int64  `json:"group_id"`
	Count           int    `json:"count"`
	NamePrefix      string `json:"name_prefix"`
	ExpiresInDays   *int   `json:"expires_in_days"`
	TokenLimitTotal int64  `json:"token_limit_total"`
	ImageLimitTotal int64  `json:"image_limit_total"`
	VideoLimitTotal int64  `json:"video_limit_total"`
}

type RetailGrokBatchGenerateResult struct {
	Keys []RetailGrokKey `json:"keys"`
}

type RetailGrokKeyRepository interface {
	Create(ctx context.Context, key *RetailGrokKey) error
	GetByID(ctx context.Context, id int64) (*RetailGrokKey, error)
	GetByKey(ctx context.Context, key string) (*RetailGrokKey, error)
	List(ctx context.Context, limit int) ([]RetailGrokKey, error)
	Delete(ctx context.Context, id int64) error
	IncrementUsage(ctx context.Context, id int64, tokens, images, videos int64) error
}

type RetailGrokUsageLogRepository interface {
	Create(ctx context.Context, log *RetailGrokUsageLog) error
	ListByKeyID(ctx context.Context, keyID int64, limit int) ([]RetailGrokUsageLog, error)
}
