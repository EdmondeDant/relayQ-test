package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestNormalizeXAIVideoGenerationBodyAcceptsSoraAlias(t *testing.T) {
	body := []byte(`{"model":"sora-2","prompt":"city at dusk","seconds":8,"size":"1280x720"}`)

	forwardBody, requestModel, err := NormalizeXAIVideoGenerationBodyForHandler(body)

	require.NoError(t, err)
	require.Equal(t, "grok-imagine-video", requestModel)
	require.Equal(t, "grok-imagine-video", gjson.GetBytes(forwardBody, "model").String())
	require.Equal(t, float64(8), gjson.GetBytes(forwardBody, "duration").Value())
	require.False(t, gjson.GetBytes(forwardBody, "seconds").Exists())
}

func TestNormalizeXAIVideoGenerationBodyPreservesGrokModel(t *testing.T) {
	body := []byte(`{"model":"grok-imagine-video","prompt":"city at dusk"}`)

	forwardBody, requestModel, err := NormalizeXAIVideoGenerationBodyForHandler(body)

	require.NoError(t, err)
	require.Equal(t, "grok-imagine-video", requestModel)
	require.Equal(t, "grok-imagine-video", gjson.GetBytes(forwardBody, "model").String())
}

func TestNormalizeXAIVideoGenerationBodyConvertsInputReference(t *testing.T) {
	body := []byte(`{"model":"sora-2","prompt":"animate","input_reference":{"image_url":"data:image/png;base64,abc"}}`)

	forwardBody, _, err := NormalizeXAIVideoGenerationBodyForHandler(body)

	require.NoError(t, err)
	require.Equal(t, "data:image/png;base64,abc", gjson.GetBytes(forwardBody, "image.url").String())
	require.False(t, gjson.GetBytes(forwardBody, "reference_images").Exists())
	require.False(t, gjson.GetBytes(forwardBody, "input_reference").Exists())
}

func TestNormalizeXAIVideoGenerationBodyMapsSizeToOfficialFields(t *testing.T) {
	body := []byte(`{"model":"sora-2","prompt":"city","size":"1280x720"}`)

	forwardBody, _, err := NormalizeXAIVideoGenerationBodyForHandler(body)

	require.NoError(t, err)
	require.Equal(t, "16:9", gjson.GetBytes(forwardBody, "aspect_ratio").String())
	require.Equal(t, "720p", gjson.GetBytes(forwardBody, "resolution").String())
	require.False(t, gjson.GetBytes(forwardBody, "size").Exists())
}

func TestNormalizeXAIVideoGenerationBodyConvertsReferenceImages(t *testing.T) {
	body := []byte(`{"model":"sora-2","prompt":"city","providerOptions":{"xai":{"mode":"reference-to-video","referenceImageUrls":["https://example.com/a.png"],"resolution":"HD","aspectRatio":"9:16"}}}`)

	forwardBody, _, err := NormalizeXAIVideoGenerationBodyForHandler(body)

	require.NoError(t, err)
	require.Equal(t, "https://example.com/a.png", gjson.GetBytes(forwardBody, "reference_images.0.url").String())
	require.Equal(t, "720p", gjson.GetBytes(forwardBody, "resolution").String())
	require.Equal(t, "9:16", gjson.GetBytes(forwardBody, "aspect_ratio").String())
	require.False(t, gjson.GetBytes(forwardBody, "providerOptions").Exists())
}

func TestNormalizeXAIVideoGenerationBodyRejectsUnknownModel(t *testing.T) {
	_, _, err := NormalizeXAIVideoGenerationBodyForHandler([]byte(`{"model":"veo-3","prompt":"city"}`))

	require.ErrorContains(t, err, "xAI-compatible video model")
}
