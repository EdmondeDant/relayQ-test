package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
	"github.com/stretchr/testify/require"
)

const pollerTestGenerationID = "123e4567-e89b-12d3-a456-426614174000"

func leonardoNSFW(value bool) *bool {
	return &value
}

type generationPollRepositoryMock struct {
	mu  sync.Mutex
	job *GenerationJob
}

func (r *generationPollRepositoryMock) GetByPublicID(context.Context, string) (*GenerationJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.job == nil {
		return nil, ErrGenerationJobNotFound
	}
	job := *r.job
	return &job, nil
}

func (r *generationPollRepositoryMock) CompareAndSwapPoll(_ context.Context, publicID string, status GenerationJobStatus, attempts int, job *GenerationJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
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

type generationPollClientMock struct {
	mu         sync.Mutex
	response   *leonardo.Generation
	err        error
	calls      int
	beforeDone chan struct{}
	release    chan struct{}
}

func (c *generationPollClientMock) GetGeneration(context.Context, string) (*leonardo.Generation, error) {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()
	if c.beforeDone != nil {
		c.beforeDone <- struct{}{}
	}
	if c.release != nil {
		<-c.release
	}
	return c.response, c.err
}

func (c *generationPollClientMock) callCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

type generationPollClockMock struct {
	now time.Time
}

type generationOutputModeratorMock struct {
	blocked map[string]bool
	err     error
	inputs  []ContentModerationCheckInput
}

func (m *generationOutputModeratorMock) Check(_ context.Context, input ContentModerationCheckInput) (*ContentModerationDecision, error) {
	m.inputs = append(m.inputs, input)
	if m.err != nil {
		return nil, m.err
	}
	var body struct {
		Images []struct {
			URL string `json:"image_url"`
		} `json:"images"`
	}
	if err := json.Unmarshal(input.Body, &body); err != nil {
		return nil, err
	}
	blocked := len(body.Images) != 1 || m.blocked[body.Images[0].URL]
	return &ContentModerationDecision{Allowed: !blocked, Blocked: blocked}, nil
}

func (c generationPollClockMock) Now() time.Time {
	return c.now
}

func TestLeonardoGenerationPollerPendingAndRebuild(t *testing.T) {
	now := time.Date(2026, time.August, 3, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	repository := &generationPollRepositoryMock{job: pollerTestJob(GenerationJobStatusRunning)}
	client := &generationPollClientMock{response: &leonardo.Generation{Status: "PENDING"}}

	first, err := NewLeonardoGenerationPoller(repository, client, generationPollClockMock{now: now}).Poll(context.Background(), "job-1")
	require.NoError(t, err)
	require.Equal(t, GenerationJobStatusRunning, first.Status)
	require.Equal(t, 1, first.PollAttempts)
	require.Equal(t, now.UTC(), *first.LastPolledAt)
	require.Equal(t, now.UTC().Add(5*time.Second), *first.NextPollAt)

	secondNow := now.Add(time.Minute)
	second, err := NewLeonardoGenerationPoller(repository, client, generationPollClockMock{now: secondNow}).Poll(context.Background(), "job-1")
	require.NoError(t, err)
	require.Equal(t, 2, second.PollAttempts)
	require.Equal(t, secondNow.UTC().Add(5*time.Second), *second.NextPollAt)
	require.Equal(t, 2, client.callCount())
}

func TestLeonardoGenerationPollerComplete(t *testing.T) {
	now := time.Date(2026, time.August, 3, 2, 0, 0, 0, time.UTC)
	stored := pollerTestJob(GenerationJobStatusQueued)
	stored.ResultPayload = map[string]any{"cost": "0.001", "apiCreditCost": 1}
	repository := &generationPollRepositoryMock{job: stored}
	client := &generationPollClientMock{response: &leonardo.Generation{
		Status: "COMPLETE",
		GeneratedImages: []leonardo.GeneratedImage{
			{ID: "image-1", URL: "https://example.com/1.png", NSFW: leonardoNSFW(false)},
			{ID: "image-2", URL: "https://example.com/2.png", NSFW: leonardoNSFW(false)},
		},
	}}

	job, err := NewLeonardoGenerationPoller(repository, client, generationPollClockMock{now: now}).Poll(context.Background(), "job-1")
	require.NoError(t, err)
	require.Equal(t, GenerationJobStatusSucceeded, job.Status)
	require.Equal(t, 2, job.OutputCount)
	require.Equal(t, now, *job.CompletedAt)
	require.Nil(t, job.NextPollAt)
	require.Equal(t, map[string]any{"cost": "0.001", "apiCreditCost": 1, "images": []map[string]any{
		{"id": "image-1", "url": "https://example.com/1.png", "nsfw": false},
		{"id": "image-2", "url": "https://example.com/2.png", "nsfw": false},
	}}, job.ResultPayload)
}

func TestLeonardoGenerationPollerBlocksNSFWOutput(t *testing.T) {
	now := time.Date(2026, time.August, 3, 2, 0, 0, 0, time.UTC)
	stored := pollerTestJob(GenerationJobStatusQueued)
	stored.ResultPayload = map[string]any{"cost": "0.001", "apiCreditCost": 1}
	repository := &generationPollRepositoryMock{job: stored}
	client := &generationPollClientMock{response: &leonardo.Generation{
		Status: "COMPLETE",
		GeneratedImages: []leonardo.GeneratedImage{
			{ID: "safe", URL: "https://example.com/safe.png", NSFW: leonardoNSFW(false)},
			{ID: "blocked", URL: "https://example.com/blocked.png", NSFW: leonardoNSFW(true)},
		},
	}}

	job, err := NewLeonardoGenerationPoller(repository, client, generationPollClockMock{now: now}).Poll(context.Background(), "job-1")
	require.NoError(t, err)
	require.Equal(t, GenerationJobStatusFailed, job.Status)
	require.Equal(t, "content_policy_violation", *job.ErrorCode)
	require.Equal(t, now, *job.FailedAt)
	require.Nil(t, job.CompletedAt)
	require.Nil(t, job.NextPollAt)
	require.Zero(t, job.OutputCount)
	require.Equal(t, GenerationJobBillingStatusManualReview, job.BillingStatus)
	require.Equal(t, map[string]any{"cost": "0.001", "apiCreditCost": 1}, job.ResultPayload)
}

func TestLeonardoGenerationPollerModeratesOutputImages(t *testing.T) {
	now := time.Date(2026, time.August, 3, 2, 0, 0, 0, time.UTC)
	stored := pollerTestJob(GenerationJobStatusQueued)
	stored.UserID = 11
	stored.APIKeyID = 12
	groupID := int64(13)
	stored.GroupID = &groupID
	stored.Model = "gpt-image-2"
	repository := &generationPollRepositoryMock{job: stored}
	client := &generationPollClientMock{response: &leonardo.Generation{Status: "COMPLETE", GeneratedImages: []leonardo.GeneratedImage{
		{ID: "safe", URL: "https://example.com/safe.png", NSFW: leonardoNSFW(false)},
		{ID: "blocked", URL: "https://example.com/blocked.png", NSFW: leonardoNSFW(false)},
	}}}
	moderator := &generationOutputModeratorMock{blocked: map[string]bool{"https://example.com/blocked.png": true}}

	job, err := NewLeonardoGenerationPoller(repository, client, generationPollClockMock{now: now}, moderator).Poll(context.Background(), "job-1")
	require.NoError(t, err)
	require.Equal(t, GenerationJobStatusFailed, job.Status)
	require.Zero(t, job.OutputCount)
	require.Equal(t, GenerationJobBillingStatusManualReview, job.BillingStatus)
	require.NotContains(t, job.ResultPayload, "images")
	require.Len(t, moderator.inputs, 2)
	require.Equal(t, ContentModerationCheckInput{
		RequestID: "job-1", UserID: 11, APIKeyID: 12, GroupID: stored.GroupID,
		Endpoint: "/v1/media/generations/:id", Provider: PlatformLeonardo,
		Model: "gpt-image-2", Protocol: ContentModerationProtocolOpenAIImages,
		Body: moderator.inputs[0].Body,
	}, moderator.inputs[0])
}

func TestLeonardoGenerationPollerModerationErrorDoesNotCommit(t *testing.T) {
	stored := pollerTestJob(GenerationJobStatusQueued)
	repository := &generationPollRepositoryMock{job: stored}
	client := &generationPollClientMock{response: &leonardo.Generation{Status: "COMPLETE", GeneratedImages: []leonardo.GeneratedImage{{ID: "image", URL: "https://example.com/image.png", NSFW: leonardoNSFW(false)}}}}
	moderationErr := errors.New("moderation unavailable")

	job, err := NewLeonardoGenerationPoller(repository, client, generationPollClockMock{now: time.Now()}, &generationOutputModeratorMock{err: moderationErr}).Poll(context.Background(), "job-1")
	require.ErrorIs(t, err, moderationErr)
	require.Equal(t, GenerationJobStatusQueued, job.Status)
	require.Equal(t, GenerationJobStatusQueued, repository.job.Status)
	require.Zero(t, repository.job.PollAttempts)
}

func TestLeonardoGenerationPollerFailsClosedWhenAllOutputSafetyIsUnknown(t *testing.T) {
	now := time.Date(2026, time.August, 3, 2, 0, 0, 0, time.UTC)
	stored := pollerTestJob(GenerationJobStatusQueued)
	stored.ResultPayload = map[string]any{"cost": "0.001"}
	repository := &generationPollRepositoryMock{job: stored}
	client := &generationPollClientMock{response: &leonardo.Generation{Status: "COMPLETE", GeneratedImages: []leonardo.GeneratedImage{{ID: "unknown", URL: "https://example.com/unknown.png"}}}}

	job, err := NewLeonardoGenerationPoller(repository, client, generationPollClockMock{now: now}).Poll(context.Background(), "job-1")
	require.NoError(t, err)
	require.Equal(t, GenerationJobStatusFailed, job.Status)
	require.Equal(t, "content_policy_violation", *job.ErrorCode)
	require.Equal(t, now, *job.FailedAt)
	require.Zero(t, job.OutputCount)
	require.Equal(t, map[string]any{"cost": "0.001"}, job.ResultPayload)
}

func TestLeonardoGenerationPollerFailed(t *testing.T) {
	now := time.Date(2026, time.August, 3, 2, 0, 0, 0, time.UTC)
	repository := &generationPollRepositoryMock{job: pollerTestJob(GenerationJobStatusRunning)}
	client := &generationPollClientMock{response: &leonardo.Generation{Status: "FAILED"}}

	job, err := NewLeonardoGenerationPoller(repository, client, generationPollClockMock{now: now}).Poll(context.Background(), "job-1")
	require.NoError(t, err)
	require.Equal(t, GenerationJobStatusFailed, job.Status)
	require.Equal(t, now, *job.FailedAt)
	require.Nil(t, job.NextPollAt)
	require.Equal(t, "upstream_failed", *job.ErrorCode)
}

func TestLeonardoGenerationPollerErrorBackoffAndRedaction(t *testing.T) {
	now := time.Date(2026, time.August, 3, 2, 0, 0, 0, time.UTC)
	job := pollerTestJob(GenerationJobStatusQueued)
	job.PollAttempts = 3
	repository := &generationPollRepositoryMock{job: job}
	pollErr := &leonardo.LeonardoError{StatusCode: 429, Message: "Authorization=secret-token", RetryAfter: 45 * time.Second}
	client := &generationPollClientMock{err: pollErr}

	updated, err := NewLeonardoGenerationPoller(repository, client, generationPollClockMock{now: now}).Poll(context.Background(), "job-1")
	require.ErrorIs(t, err, pollErr)
	require.Equal(t, 4, updated.PollAttempts)
	require.Equal(t, now.Add(45*time.Second), *updated.NextPollAt)
	require.NotContains(t, *updated.ErrorMessage, "secret-token")
	require.Equal(t, "poll_error", *updated.ErrorCode)

	job = pollerTestJob(GenerationJobStatusQueued)
	repository = &generationPollRepositoryMock{job: job}
	pollErr = &leonardo.LeonardoError{StatusCode: 429, RetryAfter: time.Hour}
	updated, err = NewLeonardoGenerationPoller(repository, &generationPollClientMock{err: pollErr}, generationPollClockMock{now: now}).Poll(context.Background(), "job-1")
	require.ErrorIs(t, err, pollErr)
	require.Equal(t, now.Add(time.Hour), *updated.NextPollAt)

	job = pollerTestJob(GenerationJobStatusQueued)
	job.PollAttempts = 100
	repository = &generationPollRepositoryMock{job: job}
	updated, err = NewLeonardoGenerationPoller(repository, &generationPollClientMock{err: errors.New("temporary")}, generationPollClockMock{now: now}).Poll(context.Background(), "job-1")
	require.Error(t, err)
	require.Equal(t, now.Add(5*time.Minute), *updated.NextPollAt)
}

func TestLeonardoGenerationPollerDoesNotGetIneligibleJobs(t *testing.T) {
	tests := []struct {
		name   string
		status GenerationJobStatus
		id     *string
	}{
		{name: "unknown with id", status: GenerationJobStatusUnknown, id: stringPointer(pollerTestGenerationID)},
		{name: "succeeded", status: GenerationJobStatusSucceeded, id: stringPointer(pollerTestGenerationID)},
		{name: "failed", status: GenerationJobStatusFailed, id: stringPointer(pollerTestGenerationID)},
		{name: "cancelled", status: GenerationJobStatusCancelled, id: stringPointer(pollerTestGenerationID)},
		{name: "queued invalid id", status: GenerationJobStatusQueued, id: stringPointer("invalid")},
		{name: "running no id", status: GenerationJobStatusRunning},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			job := pollerTestJob(test.status)
			job.UpstreamGenerationID = test.id
			if test.status == GenerationJobStatusUnknown {
				job.BillingStatus = GenerationJobBillingStatusSettled
			}
			repository := &generationPollRepositoryMock{job: job}
			client := &generationPollClientMock{}
			result, err := NewLeonardoGenerationPoller(repository, client, generationPollClockMock{}).Poll(context.Background(), "job-1")
			require.NoError(t, err)
			require.Equal(t, 0, client.callCount())
			if test.status == GenerationJobStatusUnknown {
				require.Equal(t, GenerationJobBillingStatusSettled, result.BillingStatus)
			}
		})
	}
}

