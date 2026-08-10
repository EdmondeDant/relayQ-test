package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
	"github.com/shopspring/decimal"
)

var (
	ErrLeonardoMediaCreateNotConfigured       = infraerrors.InternalServer("LEONARDO_MEDIA_CREATE_NOT_CONFIGURED", "Leonardo media create service is not configured")
	ErrLeonardoMediaCreateInputInvalid        = infraerrors.BadRequest("LEONARDO_MEDIA_CREATE_INPUT_INVALID", "Leonardo media create input is invalid")
	ErrLeonardoMediaNoAvailableAccount        = infraerrors.ServiceUnavailable("LEONARDO_MEDIA_NO_AVAILABLE_ACCOUNT", "no available Leonardo account")
	ErrLeonardoMediaAccountSelectionAmbiguous = infraerrors.ServiceUnavailable("LEONARDO_MEDIA_ACCOUNT_SELECTION_AMBIGUOUS", "multiple Leonardo accounts require scheduler selection")
	ErrLeonardoMediaCreateResultInvalid       = infraerrors.InternalServer("LEONARDO_MEDIA_CREATE_RESULT_INVALID", "Leonardo media create result is invalid")
)

var leonardoMediaCustomerPriceRate = decimal.RequireFromString("7.1")

func EstimateLeonardoCustomerPrice(ctx context.Context, request LeonardoImagePriceRequest) (*LeonardoImagePriceEstimate, decimal.Decimal, error) {
	estimate, err := NewLeonardoImagePriceResolver().Estimate(ctx, request)
	if err != nil {
		return nil, decimal.Zero, err
	}
	if estimate == nil || estimate.EstimatedCostUSD.Sign() <= 0 {
		return nil, decimal.Zero, ErrLeonardoImagePricingNotFound
	}
	return estimate, estimate.EstimatedCostUSD.Mul(leonardoMediaCustomerPriceRate), nil
}

func EstimateLeonardoVideoCustomerPrice(ctx context.Context, request LeonardoVideoPriceRequest) (*LeonardoVideoPriceEstimate, decimal.Decimal, error) {
	estimate, err := NewLeonardoVideoPriceResolver().Estimate(ctx, request)
	if err != nil {
		return nil, decimal.Zero, err
	}
	if estimate == nil || estimate.EstimatedCostUSD.Sign() <= 0 {
		return nil, decimal.Zero, ErrLeonardoVideoPricingEvidenceUnavailable
	}
	return estimate, estimate.EstimatedCostUSD.Mul(leonardoMediaCustomerPriceRate), nil
}

func LeonardoDefaultImagePriceRequest(model string) LeonardoImagePriceRequest {
	return LeonardoImagePriceRequest{Model: strings.TrimSpace(model), Width: 1024, Height: 1024, Quantity: 1, QualityTier: "low"}
}

func LeonardoDefaultVideoPriceRequest(model string) LeonardoVideoPriceRequest {
	switch strings.TrimSpace(model) {
	case "seedance-1.0-pro-fast", "seedance-1.0-pro":
		return LeonardoVideoPriceRequest{Model: strings.TrimSpace(model), Duration: 4, Width: 864, Height: 480, Quantity: 1}
	case "motion_2.0-fast":
		return LeonardoVideoPriceRequest{Model: strings.TrimSpace(model), Width: 832, Height: 480, Quantity: 1}
	case "wan-2.7":
		return LeonardoVideoPriceRequest{Model: strings.TrimSpace(model), Duration: 2, Width: 1280, Height: 720, Quantity: 1}
	default:
		return LeonardoVideoPriceRequest{Model: strings.TrimSpace(model), Quantity: 1}
	}
}

type LeonardoMediaCreateInput struct {
	IdempotencyKey  string
	UserID          int64
	APIKeyID        int64
	GroupID         int64
	Model           string
	Prompt          string
	Public          bool
	Width           int
	Height          int
	Quantity        int
	InputImage      *LeonardoImageInput
	InputImages     []*LeonardoImageInput
	ImageReferences []leonardo.ImageReference
	ImageCapability *LeonardoImageReferenceCapability
	FluxGuidances   LeonardoFluxGuidances
	RawBody         []byte
	MultipartImages map[string]*LeonardoImageInput
	QualityTier     string
	Modality        string
	Duration        int
}

