package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type mediaJobLookupRepository struct {
	job         *GenerationJob
	createCalls int
}

func (r *mediaJobLookupRepository) Create(_ context.Context, job *GenerationJob) error {
	r.createCalls++
	job.ID = 99
	r.job = job
	return nil
}
func (r *mediaJobLookupRepository) GetByPublicID(context.Context, string) (*GenerationJob, error) {
	if r.job == nil {
		return nil, ErrGenerationJobNotFound
	}
	return r.job, nil
}

type mediaRuntimeCatalogRepo struct {
	product *MediaCatalogProduct
}

func (r *mediaRuntimeCatalogRepo) List(context.Context, int, int, string, string) ([]MediaCatalogProduct, int64, error) {
	return nil, 0, nil
}
func (r *mediaRuntimeCatalogRepo) GetByID(context.Context, int64) (*MediaCatalogProduct, error) {
	return r.product, nil
}
func (r *mediaRuntimeCatalogRepo) GetRuntime(context.Context, int64, string, string, time.Time) (*MediaCatalogProduct, error) {
	return r.product, nil
}
func (r *mediaRuntimeCatalogRepo) ListRuntimeModels(context.Context, int64, time.Time) ([]string, error) {
	return nil, nil
}
func (r *mediaRuntimeCatalogRepo) GetGroups(context.Context, []int64) (map[int64]MediaCatalogGroup, error) {
	return nil, nil
}
func (r *mediaRuntimeCatalogRepo) Create(context.Context, *MediaCatalogProduct) error { return nil }
func (r *mediaRuntimeCatalogRepo) Update(context.Context, *MediaCatalogProduct) error { return nil }
func (r *mediaRuntimeCatalogRepo) Disable(context.Context, int64) error               { return nil }

type mediaRuntimeAccountRepo struct{ AccountRepository }

func (r *mediaRuntimeAccountRepo) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	return []Account{{ID: 71, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true}}, nil
}

type mediaRuntimeAdapter struct {
	calls    int
	outcomes []MediaSubmissionOutcome
	errors   []error
	poll     map[string]any
}

func (a *mediaRuntimeAdapter) Provider() string { return PlatformOpenAI }
func (a *mediaRuntimeAdapter) Submit(context.Context, *GenerationJob, MediaCanonicalRequest, MediaCatalogOffer) (MediaSubmissionOutcome, error) {
	a.calls++
	if len(a.outcomes) >= a.calls {
		return a.outcomes[a.calls-1], a.errors[a.calls-1]
	}
	return MediaSubmissionOutcome{State: MediaSubmissionSubmitted, UpstreamID: "up_1", AccountID: 71, Status: "queued"}, nil
}
func (a *mediaRuntimeAdapter) Poll(context.Context, *GenerationJob) (map[string]any, error) {
	return a.poll, nil
}
func (a *mediaRuntimeAdapter) Content(context.Context, *GenerationJob, int) (*MediaContent, error) {
	return nil, nil
}

func TestMediaOrchestratorSubmitPersistsStickyRoute(t *testing.T) {
	now := time.Now().UTC()
	product := &MediaCatalogProduct{ID: 9, PublicModel: "public-image", Modality: "image", Enabled: true, Prices: []MediaCatalogPrice{{Operation: "generations", SpecKey: "generations|n=1", UnitPriceUSD: decimal.RequireFromString("0.10"), Currency: "USD", Version: "v1", Enabled: true}}, Offers: []MediaCatalogOffer{{ID: 12, Provider: PlatformOpenAI, SourceGroupID: 5, UpstreamModel: "upstream-image", Enabled: true, Operations: []string{"generations"}, Capabilities: map[string]any{"supported_fields": map[string]any{}}, CostRules: map[string]any{"basis": "per_image", "unit_cost": 0.02, "currency": "USD"}, CostSource: "probe", CostVersion: "cost-v1", VerifiedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour)}}}
	catalog := NewMediaCatalogService(&mediaRuntimeCatalogRepo{product: product})
	repo := &mediaJobLookupRepository{}
	adapter := &mediaRuntimeAdapter{}
	orchestrator := NewMediaOrchestrator(catalog, repo, &mediaRuntimeAccountRepo{}, adapter)
	orchestrator.funds = NewMediaFundsService(&mediaFundsRepoStub{})
	orchestrator.attempts = &mediaAttemptRepoStub{}
	orchestrator.usage = NewMediaUsageAuditService(&mediaUsageAuditRepoStub{})

	job, outcome, err := orchestrator.Submit(context.Background(), MediaSubmitInput{PublicID: "media_rq_1", UserID: 1, APIKeyID: 2, GroupID: 3, Request: MediaCanonicalRequest{Operation: "generations", Model: "public-image", Modality: "image", Body: []byte(`{"model":"public-image"}`), Fields: map[string]any{"model": "public-image"}}})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.UpstreamID != "up_1" || job.OfferID == nil || *job.OfferID != 12 || job.SourceGroupID == nil || *job.SourceGroupID != 5 || job.AccountID != 71 || job.Provider != PlatformOpenAI {
		t.Fatalf("sticky route not persisted: %#v %#v", job, outcome)
	}
	if adapter.calls != 1 {
		t.Fatalf("adapter calls = %d", adapter.calls)
	}
	if repo.createCalls != 1 {
		t.Fatalf("job creates = %d", repo.createCalls)
	}
}
func (r *mediaJobLookupRepository) GetByUpstreamGenerationID(context.Context, string) (*GenerationJob, error) {
	return nil, ErrGenerationJobNotFound
}
func (r *mediaJobLookupRepository) CompareAndSwapStatus(context.Context, string, GenerationJobStatus, *GenerationJob) error {
	return nil
}