func TestLeonardoGenerationPollerDoesNotGetBeforeNextPollAt(t *testing.T) {
	now := time.Date(2026, time.August, 3, 2, 0, 0, 0, time.UTC)
	job := pollerTestJob(GenerationJobStatusQueued)
	job.NextPollAt = timePointerValue(now.Add(time.Second))
	repository := &generationPollRepositoryMock{job: job}
	client := &generationPollClientMock{}

	result, err := NewLeonardoGenerationPoller(repository, client, generationPollClockMock{now: now}).Poll(context.Background(), "job-1")
	require.NoError(t, err)
	require.Equal(t, 0, client.callCount())
	require.Equal(t, job.NextPollAt, result.NextPollAt)
}

func TestLeonardoGenerationPollerCASConflict(t *testing.T) {
	repository := &generationPollRepositoryMock{job: pollerTestJob(GenerationJobStatusQueued)}
	client := &generationPollClientMock{
		response:   &leonardo.Generation{Status: "PENDING"},
		beforeDone: make(chan struct{}, 2),
		release:    make(chan struct{}, 2),
	}
	poller := NewLeonardoGenerationPoller(repository, client, generationPollClockMock{now: time.Now()})
	errorsChannel := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, err := poller.Poll(context.Background(), "job-1")
			errorsChannel <- err
		}()
	}
	<-client.beforeDone
	<-client.beforeDone
	client.release <- struct{}{}
	client.release <- struct{}{}

	err1 := <-errorsChannel
	err2 := <-errorsChannel
	require.True(t, (err1 == nil && errors.Is(err2, ErrGenerationJobConflict)) || (err2 == nil && errors.Is(err1, ErrGenerationJobConflict)))
	require.Equal(t, 1, repository.job.PollAttempts)
}

