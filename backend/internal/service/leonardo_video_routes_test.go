package service

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
	"github.com/stretchr/testify/require"
)

func TestLeonardoVideoRoutesMatchOfficialV2Capabilities(t *testing.T) {
	tests := []struct {
		model           string
		start, end      bool
		maxReferences   int
		minimumDuration int
	}{
		{"kling-video-o-3", true, true, 7, 3},
		{"motion_2.0-fast", true, false, 0, 0},
		{"seedance-1.0-pro", true, true, 0, 4},
		{"seedance-1.0-pro-fast", true, false, 0, 4},
		{"seedance-2.0", true, true, 4, 4},
		{"seedance-2.0-fast", true, true, 4, 4},
		{"seedance-2.0-mini", true, true, 4, 4},
		{"wan-2.7", true, true, 6, 2},
	}
	for _, test := range tests {
		t.Run(test.model, func(t *testing.T) {
			route, ok := LeonardoVideoRouteFor(test.model)
			require.True(t, ok)
			require.Equal(t, test.start, route.StartFrame)
			require.Equal(t, test.end, route.EndFrame)
			require.Equal(t, test.maxReferences, route.MaxReferenceImages)
			if test.minimumDuration > 0 {
				require.Equal(t, test.minimumDuration, route.Durations[0])
			}
		})
	}
}

func TestBuildLeonardoVideoV2RequestPreservesAllGuidances(t *testing.T) {
	body, err := BuildLeonardoVideoV2Request("kling-video-o-3", "test", 3, 1280, 720, 1, false, LeonardoVideoV1References{
		StartFrame: "data:image/png;base64,start", EndFrame: "https://example.com/end.png", ReferenceImages: []string{"https://example.com/ref.png"},
	})
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(body, &decoded))
	parameters, ok := decoded["parameters"].(map[string]any)
	require.True(t, ok)
	guidances, ok := parameters["guidances"].(map[string]any)
	require.True(t, ok)
	require.Len(t, guidances["start_frame"], 1)
	require.Len(t, guidances["end_frame"], 1)
	require.Len(t, guidances["image_reference"], 1)
}

func TestBuildLeonardoVideoV2RequestRejectsMiniMax(t *testing.T) {
	// minimax-h3 is an in-house OpenAI-compatible model (account 73); it must NOT
	// be treated as a Leonardo model, so the Leonardo v2 builder must reject it.
	_, ok := LeonardoVideoRouteFor("minimax-h3")
	require.False(t, ok, "minimax-h3 must not be a Leonardo route")
	require.Equal(t, PlatformOpenAI, ExplicitCanvasVideoPlatform("minimax-h3"))
	require.Empty(t, ExplicitCanvasVideoPlatform("kling-3.0"), "provider-compatible video slugs must follow their configured account group")
	require.Empty(t, ExplicitCanvasVideoPlatform("seedance-2.0"), "provider-compatible video slugs must follow their configured account group")
	require.Empty(t, ExplicitCanvasVideoPlatform("wan-2.7"), "provider-compatible video slugs must follow their configured account group")
	_, err := BuildLeonardoVideoV2Request("minimax-h3", "test", 5, 1376, 768, 1, false, LeonardoVideoV1References{})
	require.ErrorIs(t, err, ErrLeonardoMediaCreateInputInvalid)
}

func TestBuildLeonardoVideoV2RequestRejectsUnsupportedGuidance(t *testing.T) {
	_, err := BuildLeonardoVideoV2Request("motion_2.0-fast", "test", 0, 832, 480, 1, false, LeonardoVideoV1References{EndFrame: "https://example.com/end.png"})
	require.ErrorIs(t, err, ErrLeonardoVideoParameterUnsupported)
}

