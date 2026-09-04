package service

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCalculateSearchCost_DefaultExplicitFreeAndNegative(t *testing.T) {
	s := &BillingService{}
	require.InDelta(t, 0.05, s.CalculateSearchCost(5, nil, 1).ActualCost, 1e-9)
	zero := 0.0
	require.Zero(t, s.CalculateSearchCost(5, &zero, 1).ActualCost)
	negative := -1.0
	require.Zero(t, s.CalculateSearchCost(5, &negative, 1).ActualCost)
}

func TestCalculateAudioCost_DefaultExplicitFreeAndNegative(t *testing.T) {
	s := &BillingService{}
	require.InDelta(t, 0.10, s.CalculateAudioCost("realtime", 1, nil, 1).ActualCost, 1e-9)
	zero := 0.0
	require.Zero(t, s.CalculateAudioCost("tts", 1, &audioPriceConfig{TTSPerMChars: &zero}, 1).ActualCost)
	negative := -1.0
	require.Zero(t, s.CalculateAudioCost("stt", 1, &audioPriceConfig{STTPerHour: &negative}, 1).ActualCost)
}
