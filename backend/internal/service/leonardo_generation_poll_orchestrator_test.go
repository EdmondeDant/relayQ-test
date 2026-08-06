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
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

const orchestratorGenerationID = "123e4567-e89b-42d3-a456-426614174000"

type orchestratorRepositoryMock struct {
	mu             sync.Mutex
	job            *GenerationJob
	reads          int
	readHook       func(int, *GenerationJob)
	casCalls       int
	statusCASCalls int
	statusCASErr   error
}

func (r *orchestratorRepositoryMock) CompareAndSwapStatus(_ context.Context, publicID string, status GenerationJobStatus, job *GenerationJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.statusCASCalls++
	if r.statusCASErr != nil {
		return r.statusCASErr
	}
	if r.job == nil || r.job.PublicID != publicID || r.job.Status != status {
		return ErrGenerationJobConflict
	}
	stored := *job
	r.job = &stored
	return nil
}

type orchestratorFundsMock struct {
	mu            sync.Mutex
	settleCalls   int
	releaseCalls  int
	releaseReason string
	err           error
}

type orchestratorUsageLogMock struct {
	logs     []*UsageLog
	inserted bool
	err      error
}

func (m *orchestratorUsageLogMock) Create(_ context.Context, usage *UsageLog) (bool, error) {
	m.logs = append(m.logs, usage)
	return m.inserted, m.err
}

func (f *orchestratorFundsMock) Settle(context.Context, LeonardoImageFundsSettleRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.settleCalls++
	return f.err
}

