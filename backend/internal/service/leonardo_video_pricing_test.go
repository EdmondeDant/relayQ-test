package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLeonardoVideoPriceResolverUsesTierMaximum(t *testing.T) {
	resolver := NewLeonardoVideoPriceResolver()

	estimate, err := resolver.Estimate(context.Background(), LeonardoVideoPriceRequest{Model: "grok-imagine-1.5", Duration: 5, Width: 1280, Height: 720, Quantity: 1, MotionHasAudio: true, QualityTier: "medium"})

	require.NoError(t, err)
	require.Equal(t, "6.7813", estimate.EstimatedCostUSD.String())
	require.Equal(t, "quality_tier_max", estimate.MatchType)
	require.NotEmpty(t, LeonardoVideoPricingPolicyVersion)
}

func TestLeonardoVideoPriceResolverRejectsUnknownTier(t *testing.T) {
	_, err := NewLeonardoVideoPriceResolver().Estimate(context.Background(), LeonardoVideoPriceRequest{Model: "model", Duration: 5, Width: 1280, Height: 720, Quantity: 1, QualityTier: "ultra"})
	require.ErrorIs(t, err, ErrLeonardoVideoPricingEvidenceUnavailable)
}
