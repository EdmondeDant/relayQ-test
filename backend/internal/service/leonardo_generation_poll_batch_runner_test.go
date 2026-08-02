package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type pollBatchRepositoryMock struct {
	jobs  []*GenerationJob
	err   error
	calls int
	dueAt time.Time
	limit int
}

func (r *pollBatchRepositoryMock) ListDueLeonardoPollJobs(_ context.Context, dueAt time.Time, limit int) ([]*GenerationJob, error) {
	r.calls++
	r.dueAt = dueAt
	r.limit = limit
	return r.jobs, r.err
}

type pollBatchExecutorMock struct {
	mu        sync.Mutex
	calls     []string
	errors    map[string]error
	active    int
	maxActive int
	cancel    context.CancelFunc
	cancelOn  string
}

func (e *pollBatchExecutorMock) Poll(_ context.Context, publicID string) (*GenerationJob, error) {
	e.mu.Lock()
	e.active++
	if e.active > e.maxActive {
		e.maxActive = e.active
	}
	e.calls = append(e.calls, publicID)
	err := e.errors[publicID]
	if publicID == e.cancelOn {
		e.cancel()
		err = context.Canceled
	}
	e.active--
	e.mu.Unlock()
	return nil, err
}

func TestLeonardoGenerationPollBatchRunnerSerialSuccess(t *testing.T) {
	dueAt := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	repository := &pollBatchRepositoryMock{jobs: pollBatchJobs("job-1", "job-2", "job-3")}
	executor := &pollBatchExecutorMock{}
	result, err := NewLeonardoGenerationPollBatchRunner(repository, executor).RunOnce(context.Background(), dueAt, 3)
	require.NoError(t, err)
	require.Equal(t, []string{"job-1", "job-2", "job-3"}, executor.calls)
	require.Equal(t, 1, executor.maxActive)
	require.Equal(t, 3, result.Scanned)
	require.Equal(t, 3, result.Attempted)
	require.Equal(t, 3, result.Succeeded)
	require.Zero(t, result.Failed)
	require.Empty(t, result.Failures)
	require.Equal(t, dueAt.UTC(), repository.dueAt)
}

func TestLeonardoGenerationPollBatchRunnerEmptyResult(t *testing.T) {
	for _, jobs := range [][]*GenerationJob{nil, {}} {
		result, err := NewLeonardoGenerationPollBatchRunner(&pollBatchRepositoryMock{jobs: jobs}, &pollBatchExecutorMock{}).RunOnce(context.Background(), time.Now(), 1)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Zero(t, result.Scanned)
		require.Empty(t, result.Failures)
	}
}

func TestLeonardoGenerationPollBatchRunnerValidation(t *testing.T) {
	dueAt := time.Now()
	var runner *LeonardoGenerationPollBatchRunner
	_, err := runner.RunOnce(context.Background(), dueAt, 1)
	require.ErrorIs(t, err, ErrLeonardoGenerationPollBatchRunnerNotConfigured)
	_, err = NewLeonardoGenerationPollBatchRunner(nil, &pollBatchExecutorMock{}).RunOnce(context.Background(), dueAt, 1)
	require.ErrorIs(t, err, ErrLeonardoGenerationPollBatchRunnerNotConfigured)
	_, err = NewLeonardoGenerationPollBatchRunner(&pollBatchRepositoryMock{}, nil).RunOnce(context.Background(), dueAt, 1)
	require.ErrorIs(t, err, ErrLeonardoGenerationPollBatchRunnerNotConfigured)
	repository := &pollBatchRepositoryMock{}
	runner = NewLeonardoGenerationPollBatchRunner(repository, &pollBatchExecutorMock{})
	_, err = runner.RunOnce(context.Background(), time.Time{}, 1)
	require.ErrorIs(t, err, ErrGenerationJobDuePollTimeRequired)
	_, err = runner.RunOnce(context.Background(), dueAt, 0)
	require.ErrorIs(t, err, ErrGenerationJobDuePollLimitInvalid)
	_, err = runner.RunOnce(context.Background(), dueAt, -1)
	require.ErrorIs(t, err, ErrGenerationJobDuePollLimitInvalid)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = runner.RunOnce(ctx, dueAt, 1)
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, repository.calls)
}

