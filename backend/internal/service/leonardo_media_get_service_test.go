package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type leonardoMediaGetRepositoryStub struct {
	job       *GenerationJob
	err       error
	errAtRead int
	returnNil bool
	readHook  func(int, *GenerationJob)
	reads     int
	casCalls  int
}

func (s *leonardoMediaGetRepositoryStub) GetByPublicID(context.Context, string) (*GenerationJob, error) {
	s.reads++
	if s.err != nil && (s.errAtRead == 0 || s.errAtRead == s.reads) {
		return nil, s.err
	}
	if s.returnNil {
		return nil, nil
	}
	if s.job == nil {
		return nil, ErrGenerationJobNotFound
	}
	job := *s.job
	if s.readHook != nil {
		s.readHook(s.reads, &job)
	}
	return &job, nil
}

func (s *leonardoMediaGetRepositoryStub) GetByUpstreamGenerationID(ctx context.Context, _ string) (*GenerationJob, error) {
	return s.GetByPublicID(ctx, "")
}

func (s *leonardoMediaGetRepositoryStub) CompareAndSwapPoll(context.Context, string, GenerationJobStatus, int, *GenerationJob) error {
	s.casCalls++
	return nil
}

func (s *leonardoMediaGetRepositoryStub) CompareAndSwapStatus(_ context.Context, _ string, _ GenerationJobStatus, job *GenerationJob) error {
	s.casCalls++
	stored := *job
	s.job = &stored
	return nil
}

type leonardoMediaGetAccountLoaderStub struct {
	account *Account
	err     error
	calls   int
}

func (s *leonardoMediaGetAccountLoaderStub) GetByID(context.Context, int64) (*Account, error) {
	s.calls++
	return s.account, s.err
}

type leonardoMediaGetClock struct{}

func (leonardoMediaGetClock) Now() time.Time {
	return time.Now()
}

const leonardoMediaGetPublicID = "gen_rq_0123456789abcdef0123456789abcdef"

func leonardoMediaOwnedJob() *GenerationJob {
	groupID := int64(17)
	return &GenerationJob{PublicID: leonardoMediaGetPublicID, Provider: PlatformLeonardo, Modality: "image", Model: "flux-schnell", UserID: 11, APIKeyID: 13, GroupID: &groupID, Status: GenerationJobStatusSucceeded}
}

func leonardoMediaGetInput() LeonardoMediaGetInput {
	return LeonardoMediaGetInput{PublicID: leonardoMediaGetPublicID, UserID: 11, APIKeyID: 13, GroupID: 17}
}

func leonardoMediaGetService(repository *leonardoMediaGetRepositoryStub, accounts *leonardoMediaGetAccountLoaderStub, upstream *orchestratorUpstreamMock) *LeonardoMediaGetService {
	poller := NewLeonardoGenerationPollOrchestrator(repository, accounts, upstream, &config.Config{}, leonardoMediaGetClock{}, &orchestratorFundsMock{})
	return NewLeonardoMediaGetService(repository, poller)
}

func TestLeonardoMediaGetServiceRejectsOwnedVideoBeforePolling(t *testing.T) {
	job := leonardoMediaOwnedJob()
	job.Modality = "video"
	repository := &leonardoMediaGetRepositoryStub{job: job}
	accounts := &leonardoMediaGetAccountLoaderStub{}
	upstream := &orchestratorUpstreamMock{}
	service := leonardoMediaGetService(repository, accounts, upstream)

	_, err := service.Get(context.Background(), leonardoMediaGetInput())

	require.ErrorIs(t, err, ErrLeonardoMediaVideoUnverified)
	require.Equal(t, 1, repository.reads)
	require.Zero(t, repository.casCalls)
	require.Zero(t, accounts.calls)
	require.Zero(t, upstream.calls)
}

func TestLeonardoMediaGetServiceRejectsOwned3DBeforePolling(t *testing.T) {
	job := leonardoMediaOwnedJob()
	job.Modality = "3d"
	repository := &leonardoMediaGetRepositoryStub{job: job}
	accounts := &leonardoMediaGetAccountLoaderStub{}
	upstream := &orchestratorUpstreamMock{}

	_, err := leonardoMediaGetService(repository, accounts, upstream).Get(context.Background(), leonardoMediaGetInput())

	require.ErrorIs(t, err, ErrLeonardo3DSchemaUnverified)
	require.Equal(t, 1, repository.reads)
	require.Zero(t, repository.casCalls)
	require.Zero(t, accounts.calls)
	require.Zero(t, upstream.calls)
}

