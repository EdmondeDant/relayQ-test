package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type LeonardoMediaHandler struct {
	create *service.LeonardoMediaCreateService
	get    *service.LeonardoMediaGetService
}

type leonardoMediaCreateHTTPRequest struct {
	Model      string                       `json:"model"`
	Modality   string                       `json:"modality"`
	Prompt     string                       `json:"prompt"`
	Public     bool                         `json:"public"`
	Parameters leonardoMediaImageParameters `json:"parameters"`
}

type leonardoMediaImageParameters struct {
	Width     int                           `json:"width"`
	Height    int                           `json:"height"`
	Quantity  int                           `json:"quantity"`
	Guidances service.LeonardoFluxGuidances `json:"guidances,omitempty"`
}

type leonardoRawGenerationProbe struct {
	Model      string `json:"model"`
	Public     bool   `json:"public"`
	Prompt     string `json:"prompt"`
	Size       string `json:"size"`
	N          int    `json:"n"`
	Quality    string `json:"quality"`
	Parameters *struct {
		Prompt   string `json:"prompt"`
		Width    int    `json:"width"`
		Height   int    `json:"height"`
		Quantity int    `json:"quantity"`
		Quality  string `json:"quality"`
	} `json:"parameters"`
}

type leonardoOpenAIImagesRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n,omitempty"`
	Size           string `json:"size,omitempty"`
	Quality        string `json:"quality,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	Async          bool   `json:"async,omitempty"`
}

type leonardoOpenAIImageData struct {
	URL     string `json:"url,omitempty"`
	B64JSON string `json:"b64_json,omitempty"`
}

type leonardoOpenAIImagesResult struct {
	Created int64                     `json:"created"`
	Data    []leonardoOpenAIImageData `json:"data,omitempty"`
	TaskID  string                    `json:"task_id,omitempty"`
	Status  string                    `json:"status,omitempty"`
}

type leonardoOpenAIImageEditFingerprint struct {
	Model        string   `json:"model"`
	Prompt       string   `json:"prompt"`
	Size         string   `json:"size"`
	N            int      `json:"n"`
	ImageSHA256s []string `json:"image_sha256s"`
}

func NewLeonardoMediaHandler(create *service.LeonardoMediaCreateService, get *service.LeonardoMediaGetService) *LeonardoMediaHandler {
	return &LeonardoMediaHandler{create: create, get: get}
}

func (h *LeonardoMediaHandler) Get(c *gin.Context) {
	if h == nil || h.get == nil {
		response.ErrorFrom(c, service.ErrLeonardoMediaGetNotConfigured)
		return
	}
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil || apiKey.ID <= 0 {
		response.Unauthorized(c, "Invalid API key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 || apiKey.User == nil || apiKey.User.ID != subject.UserID {
		response.Unauthorized(c, "Invalid authentication context")
		return
	}
	if apiKey.GroupID == nil || apiKey.Group == nil || apiKey.Group.ID != *apiKey.GroupID || apiKey.Group.Platform != service.PlatformLeonardo {
		response.BadRequest(c, "Invalid Leonardo group binding")
		return
	}
	publicID := strings.TrimSpace(c.Param("id"))
	if !service.ValidLeonardoMediaPublicID(publicID) {
		response.ErrorFrom(c, service.ErrLeonardoMediaGetInputInvalid)
		return
	}
	result, err := h.get.Get(c.Request.Context(), service.LeonardoMediaGetInput{PublicID: publicID, UserID: subject.UserID, APIKeyID: apiKey.ID, GroupID: *apiKey.GroupID})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.JSON(200, result)
}

func (h *LeonardoMediaHandler) Content(c *gin.Context) {
	if h == nil || h.get == nil {
		response.ErrorFrom(c, service.ErrLeonardoMediaGetNotConfigured)
		return
	}
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil || apiKey.ID <= 0 {
		response.Unauthorized(c, "Invalid API key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 || apiKey.User == nil || apiKey.User.ID != subject.UserID {
		response.Unauthorized(c, "Invalid authentication context")
		return
	}
	if apiKey.GroupID == nil || apiKey.Group == nil || apiKey.Group.ID != *apiKey.GroupID || apiKey.Group.Platform != service.PlatformLeonardo {
		response.BadRequest(c, "Invalid Leonardo group binding")
		return
	}
	publicID := strings.TrimSpace(c.Param("id"))
	index := 0
	var err error
	if raw := strings.TrimSpace(c.Query("index")); raw != "" {
		index, err = strconv.Atoi(raw)
	}
	if !service.ValidLeonardoMediaPublicID(publicID) || err != nil || index < 0 {
		response.ErrorFrom(c, service.ErrLeonardoMediaContentInvalid)
		return
	}
	content, err := h.get.Content(c.Request.Context(), service.LeonardoMediaContentInput{LeonardoMediaGetInput: service.LeonardoMediaGetInput{PublicID: publicID, UserID: subject.UserID, APIKeyID: apiKey.ID, GroupID: *apiKey.GroupID}, Index: index})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	defer func() { _ = content.Close() }()
	c.Header("Content-Type", content.ContentType)
	c.Header("Content-Disposition", "inline")
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	http.ServeContent(c.Writer, c.Request, publicID, time.Time{}, content.File)
}

func (h *LeonardoMediaHandler) OpenAIImagesGenerations(c *gin.Context) {
	if h == nil || h.create == nil || h.get == nil {
		leonardoOpenAIError(c, http.StatusInternalServerError, "api_error", "Leonardo images service is not configured", "service_not_configured")
		return
	}
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil || apiKey.ID <= 0 {
		leonardoOpenAIError(c, http.StatusUnauthorized, "authentication_error", "Invalid API key", "invalid_api_key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 || apiKey.User == nil || apiKey.User.ID != subject.UserID {
		leonardoOpenAIError(c, http.StatusUnauthorized, "authentication_error", "Invalid authentication context", "invalid_authentication_context")
		return
	}
	if apiKey.GroupID == nil || apiKey.Group == nil || apiKey.Group.ID != *apiKey.GroupID || apiKey.Group.Platform != service.PlatformLeonardo {
		leonardoOpenAIError(c, http.StatusForbidden, "permission_error", "Invalid Leonardo group binding", "invalid_group_binding")
		return
	}
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		leonardoOpenAIError(c, http.StatusBadRequest, "invalid_request_error", "Content-Type must be application/json", "invalid_content_type")
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		leonardoOpenAIError(c, http.StatusBadRequest, "invalid_request_error", "Invalid images request", "invalid_request")
		return
	}
	if probe, raw, probeErr := detectLeonardoRawGeneration(body); probeErr != nil {
		leonardoOpenAIError(c, http.StatusBadRequest, "invalid_request_error", "Invalid images request", "invalid_request")
		return
	} else if raw {
		verified, ok := leonardo.ResolveByRequestModelSlug(strings.TrimSpace(probe.Model))
		if !ok || verified.Modality != leonardo.ModelModalityImage {
			leonardoOpenAIError(c, http.StatusBadRequest, "invalid_request_error", "Model is not supported", "model_not_supported")
			return
		}
		qualityTier := strings.ToLower(strings.TrimSpace(probe.Parameters.Quality))
		if qualityTier == "" {
			qualityTier = "low"
			if probe.Model == "gpt-image-2" {
				qualityTier = "medium"
			}
		}
		key, automaticKey := leonardoOpenAIImagesIdempotencyKey(c.GetHeader("Idempotency-Key"), subject.UserID, apiKey.ID, c.FullPath(), body, time.Now())
		coordinator := service.DefaultIdempotencyCoordinator()
		if coordinator == nil {
			leonardoOpenAIError(c, http.StatusServiceUnavailable, "api_error", "Idempotency service is unavailable", "idempotency_unavailable")
			return
		}
		executed, createErr := coordinator.Execute(c.Request.Context(), service.IdempotencyExecuteOptions{Scope: leonardoOpenAIImagesScope("create", subject.UserID) + map[bool]string{true: ":automatic", false: ":explicit"}[automaticKey], ActorScope: "user:" + strconv.FormatInt(subject.UserID, 10), Method: c.Request.Method, Route: c.FullPath(), IdempotencyKey: key, Payload: json.RawMessage(body), RequireKey: true, TTL: map[bool]time.Duration{true: 20 * time.Minute, false: service.DefaultWriteIdempotencyTTL()}[automaticKey]}, func(ctx context.Context) (any, error) {
			return h.create.Create(ctx, service.LeonardoMediaCreateInput{IdempotencyKey: key, UserID: subject.UserID, APIKeyID: apiKey.ID, GroupID: *apiKey.GroupID, Model: strings.TrimSpace(probe.Model), Prompt: strings.TrimSpace(probe.Parameters.Prompt), Public: probe.Public, Width: probe.Parameters.Width, Height: probe.Parameters.Height, Quantity: probe.Parameters.Quantity, QualityTier: qualityTier, RawBody: body})
		})
		if createErr != nil {
			leonardoOpenAIError(c, http.StatusBadGateway, "api_error", "Leonardo image generation failed", "upstream_error")
			return
		}
		if executed.Replayed {
			c.Header("X-Idempotency-Replayed", "true")
		}
		result, ok := executed.Data.(map[string]any)
		if ok {
			c.JSON(http.StatusAccepted, result)
			return
		}
		c.JSON(http.StatusAccepted, executed.Data)
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	var req leonardoOpenAIImagesRequest
	if err = decoder.Decode(&req); err != nil || ensureLeonardoMediaJSONEOF(decoder) != nil {
		leonardoOpenAIError(c, http.StatusBadRequest, "invalid_request_error", "Invalid images request", "invalid_request")
		return
	}
	req.Model, req.Prompt = strings.TrimSpace(req.Model), strings.TrimSpace(req.Prompt)
	if req.N == 0 {
		req.N = 1
	}
	if req.Size == "" {
		req.Size = "1024x1024"
	}
	if req.Quality == "" {
		req.Quality = "low"
	}
	if req.ResponseFormat == "" {
		req.ResponseFormat = "b64_json"
	}
	width, height, sizeOK := leonardoOpenAIImageSize(req.Size)
	verified, modelOK := leonardo.ResolveByRequestModelSlug(req.Model)
	if req.Model == "" || req.Prompt == "" || len(req.Prompt) > 4000 || !modelOK || verified.ImageCapabilities == nil || req.N < 1 || req.N > verified.ImageCapabilities.MaxQuantity || !sizeOK || (req.Quality != "low" && req.Quality != "medium" && req.Quality != "high") || (req.ResponseFormat != "url" && req.ResponseFormat != "b64_json") {
		leonardoOpenAIError(c, http.StatusBadRequest, "invalid_request_error", "Unsupported Leonardo images request", "invalid_request")
		return
	}
	key, automaticKey := leonardoOpenAIImagesIdempotencyKey(c.GetHeader("Idempotency-Key"), subject.UserID, apiKey.ID, c.FullPath(), body, time.Now())
	if _, err := h.create.EstimateQualityQuote(c.Request.Context(), req.Model, width, height, req.N, req.Quality); err != nil {
		leonardoOpenAIError(c, http.StatusBadRequest, "invalid_request_error", "Model or image parameters are not supported", "model_not_supported")
		return
	}
	coordinator := service.DefaultIdempotencyCoordinator()
	if coordinator == nil {
		leonardoOpenAIError(c, http.StatusServiceUnavailable, "api_error", "Idempotency service is unavailable", "idempotency_unavailable")
		return
	}
	result, err := coordinator.Execute(c.Request.Context(), service.IdempotencyExecuteOptions{
		Scope: leonardoOpenAIImagesScope("create", subject.UserID) + map[bool]string{true: ":automatic", false: ":explicit"}[automaticKey], ActorScope: "user:" + strconv.FormatInt(subject.UserID, 10), Method: c.Request.Method,
		Route: c.FullPath(), IdempotencyKey: key, Payload: req, RequireKey: true, TTL: map[bool]time.Duration{true: 20 * time.Minute, false: service.DefaultWriteIdempotencyTTL()}[automaticKey],
	}, func(ctx context.Context) (any, error) {
		created, createErr := h.create.Create(ctx, service.LeonardoMediaCreateInput{IdempotencyKey: key, UserID: subject.UserID, APIKeyID: apiKey.ID, GroupID: *apiKey.GroupID, Model: req.Model, Prompt: req.Prompt, Width: width, Height: height, Quantity: req.N, QualityTier: req.Quality})
		if createErr != nil {
			return nil, createErr
		}
		return &leonardoOpenAIImagesResult{Created: created.CreatedAt, TaskID: created.ID, Status: created.Status}, nil
	})
	if err != nil {
		status := http.StatusBadGateway
		code := "upstream_error"
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			status, code = http.StatusGatewayTimeout, "generation_timeout"
		}
		leonardoOpenAIError(c, status, "api_error", "Leonardo image generation failed", code)
		return
	}
	if result.Replayed {
		c.Header("X-Idempotency-Replayed", "true")
	}
	task, taskOK := leonardoOpenAIImagesTask(result.Data)
	if taskOK {
		c.Header("X-RelayQ-Task-ID", task.TaskID)
	}
	if !req.Async {
		if !taskOK {
			leonardoOpenAIError(c, http.StatusInternalServerError, "api_error", "Stored Leonardo task response is invalid", "idempotency_response_invalid")
			return
		}
		waitCtx, cancel := context.WithTimeout(c.Request.Context(), 900*time.Second)
		result.Data, err = h.waitLeonardoOpenAIImages(waitCtx, service.LeonardoMediaGetInput{PublicID: task.TaskID, UserID: subject.UserID, APIKeyID: apiKey.ID, GroupID: *apiKey.GroupID}, task.Created, req.ResponseFormat)
		cancel()
		if err != nil {
			status, code := http.StatusBadGateway, "upstream_error"
			message := "Leonardo image generation failed"
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				status, code = http.StatusGatewayTimeout, "generation_timeout"
				message = "Leonardo image generation is still running; query task " + task.TaskID
			}
			leonardoOpenAIError(c, status, "api_error", message, code)
			return
		}
	}
	status := http.StatusOK
	if req.Async {
		status = http.StatusAccepted
	}
	c.JSON(status, result.Data)
}

func leonardoOpenAIImageSize(size string) (int, int, bool) {
	switch strings.TrimSpace(size) {
	case "1024x1024":
		return 1024, 1024, true
	case "2048x2048":
		return 2048, 2048, true
	case "2880x2880":
		return 2880, 2880, true
	default:
		return 0, 0, false
	}
}

func leonardoOpenAIImagesIdempotencyKey(explicit string, userID, apiKeyID int64, route string, body []byte, now time.Time) (string, bool) {
	if key := strings.TrimSpace(explicit); key != "" {
		return key, false
	}
	bodyHash := sha256.Sum256(body)
	bucket := now.Unix() / int64((5 * time.Minute).Seconds())
	keyHash := sha256.Sum256([]byte(strconv.FormatInt(userID, 10) + "\n" + strconv.FormatInt(apiKeyID, 10) + "\n" + route + "\n" + hex.EncodeToString(bodyHash[:]) + "\n" + strconv.FormatInt(bucket, 10)))
	return "auto-" + hex.EncodeToString(keyHash[:]), true
}

func detectLeonardoRawGeneration(body []byte) (leonardoRawGenerationProbe, bool, error) {
	var probe leonardoRawGenerationProbe
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&probe); err != nil || ensureLeonardoMediaJSONEOF(decoder) != nil {
		return probe, false, service.ErrLeonardoMediaCreateInputInvalid
	}
	if probe.Parameters == nil {
		return probe, false, nil
	}
	if strings.TrimSpace(probe.Prompt) != "" || strings.TrimSpace(probe.Size) != "" || probe.N != 0 || strings.TrimSpace(probe.Quality) != "" {
		return probe, false, service.ErrLeonardoMediaCreateInputInvalid
	}
	model := strings.TrimSpace(probe.Model)
	matchReferenceSize := (model == "nano-banana-2" || model == "nano-banana-2-lite") && probe.Parameters.Width == 0 && probe.Parameters.Height == 0
	if model == "" || strings.TrimSpace(probe.Parameters.Prompt) == "" || (!matchReferenceSize && (probe.Parameters.Width <= 0 || probe.Parameters.Height <= 0)) || probe.Parameters.Quantity <= 0 {
		return probe, false, service.ErrLeonardoMediaCreateInputInvalid
	}
	return probe, true, nil
}

func (h *LeonardoMediaHandler) OpenAIImagesEdits(c *gin.Context) {
	if h == nil || h.create == nil {
		leonardoOpenAIError(c, http.StatusInternalServerError, "api_error", "Leonardo images service is not configured", "service_not_configured")
		return
	}
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil || apiKey.ID <= 0 {
		leonardoOpenAIError(c, http.StatusUnauthorized, "authentication_error", "Invalid API key", "invalid_api_key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 || apiKey.User == nil || apiKey.User.ID != subject.UserID {
		leonardoOpenAIError(c, http.StatusUnauthorized, "authentication_error", "Invalid authentication context", "invalid_authentication_context")
		return
	}
	if apiKey.GroupID == nil || apiKey.Group == nil || apiKey.Group.ID != *apiKey.GroupID || apiKey.Group.Platform != service.PlatformLeonardo {
		leonardoOpenAIError(c, http.StatusForbidden, "permission_error", "Invalid Leonardo group binding", "invalid_group_binding")
		return
	}
	mediaType, parameters, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" || parameters["boundary"] == "" {
		leonardoOpenAIError(c, http.StatusBadRequest, "invalid_request_error", "Content-Type must be multipart/form-data", "invalid_content_type")
		return
	}
	fields, images, err := parseLeonardoImageEditMultipart(multipart.NewReader(c.Request.Body, parameters["boundary"]))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrLeonardoImageInputTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		leonardoOpenAIError(c, status, "invalid_request_error", "Invalid Leonardo image edit request", "invalid_request")
		return
	}
	model, prompt, size, quality := strings.TrimSpace(fields["model"]), strings.TrimSpace(fields["prompt"]), strings.TrimSpace(fields["size"]), strings.TrimSpace(fields["quality"])
	if size == "" {
		size = "1024x1024"
	}
	if quality == "" {
		quality = "low"
	}
	width, height, sizeOK := leonardoOpenAIImageSize(size)
	n := 1
	if fields["n"] != "" {
		n, err = strconv.Atoi(fields["n"])
	}
	if model == "" || prompt == "" || len(prompt) > 4000 || !sizeOK || n < 1 || n > 8 || (quality != "low" && quality != "medium" && quality != "high") || fields["mask"] != "" {
		leonardoOpenAIError(c, http.StatusBadRequest, "invalid_request_error", "Unsupported Leonardo image edit request", "invalid_request")
		return
	}
	capability := service.LeonardoImageReferenceCapabilityForModel(model)
	if capability == nil {
		leonardoOpenAIError(c, http.StatusBadRequest, "invalid_request_error", "Leonardo image edits are not verified for this model", "model_not_supported")
		return
	}
	if len(images) > capability.MaxItems {
		leonardoOpenAIError(c, http.StatusBadRequest, "invalid_request_error", "Too many reference images for this model", "invalid_request")
		return
	}
	imageHashes := make([]string, len(images))
	for i := range images {
		imageHashes[i] = service.LeonardoImageSHA256(images[i].Data)
	}
	key, automaticKey := leonardoOpenAIImagesIdempotencyKey(c.GetHeader("Idempotency-Key"), subject.UserID, apiKey.ID, c.FullPath(), []byte(model+"\n"+prompt+"\n"+size+"\n"+quality+"\n"+strconv.Itoa(n)+"\n"+strings.Join(imageHashes, "\n")), time.Now())
	if _, err := h.create.EstimateQualityQuote(c.Request.Context(), model, width, height, n, quality); err != nil {
		leonardoOpenAIError(c, http.StatusBadRequest, "invalid_request_error", "Model or image parameters are not supported", "model_not_supported")
		return
	}
	coordinator := service.DefaultIdempotencyCoordinator()
	if coordinator == nil {
		leonardoOpenAIError(c, http.StatusServiceUnavailable, "api_error", "Idempotency service is unavailable", "idempotency_unavailable")
		return
	}
	result, err := coordinator.Execute(c.Request.Context(), service.IdempotencyExecuteOptions{
		Scope: leonardoOpenAIImagesScope("edit", subject.UserID) + map[bool]string{true: ":automatic", false: ":explicit"}[automaticKey], ActorScope: "user:" + strconv.FormatInt(subject.UserID, 10), Method: c.Request.Method,
		Route: c.FullPath(), IdempotencyKey: key, Payload: leonardoOpenAIImageEditFingerprint{Model: model, Prompt: prompt, Size: size, N: n, ImageSHA256s: imageHashes}, RequireKey: true, TTL: map[bool]time.Duration{true: 20 * time.Minute, false: service.DefaultWriteIdempotencyTTL()}[automaticKey],
	}, func(ctx context.Context) (any, error) {
		created, createErr := h.create.Create(ctx, service.LeonardoMediaCreateInput{IdempotencyKey: key, UserID: subject.UserID, APIKeyID: apiKey.ID, GroupID: *apiKey.GroupID, Model: model, Prompt: prompt, Width: width, Height: height, Quantity: n, QualityTier: quality, InputImages: images, ImageCapability: capability})
		if createErr != nil {
			return nil, createErr
		}
		return &leonardoOpenAIImagesResult{Created: created.CreatedAt, TaskID: created.ID, Status: created.Status}, nil
	})
	if err != nil {
		leonardoOpenAIError(c, http.StatusBadGateway, "api_error", "Leonardo image edit failed", "upstream_error")
		return
	}
	if result.Replayed {
		c.Header("X-Idempotency-Replayed", "true")
	}
	c.JSON(http.StatusAccepted, result.Data)
}

func leonardoOpenAIImagesTask(value any) (*leonardoOpenAIImagesResult, bool) {
	if result, ok := value.(*leonardoOpenAIImagesResult); ok && result != nil && result.TaskID != "" {
		return result, true
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	var result leonardoOpenAIImagesResult
	if json.Unmarshal(encoded, &result) != nil || result.TaskID == "" {
		return nil, false
	}
	return &result, true
}

func leonardoOpenAIImagesScope(operation string, userID int64) string {
	return "leonardo_openai_images_" + operation + ":user:" + strconv.FormatInt(userID, 10)
}

func parseLeonardoImageEditMultipart(reader *multipart.Reader) (map[string]string, []*service.LeonardoImageInput, error) {
	allowedFields := map[string]struct{}{"model": {}, "prompt": {}, "size": {}, "quality": {}, "n": {}, "mask": {}}
	fields := map[string]string{}
	images := make([]*service.LeonardoImageInput, 0, 6)
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		name := part.FormName()
		if name != "image" {
			if _, allowed := allowedFields[name]; !allowed {
				return nil, nil, service.ErrLeonardoImageInputInvalid
			}
		}
		if name == "image" {
			if len(images) >= 6 || part.FileName() == "" {
				return nil, nil, service.ErrLeonardoImageInputInvalid
			}
			image, imageErr := service.ReadLeonardoMultipartImage(part, part.FileName(), part.Header.Get("Content-Type"), 20<<20)
			if imageErr != nil {
				return nil, nil, imageErr
			}
			images = append(images, image)
			continue
		}
		if name == "mask" {
			if _, exists := fields[name]; exists {
				return nil, nil, service.ErrLeonardoImageInputInvalid
			}
			fields["mask"] = "present"
			continue
		}
		if _, exists := fields[name]; exists {
			return nil, nil, service.ErrLeonardoImageInputInvalid
		}
		value, err := io.ReadAll(io.LimitReader(part, 4097))
		if err != nil || len(value) > 4096 {
			return nil, nil, service.ErrLeonardoImageInputInvalid
		}
		fields[name] = string(value)
	}
	if len(images) == 0 {
		return nil, nil, service.ErrLeonardoImageInputInvalid
	}
	return fields, images, nil
}

func (h *LeonardoMediaHandler) waitLeonardoOpenAIImages(ctx context.Context, input service.LeonardoMediaGetInput, createdAt int64, responseFormat string) (*leonardoOpenAIImagesResult, error) {
	for {
		result, err := h.get.Get(ctx, input)
		if err != nil {
			return nil, err
		}
		switch result.Status {
		case string(service.GenerationJobStatusSucceeded):
			out := &leonardoOpenAIImagesResult{Created: createdAt, Data: make([]leonardoOpenAIImageData, len(result.Data))}
			for i := range result.Data {
				if responseFormat == "b64_json" {
					encoded, encodeErr := h.get.ContentBase64(ctx, service.LeonardoMediaContentInput{LeonardoMediaGetInput: input, Index: i})
					if encodeErr != nil {
						return nil, encodeErr
					}
					out.Data[i].B64JSON = encoded
				} else {
					out.Data[i].URL = result.Data[i].URL
				}
			}
			return out, nil
		case string(service.GenerationJobStatusFailed), string(service.GenerationJobStatusUnknown):
			return nil, infraerrors.New(http.StatusBadGateway, "LEONARDO_GENERATION_FAILED", "Leonardo image generation failed")
		}
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func leonardoOpenAIError(c *gin.Context, status int, errorType, message, code string) {
	c.JSON(status, gin.H{"error": gin.H{"message": message, "type": errorType, "param": nil, "code": code}})
}

func (h *LeonardoMediaHandler) Create(c *gin.Context) {
	if h == nil || h.create == nil {
		response.ErrorFrom(c, service.ErrLeonardoMediaCreateNotConfigured)
		return
	}
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil || apiKey.ID <= 0 {
		response.Unauthorized(c, "Invalid API key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 || apiKey.User == nil || apiKey.User.ID != subject.UserID {
		response.Unauthorized(c, "Invalid authentication context")
		return
	}
	if apiKey.GroupID == nil || apiKey.Group == nil || apiKey.Group.ID != *apiKey.GroupID || apiKey.Group.Platform != service.PlatformLeonardo {
		response.BadRequest(c, "Invalid Leonardo group binding")
		return
	}
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		response.ErrorFrom(c, service.ErrLeonardoMediaCreateInputInvalid)
		return
	}
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	var req leonardoMediaCreateHTTPRequest
	if err = decoder.Decode(&req); err != nil {
		response.ErrorFrom(c, service.ErrLeonardoMediaCreateInputInvalid)
		return
	}
	if err = ensureLeonardoMediaJSONEOF(decoder); err != nil {
		response.ErrorFrom(c, service.ErrLeonardoMediaCreateInputInvalid)
		return
	}
	if req.Modality != "image" || strings.TrimSpace(req.Model) == "" || strings.TrimSpace(req.Prompt) == "" || len(strings.TrimSpace(req.Prompt)) > 4000 || req.Parameters.Width <= 0 || req.Parameters.Height <= 0 || req.Parameters.Quantity <= 0 {
		response.ErrorFrom(c, service.ErrLeonardoMediaCreateInputInvalid)
		return
	}
	if _, err = service.BuildLeonardoFluxGuidances(req.Model, req.Parameters.Guidances); err != nil {
		response.ErrorFrom(c, service.ErrLeonardoMediaCreateInputInvalid)
		return
	}
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		response.ErrorFrom(c, service.ErrIdempotencyKeyRequired)
		return
	}
	if service.DefaultIdempotencyCoordinator() == nil {
		response.ErrorFrom(c, service.ErrIdempotencyStoreUnavail)
		return
	}
	input := service.LeonardoMediaCreateInput{IdempotencyKey: idempotencyKey, UserID: subject.UserID, APIKeyID: apiKey.ID, GroupID: *apiKey.GroupID, Model: req.Model, Prompt: req.Prompt, Public: req.Public, Width: req.Parameters.Width, Height: req.Parameters.Height, Quantity: req.Parameters.Quantity, FluxGuidances: req.Parameters.Guidances}
	executeUserIdempotentJSON(c, "leonardo_media_create", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) { return h.create.Create(ctx, input) })
}

func ensureLeonardoMediaJSONEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if err == io.EOF {
		return nil
	}
	if err == nil {
		return service.ErrLeonardoMediaCreateInputInvalid
	}
	return err
}
