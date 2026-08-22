package service

import (
	"bytes"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestBuildXAIVideoURLHandlesVersionedBaseURL(t *testing.T) {
	svc := &OpenAIGatewayService{cfg: &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}}}
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"base_url": "https://video.example/v1"}}
	got, err := svc.buildXAIVideoURL(account, "/v1/videos/generations")
	require.NoError(t, err)
	require.Equal(t, "https://video.example/v1/videos/generations", got)
}

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

func TestNormalizeXAIVideoGenerationBodyAcceptsConfiguredOpenAIVideoModels(t *testing.T) {
	models := []string{
		"kling-3.0",
		"kling-video-o-3",
		"seedance-2.0",
		"seedance-2.0-fast",
		"seedance-2.0-mini",
		"wan-2.7",
	}
	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			forwardBody, requestModel, err := NormalizeXAIVideoGenerationBodyForHandler([]byte(`{"model":"` + model + `","prompt":"city"}`))

			require.NoError(t, err)
			require.Equal(t, model, requestModel)
			require.Equal(t, model, gjson.GetBytes(forwardBody, "model").String())
		})
	}
}

func TestNormalizeXAIVideoGenerationBodyRejectsUnknownModel(t *testing.T) {
	_, _, err := NormalizeXAIVideoGenerationBodyForHandler([]byte(`{"model":"veo-3","prompt":"city"}`))

	require.ErrorContains(t, err, "OpenAI-compatible video model")
}

func TestCanvasVideoMultipartToJSONAcceptsInfiniteCanvasFields(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "minimax-h3"))
	require.NoError(t, writer.WriteField("prompt", "a dog running"))
	require.NoError(t, writer.WriteField("seconds", "5"))
	require.NoError(t, writer.WriteField("size", "720x1280"))
	require.NoError(t, writer.WriteField("resolution_name", "480p"))
	require.NoError(t, writer.WriteField("preset", "normal"))
	require.NoError(t, writer.Close())

	got, err := CanvasVideoMultipartToJSON(writer.FormDataContentType(), body.Bytes())
	require.NoError(t, err)
	require.Equal(t, "minimax-h3", gjson.GetBytes(got, "model").String())
	require.Equal(t, "a dog running", gjson.GetBytes(got, "prompt").String())
	require.Equal(t, float64(5), gjson.GetBytes(got, "seconds").Value())
	require.Equal(t, "720x1280", gjson.GetBytes(got, "size").String())
	require.Equal(t, "480p", gjson.GetBytes(got, "resolution").String())
	require.False(t, gjson.GetBytes(got, "mode").Exists())
}

func TestNormalizeVideoGenerationBodyForHandlerAcceptsCanvasMultipart(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "minimax-h3"))
	require.NoError(t, writer.WriteField("prompt", "a dog running"))
	require.NoError(t, writer.WriteField("seconds", "5"))
	require.NoError(t, writer.WriteField("size", "720x1280"))
	require.NoError(t, writer.WriteField("resolution_name", "480p"))
	require.NoError(t, writer.Close())

	forwardBody, requestModel, err := NormalizeVideoGenerationBodyForHandler(writer.FormDataContentType(), body.Bytes())
	require.NoError(t, err)
	require.Equal(t, "minimax-h3", requestModel)
	require.Equal(t, "minimax-h3", gjson.GetBytes(forwardBody, "model").String())
	require.Equal(t, float64(5), gjson.GetBytes(forwardBody, "duration").Value())
	require.False(t, gjson.GetBytes(forwardBody, "seconds").Exists())
	require.Equal(t, "9:16", gjson.GetBytes(forwardBody, "aspect_ratio").String())
	require.Equal(t, "480p", gjson.GetBytes(forwardBody, "resolution").String())
	require.False(t, gjson.GetBytes(forwardBody, "size").Exists())
	require.Equal(t, "text", gjson.GetBytes(forwardBody, "content.0.type").String())
	require.Equal(t, "a dog running", gjson.GetBytes(forwardBody, "content.0.text").String())
}

func TestNormalizeVideoGenerationBodyMapsMiniMaxH3FirstFrame(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "minimax-h3"))
	require.NoError(t, writer.WriteField("prompt", "animate this subject"))
	require.NoError(t, writer.WriteField("seconds", "5"))
	require.NoError(t, writer.WriteField("resolution_name", "480p"))
	part, err := writer.CreateFormFile("input_reference[]", "start.png")
	require.NoError(t, err)
	_, err = part.Write([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a})
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	forwardBody, requestModel, err := NormalizeVideoGenerationBodyForHandler(writer.FormDataContentType(), body.Bytes())
	require.NoError(t, err)
	require.Equal(t, "minimax-h3", requestModel)
	require.Equal(t, "first_frame", gjson.GetBytes(forwardBody, "content.1.role").String())
	require.True(t, strings.HasPrefix(gjson.GetBytes(forwardBody, "content.1.image_url.url").String(), "data:"))
	require.False(t, gjson.GetBytes(forwardBody, "image").Exists())
	require.Equal(t, "adaptive", gjson.GetBytes(forwardBody, "ratio").String())
	require.False(t, gjson.GetBytes(forwardBody, "reference_images").Exists())
}

func TestNormalizeVideoGenerationBodyMapsMiniMaxH3ReferenceImages(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "minimax-h3"))
	require.NoError(t, writer.WriteField("prompt", "person holding the drink"))
	require.NoError(t, writer.WriteField("seconds", "5"))
	require.NoError(t, writer.WriteField("resolution_name", "480p"))
	for i, name := range []string{"person.png", "drink.png"} {
		part, err := writer.CreateFormFile("input_reference[]", name)
		require.NoError(t, err)
		_, err = part.Write([]byte{0x89, 0x50, 0x4e, 0x47, byte(i), 0x0a, 0x1a, 0x0a})
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	forwardBody, requestModel, err := NormalizeVideoGenerationBodyForHandler(writer.FormDataContentType(), body.Bytes())
	require.NoError(t, err)
	require.Equal(t, "minimax-h3", requestModel)
	require.Equal(t, "reference_image", gjson.GetBytes(forwardBody, "content.1.role").String())
	require.Equal(t, "reference_image", gjson.GetBytes(forwardBody, "content.2.role").String())
	require.Equal(t, 2, int(gjson.GetBytes(forwardBody, "content.#").Int())-1)
	require.False(t, gjson.GetBytes(forwardBody, "image").Exists())
	require.False(t, gjson.GetBytes(forwardBody, "reference_images").Exists())
	require.Equal(t, "adaptive", gjson.GetBytes(forwardBody, "ratio").String())
	require.Contains(t, gjson.GetBytes(forwardBody, "prompt").String(), "Image 1")
	require.Contains(t, gjson.GetBytes(forwardBody, "prompt").String(), "Image 2")
	require.Contains(t, gjson.GetBytes(forwardBody, "content.0.text").String(), "Image 1")
}
