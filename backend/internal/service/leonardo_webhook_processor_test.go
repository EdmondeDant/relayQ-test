package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestLeonardoGenerationPollOrchestratorApplyWebhookCompletesAndSettles(t *testing.T) {
	nsfw := false
	job := &GenerationJob{PublicID: "job-1", Provider: PlatformLeonardo, Modality: "image", Status: GenerationJobStatusQueued, BillingStatus: GenerationJobBillingStatusSubmitted, AccountID: 7, UserID: 1, APIKeyID: 2, Model: "flux-schnell", UpstreamGenerationID: stringPointer(orchestratorGenerationID), CustomerCost: decimalPointer(decimal.RequireFromString("0.005")), BillingReference: stringPointer("reservation-1")}
	repository := &orchestratorRepositoryMock{job: job}
	funds := &orchestratorFundsMock{}
	orchestrator := NewLeonardoGenerationPollOrchestrator(repository, &orchestratorAccountLoaderMock{}, &orchestratorUpstreamMock{}, &config.Config{}, generationPollClockMock{now: time.Now()}, funds)

	result, err := orchestrator.ApplyWebhook(context.Background(), 7, &leonardo.Generation{ID: orchestratorGenerationID, Status: "COMPLETE", GeneratedImages: []leonardo.GeneratedImage{{ID: "image-1", URL: "https://example.com/image.png", NSFW: &nsfw}}})

	require.NoError(t, err)
	require.Equal(t, GenerationJobStatusSucceeded, result.Status)
	require.Equal(t, GenerationJobBillingStatusSettled, result.BillingStatus)
	require.Equal(t, 1, result.OutputCount)
	require.Equal(t, 1, funds.settleCalls)
	require.Zero(t, funds.releaseCalls)
}

func TestLeonardoGenerationPollOrchestratorApplyWebhookRejectsAccountMismatchAndTerminalRegression(t *testing.T) {
	job := &GenerationJob{PublicID: "job-1", Provider: PlatformLeonardo, Modality: "image", Status: GenerationJobStatusSucceeded, BillingStatus: GenerationJobBillingStatusSettled, AccountID: 7, UpstreamGenerationID: stringPointer(orchestratorGenerationID)}
	repository := &orchestratorRepositoryMock{job: job}
	funds := &orchestratorFundsMock{}
	orchestrator := NewLeonardoGenerationPollOrchestrator(repository, &orchestratorAccountLoaderMock{}, &orchestratorUpstreamMock{}, &config.Config{}, generationPollClockMock{now: time.Now()}, funds)

	_, err := orchestrator.ApplyWebhook(context.Background(), 8, &leonardo.Generation{ID: orchestratorGenerationID, Status: "PENDING"})
	require.ErrorIs(t, err, ErrLeonardoGenerationPollAccountBinding)

	result, err := orchestrator.ApplyWebhook(context.Background(), 7, &leonardo.Generation{ID: orchestratorGenerationID, Status: "PENDING"})
	require.NoError(t, err)
	require.Equal(t, GenerationJobStatusSucceeded, result.Status)
	require.Zero(t, repository.statusCASCalls)
	require.Zero(t, funds.settleCalls)
}

func TestLeonardoGenerationPollOrchestratorRecoversStaleStates(t *testing.T) {
	t.Run("submitting becomes unknown without upstream request", func(t *testing.T) {
		job := orchestratorJob(GenerationJobStatusSubmitting)
		job.UpdatedAt = time.Now().Add(-2 * LeonardoGenerationReconciliationDelay)
		repository := &orchestratorRepositoryMock{job: job}
		loader := &orchestratorAccountLoaderMock{account: orchestratorAccount()}
		upstream := &orchestratorUpstreamMock{}
		result, err := NewLeonardoGenerationPollOrchestrator(repository, loader, upstream, &config.Config{}, generationPollClockMock{now: time.Now()}, &orchestratorFundsMock{}).Poll(context.Background(), job.PublicID)

		require.NoError(t, err)
		require.Equal(t, GenerationJobStatusUnknown, result.Status)
		require.Equal(t, GenerationJobBillingStatusManualReview, result.BillingStatus)
		require.Zero(t, loader.calls)
		require.Zero(t, upstream.calls)
	})

	t.Run("unknown with generation id is queried without resubmit", func(t *testing.T) {
		job := orchestratorJob(GenerationJobStatusUnknown)
		job.UpstreamGenerationID = stringPointer(orchestratorGenerationID)
		repository := &orchestratorRepositoryMock{job: job}
		loader := &orchestratorAccountLoaderMock{account: orchestratorAccount()}
		upstream := &orchestratorUpstreamMock{status: "COMPLETE"}
		result, err := NewLeonardoGenerationPollOrchestrator(repository, loader, upstream, &config.Config{}, generationPollClockMock{now: time.Now()}, &orchestratorFundsMock{}).Poll(context.Background(), job.PublicID)

		require.NoError(t, err)
		require.Equal(t, GenerationJobStatusSucceeded, result.Status)
		require.Equal(t, GenerationJobBillingStatusManualReview, result.BillingStatus)
		require.Equal(t, 1, upstream.calls)
		require.Zero(t, upstream.postCalls)
	})
}

