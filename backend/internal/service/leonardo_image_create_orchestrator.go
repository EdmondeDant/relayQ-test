package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
	"github.com/shopspring/decimal"
)

var (
	ErrLeonardoImageCreateNotConfigured         = errors.New("leonardo image create orchestrator is not configured")
	ErrLeonardoImageCreateRequestInvalid        = errors.New("leonardo image create request is invalid")
	ErrLeonardoImageCreateAccountBindingInvalid = errors.New("leonardo image create account binding is invalid")
	ErrLeonardoImageCreateReservationInvalid    = errors.New("leonardo image create reservation is invalid")
	ErrLeonardoImageCreateReservationConflict   = errors.New("leonardo image create reservation conflicts with existing reservation")
)

type LeonardoImageCreateRequest struct {
	PublicID         string
	UserID           int64
	APIKeyID         int64
	GroupID          *int64
	AccountID        int64
	RequestHash      string
	Model            string
	Prompt           string
	Width            int
	Height           int
	Quantity         int
	Public           bool
	QualityTier      string
	CustomerQuoteUSD decimal.Decimal
	InputImage       *LeonardoImageInput
	InputImages      []*LeonardoImageInput
	ImageReferences  []leonardo.ImageReference
	ImageCapability  *LeonardoImageReferenceCapability
	FluxGuidances    LeonardoFluxGuidances
	RawBody          []byte
	MultipartImages  map[string]*LeonardoImageInput
}

type LeonardoVideoCreateRequest struct {
	PublicID, RequestHash, Model, Prompt string
	UserID, APIKeyID, AccountID          int64
	GroupID                              *int64
	Duration, Width, Height, Quantity    int
	Public                               bool
	CustomerQuoteUSD                     decimal.Decimal
	RawBody                              []byte
}

type LeonardoImageFundsReserveRequest struct {
	UserID           int64
	PublicID         string
	AmountUSD        decimal.Decimal
	PricingVersion   string
	PricingSource    string
	PricingMatchType string
}

type LeonardoImageFundsReservation struct {
	Reference        string
	UserID           int64
	PublicID         string
	AmountUSD        decimal.Decimal
	PricingVersion   string
	PricingSource    string
	PricingMatchType string
	AlreadyReserved  bool
}

type LeonardoImageFundsReleaseRequest struct {
	UserID    int64
	PublicID  string
	Reference string
	AmountUSD decimal.Decimal
	Reason    string
}

type LeonardoImageCreateFunds interface {
	Reserve(context.Context, LeonardoImageFundsReserveRequest) (*LeonardoImageFundsReservation, error)
	Release(context.Context, LeonardoImageFundsReleaseRequest) error
}

type LeonardoImageFundsSettleRequest struct {
	UserID    int64
	PublicID  string
	Reference string
	AmountUSD decimal.Decimal
}

type LeonardoImageTerminalFunds interface {
	Settle(context.Context, LeonardoImageFundsSettleRequest) error
	Release(context.Context, LeonardoImageFundsReleaseRequest) error
}

type LeonardoImageFunds interface {
	LeonardoImageCreateFunds
	LeonardoImageTerminalFunds
}

type LeonardoImageCreateAccountReader interface {
	GetByID(context.Context, int64) (*Account, error)
}

type LeonardoImageGenerationClientFactory interface {
	Build(*Account) (LeonardoGenerationClient, error)
}

type LeonardoImageAccountAdapterFactory struct {
	upstream HTTPUpstream
	config   *config.Config
}

var _ LeonardoImageGenerationClientFactory = (*LeonardoImageAccountAdapterFactory)(nil)

func NewLeonardoImageAccountAdapterFactory(upstream HTTPUpstream, cfg *config.Config) *LeonardoImageAccountAdapterFactory {
	return &LeonardoImageAccountAdapterFactory{upstream: upstream, config: cfg}
}

func (f *LeonardoImageAccountAdapterFactory) Build(account *Account) (LeonardoGenerationClient, error) {
	if f == nil || f.upstream == nil || f.config == nil {
		return nil, ErrLeonardoImageCreateNotConfigured
	}
	return NewLeonardoGenerationAdapter(account, f.upstream, f.config)
}

