package service

import (
	"encoding/json"
	"testing"

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
		{"minimax-h3", true, true, 5, 5},
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
	parameters := decoded["parameters"].(map[string]any)
	guidances := parameters["guidances"].(map[string]any)
	require.Len(t, guidances["start_frame"], 1)
	require.Len(t, guidances["end_frame"], 1)
	require.Len(t, guidances["image_reference"], 1)
}

func TestBuildLeonardoVideoV2RequestMapsMiniMaxAlias(t *testing.T) {
	body, err := BuildLeonardoVideoV2Request("minimax-h3", "test", 5, 1376, 768, 1, false, LeonardoVideoV1References{})
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(body, &decoded))
	require.Equal(t, "minimax-h3", decoded["model"])
	require.Equal(t, "hailuo-03", LeonardoVideoUpstreamModel("minimax-h3"))
}

func TestBuildLeonardoVideoV2RequestRejectsUnsupportedGuidance(t *testing.T) {
	_, err := BuildLeonardoVideoV2Request("motion_2.0-fast", "test", 0, 832, 480, 1, false, LeonardoVideoV1References{EndFrame: "https://example.com/end.png"})
	require.ErrorIs(t, err, ErrLeonardoVideoParameterUnsupported)
}