func (f *orchestratorFundsMock) Release(_ context.Context, request LeonardoImageFundsReleaseRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.releaseCalls++
	f.releaseReason = request.Reason
	return f.err
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

func (r *orchestratorRepositoryMock) GetByUpstreamGenerationID(context.Context, string) (*GenerationJob, error) {
	return r.GetByPublicID(context.Background(), "")
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
	status      string
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
	status := u.status
	if status == "" {
		status = "PENDING"
	}
	body := fmt.Sprintf(`{"generations_by_pk":{"id":%q,"status":%q,"generated_images":[{"id":"image-1","url":"https://cdn.example/image.png","nsfw":false}]}}`, orchestratorGenerationID, status)
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

	result, err := NewLeonardoGenerationPollOrchestrator(repository, loader, upstream, &config.Config{}, generationPollClockMock{now: now}, &orchestratorFundsMock{}).Poll(context.Background(), job.PublicID)
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
		{name: "unknown without id", status: GenerationJobStatusUnknown, provider: PlatformLeonardo},
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
			result, err := NewLeonardoGenerationPollOrchestrator(repository, loader, upstream, &config.Config{}, generationPollClockMock{now: now}, &orchestratorFundsMock{}).Poll(context.Background(), job.PublicID)
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
			result, err := NewLeonardoGenerationPollOrchestrator(repository, loader, upstream, &config.Config{}, generationPollClockMock{now: time.Now()}, &orchestratorFundsMock{}).Poll(context.Background(), job.PublicID)
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
	result, err := NewLeonardoGenerationPollOrchestrator(repository, loader, upstream, &config.Config{}, generationPollClockMock{now: time.Now()}, &orchestratorFundsMock{}).Poll(context.Background(), job.PublicID)
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
	orchestrator := NewLeonardoGenerationPollOrchestrator(repository, loader, upstream, &config.Config{}, generationPollClockMock{now: time.Now()}, &orchestratorFundsMock{})
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

func TestLeonardoGenerationPollOrchestratorReconcilesStoredTerminalJobsWithoutNetwork(t *testing.T) {
	for _, test := range []struct {
		name     string
		status   GenerationJobStatus
		billing  GenerationJobBillingStatus
		settles  int
		releases int
	}{
		{name: "success", status: GenerationJobStatusSucceeded, billing: GenerationJobBillingStatusSettled, settles: 1},
		{name: "failure", status: GenerationJobStatusFailed, billing: GenerationJobBillingStatusRefunded, releases: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			job := orchestratorJob(test.status)
			job.BillingStatus = GenerationJobBillingStatusSubmitted
			cost := decimal.RequireFromString("0.005")
			job.CustomerCost = &cost
			job.BillingReference = stringPointer("leo_hold_existing")
			repository := &orchestratorRepositoryMock{job: job}
			loader := &orchestratorAccountLoaderMock{}
			upstream := &orchestratorUpstreamMock{}
			funds := &orchestratorFundsMock{}
			result, err := NewLeonardoGenerationPollOrchestrator(repository, loader, upstream, &config.Config{}, generationPollClockMock{now: time.Now()}, funds).Poll(context.Background(), job.PublicID)
			require.NoError(t, err)
			require.Equal(t, test.billing, result.BillingStatus)
			require.Equal(t, test.settles, funds.settleCalls)
			require.Equal(t, test.releases, funds.releaseCalls)
			if test.releases == 1 {
				require.Equal(t, "generation_failed", funds.releaseReason)
			}
			require.Equal(t, 1, repository.statusCASCalls)
			require.Zero(t, loader.calls)
			require.Zero(t, upstream.calls)
		})
	}
}

func TestLeonardoGenerationPollOrchestratorDoesNotBillUnverified3DTerminalJob(t *testing.T) {
	job := orchestratorJob(GenerationJobStatusSucceeded)
	job.Modality = "3d"
	job.BillingStatus = GenerationJobBillingStatusSubmitted
	job.BillingReference = stringPointer("leo_hold_existing")
	repository := &orchestratorRepositoryMock{job: job}
	funds := &orchestratorFundsMock{}

	result, err := NewLeonardoGenerationPollOrchestrator(repository, &orchestratorAccountLoaderMock{}, &orchestratorUpstreamMock{}, &config.Config{}, generationPollClockMock{now: time.Now()}, funds).Poll(context.Background(), job.PublicID)

	require.NoError(t, err)
	require.Equal(t, GenerationJobBillingStatusManualReview, result.BillingStatus)
	require.Zero(t, funds.settleCalls)
	require.Zero(t, funds.releaseCalls)
}

func TestLeonardoGenerationPollOrchestratorWritesIdempotentUsageLog(t *testing.T) {
	job := orchestratorJob(GenerationJobStatusSucceeded)
	job.BillingStatus = GenerationJobBillingStatusSubmitted
	job.OutputCount = 1
	cost := decimal.RequireFromString("0.005")
	job.CustomerCost = &cost
	job.BillingReference = stringPointer("leo_hold_existing")
	repository := &orchestratorRepositoryMock{job: job}
	usageLogs := &orchestratorUsageLogMock{inserted: true}
	orchestrator := NewLeonardoGenerationPollOrchestrator(repository, &orchestratorAccountLoaderMock{}, &orchestratorUpstreamMock{}, &config.Config{}, generationPollClockMock{now: time.Now()}, &orchestratorFundsMock{})
	orchestrator.SetUsageLogWriter(usageLogs)

	result, err := orchestrator.Poll(context.Background(), job.PublicID)

	require.NoError(t, err)
	require.Equal(t, GenerationJobBillingStatusSettled, result.BillingStatus)
	require.Len(t, usageLogs.logs, 1)
	usage := usageLogs.logs[0]
	require.Equal(t, job.PublicID, usage.RequestID)
	require.Equal(t, job.UserID, usage.UserID)
	require.Equal(t, job.APIKeyID, usage.APIKeyID)
	require.Equal(t, job.AccountID, usage.AccountID)
	require.Equal(t, job.Model, usage.Model)
	require.Equal(t, job.UpstreamModel, *usage.UpstreamModel)
	require.Equal(t, 0.005, usage.TotalCost)
	require.Equal(t, 0.005, usage.ActualCost)
	require.Equal(t, 1, usage.ImageCount)
	require.Equal(t, "1K", *usage.ImageSize)
	require.Equal(t, "image", *usage.BillingMode)
}

func TestLeonardoGenerationPollOrchestratorUsageLogFailureLeavesBillingRetryable(t *testing.T) {
	job := orchestratorJob(GenerationJobStatusSucceeded)
	job.BillingStatus = GenerationJobBillingStatusSubmitted
	cost := decimal.RequireFromString("0.005")
	job.CustomerCost = &cost
	job.BillingReference = stringPointer("leo_hold_existing")
	repository := &orchestratorRepositoryMock{job: job}
	usageErr := errors.New("usage log failed")
	usageLogs := &orchestratorUsageLogMock{err: usageErr}
	funds := &orchestratorFundsMock{}
	orchestrator := NewLeonardoGenerationPollOrchestrator(repository, &orchestratorAccountLoaderMock{}, &orchestratorUpstreamMock{}, &config.Config{}, generationPollClockMock{now: time.Now()}, funds)
	orchestrator.SetUsageLogWriter(usageLogs)

	result, err := orchestrator.Poll(context.Background(), job.PublicID)

	require.ErrorIs(t, err, usageErr)
	require.Equal(t, GenerationJobBillingStatusSubmitted, result.BillingStatus)
	require.Equal(t, 1, funds.settleCalls)
	require.Zero(t, repository.statusCASCalls)
}

func TestLeonardoGenerationPollOrchestratorTerminalFundsFailureDoesNotUpdateBilling(t *testing.T) {
	fundsErr := errors.New("funds failed")
	job := orchestratorJob(GenerationJobStatusSucceeded)
	job.BillingStatus = GenerationJobBillingStatusSubmitted
	cost := decimal.RequireFromString("0.005")
	job.CustomerCost = &cost
	job.BillingReference = stringPointer("leo_hold_existing")
	repository := &orchestratorRepositoryMock{job: job}
	funds := &orchestratorFundsMock{err: fundsErr}
	result, err := NewLeonardoGenerationPollOrchestrator(repository, &orchestratorAccountLoaderMock{}, &orchestratorUpstreamMock{}, &config.Config{}, generationPollClockMock{now: time.Now()}, funds).Poll(context.Background(), job.PublicID)
	require.ErrorIs(t, err, fundsErr)
	require.Equal(t, GenerationJobBillingStatusSubmitted, result.BillingStatus)
	require.Equal(t, 1, funds.settleCalls)
	require.Zero(t, repository.statusCASCalls)
}

func TestLeonardoGenerationPollOrchestratorPollsIntoTerminalAndReconciles(t *testing.T) {
	for _, test := range []struct {
		name          string
		upstream      string
		status        GenerationJobStatus
		billing       GenerationJobBillingStatus
		settles       int
		releases      int
		wantErrorCode string
	}{
		{name: "complete", upstream: "COMPLETE", status: GenerationJobStatusSucceeded, billing: GenerationJobBillingStatusSettled, settles: 1},
		{name: "failed", upstream: "FAILED", status: GenerationJobStatusFailed, billing: GenerationJobBillingStatusRefunded, releases: 1, wantErrorCode: "upstream_failed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			job := orchestratorJob(GenerationJobStatusQueued)
			job.BillingStatus = GenerationJobBillingStatusSubmitted
			cost := decimal.RequireFromString("0.005")
			job.CustomerCost = &cost
			job.BillingReference = stringPointer("leo_hold_existing")
			repository := &orchestratorRepositoryMock{job: job}
			loader := &orchestratorAccountLoaderMock{account: orchestratorAccount()}
			upstream := &orchestratorUpstreamMock{status: test.upstream}
			funds := &orchestratorFundsMock{}
			result, err := NewLeonardoGenerationPollOrchestrator(repository, loader, upstream, &config.Config{}, generationPollClockMock{now: time.Now()}, funds).Poll(context.Background(), job.PublicID)
			require.NoError(t, err)
			require.Equal(t, test.status, result.Status)
			require.Equal(t, test.billing, result.BillingStatus)
			require.Equal(t, 1, repository.casCalls)
			require.Equal(t, 1, repository.statusCASCalls)
			require.Equal(t, test.settles, funds.settleCalls)
			require.Equal(t, test.releases, funds.releaseCalls)
			require.Equal(t, 1, upstream.calls)
			require.Zero(t, upstream.postCalls)
			require.Equal(t, cost, *result.CustomerCost)
			require.Equal(t, "leo_hold_existing", *result.BillingReference)
			if test.status == GenerationJobStatusSucceeded {
				require.Equal(t, 1, result.OutputCount)
				require.NotEmpty(t, result.ResultPayload["images"])
			} else {
				require.Equal(t, test.wantErrorCode, stringValue(result.ErrorCode))
				require.Equal(t, "generation_failed", funds.releaseReason)
			}
		})
	}
}

