package service

import (
	"context"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

var (
	ErrLeonardoGenerationPollProviderMismatch = errors.New("generation job provider is not leonardo")
	ErrLeonardoGenerationPollAccountBinding   = errors.New("leonardo generation job account binding is invalid")
)

type LeonardoGenerationPollAccountLoader interface {
	GetByID(context.Context, int64) (*Account, error)
}

type LeonardoGenerationPollOrchestrator struct {
	repository    GenerationJobPollRepository
	accountLoader LeonardoGenerationPollAccountLoader
	upstream      HTTPUpstream
	config        *config.Config
	clock         LeonardoGenerationPollClock
}

func NewLeonardoGenerationPollOrchestrator(repository GenerationJobPollRepository, accountLoader LeonardoGenerationPollAccountLoader, upstream HTTPUpstream, cfg *config.Config, clock LeonardoGenerationPollClock) *LeonardoGenerationPollOrchestrator {
	return &LeonardoGenerationPollOrchestrator{repository: repository, accountLoader: accountLoader, upstream: upstream, config: cfg, clock: clock}
}

func (o *LeonardoGenerationPollOrchestrator) Poll(ctx context.Context, publicID string) (*GenerationJob, error) {
	if o == nil || o.repository == nil || o.accountLoader == nil || o.upstream == nil || o.config == nil || o.clock == nil {
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
	return NewLeonardoGenerationPoller(o.repository, adapter, o.clock).Poll(ctx, publicID)
}