func TestLeonardoMediaGetServiceReturnsOwnedTerminalJob(t *testing.T) {
	groupID := int64(17)
	completedAt := time.Date(2026, time.August, 3, 8, 9, 10, 0, time.UTC)
	job := &GenerationJob{
		PublicID: leonardoMediaGetPublicID, Provider: PlatformLeonardo, Modality: "image", Model: "flux-schnell",
		UserID: 11, APIKeyID: 13, GroupID: &groupID, Status: GenerationJobStatusSucceeded,
		CreatedAt: completedAt.Add(-time.Minute), UpdatedAt: completedAt, CompletedAt: &completedAt,
		ResultPayload: map[string]any{"images": []any{
			map[string]any{"id": "image-1", "url": "https://cdn.example/image.png", "nsfw": false},
			map[string]any{"id": "image-2", "url": "http://cdn.example/image.png"},
		}},
	}
	repository := &leonardoMediaGetRepositoryStub{job: job}
	accounts := &leonardoMediaGetAccountLoaderStub{}
	upstream := &orchestratorUpstreamMock{}
	result, err := leonardoMediaGetService(repository, accounts, upstream).Get(context.Background(), leonardoMediaGetInput())

	require.NoError(t, err)
	require.Equal(t, job.PublicID, result.ID)
	require.Equal(t, "media.generation", result.Object)
	require.Equal(t, GenerationJobStatusSucceeded, GenerationJobStatus(result.Status))
	require.Equal(t, completedAt.Unix(), result.CompletedAt)
	require.Equal(t, []LeonardoMediaGetImage{{ID: "image-1", URL: "https://cdn.example/image.png", NSFW: false}}, result.Data)
	require.Equal(t, 2, repository.reads)
	require.Zero(t, repository.casCalls)
	require.Zero(t, accounts.calls)
	require.Zero(t, upstream.calls)
	require.Zero(t, upstream.postCalls)
}

func TestLeonardoMediaGetServiceReconcilesTerminalBillingWithoutNetwork(t *testing.T) {
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
			job := leonardoMediaOwnedJob()
			job.Status = test.status
			job.BillingStatus = GenerationJobBillingStatusSubmitted
			cost := decimal.RequireFromString("0.005")
			job.CustomerCost = &cost
			job.BillingReference = stringPointer("leo_hold_existing")
			repository := &leonardoMediaGetRepositoryStub{job: job}
			accounts := &leonardoMediaGetAccountLoaderStub{}
			upstream := &orchestratorUpstreamMock{}
			funds := &orchestratorFundsMock{}
			poller := NewLeonardoGenerationPollOrchestrator(repository, accounts, upstream, &config.Config{}, leonardoMediaGetClock{}, funds)
			result, err := NewLeonardoMediaGetService(repository, poller).Get(context.Background(), leonardoMediaGetInput())
			require.NoError(t, err)
			require.Equal(t, string(test.status), result.Status)
			require.Equal(t, test.settles, funds.settleCalls)
			require.Equal(t, test.releases, funds.releaseCalls)
			require.Equal(t, 1, repository.casCalls)
			require.Equal(t, test.billing, repository.job.BillingStatus)
			require.Zero(t, accounts.calls)
			require.Zero(t, upstream.calls)
			body, marshalErr := json.Marshal(result)
			require.NoError(t, marshalErr)
			for _, secret := range []string{"billing_status", "billing_reference", "leo_hold_existing"} {
				require.NotContains(t, strings.ToLower(string(body)), secret)
			}
		})
	}
}

func TestLeonardoMediaGetServiceRejectsInvalidInputWithoutCalls(t *testing.T) {
	tests := []struct {
		name   string
		input  LeonardoMediaGetInput
		cancel bool
	}{
		{name: "invalid public id", input: func() LeonardoMediaGetInput { v := leonardoMediaGetInput(); v.PublicID = "bad"; return v }()},
		{name: "invalid user", input: func() LeonardoMediaGetInput { v := leonardoMediaGetInput(); v.UserID = 0; return v }()},
		{name: "invalid api key", input: func() LeonardoMediaGetInput { v := leonardoMediaGetInput(); v.APIKeyID = 0; return v }()},
		{name: "invalid group", input: func() LeonardoMediaGetInput { v := leonardoMediaGetInput(); v.GroupID = 0; return v }()},
		{name: "cancelled context", input: leonardoMediaGetInput(), cancel: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &leonardoMediaGetRepositoryStub{job: leonardoMediaOwnedJob()}
			accounts := &leonardoMediaGetAccountLoaderStub{}
			upstream := &orchestratorUpstreamMock{}
			ctx := context.Background()
			if test.cancel {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			result, err := leonardoMediaGetService(repository, accounts, upstream).Get(ctx, test.input)
			require.Nil(t, result)
			if test.cancel {
				require.ErrorIs(t, err, context.Canceled)
			} else {
				require.ErrorIs(t, err, ErrLeonardoMediaGetInputInvalid)
			}
			require.Zero(t, repository.reads)
			require.Zero(t, repository.casCalls)
			require.Zero(t, accounts.calls)
			require.Zero(t, upstream.calls)
		})
	}
}

