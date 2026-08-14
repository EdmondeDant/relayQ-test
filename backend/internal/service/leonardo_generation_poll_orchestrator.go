package service

import (
	"context"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
	"github.com/shopspring/decimal"
)

var (
	ErrLeonardoGenerationPollProviderMismatch = errors.New("generation job provider is not leonardo")
	ErrLeonardoGenerationPollAccountBinding   = errors.New("leonardo generation job account binding is invalid")
)

type LeonardoGenerationPollAccountLoader interface {
	GetByID(context.Context, int64) (*Account, error)
}

type LeonardoGenerationPollRepository interface {
	GenerationJobPollRepository
	GetByUpstreamGenerationID(context.Context, string) (*GenerationJob, error)
	CompareAndSwapStatus(context.Context, string, GenerationJobStatus, *GenerationJob) error
}

type LeonardoUsageLogWriter interface {
	Create(context.Context, *UsageLog) (bool, error)
}

type LeonardoGenerationPollOrchestrator struct {
	repository    LeonardoGenerationPollRepository
	accountLoader LeonardoGenerationPollAccountLoader
	upstream      HTTPUpstream
	config        *config.Config
	clock         LeonardoGenerationPollClock
	funds         LeonardoImageTerminalFunds
	moderator     LeonardoOutputModerator
	usageLogs     LeonardoUsageLogWriter
}

func (o *LeonardoGenerationPollOrchestrator) SetUsageLogWriter(writer LeonardoUsageLogWriter) {
	if o != nil {
		o.usageLogs = writer
	}
}

func NewLeonardoGenerationPollOrchestrator(repository LeonardoGenerationPollRepository, accountLoader LeonardoGenerationPollAccountLoader, upstream HTTPUpstream, cfg *config.Config, clock LeonardoGenerationPollClock, funds LeonardoImageTerminalFunds, moderators ...LeonardoOutputModerator) *LeonardoGenerationPollOrchestrator {
	var moderator LeonardoOutputModerator
	if len(moderators) > 0 {
		moderator = moderators[0]
	}
	return &LeonardoGenerationPollOrchestrator{repository: repository, accountLoader: accountLoader, upstream: upstream, config: cfg, clock: clock, funds: funds, moderator: moderator}
}

