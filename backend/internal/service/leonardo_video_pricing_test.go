package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLeonardoVideoPriceResolverUsesExactSeedancePrice(t *testing.T) {
	resolver := NewLeonardoVideoPriceResolver()

	estimate, err := resolver.Estimate(context.Background(), LeonardoVideoPriceRequest{Model: "seedance-1.0-pro-fast", Duration: 4, Width: 864, Height: 480, Quantity: 1})

	require.NoError(t, err)
	require.Equal(t, "0.0449", estimate.EstimatedCostUSD.String())
	require.Equal(t, "model_duration_resolution_exact", estimate.MatchType)
	require.NotEmpty(t, LeonardoVideoPricingPolicyVersion)
}

func TestLeonardoVideoPriceResolverRejectsUnpricedCombination(t *testing.T) {
	_, err := NewLeonardoVideoPriceResolver().Estimate(context.Background(), LeonardoVideoPriceRequest{Model: "seedance-1.0-pro-fast", Duration: 5, Width: 864, Height: 480, Quantity: 1})
	require.ErrorIs(t, err, ErrLeonardoVideoPricingEvidenceUnavailable)
}

func TestLeonardoVideoPriceResolverExactMatrix(t *testing.T) {
	tests := []struct {
		duration, width, height int
		price                   string
	}{
		{4, 864, 480, "0.0449"}, {10, 864, 480, "0.1346"},
		{4, 1248, 704, "0.1047"}, {10, 1248, 704, "0.2841"},
		{4, 1920, 1088, "0.2691"}, {10, 1920, 1088, "0.6578"},
	}
	for _, test := range tests {
		estimate, err := NewLeonardoVideoPriceResolver().Estimate(context.Background(), LeonardoVideoPriceRequest{Model: "seedance-1.0-pro-fast", Duration: test.duration, Width: test.width, Height: test.height, Quantity: 1})
		require.NoError(t, err)
		require.Equal(t, test.price, estimate.EstimatedCostUSD.String())
	}
	for _, request := range []LeonardoVideoPriceRequest{
		{Model: "seedance-1.0-pro-fast", Duration: 4, Width: 864, Height: 480, Quantity: 2},
		{Model: "seedance-1.0-pro-fast", Duration: 4, Width: 864, Height: 480, Quantity: 1, MotionHasAudio: true},
		{Model: "seedance-1.0-pro", Duration: 5, Width: 864, Height: 480, Quantity: 1},
	} {
		_, err := NewLeonardoVideoPriceResolver().Estimate(context.Background(), request)
		require.ErrorIs(t, err, ErrLeonardoVideoPricingEvidenceUnavailable)
	}
}

func TestLeonardoVideoPriceResolverOfficialDimensions(t *testing.T) {
	tests := []struct {
		model                   string
		duration, width, height int
		price                   string
	}{
		{"seedance-1.0-pro-fast", 6, 704, 1248, "0.1645"},
		{"seedance-1.0-pro", 10, 1440, 1440, "1.6445"},
		{"motion_2.0-fast", 0, 1280, 720, "0.1047"},
		{"wan-2.7", 3, 1280, 720, "0.246725"},
		{"wan-2.7", 10, 1080, 1920, "0.8223"},
		{"kling-video-o-3", 3, 1280, 720, "1.0046"},
		{"kling-video-o-3", 15, 3840, 2160, "5.0231999999999996"},
	}
	for _, test := range tests {
		estimate, err := NewLeonardoVideoPriceResolver().Estimate(context.Background(), LeonardoVideoPriceRequest{Model: test.model, Duration: test.duration, Width: test.width, Height: test.height, Quantity: 1})
		require.NoError(t, err)
		require.Equal(t, test.price, estimate.EstimatedCostUSD.String())
	}
	for _, request := range []LeonardoVideoPriceRequest{
		{Model: "seedance-1.0-pro", Duration: 4, Width: 1280, Height: 720, Quantity: 1},
		{Model: "motion_2.0-fast", Duration: 4, Width: 832, Height: 480, Quantity: 1},
		{Model: "wan-2.7", Duration: 11, Width: 1920, Height: 1080, Quantity: 1},
	} {
		_, err := NewLeonardoVideoPriceResolver().Estimate(context.Background(), request)
		require.ErrorIs(t, err, ErrLeonardoVideoPricingEvidenceUnavailable)
	}
}

func TestLeonardoVideoSizeUsesOfficialMatrix(t *testing.T) {
	tests := []struct{ model, resolution, ratio, size string }{
		{"seedance-1.0-pro-fast", "1080p", "21:9", "2176x928"},
		{"motion_2.0-fast", "720p", "4:5", "864x1024"},
		{"wan-2.7", "1080p", "1:1", "1440x1440"},
		{"kling-video-o-3", "2160p", "9:16", "2160x3840"},
	}
	for _, test := range tests {
		size, err := LeonardoVideoSize(test.model, test.resolution, test.ratio)
		require.NoError(t, err)
		require.Equal(t, test.size, size)
	}
	_, err := LeonardoVideoSize("wan-2.7", "480p", "16:9")
	require.ErrorIs(t, err, ErrLeonardoVideoPricingEvidenceUnavailable)
}

