package service

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/shopspring/decimal"
)

var ErrMediaProviderUnavailable = errors.New("media provider adapter unavailable")
var ErrMediaOfferExhausted = errors.New("media offer exhausted")
var ErrMediaIdempotencyConflict = errors.New("idempotency key reused with different media request")

type MediaRoute struct {
	Product MediaCatalogProduct
	Offer   MediaCatalogOffer
}

type MediaOrchestrator struct {
	catalog  *MediaCatalogService
	jobs     GenerationJobRepository
	accounts AccountRepository
	funds    *MediaFundsService
	attempts MediaJobAttemptRepository
	usage    *MediaUsageAuditService
	client   *dbent.Client
	leaser   *MediaAccountLeaser
	adapters map[string]MediaProviderAdapter
}

type MediaSubmitInput struct {
	PublicID             string
	UserID               int64
	APIKeyID             int64
	GroupID              int64
	RequestHash          string
	Request              MediaCanonicalRequest
	CustomerPriceVersion string
}

func NewMediaOrchestrator(catalog *MediaCatalogService, jobs GenerationJobRepository, accounts AccountRepository, adapters ...MediaProviderAdapter) *MediaOrchestrator {
	result := &MediaOrchestrator{catalog: catalog, jobs: jobs, accounts: accounts, adapters: map[string]MediaProviderAdapter{}}
	for _, adapter := range adapters {
		if adapter != nil {
			result.adapters[adapter.Provider()] = adapter
		}
	}
	return result
}

func ProvideMediaOrchestrator(catalog *MediaCatalogService, jobs GenerationJobRepository, accounts AccountRepository, funds *MediaFundsService, attempts MediaJobAttemptRepository, usage *MediaUsageAuditService, client *dbent.Client, concurrency *ConcurrencyService, openAI *MediaOpenAIAdapter, leonardo *MediaLeonardoAdapter) *MediaOrchestrator {
	result := NewMediaOrchestrator(catalog, jobs, accounts, openAI, leonardo)
	result.funds, result.attempts, result.usage, result.client = funds, attempts, usage, client
	result.leaser = NewMediaAccountLeaser(accounts, concurrency)
	return result
}

func (s *MediaOrchestrator) Resolve(ctx context.Context, groupID int64, request MediaCanonicalRequest) (*MediaRoute, error) {
	routes, err := s.resolveAll(ctx, groupID, request)
	if err != nil {
		return nil, err
	}
	return &routes[0], nil
}

func (s *MediaOrchestrator) resolveAll(ctx context.Context, groupID int64, request MediaCanonicalRequest) ([]MediaRoute, error) {
	if s == nil || s.catalog == nil {
		return nil, ErrMediaProductNotFound
	}
	product, err := s.catalog.GetRuntime(ctx, groupID, request.Model, request.Modality)
	if err != nil {
		return nil, err
	}
	routes := make([]MediaRoute, 0, len(product.Offers))
	for _, offer := range product.Offers {
		if !containsMediaOperation(offer.Operations, request.Operation) {
			continue
		}
		accounts, listErr := s.accounts.ListSchedulableByGroupIDAndPlatform(ctx, offer.SourceGroupID, offer.Provider)
		if listErr != nil || !hasMediaAccount(accounts, offer.UpstreamModel) {
			continue
		}
		if _, ok := s.adapters[offer.Provider]; !ok {
			continue
		}
		routes = append(routes, MediaRoute{Product: *product, Offer: offer})
	}
	if len(routes) == 0 {
		return nil, ErrNoTrustedMediaOffer
	}
	return routes, nil
}

