package service

import (
	"context"
	"errors"
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

type GenerationJob struct {
	ID                       int64
	PublicID                 string
	Provider                 string
	Modality                 string
	Model                    string
	UpstreamModel            string
	UserID                   int64
	APIKeyID                 int64
	GroupID                  *int64
	AccountID                int64
	UpstreamGenerationID     *string
	Status                   GenerationJobStatus
	UpstreamStatus           *string
	RequestHash              string
	RequestPayload           map[string]any
	ResultPayload            map[string]any
	ErrorCode                *string
	ErrorMessage             *string
	OutputCount              int
	ActualUpstreamCostAmount *decimal.Decimal
	ActualUpstreamCostUnit   *string
	CustomerCost             *decimal.Decimal
	BillingStatus            GenerationJobBillingStatus
	BillingReference         *string
	PollAttempts             int
	NextPollAt               *time.Time
	LastPolledAt             *time.Time
	SubmittedAt              *time.Time
	StartedAt                *time.Time
	CompletedAt              *time.Time
	FailedAt                 *time.Time
	CreatedAt                time.Time
	UpdatedAt                time.Time
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
	job.CustomerCost = nil
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
