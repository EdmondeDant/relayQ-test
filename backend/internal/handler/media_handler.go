package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type MediaHandler struct {
	catalog      *service.MediaCatalogService
	orchestrator *service.MediaOrchestrator
}

func NewMediaHandler(catalog *service.MediaCatalogService, orchestrator *service.MediaOrchestrator) *MediaHandler {
	return &MediaHandler{catalog: catalog, orchestrator: orchestrator}
}

func (h *MediaHandler) Submit(c *gin.Context, modality, operation string) bool {
	apiKey, subject, groupID, ok := mediaRequestIdentity(c)
	if !ok || h == nil || h.catalog == nil || h.orchestrator == nil {
		return false
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		mediaError(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return true
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	if !strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		return false
	}
	fields := map[string]any{}
	if json.Unmarshal(body, &fields) != nil {
		mediaError(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return true
	}
	model := strings.TrimSpace(mediaAnyString(fields["model"]))
	if model == "" {
		return false
	}
	if modality == "image" && !service.GroupAllowsImageGeneration(apiKey.Group) {
		mediaError(c, http.StatusForbidden, "permission_error", service.ImageGenerationPermissionMessage())
		return true
	}
	if _, err = h.catalog.GetRuntime(c.Request.Context(), groupID, model, modality); err != nil {
		if errors.Is(err, service.ErrMediaCatalogProductNotFound) {
			return false
		}
		mediaError(c, http.StatusInternalServerError, "api_error", "Failed to resolve media product")
		return true
	}
	canonicalBody, err := json.Marshal(fields)
	if err != nil {
		mediaError(c, http.StatusBadRequest, "invalid_request_error", "Failed to normalize request body")
		return true
	}
	requestSum := sha256.Sum256(append([]byte(modality+"\n"+operation+"\n"), canonicalBody...))
	requestHash := hex.EncodeToString(requestSum[:])
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		idempotencyKey = strings.TrimSpace(c.GetHeader("X-Idempotency-Key"))
	}
	if idempotencyKey == "" {
		if modality != "image" {
			mediaError(c, http.StatusBadRequest, "invalid_request_error", "Idempotency-Key is required")
			return true
		}
		idempotencyKey = requestHash
	}
	sum := sha256.Sum256([]byte(strconv.FormatInt(subject.UserID, 10) + "\n" + strconv.FormatInt(apiKey.ID, 10) + "\n" + modality + "\n" + operation + "\n" + idempotencyKey))
	publicID := "media_rq_" + hex.EncodeToString(sum[:16])
	job, outcome, err := h.orchestrator.Submit(c.Request.Context(), service.MediaSubmitInput{PublicID: publicID, UserID: subject.UserID, APIKeyID: apiKey.ID, GroupID: groupID, RequestHash: requestHash, Request: service.MediaCanonicalRequest{Operation: operation, Model: model, Modality: modality, Body: body, Fields: fields}})
	if err != nil {
		code, status := "api_error", http.StatusBadGateway
		if errors.Is(err, service.ErrNoTrustedMediaOffer) {
			code, status = "no_trusted_media_offer", http.StatusServiceUnavailable
		} else if errors.Is(err, service.ErrMediaOfferExhausted) {
			code = "media_offer_exhausted"
		} else if errors.Is(err, service.ErrMediaIdempotencyConflict) {
			code, status = "idempotency_conflict", http.StatusConflict
		}
		mediaError(c, status, code, err.Error())
		return true
	}
	if modality == "image" && outcome.Result != nil && service.IsTerminalMediaSuccessStatus(string(job.Status)) {
		c.JSON(http.StatusOK, outcome.Result)
		return true
	}
	c.JSON(http.StatusAccepted, gin.H{"id": job.PublicID, "object": "media.generation", "model": job.Model, "status": job.Status, "created_at": job.CreatedAt.Unix()})
	return true
}

func (h *MediaHandler) Lookup(c *gin.Context, content bool) bool {
	apiKey, subject, _, ok := mediaRequestIdentity(c)
	if !ok || h == nil || h.orchestrator == nil {
		return false
	}
	publicID := strings.TrimSpace(c.Param("request_id"))
	job, found, err := h.orchestrator.LookupOwnedJob(c.Request.Context(), publicID, subject.UserID, apiKey.ID)
	if err != nil {
		mediaError(c, http.StatusInternalServerError, "api_error", err.Error())
		return true
	}
	if !found {
		return false
	}
	if content {
		index, _ := strconv.Atoi(c.DefaultQuery("index", "0"))
		media, contentErr := h.orchestrator.Content(c.Request.Context(), job, index)
		if contentErr != nil {
			mediaError(c, http.StatusBadGateway, "api_error", contentErr.Error())
			return true
		}
		defer media.Body.Close()
		if media.ContentType != "" {
			c.Header("Content-Type", media.ContentType)
		}
		if media.Length >= 0 {
			c.Header("Content-Length", strconv.FormatInt(media.Length, 10))
		}
		c.Status(http.StatusOK)
		_, _ = io.Copy(c.Writer, media.Body)
		return true
	}
	result, pollErr := h.orchestrator.Poll(c.Request.Context(), job)
	if pollErr != nil {
		mediaError(c, http.StatusBadGateway, "api_error", pollErr.Error())
		return true
	}
	if result == nil {
		result = map[string]any{}
	}
	result["id"] = job.PublicID
	c.JSON(http.StatusOK, result)
	return true
}

func mediaRequestIdentity(c *gin.Context) (*service.APIKey, middleware2.AuthSubject, int64, bool) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil || apiKey.GroupID == nil || apiKey.Group == nil || apiKey.Group.Platform != service.PlatformOpenAI {
		return nil, middleware2.AuthSubject{}, 0, false
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	return apiKey, subject, *apiKey.GroupID, ok
}

func mediaAnyString(value any) string {
	text, _ := value.(string)
	return text
}

func mediaError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"type": code, "code": code, "message": message}})
}