func (s *MediaOrchestrator) Submit(ctx context.Context, input MediaSubmitInput) (*GenerationJob, MediaSubmissionOutcome, error) {
	if input.UserID <= 0 || input.APIKeyID <= 0 || input.GroupID <= 0 || strings.TrimSpace(input.PublicID) == "" {
		return nil, MediaSubmissionOutcome{}, ErrMediaProductNotFound
	}
	requestHash := strings.TrimSpace(input.RequestHash)
	if requestHash == "" {
		sum := sha256.Sum256(input.Request.Body)
		requestHash = hex.EncodeToString(sum[:])
	}
	if existing, found, lookupErr := s.LookupOwnedJob(ctx, input.PublicID, input.UserID, input.APIKeyID); lookupErr != nil {
		return nil, MediaSubmissionOutcome{}, lookupErr
	} else if found {
		if subtle.ConstantTimeCompare([]byte(existing.RequestHash), []byte(requestHash)) != 1 {
			return nil, MediaSubmissionOutcome{}, ErrMediaIdempotencyConflict
		}
		return existing, MediaSubmissionOutcome{State: MediaSubmissionSubmitted, UpstreamID: mediaStringValue(existing.UpstreamGenerationID), AccountID: existing.AccountID, Status: string(existing.Status), Result: existing.ResultPayload}, nil
	}
	product, err := s.catalog.GetRuntime(ctx, input.GroupID, input.Request.Model, input.Request.Modality)
	if err != nil {
		return nil, MediaSubmissionOutcome{}, err
	}
	selection, err := SelectMediaRuntime(*product, input.Request, time.Now().UTC())
	if err != nil {
		return nil, MediaSubmissionOutcome{}, err
	}
	routes := make([]MediaRoute, 0, len(selection.RankedEligible))
	eligible := make([]MediaOfferCandidate, 0, len(selection.RankedEligible))
	for _, candidate := range selection.RankedEligible {
		for _, offer := range product.Offers {
			if offer.ID == candidate.Offer.ID {
				accounts, listErr := s.accounts.ListSchedulableByGroupIDAndPlatform(ctx, offer.SourceGroupID, offer.Provider)
				if listErr != nil || firstMediaAccountID(accounts, offer.UpstreamModel) == 0 || s.adapters[offer.Provider] == nil {
					break
				}
				routes = append(routes, MediaRoute{Product: *product, Offer: offer})
				eligible = append(eligible, candidate)
				break
			}
		}
	}
	if len(routes) == 0 {
		return nil, MediaSubmissionOutcome{}, ErrNoTrustedMediaOffer
	}
	selection.RankedEligible = eligible
	now := time.Now().UTC()
	groupID := input.GroupID
	operation := input.Request.Operation
	productID := selection.Product.ID
	priceVersion := selection.Price.Version
	reserveRequest := MediaFundsReserveRequest{UserID: input.UserID, PublicID: input.PublicID, ProductID: productID, Amount: selection.Charge, PriceVersion: priceVersion}
	firstRoute, firstCandidate := routes[0], selection.RankedEligible[0]
	firstAccounts, err := s.accounts.ListSchedulableByGroupIDAndPlatform(ctx, firstRoute.Offer.SourceGroupID, firstRoute.Offer.Provider)
	if err != nil {
		return nil, MediaSubmissionOutcome{}, err
	}
	firstAccountID := firstMediaAccountID(firstAccounts, firstRoute.Offer.UpstreamModel)
	offerID, sourceGroupID := firstRoute.Offer.ID, firstRoute.Offer.SourceGroupID
	trustedCost := decimal.NewFromFloat(firstCandidate.TrustedCost)
	costUnit, matchType := firstCandidate.Offer.Cost.Currency, firstCandidate.Offer.Cost.Basis
	job := &GenerationJob{PublicID: strings.TrimSpace(input.PublicID), Provider: firstRoute.Offer.Provider, Modality: input.Request.Modality, Model: input.Request.Model, UpstreamModel: firstRoute.Offer.UpstreamModel, UserID: input.UserID, APIKeyID: input.APIKeyID, GroupID: &groupID, ProductID: &productID, OfferID: &offerID, SourceGroupID: &sourceGroupID, Operation: &operation, CustomerPriceVersion: &priceVersion, AccountID: firstAccountID, Status: GenerationJobStatusSubmitting, RequestHash: requestHash, RequestPayload: input.Request.Fields, ResultPayload: map[string]any{}, EstimatedUpstreamCostAmount: &trustedCost, EstimatedUpstreamCostUnit: &costUnit, PricingSnapshotVersion: &firstRoute.Offer.CostVersion, PricingSource: &firstRoute.Offer.CostSource, PricingMatchType: &matchType, BillingStatus: GenerationJobBillingStatusReserved, CreatedAt: now, UpdatedAt: now}
	_, err = s.reserveAndCreateJob(ctx, reserveRequest, job)
	if err != nil {
		if existing, found, lookupErr := s.LookupOwnedJob(ctx, input.PublicID, input.UserID, input.APIKeyID); lookupErr == nil && found {
			if subtle.ConstantTimeCompare([]byte(existing.RequestHash), []byte(requestHash)) != 1 {
				return nil, MediaSubmissionOutcome{}, ErrMediaIdempotencyConflict
			}
			return existing, MediaSubmissionOutcome{State: MediaSubmissionSubmitted, UpstreamID: mediaStringValue(existing.UpstreamGenerationID), AccountID: existing.AccountID, Status: string(existing.Status), Result: existing.ResultPayload}, nil
		}
		return nil, MediaSubmissionOutcome{}, err
	}
	for index, route := range routes {
		candidate := selection.RankedEligible[index]
		lease, leaseErr := s.acquireMediaAccount(ctx, route.Offer)
		if leaseErr != nil {
			continue
		}
		accountID := lease.Account.ID
		offerID, sourceGroupID := route.Offer.ID, route.Offer.SourceGroupID
		job.Provider, job.UpstreamModel, job.AccountID = route.Offer.Provider, route.Offer.UpstreamModel, accountID
		job.OfferID, job.SourceGroupID = &offerID, &sourceGroupID
		trustedCost := decimal.NewFromFloat(candidate.TrustedCost)
		costUnit := candidate.Offer.Cost.Currency
		job.EstimatedUpstreamCostAmount, job.EstimatedUpstreamCostUnit = &trustedCost, &costUnit
		job.PricingSnapshotVersion, job.PricingSource = &route.Offer.CostVersion, &route.Offer.CostSource
		matchType := candidate.Offer.Cost.Basis
		job.PricingMatchType = &matchType
		outcome, submitErr := s.adapters[route.Offer.Provider].Submit(ctx, job, input.Request, route.Offer)
		lease.Release()
		if outcome.AccountID == 0 {
			outcome.AccountID = accountID
		}
		attemptErr := s.recordAttempt(ctx, job, candidate, outcome, submitErr)
		if attemptErr != nil {
			if outcome.State == MediaSubmissionNotWritten {
				job.Status = GenerationJobStatusFailed
				job.ErrorMessage = stringPointer(attemptErr.Error())
				if releaseErr := s.release(ctx, job); releaseErr != nil {
					job.BillingStatus = GenerationJobBillingStatusManualReview
				}
				_ = s.jobs.CompareAndSwapStatus(ctx, job.PublicID, GenerationJobStatusSubmitting, job)
				return job, outcome, attemptErr
			}
			outcome.State = MediaSubmissionSideEffectUnknown
			submitErr = attemptErr
		}
		if submitErr != nil && outcome.State == MediaSubmissionNotWritten {
			continue
		}
		job.AccountID = outcome.AccountID
		if strings.TrimSpace(outcome.UpstreamID) != "" {
			job.UpstreamGenerationID = &outcome.UpstreamID
		}
		job.ResultPayload = outcome.Result
		if job.ResultPayload == nil {
			job.ResultPayload = map[string]any{}
		}
		switch outcome.State {
		case MediaSubmissionSubmitted:
			job.Status = mediaSubmissionJobStatus(outcome.Status)
			job.BillingStatus = GenerationJobBillingStatusSubmitted
			job.SubmittedAt = &now
			if job.Status == GenerationJobStatusSucceeded {
				job.OutputCount = mediaOutputCount(input.Request, outcome.Result)
				if err = s.settleAndAudit(ctx, job, candidate); err != nil {
					job.BillingStatus = GenerationJobBillingStatusManualReview
				}
			}
		case MediaSubmissionSideEffectUnknown:
			job.Status = GenerationJobStatusUnknown
			job.ErrorMessage = stringPointer("media generation submission status is unknown")
			NormalizeGenerationJob(job)
		default:
			continue
		}
		if err = s.jobs.CompareAndSwapStatus(ctx, job.PublicID, GenerationJobStatusSubmitting, job); err != nil {
			return nil, outcome, err
		}
		return job, outcome, submitErr
	}
	if job.ID != 0 {
		job.Status = GenerationJobStatusFailed
		code := "media_offer_exhausted"
		job.ErrorCode = &code
		if releaseErr := s.release(ctx, job); releaseErr != nil {
			job.BillingStatus = GenerationJobBillingStatusManualReview
		}
		_ = s.jobs.CompareAndSwapStatus(ctx, job.PublicID, GenerationJobStatusSubmitting, job)
	} else {
		_ = s.release(ctx, job)
	}
	return job, MediaSubmissionOutcome{State: MediaSubmissionNotWritten}, ErrMediaOfferExhausted
}

