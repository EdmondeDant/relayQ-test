package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLeonardo3DPricingFailsClosedWithoutEvidence(t *testing.T) {
	resolver := NewLeonardo3DPriceResolver()
	for _, request := range []Leonardo3DPriceRequest{{}, {Model: "rodin-v2", Quantity: 1}} {
		require.ErrorIs(t, resolver.Estimate(context.Background(), request), ErrLeonardo3DPricingEvidenceUnavailable)
	}
}
