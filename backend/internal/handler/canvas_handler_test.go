package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanvasModelFor(t *testing.T) {
	tests := []struct {
		model    string
		kind     string
		protocol string
		endpoint string
	}{
		{model: "gpt-image-2", kind: "image", protocol: "openai", endpoint: "/v1/images/generations"},
		{model: "grok-imagine-video", kind: "video", protocol: "openai-async", endpoint: "/v1/videos/generations"},
		{model: "leonardo-phoenix", kind: "image", protocol: "relayq-media", endpoint: "/v1/media/generations"},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			model := canvasModelFor(tt.model)
			require.Equal(t, tt.kind, model.Kind)
			require.Equal(t, tt.protocol, model.Protocol)
			require.Contains(t, model.Endpoints, tt.endpoint)
		})
	}
}
