package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildLeonardoVideoParameters(t *testing.T) {
	schema := map[string]json.RawMessage{
		"duration":  json.RawMessage(`{"type":"integer","enum":[5,10],"minimum":5,"maximum":10}`),
		"width":     json.RawMessage(`{"type":"integer","minimum":512,"maximum":1920}`),
		"height":    json.RawMessage(`{"type":"integer","minimum":512,"maximum":1080}`),
		"quantity":  json.RawMessage(`{"type":"integer","minimum":1,"maximum":2}`),
		"has_audio": json.RawMessage(`{"type":"boolean"}`),
	}
	binding := LeonardoVideoParameterBinding{Duration: "duration", Width: "width", Height: "height", Quantity: "quantity", MotionHasAudio: "has_audio"}
	duration, width, height, quantity, audio := 5, 1280, 720, 1, false

	parameters, err := BuildLeonardoVideoParameters(schema, binding, LeonardoVideoParameters{Duration: &duration, Width: &width, Height: &height, Quantity: &quantity, MotionHasAudio: &audio})

	require.NoError(t, err)
	require.Equal(t, map[string]any{"duration": 5, "width": 1280, "height": 720, "quantity": 1, "has_audio": false}, parameters)
}

func TestBuildLeonardoVideoParametersRejectsUnverifiedMappings(t *testing.T) {
	schema := map[string]json.RawMessage{
		"duration": json.RawMessage(`{"type":"integer","enum":[5,10]}`),
		"width":    json.RawMessage(`{"type":"integer"}`),
		"height":   json.RawMessage(`{"type":"integer"}`),
	}
	binding := LeonardoVideoParameterBinding{Duration: "duration", Width: "width", Height: "height"}
	unsupportedDuration, width, height, quantity := 6, 1280, 720, 1
	for _, input := range []LeonardoVideoParameters{
		{Duration: &unsupportedDuration},
		{Width: &width},
		{Width: &width, Height: &height, Quantity: &quantity},
	} {
		_, err := BuildLeonardoVideoParameters(schema, binding, input)
		require.ErrorIs(t, err, ErrLeonardoVideoParameterUnsupported)
	}
}

func TestRequireLeonardoVerifiedVideoModelFailsClosed(t *testing.T) {
	require.ErrorIs(t, RequireLeonardoVerifiedVideoModel("grok-imagine-1.5"), ErrLeonardoNoVerifiedVideoModels)
}
