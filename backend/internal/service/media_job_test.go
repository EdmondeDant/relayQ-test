package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMediaJobReservesBeforeAttemptAndSettlesOnce(t *testing.T) {
	job, result, now := mediaSelectedTestJob(t)
	require.NoError(t, job.ApplySelect(result, now))
	require.ErrorIs(t, func() error { _, err := job.BeginNextAttempt(now); return err }(), ErrMediaJobInvalidTransition)
	require.NoError(t, job.ReserveCustomerCharge(now.Add(time.Second)))
	attempt, err := job.BeginNextAttempt(now.Add(2 * time.Second))
	require.NoError(t, err)
	require.Equal(t, job.Reservation.Amount, attempt.CustomerChargeSnap)
	require.NoError(t, job.EndAttempt(1, AttemptEndInput{Status: MediaAttemptSucceeded, UpstreamRef: "upstream-1", BillableOutput: 2}, now.Add(3*time.Second)))
	require.NoError(t, job.SettleOrRelease(now.Add(4*time.Second)))
	require.Equal(t, MediaReservationCaptured, job.Reservation.State)
	require.Equal(t, MediaJobSettled, job.Status)
	require.ErrorIs(t, job.SettleOrRelease(now.Add(5*time.Second)), ErrMediaJobInvalidTransition)
	require.Equal(t, []string{"select", "reserve", "attempt_start", "attempt_end", "settle"}, mediaAuditTypes(job))
}

func TestMediaJobRetryableFailureFailsOverWithoutDoubleCharge(t *testing.T) {
	job, result, now := mediaSelectedTestJob(t)
	require.NoError(t, job.ApplySelect(result, now))
	require.NoError(t, job.ReserveCustomerCharge(now))
	first, err := job.BeginNextAttempt(now)
	require.NoError(t, err)
	require.NoError(t, job.EndAttempt(first.AttemptNo, AttemptEndInput{Status: MediaAttemptFailedRetryable, ErrorClass: "upstream_retryable", ErrorCode: "timeout"}, now))
	second, err := job.FailoverToNext(now)
	require.NoError(t, err)
	require.NoError(t, job.EndAttempt(second.AttemptNo, AttemptEndInput{Status: MediaAttemptSucceeded, BillableOutput: 2}, now))
	require.NoError(t, job.SettleOrRelease(now))

	require.Len(t, job.Attempts, 2)
	require.Equal(t, job.Reservation.Amount, job.Attempts[0].CustomerChargeSnap)
	require.Equal(t, job.Reservation.Amount, job.Attempts[1].CustomerChargeSnap)
	require.Equal(t, job.Reservation.Amount, job.Audit.MoneySummary["captured"])
}

func TestMediaJobReleasesAllFailedZeroOutput(t *testing.T) {
	job, result, now := mediaSelectedTestJob(t)
	require.NoError(t, job.ApplySelect(result, now))
	require.NoError(t, job.ReserveCustomerCharge(now))
	attempt, err := job.BeginNextAttempt(now)
	require.NoError(t, err)
	require.NoError(t, job.EndAttempt(attempt.AttemptNo, AttemptEndInput{Status: MediaAttemptFailedTerminal, ErrorClass: "user", ErrorCode: "invalid_prompt"}, now))
	require.ErrorIs(t, func() error { _, err := job.FailoverToNext(now); return err }(), ErrMediaJobInvalidTransition)
	require.NoError(t, job.SettleOrRelease(now))
	require.Equal(t, MediaReservationReleased, job.Reservation.State)
	require.Equal(t, MediaJobReleased, job.Status)
}

func TestMediaJobForbidsFailoverAfterBillableOutputAndMapsAuditFields(t *testing.T) {
	job, result, now := mediaSelectedTestJob(t)
	require.NoError(t, job.ApplySelect(result, now))
	require.NoError(t, job.ReserveCustomerCharge(now))
	attempt, err := job.BeginNextAttempt(now)
	require.NoError(t, err)
	require.NoError(t, job.EndAttempt(attempt.AttemptNo, AttemptEndInput{Status: MediaAttemptFailedRetryable, BillableOutput: 1, UpstreamRef: "partial"}, now))
	require.ErrorIs(t, func() error { _, err := job.FailoverToNext(now); return err }(), ErrMediaJobOutputExists)
	require.NoError(t, job.SettleOrRelease(now))

	draft := BuildUsageLogDraft(job, &job.Attempts[0])
	require.Equal(t, job.Reservation.Amount, draft.ActualCost)
	require.Equal(t, job.PublicModel, draft.RequestedModel)
	require.Equal(t, job.CustomerGroupID, draft.CustomerGroupID)
	require.NotEqual(t, int64(*job.CustomerGroupID), draft.SourceGroupID)
	require.Equal(t, job.Attempts[0].UpstreamModel, draft.UpstreamModel)
	scope, keyHash, fingerprint := BuildIdempotencyScope(job)
	require.Equal(t, "media:image:generations", scope)
	require.Equal(t, job.IdempotencyKeyHash, keyHash)
	require.Equal(t, job.RequestFingerprint, fingerprint)
}

func mediaSelectedTestJob(t *testing.T) (MediaJob, MediaSelectResult, time.Time) {
	t.Helper()
	now := time.Date(2026, time.August, 14, 8, 0, 0, 0, time.UTC)
	groupID := int64(7)
	job := NewMediaJob(MediaJobInput{
		JobID: "job-1", RequestID: "request-1", IdempotencyKeyHash: "key-hash", RequestFingerprint: "fingerprint",
		APIKeyID: 2, UserID: 3, CustomerGroupID: &groupID, ProductID: 1, PublicModel: "relay-image",
		Modality: MediaModalityImage, Op: "generations", CreatedAt: now,
	})
	result := SelectMediaOffer(mediaTestProduct(), []MediaOffer{
		mediaTestOffer(1, "openai_compat", 17, 0.01, now),
		mediaTestOffer(2, "leonardo", 18, 0.02, now),
	}, mediaTestRequest(now))
	require.NoError(t, result.Err)
	return job, result, now
}

func mediaAuditTypes(job MediaJob) []string {
	types := make([]string, 0, len(job.Audit.Events))
	for _, event := range job.Audit.Events {
		types = append(types, event.Type)
	}
	return types
}
