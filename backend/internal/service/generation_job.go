package service

import (
	"context"
	"errors"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

type GenerationJobStatus string

const (
	GenerationJobStatusCreated    GenerationJobStatus = "created"
	GenerationJobStatusSubmitting GenerationJobStatus = "submitting"
	GenerationJobStatusQueued     GenerationJobStatus = "queued"
	GenerationJobStatusRunning    GenerationJobStatus = "running"
	GenerationJobStatusSucceeded  GenerationJobStatus = "succeeded"
	GenerationJobStatusFailed     GenerationJobStatus = "failed"
	GenerationJobStatusCancelled  GenerationJobStatus = "cancelled"
	GenerationJobStatusUnknown    GenerationJobStatus = "unknown"
)

type GenerationJobBillingStatus string

const (
	GenerationJobBillingStatusUnpriced     GenerationJobBillingStatus = "unpriced"
	GenerationJobBillingStatusEstimated    GenerationJobBillingStatus = "estimated"
	GenerationJobBillingStatusReserved     GenerationJobBillingStatus = "reserved"
	GenerationJobBillingStatusSubmitted    GenerationJobBillingStatus = "submitted"
	GenerationJobBillingStatusSettled      GenerationJobBillingStatus = "settled"
	GenerationJobBillingStatusRefunded     GenerationJobBillingStatus = "refunded"
	GenerationJobBillingStatusManualReview GenerationJobBillingStatus = "manual_review"
)

var (
	ErrGenerationJobNotFound            = infraerrors.NotFound("GENERATION_JOB_NOT_FOUND", "generation job not found")
	ErrGenerationJobConflict            = infraerrors.Conflict("GENERATION_JOB_CONFLICT", "generation job status changed")
	ErrGenerationJobDuePollTimeRequired = errors.New("generation job due poll time is required")
	ErrGenerationJobDuePollLimitInvalid = errors.New("generation job due poll limit must be positive")
)

const MaxGenerationJobDuePollBatchSize = 100

const LeonardoGenerationReconciliationDelay = time.Minute

type GenerationJob struct {
	ID                          int64
	PublicID                    string
	Provider                    string
	Modality                    string
	Model                       string
	UpstreamModel               string
	UserID                      int64
	APIKeyID                    int64
	GroupID                     *int64
	ProductID                   *int64
	OfferID                     *int64
	SourceGroupID               *int64
	Operation                   *string
	CustomerPriceVersion        *string
	AccountID                   int64
	UpstreamGenerationID        *string
	Status                      GenerationJobStatus
	UpstreamStatus              *string
	RequestHash                 string
	RequestPayload              map[string]any
	ResultPayload               map[string]any
	ErrorCode                   *string
	ErrorMessage                *string
	OutputCount                 int
	EstimatedUpstreamCostAmount *decimal.Decimal
	EstimatedUpstreamCostUnit   *string
	PricingSnapshotVersion      *string
	PricingSource               *string
	PricingMatchType            *string
	ActualUpstreamCostAmount    *decimal.Decimal
	ActualUpstreamCostUnit      *string
	CustomerCost                *decimal.Decimal
	GrossMargin                 *decimal.Decimal
	CostVariance                *decimal.Decimal
	BillingStatus               GenerationJobBillingStatus
	BillingReference            *string
	PollAttempts                int
	NextPollAt                  *time.Time
	LastPolledAt                *time.Time
	SubmittedAt                 *time.Time
	StartedAt                   *time.Time
	CompletedAt                 *time.Time
	FailedAt                    *time.Time
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
}

type GenerationJobRepository interface {
	Create(ctx context.Context, job *GenerationJob) error
	GetByPublicID(ctx context.Context, publicID string) (*GenerationJob, error)
	GetByUpstreamGenerationID(ctx context.Context, upstreamGenerationID string) (*GenerationJob, error)
	CompareAndSwapStatus(ctx context.Context, publicID string, expectedStatus GenerationJobStatus, job *GenerationJob) error
}

type GenerationJobPollRepository interface {
	GetByPublicID(ctx context.Context, publicID string) (*GenerationJob, error)
	CompareAndSwapPoll(ctx context.Context, publicID string, expectedStatus GenerationJobStatus, expectedPollAttempts int, job *GenerationJob) error
}

type GenerationJobDuePollRepository interface {
	ListDueLeonardoPollJobs(ctx context.Context, dueAt time.Time, limit int) ([]*GenerationJob, error)
}

func NormalizeGenerationJob(job *GenerationJob) {
	if job == nil || job.Status != GenerationJobStatusUnknown {
		return
	}
	errorCode := "submission_unknown"
	job.ErrorCode = &errorCode
	job.BillingStatus = GenerationJobBillingStatusManualReview
	job.ActualUpstreamCostAmount = nil
	job.ActualUpstreamCostUnit = nil
	job.GrossMargin = nil
	job.CostVariance = nil
}

// NormalizeMediaStatusToken lowercases/trims status strings from internal jobs
// and OpenAI-compatible media APIs so callers can compare aliases uniformly.
func NormalizeMediaStatusToken(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

// IsTerminalMediaSuccessStatus accepts both internal ("succeeded") and
// OpenAI-compatible ("completed") success labels for image/video/audio media.
func IsTerminalMediaSuccessStatus(status string) bool {
	switch NormalizeMediaStatusToken(status) {
	case string(GenerationJobStatusSucceeded), "completed", "ready", "done":
		return true
	default:
		return false
	}
}

// IsTerminalMediaFailureStatus accepts internal and OpenAI-compatible failure /
// cancel labels for image/video/audio media polling and sync wait loops.
func IsTerminalMediaFailureStatus(status string) bool {
	switch NormalizeMediaStatusToken(status) {
	case string(GenerationJobStatusFailed), string(GenerationJobStatusUnknown), string(GenerationJobStatusCancelled),
		"error", "canceled", "rejected", "expired":
		// GenerationJobStatusCancelled already covers "cancelled".
		return true
	default:
		return false
	}
}

func CanTransitionGenerationJobStatus(from, to GenerationJobStatus) bool {
	if from == to {
		return true
	}
	switch from {
	case GenerationJobStatusCreated:
		return to == GenerationJobStatusSubmitting
	case GenerationJobStatusSubmitting:
		return to == GenerationJobStatusQueued || to == GenerationJobStatusRunning || to == GenerationJobStatusSucceeded || to == GenerationJobStatusFailed || to == GenerationJobStatusUnknown
	case GenerationJobStatusQueued:
		return to == GenerationJobStatusRunning || to == GenerationJobStatusSucceeded || to == GenerationJobStatusFailed || to == GenerationJobStatusCancelled
	case GenerationJobStatusRunning:
		return to == GenerationJobStatusSucceeded || to == GenerationJobStatusFailed || to == GenerationJobStatusCancelled
	case GenerationJobStatusUnknown:
		return to == GenerationJobStatusQueued || to == GenerationJobStatusRunning || to == GenerationJobStatusSucceeded || to == GenerationJobStatusFailed
	default:
		return false
	}
}
