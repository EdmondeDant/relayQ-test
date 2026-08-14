package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type leonardoMediaAccountRepoStub struct {
	service.AccountRepository
	calls int
}

type leonardoMediaGetRepoStub struct {
	reads int
}

func (s *leonardoMediaGetRepoStub) GetByPublicID(context.Context, string) (*service.GenerationJob, error) {
	s.reads++
	return nil, service.ErrGenerationJobNotFound
}

func (s *leonardoMediaGetRepoStub) CompareAndSwapPoll(context.Context, string, service.GenerationJobStatus, int, *service.GenerationJob) error {
	return nil
}

func (s *leonardoMediaAccountRepoStub) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]service.Account, error) {
	s.calls++
	return nil, nil
}

func TestLeonardoMediaHandlerStrictJSON(t *testing.T) {
	decoder := json.NewDecoder(strings.NewReader(`{"model":"flux","modality":"image","prompt":"cat","parameters":{"width":1,"height":1,"quantity":1}}`))
	decoder.DisallowUnknownFields()
	var request leonardoMediaCreateHTTPRequest
	require.NoError(t, decoder.Decode(&request))
	require.NoError(t, ensureLeonardoMediaJSONEOF(decoder))

	decoder = json.NewDecoder(strings.NewReader(`{} {}`))
	require.NoError(t, decoder.Decode(&request))
	require.Error(t, ensureLeonardoMediaJSONEOF(decoder))
}

func TestLeonardoOpenAIVideoValidatesOfficialSpecifications(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		body string
		ok   bool
	}{
		{`{"model":"motion_2.0-fast","prompt":"cat","seconds":0,"size":"832x480"}`, true},
		{`{"model":"motion_2.0-fast","prompt":"cat","seconds":0,"size":"1280x720"}`, true},
		{`{"model":"seedance-1.0-pro-fast","prompt":"cat","seconds":6,"size":"704x1248"}`, true},
		{`{"model":"seedance-1.0-pro","prompt":"cat","seconds":10,"size":"1920x1088"}`, true},
		{`{"model":"wan-2.7","prompt":"cat","seconds":3,"size":"1280x720"}`, true},
		{`{"model":"wan-2.7","prompt":"cat","seconds":10,"size":"1080x1920"}`, true},
		{`{"model":"wan-2.7","prompt":"cat","seconds":11,"size":"1280x720"}`, false},
		{`{"model":"motion_2.0-fast","prompt":"cat","seconds":4,"size":"832x480"}`, false},
	}
	for _, test := range tests {
		repo := &leonardoMediaAccountRepoStub{}
		handler := NewLeonardoMediaHandler(service.NewLeonardoMediaCreateService(repo, service.NewLeonardoImageCreateOrchestrator(nil, nil, nil, nil, nil)), nil)
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos/generations", strings.NewReader(test.body))
		c.Request.Header.Set("Idempotency-Key", "video-test")
		apiKey := leonardoMediaValidAPIKey()
		c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: apiKey.User.ID})

		handler.OpenAIVideoGenerations(c)
		if test.ok {
			require.NotEqual(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
		} else {
			require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
		}
	}
}

func TestLeonardoOpenAIVideoAcceptsCanvasMultipart(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "wan-2.7"))
	require.NoError(t, writer.WriteField("prompt", "cow walking"))
	require.NoError(t, writer.WriteField("seconds", "2"))
	require.NoError(t, writer.WriteField("size", "720x1280"))
	require.NoError(t, writer.WriteField("resolution_name", "480p"))
	require.NoError(t, writer.WriteField("preset", "normal"))
	require.NoError(t, writer.Close())

	repo := &leonardoMediaAccountRepoStub{}
	handler := NewLeonardoMediaHandler(service.NewLeonardoMediaCreateService(repo, service.NewLeonardoImageCreateOrchestrator(nil, nil, nil, nil, nil)), nil)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	apiKey := leonardoMediaValidAPIKey()
	c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: apiKey.User.ID})

	handler.OpenAIVideoGenerations(c)

	require.NotEqual(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
}

func TestLeonardoRawVideoValidatesModelParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		body string
		ok   bool
	}{
		{`{"model":"seedance-1.0-pro-fast","public":false,"parameters":{"prompt":"cat","width":1248,"height":704,"quantity":1,"duration":6,"mode":"RESOLUTION_720","prompt_enhance":"OFF"}}`, true},
		{`{"model":"motion_2.0-fast","public":false,"parameters":{"prompt":"cat","width":1280,"height":720,"quantity":1,"mode":"RESOLUTION_720"}}`, true},
		{`{"model":"wan-2.7","public":false,"parameters":{"prompt":"cat","width":1920,"height":1080,"quantity":1,"duration":5,"resolution":"1080p"}}`, true},
		{`{"model":"seedance-1.0-pro","public":false,"parameters":{"prompt":"cat","width":1248,"height":704,"quantity":1,"duration":6,"mode":"RESOLUTION_480","prompt_enhance":"OFF"}}`, false},
		{`{"model":"wan-2.7","public":false,"parameters":{"prompt":"cat","width":1920,"height":1080,"quantity":1,"duration":5,"mode":"RESOLUTION_1080"}}`, false},
	}
	for _, test := range tests {
		repo := &leonardoMediaAccountRepoStub{}
		handler := NewLeonardoMediaHandler(service.NewLeonardoMediaCreateService(repo, service.NewLeonardoImageCreateOrchestrator(nil, nil, nil, nil, nil)), nil)
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos/generations", strings.NewReader(test.body))
		c.Request.Header.Set("Idempotency-Key", "raw-video-test")
		apiKey := leonardoMediaValidAPIKey()
		c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: apiKey.User.ID})

		handler.OpenAIVideoGenerations(c)
		if test.ok {
			require.NotEqual(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
		} else {
			require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
		}
	}
}

func TestLeonardoMediaHandlerDecodesFluxGuidances(t *testing.T) {
	decoder := json.NewDecoder(strings.NewReader(`{"model":"flux-schnell","modality":"image","prompt":"cat","parameters":{"width":896,"height":896,"quantity":1,"guidances":{"content":[{"image":{"id":"content-1","type":"UPLOADED"},"strength":"HIGH"}],"style":[{"image":{"id":"style-1","type":"GENERATED"},"strength":"MAX"}]}}}`))
	decoder.DisallowUnknownFields()
	var request leonardoMediaCreateHTTPRequest
	require.NoError(t, decoder.Decode(&request))
	require.NoError(t, ensureLeonardoMediaJSONEOF(decoder))
	require.Equal(t, "content-1", request.Parameters.Guidances.Content[0].Image.ID)
	require.Equal(t, "MAX", request.Parameters.Guidances.Style[0].Strength)
}

func TestLeonardoMediaHandlerRejectsInvalidFluxGuidanceBeforeAccountSelection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &leonardoMediaAccountRepoStub{}
	handler := NewLeonardoMediaHandler(service.NewLeonardoMediaCreateService(repo, service.NewLeonardoImageCreateOrchestrator(nil, nil, nil, nil, nil)), nil)
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(nil, service.DefaultIdempotencyConfig()))
	t.Cleanup(func() { service.SetDefaultIdempotencyCoordinator(nil) })
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/media/generations", strings.NewReader(`{"model":"flux-schnell","modality":"image","prompt":"cat","parameters":{"width":896,"height":896,"quantity":1,"guidances":{"content":[{"image":{"id":"content-1","type":"UPLOADED"},"strength":"ULTRA"}]}}}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Idempotency-Key", "guidance-test")
	apiKey := leonardoMediaValidAPIKey()
	c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: apiKey.User.ID})

	handler.Create(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Zero(t, repo.calls)
}

