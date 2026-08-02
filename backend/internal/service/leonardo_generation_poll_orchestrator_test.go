package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

const orchestratorGenerationID = "123e4567-e89b-42d3-a456-426614174000"

type orchestratorRepositoryMock struct {
	mu       sync.Mutex
	job      *GenerationJob
	reads    int
	readHook func(int, *GenerationJob)
	casCalls int
}

func (r *orchestratorRepositoryMock) GetByPublicID(context.Context, string) (*GenerationJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reads++
	if r.job == nil {
		return nil, ErrGenerationJobNotFound
	}
	job := *r.job
	if r.readHook != nil {
		r.readHook(r.reads, &job)
	}
	return &job, nil
}

func (r *orchestratorRepositoryMock) CompareAndSwapPoll(_ context.Context, publicID string, status GenerationJobStatus, attempts int, job *GenerationJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.casCalls++
	if r.job == nil || r.job.PublicID != publicID {
		return ErrGenerationJobNotFound
	}
	if r.job.Status != status || r.job.PollAttempts != attempts {
		return ErrGenerationJobConflict
	}
	stored := *job
	r.job = &stored
	return nil
}

type orchestratorAccountLoaderMock struct {
	mu      sync.Mutex
	account *Account
	err     error
	calls   int
	ids     []int64
}

func (l *orchestratorAccountLoaderMock) GetByID(_ context.Context, id int64) (*Account, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls++
	l.ids = append(l.ids, id)
	return l.account, l.err
}

type orchestratorUpstreamMock struct {
	mu          sync.Mutex
	calls       int
	postCalls   int
	tlsCalls    int
	accountID   int64
	concurrency int
	proxyURL    string
	method      string
	path        string
	barrier     chan struct{}
	release     chan struct{}
}

func (u *orchestratorUpstreamMock) Do(req *http.Request, proxyURL string, accountID int64, concurrency int) (*http.Response, error) {
	u.mu.Lock()
	u.calls++
	if req.Method == http.MethodPost {
		u.postCalls++
	}
	u.accountID = accountID
	u.concurrency = concurrency
	u.proxyURL = proxyURL
	u.method = req.Method
	u.path = req.URL.Path
	u.mu.Unlock()
	if u.barrier != nil {
		u.barrier <- struct{}{}
	}
	if u.release != nil {
		<-u.release
	}
	body := fmt.Sprintf(`{"generations_by_pk":{"id":%q,"status":"PENDING"}}`, orchestratorGenerationID)
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
}

func (u *orchestratorUpstreamMock) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	u.mu.Lock()
	u.tlsCalls++
	u.mu.Unlock()
	return u.Do(req, proxyURL, accountID, concurrency)
}

func TestLeonardoGenerationPollOrchestratorEligibleAccountBoundGet(t *testing.T) {
	now := time.Date(2026, time.August, 3, 3, 0, 0, 0, time.UTC)
	job := orchestratorJob(GenerationJobStatusQueued)
	job.NextPollAt = timePointerValue(now)
	repository := &orchestratorRepositoryMock{job: job}
	proxyID := int64(9)
	account := orchestratorAccount()
	account.ProxyID = &proxyID
	account.Proxy = &Proxy{Protocol: "http", Host: "proxy.example", Port: 8080}
	loader := &orchestratorAccountLoaderMock{account: account}
	upstream := &orchestratorUpstreamMock{}

	result, err := NewLeonardoGenerationPollOrchestrator(repository, loader, upstream, &config.Config{}, generationPollClockMock{now: now}).Poll(context.Background(), job.PublicID)
	require.NoError(t, err)
	require.Equal(t, 1, result.PollAttempts)
	require.Equal(t, []int64{job.AccountID}, loader.ids)
	require.Equal(t, 1, upstream.calls)
	require.Equal(t, 0, upstream.postCalls)
	require.Equal(t, 0, upstream.tlsCalls)
	require.Equal(t, http.MethodGet, upstream.method)
	require.Equal(t, "/api/rest/v1/generations/"+orchestratorGenerationID, upstream.path)
	require.Equal(t, account.ID, upstream.accountID)
	require.Equal(t, account.Concurrency, upstream.concurrency)
	require.Equal(t, "http://proxy.example:8080", upstream.proxyURL)
}

