package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAICalculateUsageCostSearchIsAdditive(t *testing.T) {
	billing := NewBillingService(nil, nil)
	svc := &OpenAIGatewayService{billingService: billing}
	apiKey := &APIKey{Group: &Group{SearchPricePer1k: floatPtr(10)}}

	cost, err := svc.calculateOpenAIRecordUsageCost(context.Background(), &OpenAIForwardResult{
		Model: "claude-3-haiku", SearchCount: 100,
		Usage: OpenAIUsage{InputTokens: 1000, OutputTokens: 1000},
	}, apiKey, []string{"claude-3-haiku"}, 1, 1, UsageTokens{InputTokens: 1000, OutputTokens: 1000}, "")
	require.NoError(t, err)
	require.InDelta(t, 0.0015+1, cost.TotalCost, 1e-12)
	require.InDelta(t, cost.TotalCost, cost.ActualCost, 1e-12)
}

func TestOpenAICalculateUsageCostSearchOnlyAndPricingFailure(t *testing.T) {
	billing := NewBillingService(nil, nil)
	svc := &OpenAIGatewayService{billingService: billing}
	apiKey := &APIKey{Group: &Group{SearchPricePer1k: floatPtr(10)}}

	cost, err := svc.calculateOpenAIRecordUsageCost(context.Background(), &OpenAIForwardResult{SearchCount: 5}, apiKey, nil, 1, 1, UsageTokens{}, "")
	require.NoError(t, err)
	require.InDelta(t, 0.05, cost.ActualCost, 1e-12)

	_, err = svc.calculateOpenAIRecordUsageCost(context.Background(), &OpenAIForwardResult{Model: "not-a-priced-model", SearchCount: 5}, apiKey, []string{"not-a-priced-model"}, 1, 1, UsageTokens{}, "")
	require.Error(t, err)
}

func TestOpenAICalculateUsageCostAudioModes(t *testing.T) {
	billing := NewBillingService(nil, nil)
	svc := &OpenAIGatewayService{billingService: billing}
	apiKey := &APIKey{Group: &Group{
		AudioRealtimePricePerMin:     floatPtr(2),
		AudioTTSPricePerMillionChars: floatPtr(3),
		AudioSTTPricePerHour:         floatPtr(4),
	}}
	for _, tc := range []struct {
		mode, want  string
		units, cost float64
	}{
		{mode: "realtime", units: 2, cost: 4},
		{mode: "tts", units: 3, cost: 9},
		{mode: "stt", units: 4, cost: 16},
	} {
		result := &OpenAIForwardResult{AudioUsage: &AudioUsage{Mode: tc.mode, DurationOrUnits: tc.units}}
		got, err := svc.calculateOpenAIRecordUsageCost(context.Background(), result, apiKey, nil, 1, 1, UsageTokens{}, "")
		require.NoError(t, err)
		require.InDelta(t, tc.cost, got.ActualCost, 1e-12)
	}
}

func TestCalculateAudioCostRejectsInvalidUnitsAndPrices(t *testing.T) {
	billing := NewBillingService(nil, nil)
	negative := -1.0
	zero := 0.0
	require.Zero(t, billing.CalculateAudioCost("realtime", 1, &audioPriceConfig{RealtimePerMin: &negative}, 1).ActualCost)
	require.Zero(t, billing.CalculateAudioCost("realtime", 1, &audioPriceConfig{RealtimePerMin: &zero}, 1).ActualCost)
	require.Zero(t, billing.CalculateAudioCost("realtime", 0, nil, 1).ActualCost)
	require.Zero(t, billing.CalculateAudioCost("realtime", -1, nil, 1).ActualCost)
}

func TestGatewayCalculateUsageCostImageSearchIsAdditive(t *testing.T) {
	billing := NewBillingService(nil, nil)
	svc := &GatewayService{billingService: billing}
	apiKey := &APIKey{Group: &Group{
		ImagePrice1K:     floatPtr(0.25),
		SearchPricePer1k: floatPtr(10),
	}}

	cost, err := svc.calculateRecordUsageCost(context.Background(), &ForwardResult{
		Model: "gemini-image", ImageCount: 2, ImageSize: "1K", SearchCount: 100,
	}, apiKey, "gemini-image", 1, 1, nil)
	require.NoError(t, err)
	require.Equal(t, string(BillingModeImage), cost.BillingMode)
	require.InDelta(t, 1.5, cost.TotalCost, 1e-12)
	require.InDelta(t, 1.5, cost.ActualCost, 1e-12)
}

func TestOpenAICalculateUsageCostImageSearchIsAdditive(t *testing.T) {
	billing := NewBillingService(nil, nil)
	svc := &OpenAIGatewayService{billingService: billing}
	apiKey := &APIKey{Group: &Group{
		ImagePrice1K:     floatPtr(0.25),
		SearchPricePer1k: floatPtr(10),
	}}

	cost, err := svc.calculateOpenAIRecordUsageCost(context.Background(), &OpenAIForwardResult{
		Model: "gemini-image", ImageCount: 2, ImageSize: "1K", SearchCount: 100,
	}, apiKey, []string{"gemini-image"}, 1, 1, UsageTokens{}, "")
	require.NoError(t, err)
	require.Equal(t, string(BillingModeImage), cost.BillingMode)
	require.InDelta(t, 1.5, cost.TotalCost, 1e-12)
	require.InDelta(t, 1.5, cost.ActualCost, 1e-12)
}

func floatPtr(v float64) *float64 { return &v }
