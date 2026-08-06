package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLeonardoAudioPricingFailsClosedWithoutEvidence(t *testing.T) {
	resolver := NewLeonardoAudioPriceResolver()
	for _, request := range []LeonardoAudioPriceRequest{
		{},
		{Model: "music-v1", DurationSeconds: 30, Quantity: 1},
		{Model: "sound-effects-v2", DurationSeconds: 10, Quantity: 2},
		{Model: "dialogue-v3", DurationSeconds: 60, Quantity: 1},
	} {
		require.ErrorIs(t, resolver.Estimate(context.Background(), request), ErrLeonardoAudioPricingEvidenceUnavailable)
	}
}