func TestLeonardoGenerationPollOrchestratorBillingCASConflictConvergesOnSecondCall(t *testing.T) {
	for _, test := range []struct {
		name     string
		status   GenerationJobStatus
		billing  GenerationJobBillingStatus
		settles  int
		releases int
	}{
		{name: "success", status: GenerationJobStatusSucceeded, billing: GenerationJobBillingStatusSettled, settles: 2},
		{name: "failure", status: GenerationJobStatusFailed, billing: GenerationJobBillingStatusRefunded, releases: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			job := orchestratorJob(test.status)
			job.BillingStatus = GenerationJobBillingStatusSubmitted
			cost := decimal.RequireFromString("0.005")
			job.CustomerCost = &cost
			job.BillingReference = stringPointer("leo_hold_existing")
			repository := &orchestratorRepositoryMock{job: job, statusCASErr: ErrGenerationJobConflict}
			loader := &orchestratorAccountLoaderMock{}
			upstream := &orchestratorUpstreamMock{}
			funds := &orchestratorFundsMock{}
			orchestrator := NewLeonardoGenerationPollOrchestrator(repository, loader, upstream, &config.Config{}, generationPollClockMock{now: time.Now()}, funds)
			first, err := orchestrator.Poll(context.Background(), job.PublicID)
			require.ErrorIs(t, err, ErrGenerationJobConflict)
			require.Equal(t, GenerationJobBillingStatusSubmitted, first.BillingStatus)
			repository.statusCASErr = nil
			second, err := orchestrator.Poll(context.Background(), job.PublicID)
			require.NoError(t, err)
			require.NotNil(t, second)
			require.Equal(t, test.billing, second.BillingStatus)
			require.Equal(t, test.settles, funds.settleCalls)
			require.Equal(t, test.releases, funds.releaseCalls)
			require.Equal(t, 2, repository.statusCASCalls)
			require.Zero(t, loader.calls)
			require.Zero(t, upstream.calls)
		})
	}
}