type LeonardoImageCreateOrchestrator struct {
	quotes   *LeonardoImageQuoteGuard
	funds    LeonardoImageCreateFunds
	accounts LeonardoImageCreateAccountReader
	clients  LeonardoImageGenerationClientFactory
	jobs     GenerationJobRepository
	uploads  *LeonardoImageUploadService
}

func NewLeonardoImageCreateOrchestrator(quotes *LeonardoImageQuoteGuard, funds LeonardoImageCreateFunds, accounts LeonardoImageCreateAccountReader, clients LeonardoImageGenerationClientFactory, jobs GenerationJobRepository, uploads ...*LeonardoImageUploadService) *LeonardoImageCreateOrchestrator {
	orchestrator := &LeonardoImageCreateOrchestrator{quotes: quotes, funds: funds, accounts: accounts, clients: clients, jobs: jobs}
	if len(uploads) > 0 {
		orchestrator.uploads = uploads[0]
	}
	return orchestrator
}

func (o *LeonardoImageCreateOrchestrator) Create(ctx context.Context, request LeonardoImageCreateRequest) (*GenerationJob, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if o == nil || o.quotes == nil || o.funds == nil || o.accounts == nil || o.clients == nil || o.jobs == nil {
		return nil, ErrLeonardoImageCreateNotConfigured
	}
	publicID, requestHash, model, prompt := strings.TrimSpace(request.PublicID), strings.TrimSpace(request.RequestHash), strings.TrimSpace(request.Model), strings.TrimSpace(request.Prompt)
	matchReferenceSize := len(request.RawBody) > 0 && (model == "nano-banana-2" || model == "nano-banana-2-lite") && request.Width == 0 && request.Height == 0
	if publicID == "" || len(publicID) > 64 || request.UserID <= 0 || request.APIKeyID <= 0 || request.AccountID <= 0 || requestHash == "" || len(requestHash) > 128 || model == "" || prompt == "" || (!matchReferenceSize && (request.Width <= 0 || request.Height <= 0)) || request.Quantity <= 0 || request.CustomerQuoteUSD.Sign() <= 0 || (request.GroupID != nil && *request.GroupID <= 0) {
		return nil, ErrLeonardoImageCreateRequestInvalid
	}
	quote, err := o.quotes.Prepare(ctx, LeonardoImageQuoteRequest{UserID: request.UserID, PricingRequest: LeonardoImagePriceRequest{Model: model, Width: request.Width, Height: request.Height, Quantity: request.Quantity, Public: request.Public, QualityTier: request.QualityTier}, CustomerQuoteUSD: request.CustomerQuoteUSD})
	if err != nil {
		return nil, err
	}
	account, err := o.accounts.GetByID(ctx, request.AccountID)
	if err != nil {
		return nil, err
	}
	if !validLeonardoImageCreateAccount(account, request.AccountID, request.GroupID) {
		return nil, ErrLeonardoImageCreateAccountBindingInvalid
	}
	client, err := o.clients.Build(account)
	if err != nil {
		return nil, err
	}
	reservation, err := o.funds.Reserve(ctx, LeonardoImageFundsReserveRequest{UserID: request.UserID, PublicID: publicID, AmountUSD: quote.CustomerQuoteUSD, PricingVersion: quote.PricingVersion, PricingSource: quote.PricingSource, PricingMatchType: quote.MatchType})
	if err != nil {
		return nil, err
	}
	if !validLeonardoImageReservation(reservation, request.UserID, publicID, quote) {
		if safeLeonardoImageReservation(reservation, request.UserID, publicID, quote.CustomerQuoteUSD) {
			releaseErr := o.release(ctx, request.UserID, publicID, reservation.Reference, quote.CustomerQuoteUSD, "invalid_reservation_response")
			return nil, errors.Join(ErrLeonardoImageCreateReservationInvalid, releaseErr)
		}
		return nil, ErrLeonardoImageCreateReservationInvalid
	}
	if reservation.AlreadyReserved {
		existing, getErr := o.jobs.GetByPublicID(ctx, publicID)
		if getErr != nil {
			return nil, errors.Join(ErrLeonardoImageCreateReservationConflict, getErr)
		}
		if !sameLeonardoImageCreateJob(existing, request, reservation.Reference, quote.CustomerQuoteUSD) {
			return nil, ErrLeonardoImageCreateReservationConflict
		}
		return existing, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(err, o.release(ctx, request.UserID, publicID, reservation.Reference, quote.CustomerQuoteUSD, "context_cancelled_before_submit"))
	}
	customerCost := quote.CustomerQuoteUSD
	estimatedCost := quote.EstimatedUpstreamCostUSD
	estimatedCostUnit := "USD"
	pricingVersion := quote.PricingVersion
	pricingSource := quote.PricingSource
	pricingMatchType := quote.MatchType
	reference := strings.TrimSpace(reservation.Reference)
	job := &GenerationJob{PublicID: publicID, Provider: PlatformLeonardo, Modality: "image", Model: model, UpstreamModel: model, UserID: request.UserID, APIKeyID: request.APIKeyID, GroupID: request.GroupID, AccountID: account.ID, RequestHash: requestHash, EstimatedUpstreamCostAmount: &estimatedCost, EstimatedUpstreamCostUnit: &estimatedCostUnit, PricingSnapshotVersion: &pricingVersion, PricingSource: &pricingSource, PricingMatchType: &pricingMatchType, CustomerCost: &customerCost, BillingStatus: GenerationJobBillingStatusReserved, BillingReference: &reference}
	if len(request.RawBody) > 0 {
		body := request.RawBody
		sources, sourceErr := ParseLeonardoFluxImageSources(body)
		if sourceErr != nil {
			return nil, errors.Join(sourceErr, o.release(ctx, request.UserID, publicID, reference, customerCost, "image_reference_invalid"))
		}
		if len(sources) > 0 {
			uploadClient, ok := client.(LeonardoInitImageClient)
			if !ok || o.uploads == nil {
				return nil, errors.Join(ErrLeonardoImageUploadInvalid, o.release(ctx, request.UserID, publicID, reference, customerCost, "image_upload_failed"))
			}
			for _, source := range sources {
				input, resolveErr := ResolveLeonardoRawImageSource(ctx, source.Value, request.MultipartImages, 20<<20)
				if resolveErr != nil {
					return nil, errors.Join(resolveErr, o.release(ctx, request.UserID, publicID, reference, customerCost, "image_upload_failed"))
				}
				uploadedID, uploadErr := o.uploads.Upload(ctx, account.ID, uploadClient, input)
				if uploadErr != nil {
					return nil, errors.Join(uploadErr, o.release(ctx, request.UserID, publicID, reference, customerCost, "image_upload_failed"))
				}
				body, sourceErr = SetLeonardoRawImageID(body, source, uploadedID)
				if sourceErr != nil {
					return nil, errors.Join(sourceErr, o.release(ctx, request.UserID, publicID, reference, customerCost, "image_reference_invalid"))
				}
			}
		}
		result, submitErr := NewLeonardoGenerationService(o.jobs, client).CreateGenerationRaw(ctx, job, body)
		return o.finish(ctx, result, submitErr, request.UserID, publicID, reference, customerCost)
	}
	parameters := map[string]any{"prompt": prompt, "width": request.Width, "height": request.Height, "quantity": request.Quantity}
	if model == "gpt-image-2" {
		parameters["quality"] = strings.ToUpper(strings.TrimSpace(request.QualityTier))
	}
	if model == "kino-xl" || model == "concept-art" || model == "graphic-design" || model == "illustrative-albedo" {
		mode := map[string]string{"low": "FAST", "high": "QUALITY"}[strings.TrimSpace(request.QualityTier)]
		if mode == "" {
			return nil, errors.Join(ErrLeonardoImagePricingNotFound, o.release(ctx, request.UserID, publicID, reference, customerCost, "image_quality_invalid"))
		}
		parameters["mode"] = mode
		parameters["prompt_enhance"] = "OFF"
	}
	if (len(request.FluxGuidances.Content) > 0 || len(request.FluxGuidances.Style) > 0) && (request.InputImage != nil || len(request.InputImages) > 0 || len(request.ImageReferences) > 0 || request.ImageCapability != nil) {
		return nil, errors.Join(ErrLeonardoImageReferenceInvalid, o.release(ctx, request.UserID, publicID, reference, customerCost, "image_reference_invalid"))
	}
	fluxGuidances, err := BuildLeonardoFluxGuidances(model, request.FluxGuidances)
	if err != nil {
		return nil, errors.Join(err, o.release(ctx, request.UserID, publicID, reference, customerCost, "image_reference_invalid"))
	}
	if fluxGuidances != nil {
		parameters["guidances"] = fluxGuidances
	}
	references := append([]leonardo.ImageReference(nil), request.ImageReferences...)
	inputImages := append([]*LeonardoImageInput(nil), request.InputImages...)
	if request.InputImage != nil {
		inputImages = append([]*LeonardoImageInput{request.InputImage}, inputImages...)
	}
	if len(inputImages) > 0 {
		strength := ""
		if request.ImageCapability != nil {
			strength = request.ImageCapability.DefaultStrength
		}
		strengthRequired := request.ImageCapability != nil && (request.ImageCapability.StrengthRequired || len(request.ImageCapability.AllowedStrengths) > 0)
		if request.ImageCapability == nil || (strengthRequired && strength == "") {
			return nil, errors.Join(ErrLeonardoImageReferenceInvalid, o.release(ctx, request.UserID, publicID, reference, customerCost, "image_reference_invalid"))
		}
		pending := make([]leonardo.ImageReference, len(inputImages))
		for i := range pending {
			pending[i] = leonardo.ImageReference{Image: leonardo.ImageReferenceImage{ID: "pending", Type: "UPLOADED"}, Strength: strength}
		}
		references = append(pending, references...)
		if _, err := BuildLeonardoImageReferenceGuidance(references, request.ImageCapability); err != nil {
			return nil, errors.Join(err, o.release(ctx, request.UserID, publicID, reference, customerCost, "image_reference_invalid"))
		}
		uploadClient, ok := client.(LeonardoInitImageClient)
		if !ok || o.uploads == nil {
			return nil, errors.Join(ErrLeonardoImageUploadInvalid, o.release(ctx, request.UserID, publicID, reference, customerCost, "image_upload_failed"))
		}
		for i, image := range inputImages {
			uploadedID, uploadErr := o.uploads.Upload(ctx, account.ID, uploadClient, image)
			if uploadErr != nil {
				return nil, errors.Join(uploadErr, o.release(ctx, request.UserID, publicID, reference, customerCost, "image_upload_failed"))
			}
			references[i].Image.ID = uploadedID
		}
	}
	guidance, err := BuildLeonardoImageReferenceGuidance(references, request.ImageCapability)
	if err != nil {
		return nil, errors.Join(err, o.release(ctx, request.UserID, publicID, reference, customerCost, "image_reference_invalid"))
	}
	if guidance != nil && fluxGuidances == nil {
		parameters["guidances"] = guidance
	}
	upstreamRequest := leonardo.CreateGenerationRequest{Model: model, Public: request.Public, Parameters: parameters}
	result, submitErr := NewLeonardoGenerationService(o.jobs, client).CreateGeneration(ctx, job, upstreamRequest)
	return o.finish(ctx, result, submitErr, request.UserID, publicID, reference, customerCost)
}

