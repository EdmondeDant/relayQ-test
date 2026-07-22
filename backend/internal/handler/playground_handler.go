package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PlaygroundHandler struct {
	service *service.PlaygroundService
	storage *service.PlaygroundAssetStorage
}

func reportPlayground500HandlerDebugEvent(hypothesisID, location, msg string, data map[string]any) {
	apiURL := "http://127.0.0.1:7777/event"
	sessionID := "playground-500-blockers"
	if envBytes, err := os.ReadFile(".dbg/playground-500-blockers.env"); err == nil {
		for _, line := range strings.Split(string(envBytes), "\n") {
			if strings.HasPrefix(line, "DEBUG_SERVER_URL=") {
				apiURL = strings.TrimSpace(strings.TrimPrefix(line, "DEBUG_SERVER_URL="))
			} else if strings.HasPrefix(line, "DEBUG_SESSION_ID=") {
				sessionID = strings.TrimSpace(strings.TrimPrefix(line, "DEBUG_SESSION_ID="))
			}
		}
	}
	body, err := json.Marshal(map[string]any{
		"sessionId":    sessionID,
		"runId":        "pre-fix",
		"hypothesisId": hypothesisID,
		"location":     location,
		"msg":          msg,
		"data":         data,
		"ts":           time.Now().UnixMilli(),
	})
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err == nil && resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
}

func NewPlaygroundHandler(s *service.PlaygroundService) *PlaygroundHandler {
	return &PlaygroundHandler{service: s, storage: service.NewPlaygroundAssetStorage()}
}

type createPlaygroundTaskRequest struct {
	Kind           string          `json:"kind" binding:"required"`
	Status         string          `json:"status"`
	Model          string          `json:"model"`
	RequestID      string          `json:"request_id"`
	RequestPayload json.RawMessage `json:"request_payload"`
	ResultPayload  json.RawMessage `json:"result_payload"`
	ErrorMessage   string          `json:"error_message"`
}
type submitPlaygroundJobRequest struct {
	Kind           string          `json:"kind" binding:"required"`
	Model          string          `json:"model" binding:"required"`
	APIKey         string          `json:"api_key" binding:"required"`
	RequestPayload json.RawMessage `json:"request_payload" binding:"required"`
}
type createPlaygroundAssetRequest struct {
	TaskID      *int64          `json:"task_id"`
	Kind        string          `json:"kind" binding:"required"`
	Title       string          `json:"title"`
	Content     string          `json:"content"`
	URL         string          `json:"url"`
	StorageKey  string          `json:"storage_key"`
	ContentType string          `json:"content_type"`
	ByteSize    *int64          `json:"byte_size"`
	Metadata    json.RawMessage `json:"metadata"`
}

func (h *PlaygroundHandler) SubmitJob(c *gin.Context) {
	subject, ok := playgroundSubject(c)
	if !ok {
		return
	}
	var req submitPlaygroundJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.service.SubmitJob(c.Request.Context(), subject.UserID, service.SubmitPlaygroundJobInput{
		Kind:            req.Kind,
		Model:           req.Model,
		APIKey:          req.APIKey,
		InternalBaseURL: inferPlaygroundInternalBaseURL(c.Request),
		RequestPayload:  req.RequestPayload,
	})
	if writePlaygroundResourceError(c, err) {
		return
	}
	response.Created(c, item)
}