func TestLeonardoGenerationPollOrchestratorRejectsIneligibleBeforeAccountLookup(t *testing.T) {
	now := time.Date(2026, time.August, 3, 3, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		status    GenerationJobStatus
		provider  string
		id        *string
		next      *time.Time
		wantError error
	}{
		{name: "created", status: GenerationJobStatusCreated, provider: PlatformLeonardo, id: stringPointer(orchestratorGenerationID)},
		{name: "submitting", status: GenerationJobStatusSubmitting, provider: PlatformLeonardo, id: stringPointer(orchestratorGenerationID)},
		{name: "unknown", status: GenerationJobStatusUnknown, provider: PlatformLeonardo, id: stringPointer(orchestratorGenerationID)},
		{name: "succeeded", status: GenerationJobStatusSucceeded, provider: PlatformLeonardo, id: stringPointer(orchestratorGenerationID)},
		{name: "failed", status: GenerationJobStatusFailed, provider: PlatformLeonardo, id: stringPointer(orchestratorGenerationID)},
		{name: "cancelled", status: GenerationJobStatusCancelled, provider: PlatformLeonardo, id: stringPointer(orchestratorGenerationID)},
		{name: "missing id", status: GenerationJobStatusQueued, provider: PlatformLeonardo},
		{name: "invalid id", status: GenerationJobStatusRunning, provider: PlatformLeonardo, id: stringPointer("invalid")},
		{name: "not due", status: GenerationJobStatusQueued, provider: PlatformLeonardo, id: stringPointer(orchestratorGenerationID), next: timePointerValue(now.Add(time.Nanosecond))},
		{name: "wrong provider", status: GenerationJobStatusQueued, provider: PlatformOpenAI, id: stringPointer(orchestratorGenerationID), wantError: ErrLeonardoGenerationPollProviderMismatch},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			job := orchestratorJob(test.status)
			job.Provider = test.provider
			job.UpstreamGenerationID = test.id
			job.NextPollAt = test.next
			repository := &orchestratorRepositoryMock{job: job}
			loader := &orchestratorAccountLoaderMock{account: orchestratorAccount()}
			upstream := &orchestratorUpstreamMock{}
			result, err := NewLeonardoGenerationPollOrchestrator(repository, loader, upstream, &config.Config{}, generationPollClockMock{now: now}).Poll(context.Background(), job.PublicID)
			if test.wantError != nil {
				require.ErrorIs(t, err, test.wantError)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, job.PublicID, result.PublicID)
			require.Equal(t, 0, loader.calls)
			require.Equal(t, 0, upstream.calls)
		})
	}
}

func TestLeonardoGenerationPollOrchestratorAccountFailures(t *testing.T) {
	tests := []struct {
		name      string
		accountID int64
		account   *Account
		loaderErr error
		wantError error
		lookups   int
	}{
		{name: "invalid binding", accountID: 0, account: orchestratorAccount(), wantError: ErrLeonardoGenerationPollAccountBinding},
		{name: "not found", accountID: 41, loaderErr: ErrAccountNotFound, wantError: ErrAccountNotFound, lookups: 1},
		{name: "nil account", accountID: 41, wantError: ErrAccountNotFound, lookups: 1},
		{name: "wrong id", accountID: 41, account: func() *Account { a := orchestratorAccount(); a.ID = 42; return a }(), wantError: ErrLeonardoGenerationPollAccountBinding, lookups: 1},
		{name: "wrong platform", accountID: 41, account: func() *Account { a := orchestratorAccount(); a.Platform = PlatformOpenAI; return a }(), lookups: 1},
		{name: "wrong type", accountID: 41, account: func() *Account { a := orchestratorAccount(); a.Type = AccountTypeOAuth; return a }(), lookups: 1},
		{name: "missing key", accountID: 41, account: func() *Account { a := orchestratorAccount(); a.Credentials["api_key"] = ""; return a }(), lookups: 1},
		{name: "invalid url", accountID: 41, account: func() *Account { a := orchestratorAccount(); a.Credentials["base_url"] = "://invalid"; return a }(), lookups: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			job := orchestratorJob(GenerationJobStatusQueued)
			job.AccountID = test.accountID
			repository := &orchestratorRepositoryMock{job: job}
			loader := &orchestratorAccountLoaderMock{account: test.account, err: test.loaderErr}
			upstream := &orchestratorUpstreamMock{}
			result, err := NewLeonardoGenerationPollOrchestrator(repository, loader, upstream, &config.Config{}, generationPollClockMock{now: time.Now()}).Poll(context.Background(), job.PublicID)
			require.Error(t, err)
			if test.wantError != nil {
				require.ErrorIs(t, err, test.wantError)
			}
			require.Equal(t, job.PublicID, result.PublicID)
			require.Equal(t, test.lookups, loader.calls)
			require.Equal(t, 0, upstream.calls)
			require.Equal(t, 0, repository.casCalls)
			require.Equal(t, 0, result.PollAttempts)
		})
	}
}