func TestLeonardoMediaHandlerRequiresIdempotencyKeyInObserveOnlyMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &leonardoMediaAccountRepoStub{}
	orchestrator := service.NewLeonardoImageCreateOrchestrator(nil, nil, nil, nil, nil)
	handler := NewLeonardoMediaHandler(service.NewLeonardoMediaCreateService(repo, orchestrator), nil)
	cfg := service.DefaultIdempotencyConfig()
	require.True(t, cfg.ObserveOnly)
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(nil, cfg))
	t.Cleanup(func() { service.SetDefaultIdempotencyCoordinator(nil) })

	groupID := int64(7)
	user := &service.User{ID: 11}
	apiKey := &service.APIKey{ID: 13, UserID: user.ID, User: user, GroupID: &groupID, Group: &service.Group{ID: groupID, Platform: service.PlatformLeonardo}}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/media/generations", strings.NewReader(`{"model":"flux-schnell","modality":"image","prompt":"cat","public":false,"parameters":{"width":896,"height":896,"quantity":1}}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.Create(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, 0, repo.calls)
}

func TestLeonardoMediaHandlerRejectsUnverified3DBeforeAccountSelection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &leonardoMediaAccountRepoStub{}
	handler := NewLeonardoMediaHandler(service.NewLeonardoMediaCreateService(repo, service.NewLeonardoImageCreateOrchestrator(nil, nil, nil, nil, nil)), nil)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/media/generations", strings.NewReader(`{"model":"rodin-v2","modality":"3d","prompt":"statue","public":false,"parameters":{"width":1,"height":1,"quantity":1}}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Idempotency-Key", "3d-test")
	apiKey := leonardoMediaValidAPIKey()
	c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: apiKey.User.ID})

	handler.Create(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Zero(t, repo.calls)
}

func TestLeonardoOpenAIImagesValidatesRequestBeforeCreate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &leonardoMediaAccountRepoStub{}
	priceResolver := service.NewLeonardoImagePriceResolver()
	quotes := service.NewLeonardoImageQuoteGuard(priceResolver, nil)
	orchestrator := service.NewLeonardoImageCreateOrchestrator(quotes, nil, nil, nil, nil)
	handler := NewLeonardoMediaHandler(service.NewLeonardoMediaCreateService(repo, orchestrator), service.NewLeonardoMediaGetService(nil, nil))
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(nil, service.DefaultIdempotencyConfig()))
	t.Cleanup(func() { service.SetDefaultIdempotencyCoordinator(nil) })

	for _, body := range []string{
		`{"model":"flux-schnell","prompt":"cat","n":2}`,
		`{"model":"flux-schnell","prompt":"cat","size":"896x896"}`,
		`{"model":"unknown","prompt":"cat"}`,
		`{"model":"flux-schnell","prompt":"cat","unknown":true}`,
	} {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Request.Header.Set("Idempotency-Key", "images-test")
		apiKey := leonardoMediaValidAPIKey()
		c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: apiKey.User.ID})

		handler.OpenAIImagesGenerations(c)
		require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
		var response map[string]any
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
		require.Contains(t, response, "error")
		require.Zero(t, repo.calls)
	}
}