func (h *PlaygroundHandler) CreateTask(c *gin.Context) {
	subject, ok := playgroundSubject(c)
	if !ok {
		return
	}
	var req createPlaygroundTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.service.CreateTask(c.Request.Context(), subject.UserID, service.CreatePlaygroundTaskInput{Kind: req.Kind, Status: req.Status, Model: req.Model, RequestID: req.RequestID, RequestPayload: req.RequestPayload, ResultPayload: req.ResultPayload, ErrorMessage: req.ErrorMessage})
	if writePlaygroundResourceError(c, err) {
		return
	}
	response.Created(c, item)
}
func (h *PlaygroundHandler) ListTasks(c *gin.Context) {
	subject, ok := playgroundSubject(c)
	if !ok {
		return
	}
	params := playgroundPagination(c)
	// #region debug-point D:playground-list-tasks-start
	reportPlayground500HandlerDebugEvent("D", "playground_handler.go:ListTasks", "[DEBUG] playground list tasks start", map[string]any{
		"user_id": subject.UserID,
		"kind":    c.Query("kind"),
		"page":    params.Page,
		"limit":   params.Limit(),
	})
	// #endregion
	items, total, err := h.service.ListTasks(c.Request.Context(), subject.UserID, params, c.Query("kind"))
	// #region debug-point E:playground-list-tasks-result
	reportPlayground500HandlerDebugEvent("E", "playground_handler.go:ListTasks", "[DEBUG] playground list tasks result", map[string]any{
		"user_id":    subject.UserID,
		"err":        fmt.Sprint(err),
		"item_count": len(items),
		"total":      total,
	})
	// #endregion
	if writePlaygroundResourceError(c, err) {
		return
	}
	response.Paginated(c, items, total, params.Page, params.Limit())
}
func (h *PlaygroundHandler) GetTask(c *gin.Context) {
	subject, ok := playgroundSubject(c)
	if !ok {
		return
	}
	id, ok := playgroundID(c)
	if !ok {
		return
	}
	item, err := h.service.GetTask(c.Request.Context(), subject.UserID, id)
	if writePlaygroundResourceError(c, err) {
		return
	}
	response.Success(c, item)
}
func (h *PlaygroundHandler) CancelTask(c *gin.Context) {
	subject, ok := playgroundSubject(c)
	if !ok {
		return
	}
	id, ok := playgroundID(c)
	if !ok {
		return
	}
	if writePlaygroundResourceError(c, h.service.CancelTask(c.Request.Context(), subject.UserID, id)) {
		return
	}
	response.Success(c, gin.H{"id": id, "status": "canceled"})
}
func (h *PlaygroundHandler) CreateAsset(c *gin.Context) {
	subject, ok := playgroundSubject(c)
	if !ok {
		return
	}
	var req createPlaygroundAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.service.CreateAsset(c.Request.Context(), subject.UserID, service.CreatePlaygroundAssetInput{TaskID: req.TaskID, Kind: req.Kind, Title: req.Title, Content: req.Content, URL: req.URL, InternalBaseURL: inferPlaygroundInternalBaseURL(c.Request), StorageKey: req.StorageKey, ContentType: req.ContentType, ByteSize: req.ByteSize, Metadata: req.Metadata})
	if writePlaygroundResourceError(c, err) {
		return
	}
	response.Created(c, item)
}
func (h *PlaygroundHandler) ListAssets(c *gin.Context) {
	subject, ok := playgroundSubject(c)
	if !ok {
		return
	}
	params := playgroundPagination(c)
	items, total, err := h.service.ListAssets(c.Request.Context(), subject.UserID, params, c.Query("kind"))
	if writePlaygroundResourceError(c, err) {
		return
	}
	response.Paginated(c, items, total, params.Page, params.Limit())
}
func (h *PlaygroundHandler) GetAsset(c *gin.Context) {
	subject, ok := playgroundSubject(c)
	if !ok {
		return
	}
	id, ok := playgroundID(c)
	if !ok {
		return
	}
	item, err := h.service.GetAsset(c.Request.Context(), subject.UserID, id)
	if writePlaygroundResourceError(c, err) {
		return
	}
	response.Success(c, item)
}

func (h *PlaygroundHandler) ServeAssetByID(c *gin.Context) {
	subject, ok := playgroundSubject(c)
	if !ok {
		return
	}
	id, ok := playgroundID(c)
	if !ok {
		return
	}
	logger.L().Info("playground.asset.by_id.request",
		zap.Int64("subject_user_id", subject.UserID),
		zap.Int64("asset_id", id),
		zap.String("raw_path", c.Request.URL.Path),
	)
	asset, err := h.service.GetAsset(c.Request.Context(), subject.UserID, id)
	if writePlaygroundResourceError(c, err) {
		logger.L().Warn("playground.asset.by_id.lookup_failed",
			zap.Int64("subject_user_id", subject.UserID),
			zap.Int64("asset_id", id),
			zap.Error(err),
		)
		return
	}
	if asset != nil {
		logger.L().Info("playground.asset.by_id.asset_hit",
			zap.Int64("asset_id", asset.ID),
			zap.Int64("asset_user_id", asset.UserID),
			zap.String("asset_kind", asset.Kind),
			zap.String("asset_storage_key", asset.StorageKey),
			zap.String("asset_url", asset.URL),
		)
	}
	h.serveStoredAsset(c, asset)
}

