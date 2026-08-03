package handler

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type LeonardoMediaHandler struct {
	create *service.LeonardoMediaCreateService
	get    *service.LeonardoMediaGetService
}

type leonardoMediaCreateHTTPRequest struct {
	Model            string                       `json:"model"`
	Modality         string                       `json:"modality"`
	Prompt           string                       `json:"prompt"`
	Public           bool                         `json:"public"`
	Parameters       leonardoMediaImageParameters `json:"parameters"`
	CustomerQuoteUSD string                       `json:"customer_quote_usd"`
}

type leonardoMediaImageParameters struct {
	Width    int `json:"width"`
	Height   int `json:"height"`
	Quantity int `json:"quantity"`
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
	quote, err := decimal.NewFromString(strings.TrimSpace(req.CustomerQuoteUSD))
	if err != nil || req.Modality != "image" || strings.TrimSpace(req.Model) == "" || strings.TrimSpace(req.Prompt) == "" || len(strings.TrimSpace(req.Prompt)) > 4000 || req.Parameters.Width <= 0 || req.Parameters.Height <= 0 || req.Parameters.Quantity <= 0 || quote.Sign() <= 0 || quote.Exponent() < -8 || quote.Cmp(decimal.RequireFromString("999999999999.99999999")) > 0 {
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
	input := service.LeonardoMediaCreateInput{IdempotencyKey: idempotencyKey, UserID: subject.UserID, APIKeyID: apiKey.ID, GroupID: *apiKey.GroupID, Model: req.Model, Prompt: req.Prompt, Public: req.Public, Width: req.Parameters.Width, Height: req.Parameters.Height, Quantity: req.Parameters.Quantity, CustomerQuoteUSD: quote}
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