func TestLeonardoOpenAIImagesAsyncGeneratesIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &leonardoMediaAccountRepoStub{}
	quotes := service.NewLeonardoImageQuoteGuard(service.NewLeonardoImagePriceResolver(), nil)
	handler := NewLeonardoMediaHandler(service.NewLeonardoMediaCreateService(repo, service.NewLeonardoImageCreateOrchestrator(quotes, nil, nil, nil, nil)), service.NewLeonardoMediaGetService(nil, nil))
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"flux-schnell","prompt":"cat","output_format":"png","async":true}`))
	c.Request.Header.Set("Content-Type", "application/json")
	apiKey := leonardoMediaValidAPIKey()
	c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: apiKey.User.ID})

	handler.OpenAIImagesGenerations(c)
	require.NotEqual(t, http.StatusBadRequest, recorder.Code)
	require.NotContains(t, recorder.Body.String(), "idempotency_key_required")
	require.Zero(t, repo.calls)
}

func TestLeonardoOpenAIImagesIdempotencyScopesAreUserIsolated(t *testing.T) {
	require.Equal(t, "leonardo_openai_images_create:user:11", leonardoOpenAIImagesScope("create", 11))
	require.Equal(t, "leonardo_openai_images_create:user:12", leonardoOpenAIImagesScope("create", 12))
	require.NotEqual(t, leonardoOpenAIImagesScope("create", 11), leonardoOpenAIImagesScope("create", 12))
	require.NotEqual(t, leonardoOpenAIImagesScope("create", 11), leonardoOpenAIImagesScope("edit", 11))
}

func TestLeonardoOpenAIImagesAutomaticIdempotencyKey(t *testing.T) {
	body := []byte(`{"model":"flux-schnell","prompt":"cat"}`)
	now := time.Unix(1785571200, 0)
	first, automatic := leonardoOpenAIImagesIdempotencyKey("", 11, 13, "/v1/images/generations", body, now)
	require.True(t, automatic)
	second, _ := leonardoOpenAIImagesIdempotencyKey("", 11, 13, "/v1/images/generations", body, now.Add(4*time.Minute))
	require.Equal(t, first, second)
	third, _ := leonardoOpenAIImagesIdempotencyKey("", 11, 13, "/v1/images/generations", body, now.Add(5*time.Minute))
	require.NotEqual(t, first, third)
	explicit, automatic := leonardoOpenAIImagesIdempotencyKey(" client-key ", 11, 13, "/v1/images/generations", body, now)
	require.False(t, automatic)
	require.Equal(t, "client-key", explicit)
}

func TestLeonardoOpenAIImageProductSizes(t *testing.T) {
	for _, size := range []string{"1024x1024", "2048x2048", "2880x2880"} {
		width, height, ok := leonardoOpenAIImageSize(size)
		require.True(t, ok)
		require.Equal(t, width, height)
	}
	_, _, ok := leonardoOpenAIImageSize("896x896")
	require.False(t, ok)
}

func TestLeonardoOpenAIImagesTaskDecodesStoredReplay(t *testing.T) {
	task, ok := leonardoOpenAIImagesTask(map[string]any{"created": float64(123), "task_id": "gen_rq_0123456789abcdef0123456789abcdef", "status": "queued"})
	require.True(t, ok)
	require.Equal(t, int64(123), task.Created)
	require.Equal(t, "gen_rq_0123456789abcdef0123456789abcdef", task.TaskID)
}

func TestLeonardoOpenAIImageWaitErrorPreservesContentPolicyReason(t *testing.T) {
	status, errorType, message, code := leonardoOpenAIImageWaitError(
		infraerrors.New(http.StatusBadRequest, "content_policy_violation", "leonardo generation output was blocked by content policy"),
		"gen_rq_0123456789abcdef0123456789abcdef",
	)
	require.Equal(t, http.StatusBadRequest, status)
	require.Equal(t, "invalid_request_error", errorType)
	require.Equal(t, "leonardo generation output was blocked by content policy", message)
	require.Equal(t, "content_policy_violation", code)
}

func TestLeonardoOpenAIImagesEditsFailsClosedForUnverifiedModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &leonardoMediaAccountRepoStub{}
	quotes := service.NewLeonardoImageQuoteGuard(service.NewLeonardoImagePriceResolver(), nil)
	handler := NewLeonardoMediaHandler(service.NewLeonardoMediaCreateService(repo, service.NewLeonardoImageCreateOrchestrator(quotes, nil, nil, nil, nil)), nil)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "flux-schnell"))
	require.NoError(t, writer.WriteField("prompt", "edit this image"))
	part, err := createLeonardoImageEditPart(writer, "image", "image.png", "image/png")
	require.NoError(t, err)
	png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	require.NoError(t, err)
	_, err = part.Write(png)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Request.Header.Set("Idempotency-Key", "edit-test")
	apiKey := leonardoMediaValidAPIKey()
	c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: apiKey.User.ID})

	handler.OpenAIImagesEdits(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "model_not_supported")
	require.Zero(t, repo.calls)
}

func TestLeonardoOpenAIImagesEditsRejectsMask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewLeonardoMediaHandler(service.NewLeonardoMediaCreateService(&leonardoMediaAccountRepoStub{}, service.NewLeonardoImageCreateOrchestrator(nil, nil, nil, nil, nil)), nil)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "flux-schnell"))
	require.NoError(t, writer.WriteField("prompt", "edit"))
	image, err := createLeonardoImageEditPart(writer, "image", "image.png", "image/png")
	require.NoError(t, err)
	png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	require.NoError(t, err)
	_, _ = image.Write(png)
	mask, err := writer.CreateFormFile("mask", "mask.png")
	require.NoError(t, err)
	_, _ = mask.Write([]byte("mask"))
	require.NoError(t, writer.Close())
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	apiKey := leonardoMediaValidAPIKey()
	c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: apiKey.User.ID})

	handler.OpenAIImagesEdits(c)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestParseLeonardoImageEditMultipartAcceptsUpToSixImages(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image-2"))
	png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	require.NoError(t, err)
	for i := 0; i < 6; i++ {
		part, err := createLeonardoImageEditPart(writer, "image", fmt.Sprintf("image-%d.png", i), "image/png")
		require.NoError(t, err)
		_, err = part.Write(png)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	fields, images, err := parseLeonardoImageEditMultipart(multipart.NewReader(bytes.NewReader(body.Bytes()), writer.Boundary()))
	require.NoError(t, err)
	require.Equal(t, "gpt-image-2", fields["model"])
	require.Len(t, images, 6)
}

func TestParseLeonardoImageEditMultipartRejectsSevenImages(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	require.NoError(t, err)
	for i := 0; i < 7; i++ {
		part, err := createLeonardoImageEditPart(writer, "image", fmt.Sprintf("image-%d.png", i), "image/png")
		require.NoError(t, err)
		_, err = part.Write(png)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	_, _, err = parseLeonardoImageEditMultipart(multipart.NewReader(bytes.NewReader(body.Bytes()), writer.Boundary()))
	require.ErrorIs(t, err, service.ErrLeonardoImageInputInvalid)
}

func TestLeonardoMediaHandlerGetRejectsInvalidPublicID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewLeonardoMediaHandler(nil, service.NewLeonardoMediaGetService(nil, nil))
	groupID := int64(7)
	user := &service.User{ID: 11}
	apiKey := &service.APIKey{ID: 13, UserID: user.ID, User: user, GroupID: &groupID, Group: &service.Group{ID: groupID, Platform: service.PlatformLeonardo}}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/media/generations/not-valid", nil)
	c.Params = gin.Params{{Key: "id", Value: "not-valid"}}
	c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	handler.Get(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestLeonardoMediaHandlerGetShortCircuitsInvalidAuthenticationAndBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const validID = "gen_rq_0123456789abcdef0123456789abcdef"
	tests := []struct {
		name       string
		apiKey     *service.APIKey
		subject    *middleware2.AuthSubject
		wantStatus int
	}{
		{name: "missing api key", subject: &middleware2.AuthSubject{UserID: 11}, wantStatus: http.StatusUnauthorized},
		{name: "invalid api key id", apiKey: func() *service.APIKey { v := leonardoMediaValidAPIKey(); v.ID = 0; return v }(), subject: &middleware2.AuthSubject{UserID: 11}, wantStatus: http.StatusUnauthorized},
		{name: "missing subject", apiKey: leonardoMediaValidAPIKey(), wantStatus: http.StatusUnauthorized},
		{name: "invalid subject", apiKey: leonardoMediaValidAPIKey(), subject: &middleware2.AuthSubject{}, wantStatus: http.StatusUnauthorized},
		{name: "missing user", apiKey: func() *service.APIKey { v := leonardoMediaValidAPIKey(); v.User = nil; return v }(), subject: &middleware2.AuthSubject{UserID: 11}, wantStatus: http.StatusUnauthorized},
		{name: "foreign user", apiKey: leonardoMediaValidAPIKey(), subject: &middleware2.AuthSubject{UserID: 99}, wantStatus: http.StatusUnauthorized},
		{name: "missing group id", apiKey: func() *service.APIKey { v := leonardoMediaValidAPIKey(); v.GroupID = nil; return v }(), subject: &middleware2.AuthSubject{UserID: 11}, wantStatus: http.StatusBadRequest},
		{name: "missing group", apiKey: func() *service.APIKey { v := leonardoMediaValidAPIKey(); v.Group = nil; return v }(), subject: &middleware2.AuthSubject{UserID: 11}, wantStatus: http.StatusBadRequest},
		{name: "foreign group", apiKey: func() *service.APIKey { v := leonardoMediaValidAPIKey(); v.Group.ID = 99; return v }(), subject: &middleware2.AuthSubject{UserID: 11}, wantStatus: http.StatusBadRequest},
		{name: "wrong platform", apiKey: func() *service.APIKey {
			v := leonardoMediaValidAPIKey()
			v.Group.Platform = service.PlatformOpenAI
			return v
		}(), subject: &middleware2.AuthSubject{UserID: 11}, wantStatus: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &leonardoMediaGetRepoStub{}
			handler := NewLeonardoMediaHandler(nil, service.NewLeonardoMediaGetService(repository, &service.LeonardoGenerationPollOrchestrator{}))
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, "/v1/media/generations/"+validID, nil)
			c.Params = gin.Params{{Key: "id", Value: validID}}
			if test.apiKey != nil {
				c.Set(string(middleware2.ContextKeyAPIKey), test.apiKey)
			}
			if test.subject != nil {
				c.Set(string(middleware2.ContextKeyUser), *test.subject)
			}
			handler.Get(c)
			require.Equal(t, test.wantStatus, recorder.Code)
			require.Zero(t, repository.reads)
		})
	}
}

func TestLeonardoMediaHandlerGetRejectsPublicIDMatrixWithoutRepositoryCalls(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, publicID := range []string{"", " ", "gen_x_0123456789abcdef0123456789abcdef", "gen_rq_0123", "gen_rq_0123456789ABCDEF0123456789ABCDEF", "gen_rq_0123456789abcdef0123456789abcdeg"} {
		repository := &leonardoMediaGetRepoStub{}
		handler := NewLeonardoMediaHandler(nil, service.NewLeonardoMediaGetService(repository, &service.LeonardoGenerationPollOrchestrator{}))
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodGet, "/v1/media/generations/test", nil)
		c.Params = gin.Params{{Key: "id", Value: publicID}}
		apiKey := leonardoMediaValidAPIKey()
		c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: apiKey.User.ID})
		handler.Get(c)
		require.Equal(t, http.StatusBadRequest, recorder.Code, "publicID=%q", publicID)
		require.Zero(t, repository.reads)
	}
}

func TestLeonardoMediaHandlerGetRequiresConfiguredServiceWithoutCalls(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/media/generations/gen_rq_0123456789abcdef0123456789abcdef", nil)
	var handler *LeonardoMediaHandler
	handler.Get(c)
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func leonardoMediaValidAPIKey() *service.APIKey {
	groupID := int64(7)
	user := &service.User{ID: 11}
	return &service.APIKey{ID: 13, UserID: user.ID, User: user, GroupID: &groupID, Group: &service.Group{ID: groupID, Platform: service.PlatformLeonardo}}
}

func createLeonardoImageEditPart(writer *multipart.Writer, name, filename, contentType string) (io.Writer, error) {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": name, "filename": filename}))
	header.Set("Content-Type", contentType)
	return writer.CreatePart(header)
}