func (h *PlaygroundHandler) ServeAssetContent(c *gin.Context) {
	subject, ok := playgroundSubject(c)
	if !ok {
		return
	}
	storageKey := strings.TrimPrefix(strings.TrimSpace(c.Param("storageKey")), "/")
	role, _ := servermiddleware.GetUserRoleFromContext(c)
	logger.L().Info("playground.asset.content.request",
		zap.Int64("subject_user_id", subject.UserID),
		zap.String("role", role),
		zap.String("raw_path", c.Request.URL.Path),
		zap.String("storage_key_param", storageKey),
	)
	if storageKey == "" {
		response.NotFound(c, "Resource not found")
		return
	}
	asset, err := h.service.GetAssetByStorageKey(c.Request.Context(), subject.UserID, storageKey)
	if err != nil {
		if role == "admin" {
			logger.L().Info("playground.asset.content.admin_fallback",
				zap.Int64("subject_user_id", subject.UserID),
				zap.String("storage_key", storageKey),
			)
			asset, err = h.service.GetAssetByStorageKeyAnyUser(c.Request.Context(), storageKey)
		}
	}
	if err == nil && asset != nil {
		logger.L().Info("playground.asset.content.asset_hit",
			zap.Int64("asset_id", asset.ID),
			zap.Int64("asset_user_id", asset.UserID),
			zap.String("asset_kind", asset.Kind),
			zap.String("asset_storage_key", asset.StorageKey),
			zap.String("asset_url", asset.URL),
		)
	}
	if writePlaygroundResourceError(c, err) {
		logger.L().Warn("playground.asset.content.lookup_failed",
			zap.Int64("subject_user_id", subject.UserID),
			zap.String("storage_key", storageKey),
			zap.Error(err),
		)
		return
	}
	h.serveStoredAsset(c, asset)
}

func (h *PlaygroundHandler) serveStoredAsset(c *gin.Context, asset *service.PlaygroundAsset) {
	if asset == nil {
		response.NotFound(c, "Resource not found")
		return
	}
	storageKey := strings.TrimSpace(asset.StorageKey)
	if storageKey == "" {
		response.NotFound(c, "Resource not found")
		return
	}
	path, ok := h.storage.ResolvePath(storageKey)
	logger.L().Info("playground.asset.content.resolve",
		zap.Int64("asset_id", asset.ID),
		zap.Int64("asset_user_id", asset.UserID),
		zap.String("storage_key", storageKey),
		zap.String("resolved_path", path),
		zap.Bool("resolve_ok", ok),
	)
	if !ok {
		response.NotFound(c, "Resource not found")
		return
	}
	file, openErr := os.Open(path)
	if openErr != nil {
		logger.L().Warn("playground.asset.content.open_failed",
			zap.Int64("asset_id", asset.ID),
			zap.String("storage_key", storageKey),
			zap.String("resolved_path", path),
			zap.Error(openErr),
		)
		if os.IsNotExist(openErr) {
			response.NotFound(c, "Resource not found")
			return
		}
		response.ErrorFrom(c, openErr)
		return
	}
	defer func() { _ = file.Close() }()
	info, statErr := file.Stat()
	if statErr != nil {
		logger.L().Warn("playground.asset.content.stat_failed",
			zap.Int64("asset_id", asset.ID),
			zap.String("storage_key", storageKey),
			zap.String("resolved_path", path),
			zap.Error(statErr),
		)
		response.ErrorFrom(c, statErr)
		return
	}
	contentType := strings.TrimSpace(asset.ContentType)
	if contentType == "" {
		switch {
		case strings.HasSuffix(strings.ToLower(storageKey), ".png"):
			contentType = "image/png"
		case strings.HasSuffix(strings.ToLower(storageKey), ".jpg"), strings.HasSuffix(strings.ToLower(storageKey), ".jpeg"), strings.HasSuffix(strings.ToLower(storageKey), ".jfif"):
			contentType = "image/jpeg"
		case strings.HasSuffix(strings.ToLower(storageKey), ".webp"):
			contentType = "image/webp"
		case strings.HasSuffix(strings.ToLower(storageKey), ".wav"):
			contentType = "audio/wav"
		case strings.HasSuffix(strings.ToLower(storageKey), ".mp3"):
			contentType = "audio/mpeg"
		case strings.HasSuffix(strings.ToLower(storageKey), ".mp4"):
			contentType = "video/mp4"
		default:
			contentType = "application/octet-stream"
		}
	}
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "private, max-age=86400")
	c.Header("X-Content-Type-Options", "nosniff")
	http.ServeContent(c.Writer, c.Request, storageKey, info.ModTime(), file)
}