func (o *LeonardoGenerationPollOrchestrator) Poll(ctx context.Context, publicID string) (*GenerationJob, error) {
	if o == nil || o.repository == nil || o.accountLoader == nil || o.upstream == nil || o.config == nil || o.clock == nil || o.funds == nil {
		return nil, errors.New("leonardo generation poll orchestrator is not configured")
	}
	job, err := o.repository.GetByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, ErrGenerationJobNotFound
	}
	if job.Provider != PlatformLeonardo {
		return job, ErrLeonardoGenerationPollProviderMismatch
	}
	now := o.clock.Now().UTC()
	if job.Status == GenerationJobStatusSubmitting {
		if job.UpdatedAt.IsZero() || job.UpdatedAt.After(now.Add(-LeonardoGenerationReconciliationDelay)) {
			return job, nil
		}
		updated := *job
		updated.Status = GenerationJobStatusUnknown
		NormalizeGenerationJob(&updated)
		if err = o.repository.CompareAndSwapStatus(ctx, job.PublicID, job.Status, &updated); err != nil {
			return job, err
		}
		return &updated, nil
	}
	if terminalLeonardoGenerationJob(job) {
		return o.reconcileBilling(ctx, job)
	}
	if (job.Status == GenerationJobStatusSucceeded && job.BillingStatus == GenerationJobBillingStatusSettled) || (job.Status == GenerationJobStatusFailed && job.BillingStatus == GenerationJobBillingStatusRefunded) {
		return job, nil
	}
	if (job.Status != GenerationJobStatusQueued && job.Status != GenerationJobStatusRunning && job.Status != GenerationJobStatusUnknown) || job.UpstreamGenerationID == nil || !validLeonardoGenerationUUID(*job.UpstreamGenerationID) || (job.NextPollAt != nil && job.NextPollAt.After(now)) {
		return job, nil
	}
	if job.AccountID <= 0 {
		return job, ErrLeonardoGenerationPollAccountBinding
	}
	account, err := o.accountLoader.GetByID(ctx, job.AccountID)
	if err != nil {
		return job, err
	}
	if account == nil {
		return job, ErrAccountNotFound
	}
	if account.ID != job.AccountID {
		return job, ErrLeonardoGenerationPollAccountBinding
	}
	adapter, err := NewLeonardoGenerationAdapter(account, o.upstream, o.config)
	if err != nil {
		return job, err
	}
	if job.Status == GenerationJobStatusUnknown {
		generation, queryErr := adapter.GetGeneration(ctx, *job.UpstreamGenerationID)
		if queryErr != nil {
			return job, queryErr
		}
		if generation == nil {
			return job, errors.New("leonardo: empty generation response")
		}
		updated, applyErr := applyLeonardoGenerationResult(ctx, job, generation, now, o.moderator)
		if job.BillingStatus == GenerationJobBillingStatusManualReview && validLeonardoTerminalBilling(job) {
			updated.BillingStatus = GenerationJobBillingStatusSubmitted
		} else if job.BillingStatus != GenerationJobBillingStatusSubmitted {
			updated.BillingStatus = GenerationJobBillingStatusManualReview
		}
		if applyErr != nil && updated.ErrorCode == nil {
			return job, applyErr
		}
		if err = o.repository.CompareAndSwapStatus(ctx, job.PublicID, job.Status, &updated); err != nil {
			return job, errors.Join(applyErr, err)
		}
		if terminalLeonardoGenerationJob(&updated) {
			return o.reconcileBilling(ctx, &updated)
		}
		return &updated, applyErr
	}
	job, err = NewLeonardoGenerationPoller(o.repository, adapter, o.clock, o.moderator).Poll(ctx, publicID)
	if err != nil || !terminalLeonardoGenerationJob(job) {
		return job, err
	}
	return o.reconcileBilling(ctx, job)
}

func validLeonardoTerminalBilling(job *GenerationJob) bool {
	return job != nil && job.UserID > 0 && job.CustomerCost != nil && validLeonardoFundsAmount(*job.CustomerCost) && job.BillingReference != nil && strings.TrimSpace(*job.BillingReference) != ""
}

func (o *LeonardoGenerationPollOrchestrator) ApplyWebhook(ctx context.Context, accountID int64, generation *leonardo.Generation) (*GenerationJob, error) {
	if o == nil || o.repository == nil || o.clock == nil || o.funds == nil || generation == nil || strings.TrimSpace(generation.ID) == "" {
		return nil, errors.New("leonardo webhook processor is not configured")
	}
	job, err := o.repository.GetByUpstreamGenerationID(ctx, strings.TrimSpace(generation.ID))
	if err != nil {
		return nil, err
	}
	if job.Provider != PlatformLeonardo || job.AccountID != accountID {
		return job, ErrLeonardoGenerationPollAccountBinding
	}
	if err = requireLeonardoSupportedModality(job.Modality); err != nil {
		return job, err
	}
	if job.Status == GenerationJobStatusSucceeded || job.Status == GenerationJobStatusFailed || job.Status == GenerationJobStatusCancelled {
		return job, nil
	}
	updated, err := applyLeonardoGenerationResult(ctx, job, generation, o.clock.Now().UTC(), o.moderator)
	if err != nil && updated.ErrorCode == nil {
		return job, err
	}
	if !CanTransitionGenerationJobStatus(job.Status, updated.Status) {
		return job, ErrGenerationJobConflict
	}
	if err = o.repository.CompareAndSwapStatus(ctx, job.PublicID, job.Status, &updated); err != nil {
		latest, readErr := o.repository.GetByPublicID(ctx, job.PublicID)
		if readErr == nil && latest != nil && (latest.Status == GenerationJobStatusSucceeded || latest.Status == GenerationJobStatusFailed || latest.Status == GenerationJobStatusCancelled) {
			return latest, nil
		}
		return job, errors.Join(err, readErr)
	}
	if terminalLeonardoGenerationJob(&updated) {
		return o.reconcileBilling(ctx, &updated)
	}
	return &updated, err
}