func TestLeonardoGenerationPollOrchestratorRejectsInvalidTerminalReservationFields(t *testing.T) {
	for _, status := range []GenerationJobStatus{GenerationJobStatusSucceeded, GenerationJobStatusFailed} {
		for _, test := range []struct {
			name   string
			mutate func(*GenerationJob)
		}{
			{name: "zero user", mutate: func(job *GenerationJob) { job.UserID = 0 }},
			{name: "negative user", mutate: func(job *GenerationJob) { job.UserID = -1 }},
			{name: "empty public id", mutate: func(job *GenerationJob) { job.PublicID = "" }},
			{name: "blank public id", mutate: func(job *GenerationJob) { job.PublicID = " " }},
			{name: "long public id", mutate: func(job *GenerationJob) { job.PublicID = strings.Repeat("x", 65) }},
			{name: "nil cost", mutate: func(job *GenerationJob) { job.CustomerCost = nil }},
			{name: "zero cost", mutate: func(job *GenerationJob) { value := decimal.Zero; job.CustomerCost = &value }},
			{name: "negative cost", mutate: func(job *GenerationJob) { value := decimal.NewFromInt(-1); job.CustomerCost = &value }},
			{name: "cost precision", mutate: func(job *GenerationJob) { value := decimal.RequireFromString("0.000000001"); job.CustomerCost = &value }},
			{name: "cost overflow", mutate: func(job *GenerationJob) {
				value := decimal.RequireFromString("1000000000000")
				job.CustomerCost = &value
			}},
			{name: "nil reference", mutate: func(job *GenerationJob) { job.BillingReference = nil }},
			{name: "empty reference", mutate: func(job *GenerationJob) { job.BillingReference = stringPointer("") }},
			{name: "blank reference", mutate: func(job *GenerationJob) { job.BillingReference = stringPointer(" ") }},
			{name: "long reference", mutate: func(job *GenerationJob) { job.BillingReference = stringPointer(strings.Repeat("x", 129)) }},
		} {
			t.Run(string(status)+"/"+test.name, func(t *testing.T) {
				job := validTerminalOrchestratorJob(status)
				test.mutate(job)
				repository := &orchestratorRepositoryMock{job: job}
				loader := &orchestratorAccountLoaderMock{}
				upstream := &orchestratorUpstreamMock{}
				funds := &orchestratorFundsMock{}
				result, err := NewLeonardoGenerationPollOrchestrator(repository, loader, upstream, &config.Config{}, generationPollClockMock{now: time.Now()}, funds).Poll(context.Background(), job.PublicID)
				require.ErrorIs(t, err, ErrLeonardoImageCreateReservationInvalid)
				require.Equal(t, GenerationJobBillingStatusSubmitted, result.BillingStatus)
				require.Zero(t, funds.settleCalls)
				require.Zero(t, funds.releaseCalls)
				require.Zero(t, repository.statusCASCalls)
				require.Zero(t, loader.calls)
				require.Zero(t, upstream.calls)
			})
		}
	}
}

