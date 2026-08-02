package service

import (
	"context"
	"errors"
	"time"
)

var (
	ErrLeonardoGenerationPollBatchRunnerNotConfigured = errors.New("leonardo generation poll batch runner is not configured")
	ErrLeonardoGenerationPollBatchJobInvalid          = errors.New("leonardo generation poll batch job is invalid")
)

type LeonardoGenerationPollExecutor interface {
	Poll(context.Context, string) (*GenerationJob, error)
}

type LeonardoGenerationPollBatchResult struct {
	Scanned   int
	Attempted int
	Succeeded int
	Failed    int
	Failures  []LeonardoGenerationPollBatchFailure
}

type LeonardoGenerationPollBatchFailure struct {
	PublicID string
	Err      error
}

type LeonardoGenerationPollBatchRunner struct {
	repository GenerationJobDuePollRepository
	executor   LeonardoGenerationPollExecutor
}

func NewLeonardoGenerationPollBatchRunner(repository GenerationJobDuePollRepository, executor LeonardoGenerationPollExecutor) *LeonardoGenerationPollBatchRunner {
	return &LeonardoGenerationPollBatchRunner{repository: repository, executor: executor}
}

func (r *LeonardoGenerationPollBatchRunner) RunOnce(ctx context.Context, dueAt time.Time, limit int) (*LeonardoGenerationPollBatchResult, error) {
	if r == nil || r.repository == nil || r.executor == nil {
		return nil, ErrLeonardoGenerationPollBatchRunnerNotConfigured
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if dueAt.IsZero() {
		return nil, ErrGenerationJobDuePollTimeRequired
	}
	if limit <= 0 {
		return nil, ErrGenerationJobDuePollLimitInvalid
	}
	if limit > MaxGenerationJobDuePollBatchSize {
		limit = MaxGenerationJobDuePollBatchSize
	}
	jobs, err := r.repository.ListDueLeonardoPollJobs(ctx, dueAt.UTC(), limit)
	if err != nil {
		return nil, err
	}
	result := &LeonardoGenerationPollBatchResult{Scanned: len(jobs)}
	for _, job := range jobs {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if job == nil || job.PublicID == "" {
			result.Attempted++
			result.Failed++
			failure := LeonardoGenerationPollBatchFailure{Err: ErrLeonardoGenerationPollBatchJobInvalid}
			if job != nil {
				failure.PublicID = job.PublicID
			}
			result.Failures = append(result.Failures, failure)
			continue
		}
		result.Attempted++
		_, pollErr := r.executor.Poll(ctx, job.PublicID)
		if pollErr == nil {
			result.Succeeded++
			continue
		}
		result.Failed++
		result.Failures = append(result.Failures, LeonardoGenerationPollBatchFailure{PublicID: job.PublicID, Err: pollErr})
		if err := ctx.Err(); err != nil {
			return result, err
		}
	}
	return result, nil
}