func TestLeonardoWebhookAndPollRaceConvergesOnce(t *testing.T) {
	nsfw := false
	job := orchestratorJob(GenerationJobStatusQueued)
	job.BillingStatus = GenerationJobBillingStatusSubmitted
	job.CustomerCost = decimalPointer(decimal.RequireFromString("0.005"))
	job.BillingReference = stringPointer("reservation-1")
	repository := &orchestratorRepositoryMock{job: job}
	loader := &orchestratorAccountLoaderMock{account: orchestratorAccount()}
	upstream := &orchestratorUpstreamMock{status: "COMPLETE", barrier: make(chan struct{}, 1), release: make(chan struct{}, 1)}
	funds := &orchestratorFundsMock{}
	orchestrator := NewLeonardoGenerationPollOrchestrator(repository, loader, upstream, &config.Config{}, generationPollClockMock{now: time.Now()}, funds)
	pollResult := make(chan error, 1)
	go func() {
		_, err := orchestrator.Poll(context.Background(), job.PublicID)
		pollResult <- err
	}()
	<-upstream.barrier

	result, err := orchestrator.ApplyWebhook(context.Background(), job.AccountID, &leonardo.Generation{ID: orchestratorGenerationID, Status: "COMPLETE", GeneratedImages: []leonardo.GeneratedImage{{ID: "image-1", URL: "https://example.com/image.png", NSFW: &nsfw}}})
	require.NoError(t, err)
	require.Equal(t, GenerationJobStatusSucceeded, result.Status)
	upstream.release <- struct{}{}
	require.ErrorIs(t, <-pollResult, ErrGenerationJobConflict)
	require.Equal(t, GenerationJobStatusSucceeded, repository.job.Status)
	require.Equal(t, GenerationJobBillingStatusSettled, repository.job.BillingStatus)
	require.Equal(t, 1, funds.settleCalls)
	require.Zero(t, funds.releaseCalls)
}

func TestLeonardoWebhookProcessorHeartbeat(t *testing.T) {
	var heartbeat *OpsUpsertJobHeartbeatInput
	ops := &opsRepoMock{UpsertJobHeartbeatFn: func(_ context.Context, input *OpsUpsertJobHeartbeatInput) error {
		heartbeat = input
		return nil
	}}
	processor := &LeonardoWebhookProcessor{ops: ops, lastClaimed: 3, lastProcessed: 2, lastFailed: 1}
	processor.heartbeat(time.Now(), nil)

	require.NotNil(t, heartbeat)
	require.Equal(t, "leonardo_webhook_worker", heartbeat.JobName)
	require.NotNil(t, heartbeat.LastSuccessAt)
	require.Equal(t, "claimed=3 processed=2 failed=1", *heartbeat.LastResult)
}

func TestLeonardoGenerationPollOrchestratorDoesNotBillVideoTerminalJob(t *testing.T) {
	job := validTerminalOrchestratorJob(GenerationJobStatusSucceeded)
	job.Modality = "video"
	repository := &orchestratorRepositoryMock{job: job}
	funds := &orchestratorFundsMock{}
	orchestrator := NewLeonardoGenerationPollOrchestrator(repository, &orchestratorAccountLoaderMock{}, &orchestratorUpstreamMock{}, &config.Config{}, generationPollClockMock{now: time.Now()}, funds)

	result, err := orchestrator.Poll(context.Background(), job.PublicID)

	require.NoError(t, err)
	require.Equal(t, GenerationJobBillingStatusManualReview, result.BillingStatus)
	require.Zero(t, funds.settleCalls)
	require.Zero(t, funds.releaseCalls)
}

func TestLeonardoGenerationPollOrchestratorRejectsVideoWebhookBeforeMutation(t *testing.T) {
	job := orchestratorJob(GenerationJobStatusQueued)
	job.Modality = "video"
	repository := &orchestratorRepositoryMock{job: job}
	funds := &orchestratorFundsMock{}
	orchestrator := NewLeonardoGenerationPollOrchestrator(repository, &orchestratorAccountLoaderMock{}, &orchestratorUpstreamMock{}, &config.Config{}, generationPollClockMock{now: time.Now()}, funds)

	result, err := orchestrator.ApplyWebhook(context.Background(), job.AccountID, &leonardo.Generation{ID: orchestratorGenerationID, Status: "COMPLETE"})

	require.ErrorIs(t, err, ErrLeonardoVideoSchemaUnverified)
	require.Equal(t, GenerationJobStatusQueued, result.Status)
	require.Zero(t, repository.statusCASCalls)
	require.Zero(t, funds.settleCalls)
	require.Zero(t, funds.releaseCalls)
}

func decimalPointer(value decimal.Decimal) *decimal.Decimal {
	return &value
}