func TestLeonardoGenerationPollOrchestratorSecondReadPreventsDrift(t *testing.T) {
	job := orchestratorJob(GenerationJobStatusQueued)
	repository := &orchestratorRepositoryMock{job: job, readHook: func(read int, loaded *GenerationJob) {
		if read == 2 {
			loaded.Status = GenerationJobStatusSucceeded
		}
	}}
	loader := &orchestratorAccountLoaderMock{account: orchestratorAccount()}
	upstream := &orchestratorUpstreamMock{}
	result, err := NewLeonardoGenerationPollOrchestrator(repository, loader, upstream, &config.Config{}, generationPollClockMock{now: time.Now()}).Poll(context.Background(), job.PublicID)
	require.NoError(t, err)
	require.Equal(t, GenerationJobStatusSucceeded, result.Status)
	require.Equal(t, 1, loader.calls)
	require.Equal(t, 0, upstream.calls)
	require.Equal(t, 0, repository.casCalls)
}

func TestLeonardoGenerationPollOrchestratorConcurrentCAS(t *testing.T) {
	job := orchestratorJob(GenerationJobStatusQueued)
	repository := &orchestratorRepositoryMock{job: job}
	loader := &orchestratorAccountLoaderMock{account: orchestratorAccount()}
	upstream := &orchestratorUpstreamMock{barrier: make(chan struct{}, 2), release: make(chan struct{}, 2)}
	orchestrator := NewLeonardoGenerationPollOrchestrator(repository, loader, upstream, &config.Config{}, generationPollClockMock{now: time.Now()})
	errorsChannel := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, err := orchestrator.Poll(context.Background(), job.PublicID)
			errorsChannel <- err
		}()
	}
	<-upstream.barrier
	<-upstream.barrier
	upstream.release <- struct{}{}
	upstream.release <- struct{}{}
	err1 := <-errorsChannel
	err2 := <-errorsChannel
	require.True(t, (err1 == nil && errors.Is(err2, ErrGenerationJobConflict)) || (err2 == nil && errors.Is(err1, ErrGenerationJobConflict)))
	require.Equal(t, 2, upstream.calls)
	require.Equal(t, 0, upstream.postCalls)
	require.Equal(t, 0, upstream.tlsCalls)
	require.Equal(t, 1, repository.job.PollAttempts)
}

func TestLeonardoGenerationPollOrchestratorRequiresConfiguration(t *testing.T) {
	loader := &orchestratorAccountLoaderMock{}
	upstream := &orchestratorUpstreamMock{}
	var orchestrator *LeonardoGenerationPollOrchestrator
	_, err := orchestrator.Poll(context.Background(), "job-1")
	require.ErrorContains(t, err, "not configured")
	require.Equal(t, 0, loader.calls)
	require.Equal(t, 0, upstream.calls)
}

func orchestratorJob(status GenerationJobStatus) *GenerationJob {
	return &GenerationJob{
		PublicID:             "job-1",
		Provider:             PlatformLeonardo,
		Status:               status,
		AccountID:            41,
		UpstreamGenerationID: stringPointer(orchestratorGenerationID),
		ResultPayload:        map[string]any{},
		BillingStatus:        GenerationJobBillingStatusSubmitted,
	}
}

func orchestratorAccount() *Account {
	return &Account{ID: 41, Platform: PlatformLeonardo, Type: AccountTypeAPIKey, Concurrency: 7, Credentials: map[string]any{"api_key": "test-key", "base_url": "https://leonardo.example/api/rest"}}
}