func (o *LeonardoImageCreateOrchestrator) CreateVideo(ctx context.Context, request LeonardoVideoCreateRequest) (*GenerationJob, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if o == nil || o.funds == nil || o.accounts == nil || o.clients == nil || o.jobs == nil {
		return nil, ErrLeonardoImageCreateNotConfigured
	}
	model, prompt, publicID, requestHash := strings.TrimSpace(request.Model), strings.TrimSpace(request.Prompt), strings.TrimSpace(request.PublicID), strings.TrimSpace(request.RequestHash)
	if publicID == "" || requestHash == "" || request.UserID <= 0 || request.APIKeyID <= 0 || request.AccountID <= 0 || prompt == "" || request.CustomerQuoteUSD.Sign() <= 0 {
		return nil, ErrLeonardoImageCreateRequestInvalid
	}
	var estimate *LeonardoVideoPriceEstimate
	var err error
	if model == "minimax-h3" {
		if request.Duration < 5 || request.Duration > 15 || request.Width != 1376 || request.Height != 768 || request.Quantity != 1 {
			return nil, ErrLeonardoVideoPricingEvidenceUnavailable
		}
		cost := interpolateLeonardoVideoPrice("0.897", "2.691", request.Duration, 5, 15)
		estimate = &LeonardoVideoPriceEstimate{EstimatedCostUSD: cost, PricingVersion: LeonardoVideoPricingPolicyVersion, PricingSource: "leonardo_authenticated_pricing_calculator", MatchType: "model_duration_resolution_exact"}
	} else {
		estimate, err = NewLeonardoVideoPriceResolver().Estimate(ctx, LeonardoVideoPriceRequest{Model: model, Duration: request.Duration, Width: request.Width, Height: request.Height, Quantity: request.Quantity})
	}
	if err != nil || estimate == nil || !request.CustomerQuoteUSD.Equal(estimate.EstimatedCostUSD.Mul(leonardoMediaCustomerPriceRate)) {
		return nil, ErrLeonardoVideoPricingEvidenceUnavailable
	}
	account, err := o.accounts.GetByID(ctx, request.AccountID)
	if err != nil {
		return nil, err
	}
	if !validLeonardoImageCreateAccount(account, request.AccountID, request.GroupID) {
		return nil, ErrLeonardoImageCreateAccountBindingInvalid
	}
	client, err := o.clients.Build(account)
	if err != nil {
		return nil, err
	}
	reservation, err := o.funds.Reserve(ctx, LeonardoImageFundsReserveRequest{UserID: request.UserID, PublicID: publicID, AmountUSD: request.CustomerQuoteUSD, PricingVersion: estimate.PricingVersion, PricingSource: estimate.PricingSource, PricingMatchType: estimate.MatchType})
	if err != nil {
		return nil, err
	}
	if reservation == nil || strings.TrimSpace(reservation.Reference) == "" || reservation.UserID != request.UserID || reservation.PublicID != publicID || !reservation.AmountUSD.Equal(request.CustomerQuoteUSD) || reservation.PricingVersion != estimate.PricingVersion || reservation.PricingSource != estimate.PricingSource || reservation.PricingMatchType != estimate.MatchType {
		if reservation != nil && reservation.UserID == request.UserID && reservation.PublicID == publicID && reservation.AmountUSD.Equal(request.CustomerQuoteUSD) && strings.TrimSpace(reservation.Reference) != "" {
			return nil, errors.Join(ErrLeonardoImageCreateReservationInvalid, o.release(ctx, request.UserID, publicID, reservation.Reference, request.CustomerQuoteUSD, "invalid_reservation_response"))
		}
		return nil, ErrLeonardoImageCreateReservationInvalid
	}
	if reservation.AlreadyReserved {
		existing, getErr := o.jobs.GetByPublicID(ctx, publicID)
		if getErr != nil {
			return nil, errors.Join(ErrLeonardoImageCreateReservationConflict, getErr)
		}
		if existing == nil || existing.Modality != "video" || existing.RequestHash != requestHash || existing.BillingReference == nil || *existing.BillingReference != reservation.Reference {
			return nil, ErrLeonardoImageCreateReservationConflict
		}
		return existing, nil
	}
	estimatedCost, customerCost, unit, reference := estimate.EstimatedCostUSD, request.CustomerQuoteUSD, "USD", strings.TrimSpace(reservation.Reference)
	job := &GenerationJob{PublicID: publicID, Provider: PlatformLeonardo, Modality: "video", Model: model, UpstreamModel: model, UserID: request.UserID, APIKeyID: request.APIKeyID, GroupID: request.GroupID, AccountID: account.ID, RequestHash: requestHash, EstimatedUpstreamCostAmount: &estimatedCost, EstimatedUpstreamCostUnit: &unit, PricingSnapshotVersion: &estimate.PricingVersion, PricingSource: &estimate.PricingSource, PricingMatchType: &estimate.MatchType, CustomerCost: &customerCost, BillingStatus: GenerationJobBillingStatusReserved, BillingReference: &reference}
	parameters, parameterErr := LeonardoVideoGenerationParameters(model, prompt, request.Duration, request.Width, request.Height, request.Quantity)
	if parameterErr != nil {
		return nil, errors.Join(ErrLeonardoVideoPricingEvidenceUnavailable, o.release(ctx, request.UserID, publicID, reference, customerCost, "video_parameters_invalid"))
	}
	if len(request.RawBody) > 0 {
		body := request.RawBody
		if upstreamModel := LeonardoVideoUpstreamModel(model); upstreamModel != model {
			var payload map[string]any
			if json.Unmarshal(body, &payload) != nil {
				return nil, errors.Join(ErrLeonardoImageReferenceInvalid, o.release(ctx, request.UserID, publicID, reference, customerCost, "video_request_invalid"))
			}
			payload["model"] = upstreamModel
			body, _ = json.Marshal(payload)
		}
		sources, sourceErr := ParseLeonardoRawImageSources(body)
		if sourceErr != nil || ValidateLeonardoVideoV2Sources(model, sources) != nil {
			return nil, errors.Join(ErrLeonardoImageReferenceInvalid, o.release(ctx, request.UserID, publicID, reference, customerCost, "video_guidance_invalid"))
		}
		if len(sources) > 0 {
			uploadClient, ok := client.(LeonardoInitImageClient)
			if !ok || o.uploads == nil {
				return nil, errors.Join(ErrLeonardoImageUploadInvalid, o.release(ctx, request.UserID, publicID, reference, customerCost, "image_upload_failed"))
			}
			for _, source := range sources {
				input, resolveErr := ResolveLeonardoRawImageSource(ctx, source.Value, nil, 20<<20)
				if resolveErr != nil {
					return nil, errors.Join(resolveErr, o.release(ctx, request.UserID, publicID, reference, customerCost, "image_upload_failed"))
				}
				uploadedID, uploadErr := o.uploads.Upload(ctx, account.ID, uploadClient, input)
				if uploadErr != nil {
					return nil, errors.Join(uploadErr, o.release(ctx, request.UserID, publicID, reference, customerCost, "image_upload_failed"))
				}
				body, sourceErr = SetLeonardoRawImageID(body, source, uploadedID)
				if sourceErr != nil {
					return nil, errors.Join(sourceErr, o.release(ctx, request.UserID, publicID, reference, customerCost, "video_guidance_invalid"))
				}
			}
		}
		result, submitErr := NewLeonardoGenerationService(o.jobs, client).CreateGenerationRaw(ctx, job, body)
		return o.finish(ctx, result, submitErr, request.UserID, publicID, reference, customerCost)
	}
	result, submitErr := NewLeonardoGenerationService(o.jobs, client).CreateGeneration(ctx, job, leonardo.CreateGenerationRequest{Model: model, Public: request.Public, Parameters: parameters})
	return o.finish(ctx, result, submitErr, request.UserID, publicID, reference, customerCost)
}

