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
	estimatedAmount := decimal.RequireFromString("0.75")
	actualAmount := decimal.RequireFromString("1.25")
	unit := "USD"
	pricingVersion := "2026-08-01"
	pricingSource := "leonardo_authenticated_pricing_calculator"
	pricingMatchType := "exact"
	grossMargin := decimal.RequireFromString("0.5")
	costVariance := decimal.RequireFromString("0.5")
	job := &GenerationJob{
		Status:                      GenerationJobStatusUnknown,
		EstimatedUpstreamCostAmount: &estimatedAmount,
		EstimatedUpstreamCostUnit:   &unit,
		PricingSnapshotVersion:      &pricingVersion,
		PricingSource:               &pricingSource,
		PricingMatchType:            &pricingMatchType,
		ActualUpstreamCostAmount:    &actualAmount,
		ActualUpstreamCostUnit:      &unit,
		CustomerCost:                &actualAmount,
		GrossMargin:                 &grossMargin,
		CostVariance:                &costVariance,
		BillingStatus:               GenerationJobBillingStatusSettled,
	}

	NormalizeGenerationJob(job)

	require.Equal(t, "submission_unknown", *job.ErrorCode)
	require.Equal(t, GenerationJobBillingStatusManualReview, job.BillingStatus)
	require.Equal(t, "0.75", job.EstimatedUpstreamCostAmount.String())
	require.Equal(t, "USD", *job.EstimatedUpstreamCostUnit)
	require.Equal(t, "2026-08-01", *job.PricingSnapshotVersion)
	require.Equal(t, "leonardo_authenticated_pricing_calculator", *job.PricingSource)
	require.Equal(t, "exact", *job.PricingMatchType)
	require.Nil(t, job.ActualUpstreamCostAmount)
	require.Nil(t, job.ActualUpstreamCostUnit)
	require.Equal(t, actualAmount, *job.CustomerCost)
	require.Nil(t, job.GrossMargin)
	require.Nil(t, job.CostVariance)
}