func inferPlaygroundInternalBaseURL(r *http.Request) string {
	if r == nil {
		return ""
	}
	if addr, ok := r.Context().Value(http.LocalAddrContextKey).(fmt.Stringer); ok {
		if baseURL := normalizePlaygroundInternalBaseURLCandidate(addr.String()); baseURL != "" {
			return baseURL
		}
	}
	if baseURL := normalizePlaygroundInternalBaseURLCandidate(r.Host); baseURL != "" {
		return baseURL
	}
	if forwardedHost := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Host"), ",")[0]); forwardedHost != "" {
		return normalizePlaygroundInternalBaseURLCandidate(forwardedHost)
	}
	return ""
}

func normalizePlaygroundInternalBaseURLCandidate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "://") {
		return raw
	}
	return "http://" + raw
}

func (h *PlaygroundHandler) DeleteAsset(c *gin.Context) {
	subject, ok := playgroundSubject(c)
	if !ok {
		return
	}
	id, ok := playgroundID(c)
	if !ok {
		return
	}
	if writePlaygroundResourceError(c, h.service.DeleteAsset(c.Request.Context(), subject.UserID, id)) {
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *PlaygroundHandler) ListRecords(c *gin.Context) {
	subject, ok := playgroundSubject(c)
	if !ok {
		return
	}
	params := playgroundPagination(c)
	// #region debug-point F:playground-list-records-start
	reportPlayground500HandlerDebugEvent("F", "playground_handler.go:ListRecords", "[DEBUG] playground list records start", map[string]any{
		"user_id": subject.UserID,
		"kind":    c.Query("kind"),
		"page":    params.Page,
		"limit":   params.Limit(),
	})
	// #endregion
	items, total, err := h.service.ListRecords(c.Request.Context(), subject.UserID, params, c.Query("kind"))
	// #region debug-point G:playground-list-records-result
	reportPlayground500HandlerDebugEvent("G", "playground_handler.go:ListRecords", "[DEBUG] playground list records result", map[string]any{
		"user_id":    subject.UserID,
		"err":        fmt.Sprint(err),
		"item_count": len(items),
		"total":      total,
	})
	// #endregion
	if writePlaygroundResourceError(c, err) {
		return
	}
	response.Paginated(c, items, total, params.Page, params.Limit())
}

func (h *PlaygroundHandler) DeleteRecord(c *gin.Context) {
	subject, ok := playgroundSubject(c)
	if !ok {
		return
	}
	id, ok := playgroundID(c)
	if !ok {
		return
	}
	if writePlaygroundResourceError(c, h.service.DeleteRecord(c.Request.Context(), subject.UserID, id)) {
		return
	}
	c.Status(http.StatusNoContent)
}

func playgroundSubject(c *gin.Context) (servermiddleware.AuthSubject, bool) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
	}
	return subject, ok
}
func playgroundID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		response.BadRequest(c, "Invalid id")
		return 0, false
	}
	return id, true
}
func playgroundPagination(c *gin.Context) pagination.PaginationParams {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return pagination.PaginationParams{Page: page, PageSize: size}
}
func writePlaygroundResourceError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, service.ErrPlaygroundNotFound):
		response.NotFound(c, "Resource not found")
	case errors.Is(err, service.ErrPlaygroundInvalidState):
		response.Error(c, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrPlaygroundInvalidInput):
		response.BadRequest(c, err.Error())
	default:
		response.ErrorFrom(c, err)
	}
	return true
}