type mediaAttemptRepoStub struct{ attempts []MediaJobAttempt }

func (r *mediaAttemptRepoStub) Create(_ context.Context, attempt *MediaJobAttempt) error {
	r.attempts = append(r.attempts, *attempt)
	return nil
}

type mediaRuntimeFundsRepo struct {
	reserveCalls int
	settleCalls  int
	releaseCalls int
}

func (r *mediaRuntimeFundsRepo) Reserve(_ context.Context, request MediaFundsReserveRequest) (*MediaFundsReservation, error) {
	r.reserveCalls++
	return &MediaFundsReservation{Reference: "hold-1", UserID: request.UserID, PublicID: request.PublicID, ProductID: request.ProductID, Amount: request.Amount, PriceVersion: request.PriceVersion, Status: "reserved", AlreadyExists: r.reserveCalls > 1}, nil
}
func (r *mediaRuntimeFundsRepo) Settle(context.Context, MediaFundsTransitionRequest) error {
	r.settleCalls++
	return nil
}
func (r *mediaRuntimeFundsRepo) Release(context.Context, MediaFundsTransitionRequest) error {
	r.releaseCalls++
	return nil
}

func TestMediaRuntimeSelectorUsesExactSpecAndTrustedCostOrder(t *testing.T) {
	now := time.Now().UTC()
	request := MediaCanonicalRequest{Operation: "generations", Model: "public-image", Modality: "image", Fields: map[string]any{"model": "public-image", "size": "1024x1024", "quality": "high", "n": float64(2)}}
	spec := MediaSpecKey(request)
	product := MediaCatalogProduct{ID: 9, PublicModel: "public-image", Modality: "image", Enabled: true, Prices: []MediaCatalogPrice{{Operation: "generations", SpecKey: spec, UnitPriceUSD: decimal.RequireFromString("0.10"), Currency: "USD", Version: "v2", Enabled: true}}}
	for id, cost := range []float64{0.04, 0.02} {
		product.Offers = append(product.Offers, MediaCatalogOffer{ID: int64(id + 1), Provider: PlatformOpenAI, SourceGroupID: int64(id + 5), UpstreamModel: "upstream-image", Enabled: true, Operations: []string{"generations"}, Capabilities: map[string]any{"max_n": float64(4), "supported_fields": map[string]any{"size": map[string]any{"enum": []any{"1024x1024"}}, "quality": map[string]any{"enum": []any{"high"}}, "n": map[string]any{"max": float64(4)}}}, CostRules: map[string]any{"basis": "per_image", "unit_cost": cost, "currency": "USD"}, CostSource: "probe", CostVersion: "cost-v1", VerifiedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour)})
	}
	selection, err := SelectMediaRuntime(product, request, now)
	if err != nil {
		t.Fatal(err)
	}
	if !selection.Charge.Equal(decimal.RequireFromString("0.20")) || selection.RankedEligible[0].Offer.ID != 2 || selection.RankedEligible[0].TrustedCost != 0.04 {
		t.Fatalf("selection = %#v", selection)
	}
	request.Fields["quality"] = "ultra"
	if _, err = SelectMediaRuntime(product, request, now); !errors.Is(err, ErrMediaCustomerPrice) {
		t.Fatalf("missing exact price err = %v", err)
	}
}

