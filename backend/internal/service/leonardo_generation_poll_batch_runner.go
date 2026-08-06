package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
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
	interval   time.Duration
	batchSize  int
	mu         sync.Mutex
	cancel     context.CancelFunc
	started    bool
	wg         sync.WaitGroup
	ops        OpsRepository
}

func NewLeonardoGenerationPollBatchRunner(repository GenerationJobDuePollRepository, executor LeonardoGenerationPollExecutor, ops ...OpsRepository) *LeonardoGenerationPollBatchRunner {
	runner := &LeonardoGenerationPollBatchRunner{repository: repository, executor: executor, interval: time.Second, batchSize: MaxGenerationJobDuePollBatchSize}
	if len(ops) > 0 {
		runner.ops = ops[0]
	}
	return runner
}

func (r *LeonardoGenerationPollBatchRunner) Start() {
	if r == nil || r.repository == nil || r.executor == nil || r.interval <= 0 || r.batchSize <= 0 {
		return
	}
	r.mu.Lock()
	if r.started {
		r.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.started = true
	r.wg.Add(1)
	r.mu.Unlock()
	go r.run(ctx)
}

func (r *LeonardoGenerationPollBatchRunner) Stop() {
	if r == nil {
		return
	}
	r.mu.Lock()
	cancel := r.cancel
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	r.wg.Wait()
}

func (r *LeonardoGenerationPollBatchRunner) run(ctx context.Context) {
	defer r.wg.Done()
	r.runScheduledBatch(ctx)
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.runScheduledBatch(ctx)
		}
	}
}

func (r *LeonardoGenerationPollBatchRunner) runScheduledBatch(ctx context.Context) {
	started := time.Now().UTC()
	result, err := r.RunOnce(ctx, time.Now().UTC(), r.batchSize)
	r.heartbeat(started, result, err)
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("[LeonardoPoll] Batch failed: %v", err)
		return
	}
	if result != nil && result.Failed > 0 {
		log.Printf("[LeonardoPoll] Batch completed with failures: scanned=%d attempted=%d succeeded=%d failed=%d", result.Scanned, result.Attempted, result.Succeeded, result.Failed)
	}
}

func (r *LeonardoGenerationPollBatchRunner) heartbeat(started time.Time, result *LeonardoGenerationPollBatchResult, runErr error) {
	if r.ops == nil {
		return
	}
	now, duration := time.Now().UTC(), time.Since(started).Milliseconds()
	input := &OpsUpsertJobHeartbeatInput{JobName: "leonardo_generation_poll_worker", LastRunAt: &now, LastDurationMs: &duration}
	if runErr != nil {
		message := runErr.Error()
		input.LastErrorAt, input.LastError = &now, &message
	} else {
		input.LastSuccessAt = &now
		if result != nil {
			summary := fmt.Sprintf("scanned=%d attempted=%d succeeded=%d failed=%d", result.Scanned, result.Attempted, result.Succeeded, result.Failed)
			input.LastResult = &summary
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = r.ops.UpsertJobHeartbeat(ctx, input)
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