func TestLeonardoVideoPriceResolverNewModelMinimums(t *testing.T) {
	tests := []struct {
		model                   string
		duration, width, height int
		price                   string
	}{
		{"motion_2.0-fast", 0, 832, 480, "0.1047"},
		{"seedance-1.0-pro", 4, 864, 480, "0.1346"},
		{"wan-2.7", 2, 1280, 720, "0.1645"},
		{"kling-video-o-3", 3, 1280, 720, "1.0046"},
	}
	for _, test := range tests {
		estimate, err := NewLeonardoVideoPriceResolver().Estimate(context.Background(), LeonardoVideoPriceRequest{Model: test.model, Duration: test.duration, Width: test.width, Height: test.height, Quantity: 1})
		require.NoError(t, err)
		require.Equal(t, test.price, estimate.EstimatedCostUSD.String())
	}
}

func TestLeonardoVideoGenerationParametersAreModelSpecific(t *testing.T) {
	motion, err := LeonardoVideoGenerationParameters("motion_2.0-fast", "prompt", 0, 832, 480, 1)
	require.NoError(t, err)
	require.NotContains(t, motion, "duration")
	require.Equal(t, "RESOLUTION_480", motion["mode"])

	wan, err := LeonardoVideoGenerationParameters("wan-2.7", "prompt", 2, 1280, 720, 1)
	require.NoError(t, err)
	require.Equal(t, 2, wan["duration"])
	require.Equal(t, "720p", wan["resolution"])
	require.NotContains(t, wan, "mode")

	motion720, err := LeonardoVideoGenerationParameters("motion_2.0-fast", "prompt", 0, 1280, 720, 1)
	require.NoError(t, err)
	require.Equal(t, "RESOLUTION_720", motion720["mode"])

	wan1080, err := LeonardoVideoGenerationParameters("wan-2.7", "prompt", 10, 1920, 1080, 1)
	require.NoError(t, err)
	require.Equal(t, "1080p", wan1080["resolution"])

	kling, err := LeonardoVideoGenerationParameters("kling-video-o-3", "prompt", 3, 1280, 720, 1)
	require.NoError(t, err)
	require.Equal(t, 3, kling["duration"])
	require.Equal(t, "RESOLUTION_720", kling["mode"])
	require.NotContains(t, kling, "prompt_enhance")
}

func TestLeonardoVideoPriceResolverExpandedCatalog(t *testing.T) {
	tests := []struct {
		model                   string
		duration, width, height int
		price                   string
	}{
		{"seedance-2.0-mini", 4, 864, 496, "0.3588"},
		{"seedance-2.0-fast", 15, 1280, 720, "5.4239"},
		{"seedance-2.0", 4, 3840, 2160, "11.3859"},
		{"kling-2.1", 5, 1920, 1080, "0.613"},
		{"kling-2.5", 10, 1280, 720, "0.7027"},
		{"kling-2.5-turbo-standard", 5, 1280, 720, "0.2841"},
		{"kling-2.6", 10, 1440, 1440, "1.806"},
		{"kling-3.0", 3, 1280, 720, "0.5651"},
		{"kling-3.0-turbo", 15, 1920, 1080, "3.588"},
		{"kling-video-o-1", 5, 1920, 1080, "0.755"},
	}
	for _, test := range tests {
		estimate, err := NewLeonardoVideoPriceResolver().Estimate(context.Background(), LeonardoVideoPriceRequest{Model: test.model, Duration: test.duration, Width: test.width, Height: test.height, Quantity: 1})
		require.NoError(t, err)
		require.Equal(t, test.price, estimate.EstimatedCostUSD.String())
	}
	_, err := NewLeonardoVideoPriceResolver().Estimate(context.Background(), LeonardoVideoPriceRequest{Model: "bytedance/seedance-2.5", Duration: 4, Width: 1280, Height: 720, Quantity: 1})
	require.ErrorIs(t, err, ErrLeonardoVideoPricingEvidenceUnavailable)
}

func TestLeonardoVideoExpandedParameters(t *testing.T) {
	seedance, err := LeonardoVideoGenerationParameters("seedance-2.0-mini", "prompt", 4, 864, 496, 1)
	require.NoError(t, err)
	require.NotContains(t, seedance, "mode")
	require.Equal(t, false, seedance["motion_has_audio"])

	kling, err := LeonardoVideoGenerationParameters("kling-3.0", "prompt", 3, 1280, 720, 1)
	require.NoError(t, err)
	require.Equal(t, "RESOLUTION_720", kling["mode"])
	require.NotContains(t, kling, "prompt_enhance")

	require.True(t, SupportsLeonardoVideoStartFrame("kling-2.1"))
	require.False(t, SupportsLeonardoVideoStartFrame("unknown"))
}