func terminalLeonardoGenerationJob(job *GenerationJob) bool {
	return job != nil && job.ProductID == nil && job.BillingStatus == GenerationJobBillingStatusSubmitted && (job.Status == GenerationJobStatusSucceeded || job.Status == GenerationJobStatusFailed)
}

func (o *LeonardoGenerationPollOrchestrator) reconcileBilling(ctx context.Context, job *GenerationJob) (*GenerationJob, error) {
	if requireLeonardoSupportedModality(job.Modality) != nil {
		updated := *job
		updated.BillingStatus = GenerationJobBillingStatusManualReview
		if err := o.repository.CompareAndSwapStatus(ctx, job.PublicID, job.Status, &updated); err != nil {
			return job, err
		}
		return &updated, nil
	}
	if job.UserID <= 0 || strings.TrimSpace(job.PublicID) == "" || len(strings.TrimSpace(job.PublicID)) > 64 || job.CustomerCost == nil || !validLeonardoFundsAmount(*job.CustomerCost) || job.BillingReference == nil || strings.TrimSpace(*job.BillingReference) == "" || len(strings.TrimSpace(*job.BillingReference)) > 128 {
		return job, ErrLeonardoImageCreateReservationInvalid
	}
	reference := strings.TrimSpace(*job.BillingReference)
	var err error
	updated := *job
	if job.Status == GenerationJobStatusSucceeded {
		err = o.funds.Settle(ctx, LeonardoImageFundsSettleRequest{UserID: job.UserID, PublicID: strings.TrimSpace(job.PublicID), Reference: reference, AmountUSD: *job.CustomerCost})
		updated.BillingStatus = GenerationJobBillingStatusSettled
	} else {
		err = o.funds.Release(ctx, LeonardoImageFundsReleaseRequest{UserID: job.UserID, PublicID: strings.TrimSpace(job.PublicID), Reference: reference, AmountUSD: *job.CustomerCost, Reason: "generation_failed"})
		updated.BillingStatus = GenerationJobBillingStatusRefunded
	}
	if err != nil {
		return job, err
	}
	if job.Status == GenerationJobStatusSucceeded && o.usageLogs != nil {
		if _, err = o.usageLogs.Create(ctx, leonardoUsageLog(job)); err != nil {
			return job, err
		}
	}
	if err = o.repository.CompareAndSwapStatus(ctx, job.PublicID, job.Status, &updated); err != nil {
		return job, err
	}
	return &updated, nil
}

func leonardoUsageLog(job *GenerationJob) *UsageLog {
	customerCost, _ := job.CustomerCost.Float64()
	billingMode := strings.ToLower(strings.TrimSpace(job.Modality))
	if billingMode == "" {
		billingMode = "image"
	}
	imageSize := "1K"
	inboundEndpoint := "/v1/media/generations"
	upstreamEndpoint := "/api/rest/v2/generations"
	upstreamModel := job.UpstreamModel
	return &UsageLog{
		UserID:         job.UserID,
		APIKeyID:       job.APIKeyID,
		AccountID:      job.AccountID,
		RequestID:      job.PublicID,
		Model:          job.Model,
		RequestedModel: job.Model,
		UpstreamModel:  &upstreamModel,
		GroupID:        job.GroupID,
		TotalCost:      customerCost,
		ActualCost:     customerCost,
		RateMultiplier: 1,
		ImageCount: func() int {
			if billingMode == "image" {
				return job.OutputCount
			}
			return 0
		}(),
		ImageSize:        &imageSize,
		BillingMode:      &billingMode,
		InboundEndpoint:  &inboundEndpoint,
		UpstreamEndpoint: &upstreamEndpoint,
	}
}

func validLeonardoFundsAmount(amount decimal.Decimal) bool {
	return amount.Sign() > 0 && amount.Cmp(decimal.RequireFromString("999999999999.99999999")) <= 0 && amount.Equal(amount.Truncate(8))
}
