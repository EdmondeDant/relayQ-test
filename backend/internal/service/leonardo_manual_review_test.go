package service

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestLeonardoManualReviewRefundReleasesReservation(t *testing.T) {
	cost := decimal.RequireFromString("0.005")
	job := orchestratorJob(GenerationJobStatusUnknown)
	job.BillingStatus = GenerationJobBillingStatusManualReview
	job.CustomerCost = &cost
	job.BillingReference = stringPointer("leo_hold_existing")
	repository := &manualReviewRepositoryMock{job: job}
	funds := &orchestratorFundsMock{}

	result, err := NewLeonardoManualReviewService(repository, funds).Refund(context.Background(), job.PublicID, "confirmed not accepted")

	require.NoError(t, err)
	require.Equal(t, GenerationJobStatusFailed, result.Status)
	require.Equal(t, GenerationJobBillingStatusRefunded, result.BillingStatus)
	require.Equal(t, 1, funds.releaseCalls)
	require.Equal(t, "confirmed not accepted", funds.releaseReason)
}

func TestLeonardoManualReviewAttachUpstreamIDMakesUnknownRecoverable(t *testing.T) {
	job := orchestratorJob(GenerationJobStatusUnknown)
	job.BillingStatus = GenerationJobBillingStatusManualReview
	job.UpstreamGenerationID = nil
	repository := &manualReviewRepositoryMock{job: job}

	result, err := NewLeonardoManualReviewService(repository, &orchestratorFundsMock{}).AttachUpstreamID(context.Background(), job.PublicID, orchestratorGenerationID)

	require.NoError(t, err)
	require.Equal(t, orchestratorGenerationID, *result.UpstreamGenerationID)
	require.NotNil(t, result.NextPollAt)
}

type manualReviewRepositoryMock struct {
	job *GenerationJob
}

func (r *manualReviewRepositoryMock) Create(context.Context, *GenerationJob) error { return nil }
func (r *manualReviewRepositoryMock) GetByPublicID(context.Context, string) (*GenerationJob, error) {
	job := *r.job
	return &job, nil
}
func (r *manualReviewRepositoryMock) GetByUpstreamGenerationID(context.Context, string) (*GenerationJob, error) {
	return nil, ErrGenerationJobNotFound
}
func (r *manualReviewRepositoryMock) CompareAndSwapStatus(_ context.Context, _ string, _ GenerationJobStatus, job *GenerationJob) error {
	r.job = job
	return nil
}