func (s *MediaOrchestrator) acquireMediaAccount(ctx context.Context, offer MediaCatalogOffer) (*MediaAccountLease, error) {
	if s.leaser != nil {
		return s.leaser.Acquire(ctx, offer.SourceGroupID, offer.Provider, offer.UpstreamModel, nil)
	}
	accounts, err := s.accounts.ListSchedulableByGroupIDAndPlatform(ctx, offer.SourceGroupID, offer.Provider)
	if err != nil {
		return nil, err
	}
	for i := range accounts {
		if accounts[i].IsSchedulable() && accounts[i].IsModelSupported(offer.UpstreamModel) {
			return &MediaAccountLease{Account: &accounts[i]}, nil
		}
	}
	return nil, ErrMediaAccountUnavailable
}

func (s *MediaOrchestrator) reserveAndCreateJob(ctx context.Context, request MediaFundsReserveRequest, job *GenerationJob) (*MediaFundsReservation, error) {
	if s.client == nil {
		reservation, err := s.funds.Reserve(ctx, request)
		if err != nil {
			return nil, err
		}
		job.CustomerCost, job.BillingReference = &reservation.Amount, &reservation.Reference
		if err = s.jobs.Create(ctx, job); err != nil {
			if !reservation.AlreadyExists {
				_ = s.funds.Release(ctx, MediaFundsTransitionRequest{UserID: request.UserID, PublicID: request.PublicID, Reference: reservation.Reference, Amount: reservation.Amount})
			}
			return nil, err
		}
		return reservation, nil
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	reservation, err := s.funds.Reserve(txCtx, request)
	if err != nil {
		return nil, err
	}
	job.CustomerCost, job.BillingReference = &reservation.Amount, &reservation.Reference
	if err = s.jobs.Create(txCtx, job); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return reservation, nil
}

func (s *MediaOrchestrator) Poll(ctx context.Context, job *GenerationJob) (map[string]any, error) {
	if job == nil {
		return nil, ErrGenerationJobNotFound
	}
	adapter := s.adapters[job.Provider]
	if adapter == nil {
		return nil, ErrMediaProviderUnavailable
	}
	result, err := adapter.Poll(ctx, job)
	if err != nil || result == nil {
		return result, err
	}
	status := NormalizeMediaStatusToken(mediaStringValueAny(result["status"]))
	if !IsTerminalMediaSuccessStatus(status) && !IsTerminalMediaFailureStatus(status) {
		return result, nil
	}
	latest, latestErr := s.jobs.GetByPublicID(ctx, job.PublicID)
	if latestErr != nil {
		return nil, latestErr
	}
	if latest == nil {
		return nil, ErrGenerationJobNotFound
	}
	updated := *latest
	updated.ResultPayload = result
	if IsTerminalMediaSuccessStatus(status) {
		updated.Status = GenerationJobStatusSucceeded
		updated.OutputCount = mediaOutputCount(MediaCanonicalRequest{Modality: latest.Modality, Fields: latest.RequestPayload}, result)
		candidate := mediaCandidateFromJob(latest)
		if err = s.settleAndAudit(ctx, &updated, candidate); err != nil {
			updated.BillingStatus = GenerationJobBillingStatusManualReview
		}
	} else {
		updated.Status = GenerationJobStatusFailed
		if err = s.release(ctx, &updated); err != nil {
			updated.BillingStatus = GenerationJobBillingStatusManualReview
		}
	}
	if casErr := s.jobs.CompareAndSwapStatus(ctx, latest.PublicID, latest.Status, &updated); casErr != nil {
		if !errors.Is(casErr, ErrGenerationJobConflict) {
			return nil, casErr
		}
		reloaded, reloadErr := s.jobs.GetByPublicID(ctx, latest.PublicID)
		if reloadErr != nil {
			return nil, reloadErr
		}
		*job = *reloaded
		return result, err
	}
	*job = updated
	return result, err
}

func (s *MediaOrchestrator) Content(ctx context.Context, job *GenerationJob, index int) (*MediaContent, error) {
	if job == nil || index < 0 {
		return nil, ErrGenerationJobNotFound
	}
	adapter := s.adapters[job.Provider]
	if adapter == nil {
		return nil, ErrMediaProviderUnavailable
	}
	return adapter.Content(ctx, job, index)
}

func (s *MediaOrchestrator) LookupOwnedJob(ctx context.Context, publicID string, userID, apiKeyID int64) (*GenerationJob, bool, error) {
	if s == nil || s.jobs == nil || strings.TrimSpace(publicID) == "" {
		return nil, false, nil
	}
	job, err := s.jobs.GetByPublicID(ctx, strings.TrimSpace(publicID))
	if errors.Is(err, ErrGenerationJobNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if job.ProductID == nil || job.UserID != userID || job.APIKeyID != apiKeyID {
		return nil, false, nil
	}
	return job, true, nil
}

func (s *MediaOrchestrator) recordAttempt(ctx context.Context, job *GenerationJob, candidate MediaOfferCandidate, outcome MediaSubmissionOutcome, submitErr error) error {
	if s.attempts == nil || job == nil || job.ID <= 0 {
		return ErrMediaAttemptInvalid
	}
	accountID := outcome.AccountID
	attempt := &MediaJobAttempt{JobID: job.ID, OfferID: candidate.Offer.ID, Provider: candidate.Offer.Provider, SourceGroupID: candidate.Offer.SourceGroupID, UpstreamModel: candidate.Offer.UpstreamModel, TrustedCostSnapshot: map[string]any{"amount": candidate.TrustedCost, "currency": candidate.Offer.Cost.Currency, "basis": candidate.Offer.Cost.Basis, "source": candidate.Offer.Cost.Source, "version": candidate.Offer.Cost.Version}, SubmissionState: outcome.State}
	if accountID > 0 {
		attempt.AccountID = &accountID
	}
	if submitErr != nil {
		code, message := "upstream_submit_error", submitErr.Error()
		attempt.ErrorCode, attempt.ErrorMessage = &code, &message
	}
	return s.attempts.Create(ctx, attempt)
}

func (s *MediaOrchestrator) settleAndAudit(ctx context.Context, job *GenerationJob, candidate MediaOfferCandidate) error {
	if job == nil || job.CustomerCost == nil || job.BillingReference == nil {
		return ErrMediaReservationInvalid
	}
	transition := MediaFundsTransitionRequest{UserID: job.UserID, PublicID: job.PublicID, Reference: *job.BillingReference, Amount: *job.CustomerCost}
	if err := s.funds.Settle(ctx, transition); err != nil {
		return err
	}
	job.BillingStatus = GenerationJobBillingStatusSettled
	trustedCost := candidate.TrustedCost
	if trustedCost <= 0 && job.EstimatedUpstreamCostAmount != nil {
		trustedCost = job.EstimatedUpstreamCostAmount.InexactFloat64()
	}
	accountID := job.AccountID
	imageSize := ImageBillingSizeResolution{}
	imageCount := 0
	if job.Modality == string(MediaModalityImage) {
		imageSize = ResolveImageBillingSize(mediaUsageInputSize(job), nil)
		imageCount = max(job.OutputCount, 1)
	}
	_, err := s.usage.Write(ctx, UsageLogDraft{RequestID: job.PublicID, APIKeyID: job.APIKeyID, UserID: job.UserID, CustomerGroupID: job.GroupID, AccountID: &accountID, RequestedModel: job.Model, UpstreamModel: job.UpstreamModel, MediaType: job.Modality, ImageCount: imageCount, ImageSize: imageSize, ActualCost: job.CustomerCost.InexactFloat64(), ProductID: pointerInt64(job.ProductID), OfferID: pointerInt64(job.OfferID), UpstreamPlatform: job.Provider, SourceGroupID: pointerInt64(job.SourceGroupID), TrustedCost: trustedCost, TrustedCostUnit: mediaStringValue(job.EstimatedUpstreamCostUnit), TrustedCostSource: mediaStringValue(job.PricingSource), TrustedCostVersion: mediaStringValue(job.PricingSnapshotVersion), CustomerPriceVersion: mediaStringValue(job.CustomerPriceVersion)})
	return err
}

func mediaUsageInputSize(job *GenerationJob) string {
	if job == nil {
		return ""
	}
	if width, height := mediaUnifiedDimensions(job.RequestPayload); width > 0 && height > 0 {
		return strconv.Itoa(width) + "x" + strconv.Itoa(height)
	}
	if job.Modality == string(MediaModalityImage) && job.Provider == PlatformLeonardo {
		width, height := leonardoUnifiedImageDefaultDimensions(job.UpstreamModel)
		return strconv.Itoa(width) + "x" + strconv.Itoa(height)
	}
	return ""
}

func (s *MediaOrchestrator) release(ctx context.Context, job *GenerationJob) error {
	if s == nil || s.funds == nil || job == nil || job.CustomerCost == nil || job.BillingReference == nil {
		return ErrMediaReservationInvalid
	}
	err := s.funds.Release(ctx, MediaFundsTransitionRequest{UserID: job.UserID, PublicID: job.PublicID, Reference: *job.BillingReference, Amount: *job.CustomerCost})
	if err == nil {
		job.BillingStatus = GenerationJobBillingStatusRefunded
	}
	return err
}

func mediaCandidateFromJob(job *GenerationJob) MediaOfferCandidate {
	if job == nil {
		return MediaOfferCandidate{}
	}
	candidate := MediaOfferCandidate{Offer: MediaOffer{ID: pointerInt64(job.OfferID), Provider: job.Provider, SourceGroupID: pointerInt64(job.SourceGroupID), UpstreamModel: job.UpstreamModel, Cost: TrustedCostPolicy{Basis: mediaStringValue(job.PricingMatchType), Currency: mediaStringValue(job.EstimatedUpstreamCostUnit), Source: mediaStringValue(job.PricingSource), Version: mediaStringValue(job.PricingSnapshotVersion)}}}
	if job.EstimatedUpstreamCostAmount != nil {
		candidate.TrustedCost = job.EstimatedUpstreamCostAmount.InexactFloat64()
	}
	return candidate
}

func mediaOutputCount(request MediaCanonicalRequest, result map[string]any) int {
	if data, ok := result["data"].([]any); ok && len(data) > 0 {
		return len(data)
	}
	return mediaRequestQuantity(request.Fields)
}

func pointerInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func containsMediaOperation(operations []string, operation string) bool {
	for _, value := range operations {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(operation)) {
			return true
		}
	}
	return false
}

func hasMediaAccount(accounts []Account, model string) bool {
	for i := range accounts {
		if accounts[i].IsSchedulable() && accounts[i].IsModelSupported(model) {
			return true
		}
	}
	return false
}

func firstMediaAccountID(accounts []Account, model string) int64 {
	for i := range accounts {
		if accounts[i].IsSchedulable() && accounts[i].IsModelSupported(model) {
			return accounts[i].ID
		}
	}
	return 0
}

func mediaSubmissionJobStatus(status string) GenerationJobStatus {
	switch NormalizeMediaStatusToken(status) {
	case "completed", "succeeded", "done", "ready":
		return GenerationJobStatusSucceeded
	case "running", "processing", "in_progress":
		return GenerationJobStatusRunning
	default:
		return GenerationJobStatusQueued
	}
}

func stringPointerOrNil(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func mediaStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