func TestLeonardoVideoGenerationParametersMatchOfficialV2Schema(t *testing.T) {
	t.Run("motion_2.0-fast has no duration", func(t *testing.T) {
		p, err := LeonardoVideoGenerationParameters("motion_2.0-fast", "hello", 8, 512, 768, 1)
		require.NoError(t, err)
		_, hasDuration := p["duration"]
		require.False(t, hasDuration, "motion_2.0-fast must not emit duration")
		require.Equal(t, 512, p["width"])
		require.Equal(t, 768, p["height"])
		_, hasMode := p["mode"]
		require.False(t, hasMode, "mode is deprecated; width/height are canonical")
	})

	t.Run("seedance-1.0-pro uses duration enum", func(t *testing.T) {
		p, err := LeonardoVideoGenerationParameters("seedance-1.0-pro", "hello", 6, 0, 0, 1)
		require.NoError(t, err)
		require.Equal(t, 6, p["duration"])
		require.Equal(t, 1248, p["width"])
		require.Equal(t, 704, p["height"])
		require.Equal(t, -1, p["seed"])
		_, hasMode := p["mode"]
		require.False(t, hasMode)
	})

	t.Run("seedance-2.0 keeps motion_has_audio default true", func(t *testing.T) {
		p, err := LeonardoVideoGenerationParameters("seedance-2.0", "hello", 4, 0, 0, 1)
		require.NoError(t, err)
		require.Equal(t, 4, p["duration"])
		require.Equal(t, 1280, p["width"])
		require.Equal(t, 720, p["height"])
		_, hasAudio := p["motion_has_audio"]
		require.False(t, hasAudio, "must not force motion_has_audio:false; official default is true")
	})

	t.Run("kling-video-o-3", func(t *testing.T) {
		p, err := LeonardoVideoGenerationParameters("kling-video-o-3", "hello", 3, 0, 0, 1)
		require.NoError(t, err)
		require.Equal(t, 3, p["duration"])
		require.Equal(t, 1920, p["width"])
		require.Equal(t, 1080, p["height"])
		_, hasMode := p["mode"]
		require.False(t, hasMode)
	})

	t.Run("minimax-h3 forces audio true", func(t *testing.T) {
		p, err := LeonardoVideoGenerationParameters("minimax-h3", "hello", 5, 0, 0, 1)
		require.NoError(t, err)
		require.Equal(t, true, p["motion_has_audio"])
		require.Equal(t, 1376, p["width"])
		require.Equal(t, 768, p["height"])
		require.Equal(t, 5, p["duration"])
	})

	t.Run("rejects out-of-range duration", func(t *testing.T) {
		_, err := LeonardoVideoGenerationParameters("seedance-1.0-pro", "hello", 5, 0, 0, 1)
		require.Error(t, err, "seedance-1.0-pro only allows 4/6/8/10")
	})
}

func TestAllVerifiedLeonardoVideoModelsHaveIndependentRoutes(t *testing.T) {
	for _, model := range leonardo.ListVerifiedVideoModels() {
		t.Run(model.RequestModelSlug, func(t *testing.T) {
			route, ok := LeonardoVideoRouteFor(model.RequestModelSlug)
			require.True(t, ok, "verified Leonardo video model must have an independent route")
			require.Equal(t, model.RequestModelSlug, route.Model)
			require.NotNil(t, route.BuildParameters)
		})
	}
}

func TestKling30CanvasSquareImageRequestUsesOfficialSizeAndStartFrame(t *testing.T) {
	width, height, err := NormalizeLeonardoVideoRequestSize("kling-3.0", 768, 768)
	require.NoError(t, err)
	require.Equal(t, 960, width)
	require.Equal(t, 960, height)

	body, err := BuildLeonardoVideoV2Request("kling-3.0", "cow slowly turns its head", 3, width, height, 1, false, LeonardoVideoV1References{StartFrame: "data:image/png;base64,aGVsbG8="})
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(body, &decoded))
	parameters, ok := decoded["parameters"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(960), parameters["width"])
	require.Equal(t, float64(960), parameters["height"])
	guidances, ok := parameters["guidances"].(map[string]any)
	require.True(t, ok)
	require.Len(t, guidances["start_frame"], 1)
}
