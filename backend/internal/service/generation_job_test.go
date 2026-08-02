package service

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestCanTransitionGenerationJobStatus(t *testing.T) {
	tests := []struct {
		from    GenerationJobStatus
		to      GenerationJobStatus
		allowed bool
	}{
		{GenerationJobStatusCreated, GenerationJobStatusSubmitting, true},
		{GenerationJobStatusSubmitting, GenerationJobStatusUnknown, true},
		{GenerationJobStatusQueued, GenerationJobStatusRunning, true},
		{GenerationJobStatusRunning, GenerationJobStatusSucceeded, true},
		{GenerationJobStatusUnknown, GenerationJobStatusRunning, true},
		{GenerationJobStatusSucceeded, GenerationJobStatusRunning, false},
		{GenerationJobStatusFailed, GenerationJobStatusQueued, false},
		{GenerationJobStatusCancelled, GenerationJobStatusRunning, false},
		{GenerationJobStatusCreated, GenerationJobStatusSucceeded, false},
	}

	for _, test := range tests {
		require.Equal(t, test.allowed, CanTransitionGenerationJobStatus(test.from, test.to), "%s -> %s", test.from, test.to)
	}
}

func TestNormalizeGenerationJobUnknown(t *testing.T) {
	amount := decimal.RequireFromString("1.25")
	unit := "USD"
	job := &GenerationJob{
		Status:                   GenerationJobStatusUnknown,
		ActualUpstreamCostAmount: &amount,
		ActualUpstreamCostUnit:   &unit,
		CustomerCost:             &amount,
		BillingStatus:            GenerationJobBillingStatusSettled,
	}

	NormalizeGenerationJob(job)

	require.Equal(t, "submission_unknown", *job.ErrorCode)
	require.Equal(t, GenerationJobBillingStatusManualReview, job.BillingStatus)
	require.Nil(t, job.ActualUpstreamCostAmount)
	require.Nil(t, job.ActualUpstreamCostUnit)
	require.Nil(t, job.CustomerCost)
}