func pollerTestJob(status GenerationJobStatus) *GenerationJob {
	return &GenerationJob{
		PublicID:             "job-1",
		Status:               status,
		UpstreamGenerationID: stringPointer(pollerTestGenerationID),
		ResultPayload:        map[string]any{},
		BillingStatus:        GenerationJobBillingStatusSubmitted,
		Modality:             "image",
	}
}

func TestLeonardoGenerationPollerCompletesVideo(t *testing.T) {
	job := pollerTestJob(GenerationJobStatusQueued)
	job.Modality = "video"
	job.RequestPayload = map[string]any{"parameters": map[string]any{"duration": 4, "width": 864, "height": 480}}
	repository := &generationPollRepositoryMock{job: job}
	client := &generationPollClientMock{response: &leonardo.Generation{Status: "COMPLETE", GeneratedImages: []leonardo.GeneratedImage{{ID: "video-1", MotionMP4URL: "https://example.com/video.mp4", NSFW: leonardoNSFW(false)}}}}

	result, err := NewLeonardoGenerationPoller(repository, client, generationPollClockMock{now: time.Now()}).Poll(context.Background(), job.PublicID)

	require.NoError(t, err)
	require.Equal(t, GenerationJobStatusSucceeded, result.Status)
	require.Equal(t, 1, result.OutputCount)
	require.Equal(t, 1, client.callCount())
	require.Equal(t, []map[string]any{{"id": "video-1", "url": "https://example.com/video.mp4", "mime": "video/mp4", "nsfw": false, "duration": 4, "width": 864, "height": 480}}, result.ResultPayload["videos"])
}