func TestMediaOrchestratorFailoverReservesOnceAndAuditsAttempts(t *testing.T) {
	now := time.Now().UTC()
	product := &MediaCatalogProduct{ID: 9, PublicModel: "public-image", Modality: "image", Enabled: true, Prices: []MediaCatalogPrice{{Operation: "generations", SpecKey: "generations|n=1", UnitPriceUSD: decimal.RequireFromString("0.10"), Currency: "USD", Version: "v1", Enabled: true}}}
	for id, cost := range []float64{0.01, 0.02} {
		product.Offers = append(product.Offers, MediaCatalogOffer{ID: int64(id + 1), Provider: PlatformOpenAI, SourceGroupID: int64(id + 5), UpstreamModel: "upstream-image", Enabled: true, Operations: []string{"generations"}, Capabilities: map[string]any{"supported_fields": map[string]any{}}, CostRules: map[string]any{"basis": "per_image", "unit_cost": cost, "currency": "USD"}, CostSource: "probe", CostVersion: "cost-v1", VerifiedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour)})
	}
	repo := &mediaJobLookupRepository{}
	funds := &mediaRuntimeFundsRepo{}
	attempts := &mediaAttemptRepoStub{}
	adapter := &mediaRuntimeAdapter{outcomes: []MediaSubmissionOutcome{{State: MediaSubmissionNotWritten, AccountID: 71}, {State: MediaSubmissionSubmitted, UpstreamID: "up-2", AccountID: 71, Status: "queued"}}, errors: []error{errors.New("connect refused"), nil}}
	orchestrator := NewMediaOrchestrator(NewMediaCatalogService(&mediaRuntimeCatalogRepo{product: product}), repo, &mediaRuntimeAccountRepo{}, adapter)
	orchestrator.funds, orchestrator.attempts, orchestrator.usage = NewMediaFundsService(funds), attempts, NewMediaUsageAuditService(&mediaUsageAuditRepoStub{})
	job, _, err := orchestrator.Submit(context.Background(), MediaSubmitInput{PublicID: "media_rq_failover", UserID: 1, APIKeyID: 2, GroupID: 3, Request: MediaCanonicalRequest{Operation: "generations", Model: "public-image", Modality: "image", Fields: map[string]any{"model": "public-image"}}})
	if err != nil {
		t.Fatal(err)
	}
	if funds.reserveCalls != 1 || funds.settleCalls != 0 || funds.releaseCalls != 0 || adapter.calls != 2 || len(attempts.attempts) != 2 || attempts.attempts[0].SubmissionState != MediaSubmissionNotWritten || job.OfferID == nil || *job.OfferID != 2 {
		t.Fatalf("funds=%#v calls=%d attempts=%#v job=%#v", funds, adapter.calls, attempts.attempts, job)
	}
}

func TestMediaOrchestratorUnknownNeverFailoversAndExhaustionReleases(t *testing.T) {
	now := time.Now().UTC()
	product := &MediaCatalogProduct{ID: 9, PublicModel: "public-image", Modality: "image", Enabled: true, Prices: []MediaCatalogPrice{{Operation: "generations", SpecKey: "generations|n=1", UnitPriceUSD: decimal.RequireFromString("0.10"), Currency: "USD", Version: "v1", Enabled: true}}, Offers: []MediaCatalogOffer{{ID: 1, Provider: PlatformOpenAI, SourceGroupID: 5, UpstreamModel: "upstream-image", Enabled: true, Operations: []string{"generations"}, Capabilities: map[string]any{"supported_fields": map[string]any{}}, CostRules: map[string]any{"basis": "per_image", "unit_cost": 0.01}, CostSource: "probe", CostVersion: "v1", VerifiedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour)}, {ID: 2, Provider: PlatformOpenAI, SourceGroupID: 6, UpstreamModel: "upstream-image", Enabled: true, Operations: []string{"generations"}, Capabilities: map[string]any{"supported_fields": map[string]any{}}, CostRules: map[string]any{"basis": "per_image", "unit_cost": 0.02}, CostSource: "probe", CostVersion: "v1", VerifiedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour)}}}
	for _, unknown := range []bool{true, false} {
		repo, funds, attempts := &mediaJobLookupRepository{}, &mediaRuntimeFundsRepo{}, &mediaAttemptRepoStub{}
		state := MediaSubmissionNotWritten
		if unknown {
			state = MediaSubmissionSideEffectUnknown
		}
		adapter := &mediaRuntimeAdapter{outcomes: []MediaSubmissionOutcome{{State: state, AccountID: 71}, {State: state, AccountID: 71}}, errors: []error{errors.New("submit failed"), errors.New("submit failed")}}
		orchestrator := NewMediaOrchestrator(NewMediaCatalogService(&mediaRuntimeCatalogRepo{product: product}), repo, &mediaRuntimeAccountRepo{}, adapter)
		orchestrator.funds, orchestrator.attempts, orchestrator.usage = NewMediaFundsService(funds), attempts, NewMediaUsageAuditService(&mediaUsageAuditRepoStub{})
		job, _, _ := orchestrator.Submit(context.Background(), MediaSubmitInput{PublicID: "media_rq_terminal", UserID: 1, APIKeyID: 2, GroupID: 3, Request: MediaCanonicalRequest{Operation: "generations", Model: "public-image", Modality: "image", Fields: map[string]any{"model": "public-image"}}})
		if unknown && (adapter.calls != 1 || funds.releaseCalls != 0 || job.Status != GenerationJobStatusUnknown) {
			t.Fatalf("unknown branch calls=%d funds=%#v job=%#v", adapter.calls, funds, job)
		}
		if !unknown && (adapter.calls != 2 || funds.releaseCalls != 1 || job.Status != GenerationJobStatusFailed) {
			t.Fatalf("exhausted branch calls=%d funds=%#v job=%#v", adapter.calls, funds, job)
		}
	}
}
func (r *mediaAttemptRepoStub) ListByJobID(context.Context, int64) ([]MediaJobAttempt, error) {
	return r.attempts, nil
}