func TestLeonardoMediaGetServiceRequiresConfigurationWithoutCalls(t *testing.T) {
	repository := &leonardoMediaGetRepositoryStub{job: leonardoMediaOwnedJob()}
	tests := []struct {
		name    string
		service *LeonardoMediaGetService
	}{
		{name: "nil receiver"},
		{name: "nil repository", service: NewLeonardoMediaGetService(nil, &LeonardoGenerationPollOrchestrator{})},
		{name: "nil poller", service: NewLeonardoMediaGetService(repository, nil)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.service.Get(context.Background(), leonardoMediaGetInput())
			require.Nil(t, result)
			require.ErrorIs(t, err, ErrLeonardoMediaGetNotConfigured)
			require.Zero(t, repository.reads)
		})
	}
}

func TestLeonardoMediaGetServiceHidesMissingAndForeignJobsBeforePoll(t *testing.T) {
	groupID := int64(17)
	repositoryError := errors.New("read failed")
	tests := []struct {
		name      string
		job       *GenerationJob
		repoErr   error
		returnNil bool
		wantErr   error
	}{
		{name: "not found", wantErr: ErrGenerationJobNotFound},
		{name: "repository error", repoErr: repositoryError, wantErr: repositoryError},
		{name: "nil job", returnNil: true, wantErr: ErrGenerationJobNotFound},
		{name: "foreign user", job: func() *GenerationJob { v := leonardoMediaOwnedJob(); v.UserID = 99; return v }(), wantErr: ErrGenerationJobNotFound},
		{name: "foreign api key", job: func() *GenerationJob { v := leonardoMediaOwnedJob(); v.APIKeyID = 99; return v }(), wantErr: ErrGenerationJobNotFound},
		{name: "missing group", job: func() *GenerationJob { v := leonardoMediaOwnedJob(); v.GroupID = nil; return v }(), wantErr: ErrGenerationJobNotFound},
		{name: "foreign group", job: func() *GenerationJob { v := leonardoMediaOwnedJob(); v.GroupID = &groupID; *v.GroupID = 99; return v }(), wantErr: ErrGenerationJobNotFound},
		{name: "foreign provider", job: func() *GenerationJob { v := leonardoMediaOwnedJob(); v.Provider = PlatformOpenAI; return v }(), wantErr: ErrGenerationJobNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &leonardoMediaGetRepositoryStub{job: test.job, err: test.repoErr, returnNil: test.returnNil}
			accounts := &leonardoMediaGetAccountLoaderStub{}
			upstream := &orchestratorUpstreamMock{}
			result, err := leonardoMediaGetService(repository, accounts, upstream).Get(context.Background(), leonardoMediaGetInput())
			require.Nil(t, result)
			require.ErrorIs(t, err, test.wantErr)
			require.Equal(t, 1, repository.reads)
			require.Zero(t, repository.casCalls)
			require.Zero(t, accounts.calls)
			require.Zero(t, upstream.calls)
			require.Zero(t, upstream.postCalls)
		})
	}
}