func (o *LeonardoImageCreateOrchestrator) finish(ctx context.Context, job *GenerationJob, submitErr error, userID int64, publicID, reference string, amount decimal.Decimal) (*GenerationJob, error) {
	if submitErr == nil {
		return job, nil
	}
	reason := ""
	if job == nil {
		reason = "job_create_failed"
	} else if job.Status == GenerationJobStatusCreated {
		reason = "job_submit_gate_failed"
	} else if job.Status == GenerationJobStatusFailed && errors.Is(submitErr, ErrLeonardoGenerationRequestNotWritten) {
		reason = "request_not_written"
	}
	if reason == "" {
		if job != nil && job.Status == GenerationJobStatusSubmitting {
			unknown := *job
			unknown.Status = GenerationJobStatusUnknown
			NormalizeGenerationJob(&unknown)
			if err := o.jobs.CompareAndSwapStatus(context.WithoutCancel(ctx), publicID, GenerationJobStatusSubmitting, &unknown); err != nil {
				return job, errors.Join(submitErr, err)
			}
			return &unknown, submitErr
		}
		return job, submitErr
	}
	err := o.release(ctx, userID, publicID, reference, amount, reason)
	if err != nil {
		return job, errors.Join(submitErr, err)
	}
	if job != nil {
		refunded := *job
		refunded.BillingStatus = GenerationJobBillingStatusRefunded
		err = o.jobs.CompareAndSwapStatus(context.WithoutCancel(ctx), publicID, job.Status, &refunded)
		job = &refunded
	}
	return job, errors.Join(submitErr, err)
}

