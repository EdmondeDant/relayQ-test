package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type PlaygroundHandler struct {
	service *service.PlaygroundService
	storage *service.PlaygroundAssetStorage
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
	items, total, err := h.service.ListTasks(c.Request.Context(), subject.UserID, params, c.Query("kind"))
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
	item, err := h.service.CreateAsset(c.Request.Context(), subject.UserID, service.CreatePlaygroundAssetInput{TaskID: req.TaskID, Kind: req.Kind, Title: req.Title, Content: req.Content, URL: req.URL, StorageKey: req.StorageKey, ContentType: req.ContentType, ByteSize: req.ByteSize, Metadata: req.Metadata})
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
	asset, err := h.service.GetAsset(c.Request.Context(), subject.UserID, id)
	if writePlaygroundResourceError(c, err) {
		return
	}
	h.serveStoredAsset(c, asset)
}

func (h *PlaygroundHandler) ServeAssetContent(c *gin.Context) {
	subject, ok := playgroundSubject(c)
	if !ok {
		return
	}
	storageKey := strings.TrimSpace(c.Param("key"))
	if storageKey == "" {
		response.NotFound(c, "Resource not found")
		return
	}
	asset, err := h.service.GetAssetByStorageKey(c.Request.Context(), subject.UserID, storageKey)
	if writePlaygroundResourceError(c, err) {
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
	if !ok {
		response.NotFound(c, "Resource not found")
		return
	}
	file, openErr := os.Open(path)
	if openErr != nil {
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
	items, total, err := h.service.ListRecords(c.Request.Context(), subject.UserID, params, c.Query("kind"))
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