func TestLeonardoMediaGetServiceNonDueStatesNeverCallUpstream(t *testing.T) {
	future := time.Now().Add(time.Hour)
	tests := []struct {
		name   string
		status GenerationJobStatus
		id     *string
		next   *time.Time
	}{
		{name: "created", status: GenerationJobStatusCreated},
		{name: "submitting", status: GenerationJobStatusSubmitting},
		{name: "succeeded", status: GenerationJobStatusSucceeded},
		{name: "failed", status: GenerationJobStatusFailed},
		{name: "cancelled", status: GenerationJobStatusCancelled},
		{name: "unknown", status: GenerationJobStatusUnknown},
		{name: "queued missing upstream id", status: GenerationJobStatusQueued},
		{name: "running empty upstream id", status: GenerationJobStatusRunning, id: stringPointer("")},
		{name: "queued invalid upstream id", status: GenerationJobStatusQueued, id: stringPointer("invalid")},
		{name: "running not due", status: GenerationJobStatusRunning, id: stringPointer(orchestratorGenerationID), next: &future},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			job := leonardoMediaOwnedJob()
			job.Status, job.UpstreamGenerationID, job.NextPollAt = test.status, test.id, test.next
			repository := &leonardoMediaGetRepositoryStub{job: job}
			accounts := &leonardoMediaGetAccountLoaderStub{}
			upstream := &orchestratorUpstreamMock{}
			result, err := leonardoMediaGetService(repository, accounts, upstream).Get(context.Background(), leonardoMediaGetInput())
			require.NoError(t, err)
			require.Equal(t, string(test.status), result.Status)
			require.Equal(t, 2, repository.reads)
			require.Zero(t, repository.casCalls)
			require.Zero(t, accounts.calls)
			require.Zero(t, upstream.calls)
			require.Zero(t, upstream.postCalls)
		})
	}
}

func TestLeonardoMediaGetServiceDuePollUsesOneGetAndNeverPosts(t *testing.T) {
	job := leonardoMediaOwnedJob()
	job.Status = GenerationJobStatusQueued
	job.AccountID = 41
	job.UpstreamGenerationID = stringPointer(orchestratorGenerationID)
	repository := &leonardoMediaGetRepositoryStub{job: job}
	account := orchestratorAccount()
	accounts := &leonardoMediaGetAccountLoaderStub{account: account}
	upstream := &orchestratorUpstreamMock{}
	result, err := leonardoMediaGetService(repository, accounts, upstream).Get(context.Background(), leonardoMediaGetInput())
	require.NoError(t, err)
	require.Equal(t, string(GenerationJobStatusQueued), result.Status)
	require.Equal(t, 3, repository.reads)
	require.Equal(t, 1, repository.casCalls)
	require.Equal(t, 1, accounts.calls)
	require.Equal(t, 1, upstream.calls)
	require.Zero(t, upstream.postCalls)
}

func TestLeonardoMediaGetServiceDuePollPreflightFailuresNeverCallUpstream(t *testing.T) {
	tests := []struct {
		name       string
		accountID  int64
		account    *Account
		accountErr error
		wantErr    error
	}{
		{name: "invalid account binding", wantErr: ErrLeonardoGenerationPollAccountBinding},
		{name: "account lookup error", accountID: 41, accountErr: ErrAccountNotFound, wantErr: ErrAccountNotFound},
		{name: "nil account", accountID: 41, wantErr: ErrAccountNotFound},
		{name: "foreign account", accountID: 41, account: func() *Account { v := orchestratorAccount(); v.ID = 42; return v }(), wantErr: ErrLeonardoGenerationPollAccountBinding},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			job := leonardoMediaOwnedJob()
			job.Status = GenerationJobStatusQueued
			job.AccountID = test.accountID
			job.UpstreamGenerationID = stringPointer(orchestratorGenerationID)
			repository := &leonardoMediaGetRepositoryStub{job: job}
			accounts := &leonardoMediaGetAccountLoaderStub{account: test.account, err: test.accountErr}
			upstream := &orchestratorUpstreamMock{}
			result, err := leonardoMediaGetService(repository, accounts, upstream).Get(context.Background(), leonardoMediaGetInput())
			require.Nil(t, result)
			require.ErrorIs(t, err, test.wantErr)
			require.Zero(t, repository.casCalls)
			require.Zero(t, upstream.calls)
			require.Zero(t, upstream.postCalls)
		})
	}
}

func TestLeonardoMediaGetServiceRechecksOwnershipAfterPoll(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*GenerationJob)
	}{
		{name: "nil group", mutate: func(job *GenerationJob) { job.GroupID = nil }},
		{name: "changed user", mutate: func(job *GenerationJob) { job.UserID++ }},
		{name: "changed api key", mutate: func(job *GenerationJob) { job.APIKeyID++ }},
		{name: "changed group", mutate: func(job *GenerationJob) { id := int64(99); job.GroupID = &id }},
		{name: "changed provider", mutate: func(job *GenerationJob) { job.Provider = PlatformOpenAI }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &leonardoMediaGetRepositoryStub{job: leonardoMediaOwnedJob(), readHook: func(read int, job *GenerationJob) {
				if read == 2 {
					test.mutate(job)
				}
			}}
			accounts := &leonardoMediaGetAccountLoaderStub{}
			upstream := &orchestratorUpstreamMock{}
			result, err := leonardoMediaGetService(repository, accounts, upstream).Get(context.Background(), leonardoMediaGetInput())
			require.Nil(t, result)
			require.ErrorIs(t, err, ErrGenerationJobNotFound)
			require.Equal(t, 2, repository.reads)
			require.Zero(t, repository.casCalls)
			require.Zero(t, accounts.calls)
			require.Zero(t, upstream.calls)
			require.Zero(t, upstream.postCalls)
		})
	}
}