func (o *LeonardoImageCreateOrchestrator) release(ctx context.Context, userID int64, publicID, reference string, amount decimal.Decimal, reason string) error {
	return o.funds.Release(context.WithoutCancel(ctx), LeonardoImageFundsReleaseRequest{UserID: userID, PublicID: publicID, Reference: strings.TrimSpace(reference), AmountUSD: amount, Reason: reason})
}

func validLeonardoImageCreateAccount(account *Account, accountID int64, groupID *int64) bool {
	if account == nil || account.ID != accountID || account.Platform != PlatformLeonardo || account.Type != AccountTypeAPIKey || !account.IsSchedulable() {
		return false
	}
	if groupID == nil {
		return true
	}
	for _, id := range account.GroupIDs {
		if id == *groupID {
			return true
		}
	}
	for _, relation := range account.AccountGroups {
		if relation.GroupID == *groupID {
			return true
		}
	}
	for _, group := range account.Groups {
		if group != nil && group.ID == *groupID {
			return true
		}
	}
	return false
}

func validLeonardoImageReservation(reservation *LeonardoImageFundsReservation, userID int64, publicID string, quote *LeonardoImageQuote) bool {
	return reservation != nil && strings.TrimSpace(reservation.Reference) != "" && reservation.UserID == userID && reservation.PublicID == publicID && reservation.AmountUSD.Equal(quote.CustomerQuoteUSD) && reservation.PricingVersion == quote.PricingVersion && reservation.PricingSource == quote.PricingSource && reservation.PricingMatchType == quote.MatchType
}

func safeLeonardoImageReservation(reservation *LeonardoImageFundsReservation, userID int64, publicID string, amount decimal.Decimal) bool {
	return reservation != nil && strings.TrimSpace(reservation.Reference) != "" && reservation.UserID == userID && reservation.PublicID == publicID && reservation.AmountUSD.Equal(amount)
}

func sameLeonardoImageCreateJob(job *GenerationJob, request LeonardoImageCreateRequest, reference string, amount decimal.Decimal) bool {
	return job != nil && job.PublicID == strings.TrimSpace(request.PublicID) && job.UserID == request.UserID && job.APIKeyID == request.APIKeyID && job.AccountID == request.AccountID && sameOptionalLeonardoGroupID(job.GroupID, request.GroupID) && job.RequestHash == strings.TrimSpace(request.RequestHash) && job.Provider == PlatformLeonardo && job.Modality == "image" && job.Model == strings.TrimSpace(request.Model) && job.BillingReference != nil && strings.TrimSpace(*job.BillingReference) == strings.TrimSpace(reference) && job.CustomerCost != nil && job.CustomerCost.Equal(amount)
}

func sameOptionalLeonardoGroupID(left, right *int64) bool {
	return (left == nil && right == nil) || (left != nil && right != nil && *left == *right)
}
