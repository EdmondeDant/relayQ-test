//go:build unit

package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
	"github.com/stretchr/testify/require"
)

type leonardoTestTransport struct {
	responses []*http.Response
	requests  []*http.Request
}

func (t *leonardoTestTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	t.requests = append(t.requests, request)
	if len(t.responses) == 0 {
		return nil, fmt.Errorf("no mocked response")
	}
	response := t.responses[0]
	t.responses = t.responses[1:]
	return response, nil
}

func TestAccountTestService_LeonardoVideoModelUsesVideoParametersAndOutput(t *testing.T) {
	ctx, recorder := newTestContext()
	transport := &leonardoTestTransport{responses: []*http.Response{
		newJSONResponse(http.StatusOK, `{"generationId":"1f193a96-c1fd-6e60-a120-72abf3262861"}`),
		newJSONResponse(http.StatusOK, `{"generations_by_pk":{"id":"1f193a96-c1fd-6e60-a120-72abf3262861","status":"COMPLETE","generated_images":[{"id":"video-1","url":"","motionMP4URL":"https://cdn.example.test/video.mp4","nsfw":false}]}}`),
	}}
	svc := &AccountTestService{cfg: &config.Config{}}
	client, err := leonardo.NewClientWithHTTPClient(leonardo.DefaultBaseURL, "test-key", leonardo.DefaultTimeout, &http.Client{Transport: transport})
	require.NoError(t, err)

	err = svc.testLeonardoPaidGeneration(ctx, client, "seedance-1.0-pro-fast", "calm lake")
	require.NoError(t, err)
	require.Len(t, transport.requests, 2)
	body, err := io.ReadAll(transport.requests[0].Body)
	require.NoError(t, err)
	var request leonardo.CreateGenerationRequest
	require.NoError(t, json.Unmarshal(body, &request))
	require.Equal(t, float64(4), request.Parameters["duration"])
	require.Equal(t, float64(864), request.Parameters["width"])
	require.Equal(t, float64(480), request.Parameters["height"])
	require.Equal(t, "RESOLUTION_480", request.Parameters["mode"])
	require.Contains(t, recorder.Body.String(), `"type":"video"`)
	require.Contains(t, recorder.Body.String(), `"video_url":"https://cdn.example.test/video.mp4"`)
	require.NotContains(t, recorder.Body.String(), `"type":"image"`)
}

func TestAccountTestService_LeonardoImageModelUsesImageParametersAndOutput(t *testing.T) {
	ctx, recorder := newTestContext()
	transport := &leonardoTestTransport{responses: []*http.Response{
		newJSONResponse(http.StatusOK, `{"generationId":"1f193a96-c1fd-6e60-a120-72abf3262861"}`),
		newJSONResponse(http.StatusOK, `{"generations_by_pk":{"id":"1f193a96-c1fd-6e60-a120-72abf3262861","status":"COMPLETE","generated_images":[{"id":"image-1","url":"https://cdn.example.test/image.png","motionMP4URL":"","nsfw":false}]}}`),
	}}
	svc := &AccountTestService{cfg: &config.Config{}}
	client, err := leonardo.NewClientWithHTTPClient(leonardo.DefaultBaseURL, "test-key", leonardo.DefaultTimeout, &http.Client{Transport: transport})
	require.NoError(t, err)

	err = svc.testLeonardoPaidGeneration(ctx, client, "kino-xl", "orange cat")
	require.NoError(t, err)
	body, err := io.ReadAll(transport.requests[0].Body)
	require.NoError(t, err)
	var request leonardo.CreateGenerationRequest
	require.NoError(t, json.Unmarshal(body, &request))
	require.True(t, strings.Contains(string(body), `"width":896`))
	require.Equal(t, "FAST", request.Parameters["mode"])
	require.Equal(t, "OFF", request.Parameters["prompt_enhance"])
	require.Contains(t, recorder.Body.String(), `"type":"image"`)
	require.NotContains(t, recorder.Body.String(), `"type":"video"`)
}

func TestAccountTestService_LeonardoGraphicDesignUsesVerifiedParameters(t *testing.T) {
	ctx, _ := newTestContext()
	transport := &leonardoTestTransport{responses: []*http.Response{
		newJSONResponse(http.StatusOK, `{"generate":{"generationId":"1f193a96-c1fd-6e60-a120-72abf3262861"}}`),
		newJSONResponse(http.StatusOK, `{"generations_by_pk":{"id":"1f193a96-c1fd-6e60-a120-72abf3262861","status":"COMPLETE","generated_images":[{"id":"image-1","url":"https://cdn.example.test/image.png","nsfw":false}]}}`),
	}}
	svc := &AccountTestService{cfg: &config.Config{}}
	client, err := leonardo.NewClientWithHTTPClient(leonardo.DefaultBaseURL, "test-key", leonardo.DefaultTimeout, &http.Client{Transport: transport})
	require.NoError(t, err)

	require.NoError(t, svc.testLeonardoPaidGeneration(ctx, client, "graphic-design", "orange cat"))
	body, err := io.ReadAll(transport.requests[0].Body)
	require.NoError(t, err)
	var request leonardo.CreateGenerationRequest
	require.NoError(t, json.Unmarshal(body, &request))
	require.Equal(t, float64(888), request.Parameters["width"])
	require.Equal(t, float64(888), request.Parameters["height"])
	require.Equal(t, "FAST", request.Parameters["mode"])
	require.Equal(t, "OFF", request.Parameters["prompt_enhance"])
}
