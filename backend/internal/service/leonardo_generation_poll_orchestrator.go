package service

import (
	"context"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
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
	CompareAndSwapStatus(context.Context, string, GenerationJobStatus, *GenerationJob) error
}

type LeonardoGenerationPollOrchestrator struct {
	repository    LeonardoGenerationPollRepository
	accountLoader LeonardoGenerationPollAccountLoader
	upstream      HTTPUpstream
	config        *config.Config
	clock         LeonardoGenerationPollClock
	funds         LeonardoImageTerminalFunds
}

func NewLeonardoGenerationPollOrchestrator(repository LeonardoGenerationPollRepository, accountLoader LeonardoGenerationPollAccountLoader, upstream HTTPUpstream, cfg *config.Config, clock LeonardoGenerationPollClock, funds LeonardoImageTerminalFunds) *LeonardoGenerationPollOrchestrator {
	return &LeonardoGenerationPollOrchestrator{repository: repository, accountLoader: accountLoader, upstream: upstream, config: cfg, clock: clock, funds: funds}
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
	if terminalLeonardoGenerationJob(job) {
		return o.reconcileBilling(ctx, job)
	}
	if (job.Status == GenerationJobStatusSucceeded && job.BillingStatus == GenerationJobBillingStatusSettled) || (job.Status == GenerationJobStatusFailed && job.BillingStatus == GenerationJobBillingStatusRefunded) {
		return job, nil
	}
	now := o.clock.Now().UTC()
	if (job.Status != GenerationJobStatusQueued && job.Status != GenerationJobStatusRunning) || job.UpstreamGenerationID == nil || !validLeonardoGenerationUUID(*job.UpstreamGenerationID) || (job.NextPollAt != nil && job.NextPollAt.After(now)) {
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
	job, err = NewLeonardoGenerationPoller(o.repository, adapter, o.clock).Poll(ctx, publicID)
	if err != nil || !terminalLeonardoGenerationJob(job) {
		return job, err
	}
	return o.reconcileBilling(ctx, job)
}

func terminalLeonardoGenerationJob(job *GenerationJob) bool {
	return job != nil && job.BillingStatus == GenerationJobBillingStatusSubmitted && (job.Status == GenerationJobStatusSucceeded || job.Status == GenerationJobStatusFailed)
}

func (o *LeonardoGenerationPollOrchestrator) reconcileBilling(ctx context.Context, job *GenerationJob) (*GenerationJob, error) {
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
	if err = o.repository.CompareAndSwapStatus(ctx, job.PublicID, job.Status, &updated); err != nil {
		return job, err
	}
	return &updated, nil
}

func validLeonardoFundsAmount(amount decimal.Decimal) bool {
	return amount.Sign() > 0 && amount.Cmp(decimal.RequireFromString("999999999999.99999999")) <= 0 && amount.Equal(amount.Truncate(8))
}