type LeonardoMediaCreateResult struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	Modality  string `json:"modality"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
}

type LeonardoMediaCreateService struct {
	accounts     AccountRepository
	orchestrator *LeonardoImageCreateOrchestrator
}

func NewLeonardoMediaCreateService(accounts AccountRepository, orchestrator *LeonardoImageCreateOrchestrator) *LeonardoMediaCreateService {
	return &LeonardoMediaCreateService{accounts: accounts, orchestrator: orchestrator}
}

func (s *LeonardoMediaCreateService) EstimateQuote(ctx context.Context, model string, width, height, quantity int) (decimal.Decimal, error) {
	return s.EstimateQualityQuote(ctx, model, width, height, quantity, "low")
}

func (s *LeonardoMediaCreateService) EstimateQualityQuote(ctx context.Context, model string, width, height, quantity int, qualityTier string) (decimal.Decimal, error) {
	if s == nil || s.orchestrator == nil || s.orchestrator.quotes == nil || s.orchestrator.quotes.priceResolver == nil {
		return decimal.Zero, ErrLeonardoMediaCreateNotConfigured
	}
	estimate, err := s.orchestrator.quotes.priceResolver.Estimate(ctx, LeonardoImagePriceRequest{Model: strings.TrimSpace(model), Width: width, Height: height, Quantity: quantity, QualityTier: qualityTier})
	if err != nil {
		return decimal.Zero, err
	}
	if estimate == nil || estimate.EstimatedCostUSD.Sign() <= 0 {
		return decimal.Zero, ErrLeonardoImagePricingNotFound
	}
	return estimate.EstimatedCostUSD.Mul(leonardoMediaCustomerPriceRate), nil
}

func (s *LeonardoMediaCreateService) Create(ctx context.Context, input LeonardoMediaCreateInput) (*LeonardoMediaCreateResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.accounts == nil || s.orchestrator == nil {
		return nil, ErrLeonardoMediaCreateNotConfigured
	}
	model, prompt := strings.TrimSpace(input.Model), strings.TrimSpace(input.Prompt)
	modality := strings.ToLower(strings.TrimSpace(input.Modality))
	if modality == "" {
		modality = "image"
	}
	matchReferenceSize := len(input.RawBody) > 0 && (model == "nano-banana-2" || model == "nano-banana-2-lite") && input.Width == 0 && input.Height == 0
	if input.UserID <= 0 || input.APIKeyID <= 0 || input.GroupID <= 0 || (modality != "image" && modality != "video") || model == "" || prompt == "" || len(prompt) > 4000 || (!matchReferenceSize && (input.Width <= 0 || input.Height <= 0)) || input.Quantity <= 0 || (modality == "video" && input.Duration <= 0 && model != "motion_2.0-fast") {
		return nil, ErrLeonardoMediaCreateInputInvalid
	}
	verified, ok := leonardo.ResolveByRequestModelSlug(model)
	if !ok || (modality == "image" && verified.Modality != leonardo.ModelModalityImage) || (modality == "video" && verified.Modality != leonardo.ModelModalityVideo) {
		return nil, ErrLeonardoMediaCreateInputInvalid
	}
	if modality == "image" && len(input.RawBody) == 0 {
		if _, err := BuildLeonardoFluxGuidances(model, input.FluxGuidances); err != nil {
			return nil, ErrLeonardoMediaCreateInputInvalid
		}
	} else if len(input.RawBody) > 0 && !json.Valid(input.RawBody) {
		return nil, ErrLeonardoMediaCreateInputInvalid
	}
	var quote decimal.Decimal
	var err error
	if modality == "video" {
		estimate, priceErr := NewLeonardoVideoPriceResolver().Estimate(ctx, LeonardoVideoPriceRequest{Model: model, Duration: input.Duration, Width: input.Width, Height: input.Height, Quantity: input.Quantity})
		if priceErr != nil {
			return nil, priceErr
		}
		quote = estimate.EstimatedCostUSD.Mul(leonardoMediaCustomerPriceRate)
	} else {
		quote, err = s.EstimateQualityQuote(ctx, model, input.Width, input.Height, input.Quantity, input.QualityTier)
	}
	if err != nil {
		return nil, err
	}
	key, err := NormalizeIdempotencyKey(input.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	publicSum := sha256.Sum256([]byte("leonardo_media_create\n" + strconv.FormatInt(input.UserID, 10) + "\n" + key))
	publicID := "gen_rq_" + hex.EncodeToString(publicSum[:16])
	fingerprint, err := json.Marshal(struct {
		Model             string                    `json:"model"`
		Modality          string                    `json:"modality"`
		Prompt            string                    `json:"prompt"`
		Public            bool                      `json:"public"`
		Width             int                       `json:"width"`
		Height            int                       `json:"height"`
		Quantity          int                       `json:"quantity"`
		Duration          int                       `json:"duration,omitempty"`
		CustomerQuoteUSD  string                    `json:"customer_quote_usd"`
		InputImageSHA256s []string                  `json:"input_image_sha256s,omitempty"`
		ImageReferences   []leonardo.ImageReference `json:"image_references,omitempty"`
		FluxGuidances     LeonardoFluxGuidances     `json:"flux_guidances,omitempty"`
		RawBodySHA256     string                    `json:"raw_body_sha256,omitempty"`
	}{model, modality, prompt, input.Public, input.Width, input.Height, input.Quantity, input.Duration, quote.String(), leonardoMediaInputImageSHA256s(input.InputImage, input.InputImages), input.ImageReferences, input.FluxGuidances, leonardoMediaRawBodySHA256(input.RawBody)})
	if err != nil {
		return nil, ErrLeonardoMediaCreateInputInvalid
	}
	hash := sha256.Sum256(fingerprint)
	accounts, err := s.accounts.ListSchedulableByGroupIDAndPlatform(ctx, input.GroupID, PlatformLeonardo)
	if err != nil {
		return nil, err
	}
	valid := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if validLeonardoMediaAccount(&account, input.GroupID, model) {
			valid = append(valid, account)
		}
	}
	if len(valid) == 0 {
		return nil, ErrLeonardoMediaNoAvailableAccount
	}
	if len(valid) > 1 {
		return nil, ErrLeonardoMediaAccountSelectionAmbiguous
	}
	var job *GenerationJob
	if modality == "video" {
		job, err = s.orchestrator.CreateVideo(ctx, LeonardoVideoCreateRequest{PublicID: publicID, UserID: input.UserID, APIKeyID: input.APIKeyID, GroupID: &input.GroupID, AccountID: valid[0].ID, RequestHash: hex.EncodeToString(hash[:]), Model: model, Prompt: prompt, Width: input.Width, Height: input.Height, Duration: input.Duration, Quantity: input.Quantity, Public: input.Public, CustomerQuoteUSD: quote, RawBody: input.RawBody})
	} else {
		job, err = s.orchestrator.Create(ctx, LeonardoImageCreateRequest{PublicID: publicID, UserID: input.UserID, APIKeyID: input.APIKeyID, GroupID: &input.GroupID, AccountID: valid[0].ID, RequestHash: hex.EncodeToString(hash[:]), Model: model, Prompt: prompt, Width: input.Width, Height: input.Height, Quantity: input.Quantity, Public: input.Public, QualityTier: input.QualityTier, CustomerQuoteUSD: quote, InputImage: input.InputImage, InputImages: input.InputImages, ImageReferences: input.ImageReferences, ImageCapability: input.ImageCapability, FluxGuidances: input.FluxGuidances, RawBody: input.RawBody, MultipartImages: input.MultipartImages})
	}
	if err != nil {
		return nil, err
	}
	if job == nil || strings.TrimSpace(job.PublicID) == "" {
		return nil, ErrLeonardoMediaCreateResultInvalid
	}
	createdAt := int64(0)
	if !job.CreatedAt.IsZero() {
		createdAt = job.CreatedAt.Unix()
	}
	return &LeonardoMediaCreateResult{ID: job.PublicID, Object: "media.generation", Provider: PlatformLeonardo, Model: job.Model, Modality: modality, Status: string(job.Status), CreatedAt: createdAt}, nil
}

func leonardoMediaRawBodySHA256(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:])
}

func leonardoMediaInputImageSHA256(input *LeonardoImageInput) string {
	if input == nil || len(input.Data) == 0 {
		return ""
	}
	return LeonardoImageSHA256(input.Data)
}

func leonardoMediaInputImageSHA256s(input *LeonardoImageInput, inputs []*LeonardoImageInput) []string {
	all := append([]*LeonardoImageInput(nil), inputs...)
	if input != nil {
		all = append([]*LeonardoImageInput{input}, all...)
	}
	hashes := make([]string, 0, len(all))
	for _, image := range all {
		if hash := leonardoMediaInputImageSHA256(image); hash != "" {
			hashes = append(hashes, hash)
		}
	}
	return hashes
}

func validLeonardoMediaAccount(account *Account, groupID int64, model string) bool {
	if account == nil || account.ID <= 0 || account.Platform != PlatformLeonardo || account.Type != AccountTypeAPIKey || !account.IsSchedulable() || !account.IsModelSupported(model) {
		return false
	}
	for _, id := range account.GroupIDs {
		if id == groupID {
			return true
		}
	}
	for _, relation := range account.AccountGroups {
		if relation.GroupID == groupID {
			return true
		}
	}
	for _, group := range account.Groups {
		if group != nil && group.ID == groupID {
			return true
		}
	}
	return false
}
