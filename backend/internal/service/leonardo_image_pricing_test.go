package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLeonardoImagePriceResolverExactEstimate(t *testing.T) {
	estimate, err := NewLeonardoImagePriceResolver().Estimate(context.Background(), leonardoImagePriceRequest())
	require.NoError(t, err)
	require.Equal(t, "0.003", estimate.UnitCostUSD.String())
	require.Equal(t, 1, estimate.Quantity)
	require.Equal(t, "0.003", estimate.EstimatedCostUSD.String())
	require.Equal(t, "2026-08-08", estimate.PricingVersion)
	require.Equal(t, "leonardo_authenticated_pricing_calculator", estimate.PricingSource)
	require.Equal(t, "exact", estimate.MatchType)
}

func TestLeonardoImagePriceResolverStrictModel(t *testing.T) {
	for _, model := range []string{"FLUX Schnell", "1dd50843-d653-4516-a8e3-f0238ee453ff", "unknown", "FLUX-SCHNELL"} {
		request := leonardoImagePriceRequest()
		request.Model = model
		_, err := NewLeonardoImagePriceResolver().Estimate(context.Background(), request)
		require.ErrorIs(t, err, ErrLeonardoImagePricingNotFound)
	}
	request := leonardoImagePriceRequest()
	request.Model = " flux-schnell "
	_, err := NewLeonardoImagePriceResolver().Estimate(context.Background(), request)
	require.NoError(t, err)
}

func TestLeonardoImagePriceResolverDimensions(t *testing.T) {
	for _, dimensions := range [][2]int{{896, 1024}, {1024, 896}, {2880, 2880}} {
		request := leonardoImagePriceRequest()
		request.Width, request.Height = dimensions[0], dimensions[1]
		_, err := NewLeonardoImagePriceResolver().Estimate(context.Background(), request)
		require.ErrorIs(t, err, ErrLeonardoImagePricingNotFound)
	}
	for _, dimensions := range [][2]int{{0, 896}, {-1, 896}, {896, 0}, {896, -1}} {
		request := leonardoImagePriceRequest()
		request.Width, request.Height = dimensions[0], dimensions[1]
		_, err := NewLeonardoImagePriceResolver().Estimate(context.Background(), request)
		require.ErrorIs(t, err, ErrLeonardoImagePricingRequestInvalid)
	}
}

func TestLeonardoImagePriceResolverQualityTierMaximum(t *testing.T) {
	request := leonardoImagePriceRequest()
	request.Width, request.Height = 2048, 2048
	estimate, err := NewLeonardoImagePriceResolver().Estimate(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, "0.0045", estimate.EstimatedCostUSD.String())
	require.Equal(t, "quality_tier_max", estimate.MatchType)
}

func TestLeonardoImagePriceResolverQuantityAndPublic(t *testing.T) {
	for _, quantity := range []int{0, -1} {
		request := leonardoImagePriceRequest()
		request.Quantity = quantity
		_, err := NewLeonardoImagePriceResolver().Estimate(context.Background(), request)
		require.ErrorIs(t, err, ErrLeonardoImagePricingRequestInvalid)
	}
	request := leonardoImagePriceRequest()
	request.Quantity = 2
	estimate, err := NewLeonardoImagePriceResolver().Estimate(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, "0.006", estimate.EstimatedCostUSD.String())
	request = leonardoImagePriceRequest()
	request.Public = true
	_, err = NewLeonardoImagePriceResolver().Estimate(context.Background(), request)
	require.ErrorIs(t, err, ErrLeonardoImagePricingNotFound)
}

func TestLeonardoImagePriceResolverNewModels(t *testing.T) {
	tests := []struct {
		model, quality, cost string
		width                int
	}{
		{"gpt-image-2", "low", "0.012", 1024},
		{"gpt-image-2", "medium", "0.2153", 2048},
		{"gpt-image-2", "high", "2.6596", 2880},
		{"nano-banana-2", "low", "0.0389", 1024},
		{"nano-banana-2", "low", "0.0583", 2048},
		{"nano-banana-2-lite", "low", "0.0449", 1024},
		{"kino-xl", "low", "0.0045", 896},
		{"kino-xl", "high", "0.0269", 1120},
		{"concept-art", "low", "0.0045", 888},
		{"concept-art", "high", "0.0239", 1024},
		{"graphic-design", "low", "0.0045", 960},
		{"illustrative-albedo", "high", "0.0239", 888},
	}
	for _, test := range tests {
		estimate, err := NewLeonardoImagePriceResolver().Estimate(context.Background(), LeonardoImagePriceRequest{Model: test.model, Width: test.width, Height: test.width, Quantity: 1, QualityTier: test.quality})
		require.NoError(t, err, test.model)
		require.Equal(t, test.cost, estimate.UnitCostUSD.String(), test.model)
	}
}

func TestLeonardoImagePriceResolverInvalidModel(t *testing.T) {
	for _, model := range []string{"", "   "} {
		request := leonardoImagePriceRequest()
		request.Model = model
		_, err := NewLeonardoImagePriceResolver().Estimate(context.Background(), request)
		require.ErrorIs(t, err, ErrLeonardoImagePricingRequestInvalid)
	}
}

func TestLeonardoImagePriceResolverCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewLeonardoImagePriceResolver().Estimate(ctx, leonardoImagePriceRequest())
	require.ErrorIs(t, err, context.Canceled)
}

func TestLeonardoImagePriceResolverReturnsIndependentEstimates(t *testing.T) {
	resolver := NewLeonardoImagePriceResolver()
	first, err := resolver.Estimate(context.Background(), leonardoImagePriceRequest())
	require.NoError(t, err)
	second, err := resolver.Estimate(context.Background(), leonardoImagePriceRequest())
	require.NoError(t, err)
	require.NotSame(t, first, second)
	first.Quantity = 99
	require.Equal(t, 1, second.Quantity)
}

func leonardoImagePriceRequest() LeonardoImagePriceRequest {
	return LeonardoImagePriceRequest{Model: "flux-schnell", Width: 896, Height: 896, Quantity: 1}
}