func TestLeonardoGenerationPollBatchRunnerClampsLimit(t *testing.T) {
	repository := &pollBatchRepositoryMock{}
	_, err := NewLeonardoGenerationPollBatchRunner(repository, &pollBatchExecutorMock{}).RunOnce(context.Background(), time.Now(), 101)
	require.NoError(t, err)
	require.Equal(t, MaxGenerationJobDuePollBatchSize, repository.limit)
	require.Equal(t, 1, repository.calls)
}

func TestLeonardoGenerationPollBatchRunnerIsolatesFailures(t *testing.T) {
	for _, middleErr := range []error{errors.New("temporary"), ErrGenerationJobConflict} {
		repository := &pollBatchRepositoryMock{jobs: pollBatchJobs("job-1", "job-2", "job-3")}
		executor := &pollBatchExecutorMock{errors: map[string]error{"job-2": middleErr}}
		result, err := NewLeonardoGenerationPollBatchRunner(repository, executor).RunOnce(context.Background(), time.Now(), 3)
		require.NoError(t, err)
		require.Equal(t, []string{"job-1", "job-2", "job-3"}, executor.calls)
		require.Equal(t, 3, result.Attempted)
		require.Equal(t, 2, result.Succeeded)
		require.Equal(t, 1, result.Failed)
		require.Len(t, result.Failures, 1)
		require.Equal(t, "job-2", result.Failures[0].PublicID)
		require.ErrorIs(t, result.Failures[0].Err, middleErr)
	}
}

func TestLeonardoGenerationPollBatchRunnerRejectsInvalidJobs(t *testing.T) {
	repository := &pollBatchRepositoryMock{jobs: []*GenerationJob{nil, {PublicID: ""}, {PublicID: "job-3"}}}
	executor := &pollBatchExecutorMock{}
	result, err := NewLeonardoGenerationPollBatchRunner(repository, executor).RunOnce(context.Background(), time.Now(), 3)
	require.NoError(t, err)
	require.Equal(t, []string{"job-3"}, executor.calls)
	require.Equal(t, 3, result.Scanned)
	require.Equal(t, 3, result.Attempted)
	require.Equal(t, 1, result.Succeeded)
	require.Equal(t, 2, result.Failed)
	require.Len(t, result.Failures, 2)
	for _, failure := range result.Failures {
		require.ErrorIs(t, failure.Err, ErrLeonardoGenerationPollBatchJobInvalid)
	}
}

func TestLeonardoGenerationPollBatchRunnerStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	repository := &pollBatchRepositoryMock{jobs: pollBatchJobs("job-1", "job-2", "job-3")}
	executor := &pollBatchExecutorMock{cancel: cancel, cancelOn: "job-2"}
	result, err := NewLeonardoGenerationPollBatchRunner(repository, executor).RunOnce(ctx, time.Now(), 3)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, []string{"job-1", "job-2"}, executor.calls)
	require.Equal(t, 3, result.Scanned)
	require.Equal(t, 2, result.Attempted)
	require.Equal(t, 1, result.Succeeded)
	require.Equal(t, 1, result.Failed)
	require.Len(t, result.Failures, 1)
}

func TestLeonardoGenerationPollBatchRunnerRepositoryFailure(t *testing.T) {
	repositoryErr := errors.New("scan failed")
	repository := &pollBatchRepositoryMock{err: repositoryErr}
	executor := &pollBatchExecutorMock{}
	result, err := NewLeonardoGenerationPollBatchRunner(repository, executor).RunOnce(context.Background(), time.Now(), 1)
	require.Nil(t, result)
	require.ErrorIs(t, err, repositoryErr)
	require.Equal(t, 1, repository.calls)
	require.Empty(t, executor.calls)
}

func pollBatchJobs(publicIDs ...string) []*GenerationJob {
	jobs := make([]*GenerationJob, len(publicIDs))
	for i, publicID := range publicIDs {
		jobs[i] = &GenerationJob{PublicID: publicID}
	}
	return jobs
}