func TestLeonardoGenerationPollerDoesNotCompleteVideoWithoutSafeMP4(t *testing.T) {
	job := pollerTestJob(GenerationJobStatusQueued)
	job.Modality = "video"
	repository := &generationPollRepositoryMock{job: job}
	client := &generationPollClientMock{response: &leonardo.Generation{Status: "COMPLETE", GeneratedImages: []leonardo.GeneratedImage{{ID: "video-1", MotionMP4URL: "http://example.com/video.mp4", NSFW: leonardoNSFW(false)}}}}

	result, err := NewLeonardoGenerationPoller(repository, client, generationPollClockMock{now: time.Now()}).Poll(context.Background(), job.PublicID)

	require.Error(t, err)
	require.Equal(t, GenerationJobStatusQueued, result.Status)
	require.Equal(t, "invalid_upstream_output", *result.ErrorCode)
}

func TestLeonardoGenerationPollerRejectsUnverifiedAudioWithoutNetwork(t *testing.T) {
	job := pollerTestJob(GenerationJobStatusQueued)
	job.Modality = "audio"
	repository := &generationPollRepositoryMock{job: job}
	client := &generationPollClientMock{}
	poller := NewLeonardoGenerationPoller(repository, client, generationPollClockMock{now: time.Now()})

	result, err := poller.Poll(context.Background(), job.PublicID)

	require.ErrorIs(t, err, ErrLeonardoAudioSchemaUnverified)
	require.Equal(t, GenerationJobStatusQueued, result.Status)
	require.Zero(t, result.PollAttempts)
	require.Zero(t, client.callCount())
}