func TestLeonardoGenerationPollOrchestratorConvergedTerminalJobsHaveZeroActions(t *testing.T) {
	for _, test := range []struct {
		status  GenerationJobStatus
		billing GenerationJobBillingStatus
	}{
		{status: GenerationJobStatusSucceeded, billing: GenerationJobBillingStatusSettled},
		{status: GenerationJobStatusFailed, billing: GenerationJobBillingStatusRefunded},
	} {
		t.Run(string(test.status), func(t *testing.T) {
			job := validTerminalOrchestratorJob(test.status)
			job.BillingStatus = test.billing
			completedAt := time.Now().UTC()
			job.CompletedAt = &completedAt
			job.FailedAt = &completedAt
			job.ErrorCode = stringPointer("preserved")
			job.ErrorMessage = stringPointer("preserved message")
			job.PollAttempts = 3
			repository := &orchestratorRepositoryMock{job: job}
			loader := &orchestratorAccountLoaderMock{}
			upstream := &orchestratorUpstreamMock{}
			funds := &orchestratorFundsMock{err: errors.New("must not be called")}
			result, err := NewLeonardoGenerationPollOrchestrator(repository, loader, upstream, &config.Config{}, generationPollClockMock{now: time.Now()}, funds).Poll(context.Background(), job.PublicID)
			require.NoError(t, err)
			require.Equal(t, job, result)
			require.Equal(t, 1, repository.reads)
			require.Zero(t, funds.settleCalls)
			require.Zero(t, funds.releaseCalls)
			require.Zero(t, repository.statusCASCalls)
			require.Zero(t, repository.casCalls)
			require.Zero(t, loader.calls)
			require.Zero(t, upstream.calls)
		})
	}
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
		Modality:             "image",
		Status:               status,
		UserID:               7,
		AccountID:            41,
		UpstreamGenerationID: stringPointer(orchestratorGenerationID),
		ResultPayload:        map[string]any{},
		BillingStatus:        GenerationJobBillingStatusReserved,
	}
}

func orchestratorAccount() *Account {
	return &Account{ID: 41, Platform: PlatformLeonardo, Type: AccountTypeAPIKey, Concurrency: 7, Credentials: map[string]any{"api_key": "test-key", "base_url": "https://leonardo.example/api/rest"}}
}

func validTerminalOrchestratorJob(status GenerationJobStatus) *GenerationJob {
	job := orchestratorJob(status)
	job.BillingStatus = GenerationJobBillingStatusSubmitted
	cost := decimal.RequireFromString("0.005")
	job.CustomerCost = &cost
	job.BillingReference = stringPointer("leo_hold_existing")
	job.ResultPayload = map[string]any{"images": []any{map[string]any{"id": "image-1"}}, "cost": "999"}
	actualCost := decimal.RequireFromString("999")
	job.ActualUpstreamCostAmount = &actualCost
	return job
}