func TestLeonardoMediaGetServicePropagatesPollNilError(t *testing.T) {
	pollErr := errors.New("poll failed before job result")
	repository := &leonardoMediaGetRepositoryStub{job: leonardoMediaOwnedJob(), err: pollErr, errAtRead: 2}
	accounts := &leonardoMediaGetAccountLoaderStub{}
	upstream := &orchestratorUpstreamMock{}
	result, err := leonardoMediaGetService(repository, accounts, upstream).Get(context.Background(), leonardoMediaGetInput())
	require.Nil(t, result)
	require.ErrorIs(t, err, pollErr)
	require.NotErrorIs(t, err, ErrGenerationJobNotFound)
	require.Equal(t, 2, repository.reads)
	require.Zero(t, repository.casCalls)
	require.Zero(t, accounts.calls)
	require.Zero(t, upstream.calls)
	require.Zero(t, upstream.postCalls)
}

func TestLeonardoMediaGetServiceConcurrentCASOneSuccessOneConflict(t *testing.T) {
	job := leonardoMediaOwnedJob()
	job.Status = GenerationJobStatusQueued
	job.AccountID = 41
	job.UpstreamGenerationID = stringPointer(orchestratorGenerationID)
	repository := &orchestratorRepositoryMock{job: job}
	accounts := &orchestratorAccountLoaderMock{account: orchestratorAccount()}
	upstream := &orchestratorUpstreamMock{barrier: make(chan struct{}, 2), release: make(chan struct{}, 2)}
	poller := NewLeonardoGenerationPollOrchestrator(repository, accounts, upstream, &config.Config{}, leonardoMediaGetClock{}, &orchestratorFundsMock{})
	service := NewLeonardoMediaGetService(repository, poller)

	type getOutcome struct {
		result *LeonardoMediaGetResult
		err    error
	}
	outcomes := make(chan getOutcome, 2)
	for i := 0; i < 2; i++ {
		go func() {
			result, err := service.Get(context.Background(), leonardoMediaGetInput())
			outcomes <- getOutcome{result: result, err: err}
		}()
	}
	<-upstream.barrier
	<-upstream.barrier
	upstream.release <- struct{}{}
	upstream.release <- struct{}{}

	first, second := <-outcomes, <-outcomes
	successes, conflicts := 0, 0
	for _, outcome := range []getOutcome{first, second} {
		switch {
		case outcome.err == nil:
			successes++
			require.NotNil(t, outcome.result)
			require.Equal(t, string(GenerationJobStatusQueued), outcome.result.Status)
		case errors.Is(outcome.err, ErrGenerationJobConflict):
			conflicts++
			require.Nil(t, outcome.result)
		default:
			require.NoError(t, outcome.err)
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, conflicts)
	require.Equal(t, 2, repository.casCalls)
	require.Equal(t, 1, repository.job.PollAttempts)
	require.Equal(t, 2, upstream.calls)
	require.Zero(t, upstream.postCalls)
	require.Zero(t, upstream.tlsCalls)
}

func TestLeonardoMediaGetServiceFiltersUnsafeImagesAndSensitiveFields(t *testing.T) {
	job := leonardoMediaOwnedJob()
	job.ResultPayload = map[string]any{"images": []any{
		map[string]any{"id": "first", "url": "https://cdn.example/first.png", "nsfw": true},
		map[string]any{"id": "http", "url": "http://cdn.example/image.png"},
		map[string]any{"id": "ftp", "url": "ftp://cdn.example/image.png"},
		map[string]any{"id": "data", "url": "data:image/png;base64,AA"},
		map[string]any{"id": "relative", "url": "//cdn.example/image.png"},
		map[string]any{"id": "hostless", "url": "https:///image.png"},
		map[string]any{"id": " ", "url": "https://cdn.example/image.png"},
		map[string]any{"id": 1, "url": "https://cdn.example/image.png"},
		map[string]any{"id": "bad-url-type", "url": 1},
		"not-an-object",
		map[string]any{"id": "second", "url": " https://cdn.example/second.png ", "nsfw": false},
		map[string]any{"id": "unknown", "url": "https://cdn.example/unknown.png", "nsfw": "invalid"},
	}}
	result := newLeonardoMediaGetResult(job)
	require.Equal(t, []LeonardoMediaGetImage{{ID: "second", URL: "https://cdn.example/second.png", NSFW: false}}, result.Data)

	body, err := json.Marshal(result)
	require.NoError(t, err)
	require.NotContains(t, string(body), "https://cdn.example/first.png")
	require.NotContains(t, string(body), "https://cdn.example/unknown.png")
	for _, secret := range []string{"account_id", "upstream_generation_id", "request_payload", "result_payload", "billing_reference", "actual_upstream_cost", "api_key", "authorization", "credentials", "proxy", "cookie", "signature"} {
		require.NotContains(t, strings.ToLower(string(body)), secret)
	}
}

func TestLeonardoMediaGetServiceFailedResultAndEmptyImages(t *testing.T) {
	job := leonardoMediaOwnedJob()
	job.Status = GenerationJobStatusFailed
	job.ErrorCode = stringPointer("upstream_failed")
	job.ErrorMessage = stringPointer("safe failure")
	job.ResultPayload = map[string]any{"images": "invalid"}
	result := newLeonardoMediaGetResult(job)
	require.Equal(t, &LeonardoMediaGetError{Code: "upstream_failed", Message: "safe failure"}, result.Error)
	require.NotNil(t, result.Data)
	require.Empty(t, result.Data)
}

func TestLeonardoMediaGetServiceContentDownloadsValidatedImage(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0}
	job := leonardoMediaOwnedJob()
	job.Status = GenerationJobStatusSucceeded
	job.BillingStatus = GenerationJobBillingStatusSettled
	job.ResultPayload = map[string]any{"images": []map[string]any{{"id": "image", "url": "https://cdn.example/image.png", "nsfw": false}}}
	repository := &leonardoMediaGetRepositoryStub{job: job}
	getService := leonardoMediaGetService(repository, &leonardoMediaGetAccountLoaderStub{}, &orchestratorUpstreamMock{})
	getService.content = &http.Client{Transport: leonardoMediaRoundTripper(func(r *http.Request) (*http.Response, error) {
		require.Empty(t, r.Header.Get("Authorization"))
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"image/png"}}, Body: io.NopCloser(strings.NewReader(string(png))), ContentLength: int64(len(png))}, nil
	})}

	content, err := getService.Content(context.Background(), LeonardoMediaContentInput{LeonardoMediaGetInput: ownedLeonardoMediaGetInput(job), Index: 0})
	require.NoError(t, err)
	path := content.path
	result, err := io.ReadAll(content.File)
	require.NoError(t, err)
	require.Equal(t, png, result)
	require.Equal(t, "image/png", content.ContentType)
	require.NoError(t, content.Close())
	_, err = os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestLeonardoMediaGetServiceContentRejectsMIMEConfusion(t *testing.T) {
	job := leonardoMediaOwnedJob()
	job.Status = GenerationJobStatusSucceeded
	job.BillingStatus = GenerationJobBillingStatusSettled
	job.ResultPayload = map[string]any{"images": []map[string]any{{"id": "image", "url": "https://cdn.example/image.png", "nsfw": false}}}
	repository := &leonardoMediaGetRepositoryStub{job: job}
	getService := leonardoMediaGetService(repository, &leonardoMediaGetAccountLoaderStub{}, &orchestratorUpstreamMock{})
	getService.content = &http.Client{Transport: leonardoMediaRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"image/png"}}, Body: io.NopCloser(strings.NewReader("<html>not an image</html>")), ContentLength: 25}, nil
	})}

	content, err := getService.Content(context.Background(), LeonardoMediaContentInput{LeonardoMediaGetInput: ownedLeonardoMediaGetInput(job), Index: 0})
	require.ErrorIs(t, err, ErrLeonardoMediaContentFailed)
	require.Nil(t, content)
}

func ownedLeonardoMediaGetInput(job *GenerationJob) LeonardoMediaGetInput {
	return LeonardoMediaGetInput{PublicID: job.PublicID, UserID: job.UserID, APIKeyID: job.APIKeyID, GroupID: *job.GroupID}
}

type leonardoMediaRoundTripper func(*http.Request) (*http.Response, error)

func (f leonardoMediaRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