func TestLeonardoGenerationPollerRejectsUnverified3DWithoutNetwork(t *testing.T) {
	job := pollerTestJob(GenerationJobStatusQueued)
	job.Modality = "3d"
	repository := &generationPollRepositoryMock{job: job}
	client := &generationPollClientMock{}
	poller := NewLeonardoGenerationPoller(repository, client, generationPollClockMock{now: time.Now()})

	result, err := poller.Poll(context.Background(), job.PublicID)

	require.ErrorIs(t, err, ErrLeonardo3DSchemaUnverified)
	require.Equal(t, GenerationJobStatusQueued, result.Status)
	require.Zero(t, result.PollAttempts)
	require.Zero(t, client.callCount())
}

func TestLeonardoGenerationPollerDoesNotSucceedWithoutImageOutputs(t *testing.T) {
	job := pollerTestJob(GenerationJobStatusQueued)
	repository := &generationPollRepositoryMock{job: job}
	client := &generationPollClientMock{response: &leonardo.Generation{Status: "COMPLETE"}}

	result, err := NewLeonardoGenerationPoller(repository, client, generationPollClockMock{now: time.Now()}).Poll(context.Background(), job.PublicID)

	require.Error(t, err)
	require.Equal(t, GenerationJobStatusQueued, result.Status)
	require.Equal(t, "invalid_upstream_output", *result.ErrorCode)
	require.Equal(t, GenerationJobBillingStatusSubmitted, result.BillingStatus)
}
