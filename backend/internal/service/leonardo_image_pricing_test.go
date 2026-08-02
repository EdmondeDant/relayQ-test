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
	require.Equal(t, "2026-08-01", estimate.PricingVersion)
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
	for _, dimensions := range [][2]int{{1024, 1024}, {896, 1024}, {1024, 896}} {
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

func TestLeonardoImagePriceResolverQuantityAndPublic(t *testing.T) {
	for _, quantity := range []int{0, -1} {
		request := leonardoImagePriceRequest()
		request.Quantity = quantity
		_, err := NewLeonardoImagePriceResolver().Estimate(context.Background(), request)
		require.ErrorIs(t, err, ErrLeonardoImagePricingRequestInvalid)
	}
	request := leonardoImagePriceRequest()
	request.Quantity = 2
	_, err := NewLeonardoImagePriceResolver().Estimate(context.Background(), request)
	require.ErrorIs(t, err, ErrLeonardoImagePricingNotFound)
	request = leonardoImagePriceRequest()
	request.Public = true
	_, err = NewLeonardoImagePriceResolver().Estimate(context.Background(), request)
	require.ErrorIs(t, err, ErrLeonardoImagePricingNotFound)
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