type mediaUsageAuditErrorRepo struct{ err error }

func (r mediaUsageAuditErrorRepo) CreateMediaUsageAudit(context.Context, *UsageLog) (bool, error) {
	return false, r.err
}

func TestMediaOrchestratorPollPersistsTerminalManualReviewAfterAuditFailure(t *testing.T) {
	productID, offerID, sourceGroupID, groupID := int64(9), int64(12), int64(5), int64(3)
	amount := decimal.RequireFromString("0.10")
	reference := "hold-1"
	unit, source, version := "USD", "probe", "cost-v1"
	repo := &mediaJobLookupRepository{job: &GenerationJob{PublicID: "media_rq_poll", UserID: 1, APIKeyID: 2, GroupID: &groupID, ProductID: &productID, OfferID: &offerID, SourceGroupID: &sourceGroupID, Provider: PlatformOpenAI, Modality: "image", Model: "public-image", UpstreamModel: "upstream-image", AccountID: 71, Status: GenerationJobStatusRunning, BillingStatus: GenerationJobBillingStatusSubmitted, CustomerCost: &amount, BillingReference: &reference, EstimatedUpstreamCostAmount: &amount, EstimatedUpstreamCostUnit: &unit, PricingSource: &source, PricingSnapshotVersion: &version, RequestPayload: map[string]any{"size": "2048x1152"}}}
	adapter := &mediaRuntimeAdapter{poll: map[string]any{"status": "completed", "data": []any{map[string]any{"url": "https://example.com/image.jpg"}}}}
	orchestrator := NewMediaOrchestrator(nil, repo, nil, adapter)
	orchestrator.funds = NewMediaFundsService(&mediaRuntimeFundsRepo{})
	orchestrator.usage = NewMediaUsageAuditService(mediaUsageAuditErrorRepo{err: errors.New("audit unavailable")})

	result, err := orchestrator.Poll(context.Background(), repo.job)
	require.Error(t, err)
	require.Equal(t, "completed", result["status"])
	require.Equal(t, GenerationJobStatusSucceeded, repo.job.Status)
	require.Equal(t, GenerationJobBillingStatusManualReview, repo.job.BillingStatus)
}

func TestMediaOrchestratorLookupOwnedJobRequiresUnifiedOwnership(t *testing.T) {
	productID := int64(7)
	repo := &mediaJobLookupRepository{job: &GenerationJob{PublicID: "job_1", ProductID: &productID, UserID: 11, APIKeyID: 22}}
	orchestrator := NewMediaOrchestrator(nil, repo, nil)

	job, found, err := orchestrator.LookupOwnedJob(context.Background(), "job_1", 11, 22)
	if err != nil || !found || job == nil {
		t.Fatalf("LookupOwnedJob() = %#v, %v, %v", job, found, err)
	}
	if _, found, err = orchestrator.LookupOwnedJob(context.Background(), "job_1", 11, 23); err != nil || found {
		t.Fatalf("foreign API key lookup found=%v err=%v", found, err)
	}
	repo.job.ProductID = nil
	if _, found, err = orchestrator.LookupOwnedJob(context.Background(), "job_1", 11, 22); err != nil || found {
		t.Fatalf("legacy job lookup found=%v err=%v", found, err)
	}
}

func TestMediaOrchestratorRejectsIdempotencyKeyWithDifferentRequest(t *testing.T) {
	productID := int64(7)
	repo := &mediaJobLookupRepository{job: &GenerationJob{PublicID: "job_1", ProductID: &productID, UserID: 11, APIKeyID: 22, RequestHash: "first"}}
	orchestrator := NewMediaOrchestrator(nil, repo, nil)

	if _, _, err := orchestrator.Submit(context.Background(), MediaSubmitInput{PublicID: "job_1", UserID: 11, APIKeyID: 22, GroupID: 33, RequestHash: "second"}); !errors.Is(err, ErrMediaIdempotencyConflict) {
		t.Fatalf("Submit() error = %v", err)
	}
	if repo.createCalls != 0 {
		t.Fatalf("job creates = %d", repo.createCalls)
	}
}
