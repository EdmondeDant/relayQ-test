package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type leonardoImageCreateFundsFake struct {
	reservation  *LeonardoImageFundsReservation
	reserveErr   error
	releaseErr   error
	reserveCalls int
	releaseCalls int
	reserve      LeonardoImageFundsReserveRequest
	release      LeonardoImageFundsReleaseRequest
}

func (f *leonardoImageCreateFundsFake) Reserve(_ context.Context, request LeonardoImageFundsReserveRequest) (*LeonardoImageFundsReservation, error) {
	f.reserveCalls++
	f.reserve = request
	return f.reservation, f.reserveErr
}

func (f *leonardoImageCreateFundsFake) Release(_ context.Context, request LeonardoImageFundsReleaseRequest) error {
	f.releaseCalls++
	f.release = request
	return f.releaseErr
}

type leonardoImageCreateAccountReaderFake struct {
	account *Account
	err     error
	calls   int
}

func (f *leonardoImageCreateAccountReaderFake) GetByID(context.Context, int64) (*Account, error) {
	f.calls++
	return f.account, f.err
}

type leonardoImageCreateClientFactoryFake struct {
	client  LeonardoGenerationClient
	err     error
	calls   int
	account *Account
}

func (f *leonardoImageCreateClientFactoryFake) Build(account *Account) (LeonardoGenerationClient, error) {
	f.calls++
	f.account = account
	return f.client, f.err
}

func TestLeonardoImageCreateOrchestratorSuccess(t *testing.T) {
	repository := &leonardoGenerationRepositoryMock{}
	client := &leonardoGenerationClientMock{response: &leonardo.CreateGenerationResponse{GenerationID: "1dd50843-d653-4516-a8e3-f0238ee453ff"}}
	funds := createFundsFake("0.005")
	accounts := &leonardoImageCreateAccountReaderFake{account: createLeonardoAccount()}
	factory := &leonardoImageCreateClientFactoryFake{client: client}
	job, err := createOrchestrator(funds, accounts, factory, repository).Create(context.Background(), createImageRequest("0.005"))
	require.NoError(t, err)
	require.Equal(t, GenerationJobStatusQueued, job.Status)
	require.Equal(t, GenerationJobBillingStatusSubmitted, job.BillingStatus)
	require.Equal(t, "0.003", job.EstimatedUpstreamCostAmount.String())
	require.Equal(t, "USD", *job.EstimatedUpstreamCostUnit)
	require.Equal(t, "2026-08-01", *job.PricingSnapshotVersion)
	require.Equal(t, "leonardo_authenticated_pricing_calculator", *job.PricingSource)
	require.Equal(t, "exact", *job.PricingMatchType)
	require.Equal(t, "0.005", job.CustomerCost.String())
	require.Equal(t, "reservation-1", *job.BillingReference)
	require.Equal(t, 1, client.calls)
	require.Equal(t, 1, funds.reserveCalls)
	require.Zero(t, funds.releaseCalls)
	require.Same(t, accounts.account, factory.account)
	require.Equal(t, "0.005", funds.reserve.AmountUSD.String())
}

func TestLeonardoImageCreateOrchestratorUploadsEditImage(t *testing.T) {
	repository := &leonardoGenerationRepositoryMock{}
	client := &leonardoGenerationClientMock{
		response:   &leonardo.CreateGenerationResponse{GenerationID: "1dd50843-d653-4516-a8e3-f0238ee453ff"},
		initUpload: &leonardo.InitImageUpload{ID: "uploaded-1", URL: "https://upload.example"},
	}
	funds := createFundsFake("0.005")
	request := createImageRequest("0.005")
	request.InputImage = &LeonardoImageInput{Data: []byte("image"), Extension: "png", FileName: "image.png"}
	request.ImageCapability = &LeonardoImageReferenceCapability{MaxItems: 1, AllowedStrengths: []string{"MID"}, DefaultStrength: "MID"}
	orchestrator := createOrchestrator(funds, &leonardoImageCreateAccountReaderFake{account: createLeonardoAccount()}, &leonardoImageCreateClientFactoryFake{client: client}, repository)
	orchestrator.uploads = NewLeonardoImageUploadService(nil)

	job, err := orchestrator.Create(context.Background(), request)

	require.NoError(t, err)
	require.Equal(t, GenerationJobStatusQueued, job.Status)
	require.Equal(t, 1, client.initCalls)
	require.Equal(t, 1, client.uploadCalls)
	require.Equal(t, 1, client.calls)
	require.Zero(t, funds.releaseCalls)
	require.JSONEq(t, `{"image_reference":[{"image":{"id":"uploaded-1","type":"UPLOADED"},"strength":"MID"}]}`, marshalLeonardoTestJSON(t, client.request.Parameters["guidances"]))
}

func TestLeonardoImageCreateOrchestratorBuildsFluxGuidances(t *testing.T) {
	repository := &leonardoGenerationRepositoryMock{}
	client := &leonardoGenerationClientMock{response: &leonardo.CreateGenerationResponse{GenerationID: "1dd50843-d653-4516-a8e3-f0238ee453ff"}}
	funds := createFundsFake("0.005")
	request := createImageRequest("0.005")
	request.FluxGuidances = LeonardoFluxGuidances{
		Content: []leonardo.ImageReference{{Image: leonardo.ImageReferenceImage{ID: "content-1", Type: "UPLOADED"}, Strength: "HIGH"}},
		Style:   []leonardo.ImageReference{{Image: leonardo.ImageReferenceImage{ID: "style-1", Type: "GENERATED"}, Strength: "MAX"}},
	}

	job, err := createOrchestrator(funds, &leonardoImageCreateAccountReaderFake{account: createLeonardoAccount()}, &leonardoImageCreateClientFactoryFake{client: client}, repository).Create(context.Background(), request)

	require.NoError(t, err)
	require.Equal(t, GenerationJobStatusQueued, job.Status)
	guidances := client.request.Parameters["guidances"].(map[string]any)
	require.JSONEq(t, `[{"image":{"id":"content-1","type":"UPLOADED"},"strength":"HIGH"}]`, marshalLeonardoTestJSON(t, guidances["content"]))
	require.JSONEq(t, `[{"image":{"id":"style-1","type":"GENERATED"},"strength":"MAX"}]`, marshalLeonardoTestJSON(t, guidances["style"]))
	require.NotContains(t, guidances, "image_reference")
}

func marshalLeonardoTestJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	return string(encoded)
}

func TestLeonardoImageCreateOrchestratorRejectsMixedGuidanceProtocols(t *testing.T) {
	client := &leonardoGenerationClientMock{}
	funds := createFundsFake("0.005")
	request := createImageRequest("0.005")
	request.FluxGuidances = LeonardoFluxGuidances{Content: []leonardo.ImageReference{{Image: leonardo.ImageReferenceImage{ID: "content-1", Type: "UPLOADED"}, Strength: "MID"}}}
	request.ImageReferences = []leonardo.ImageReference{{Image: leonardo.ImageReferenceImage{ID: "legacy-1", Type: "UPLOADED"}, Strength: "MID"}}

	_, err := createOrchestrator(funds, &leonardoImageCreateAccountReaderFake{account: createLeonardoAccount()}, &leonardoImageCreateClientFactoryFake{client: client}, &leonardoGenerationRepositoryMock{}).Create(context.Background(), request)

	require.ErrorIs(t, err, ErrLeonardoImageReferenceInvalid)
	require.Equal(t, 1, funds.releaseCalls)
	require.Zero(t, client.calls)
}

func TestLeonardoImageCreateOrchestratorReleasesFundsWhenEditUploadFails(t *testing.T) {
	client := &leonardoGenerationClientMock{initUploadErr: errors.New("upload unavailable")}
	funds := createFundsFake("0.005")
	request := createImageRequest("0.005")
	request.InputImage = &LeonardoImageInput{Data: []byte("image"), Extension: "png"}
	request.ImageCapability = &LeonardoImageReferenceCapability{MaxItems: 1, AllowedStrengths: []string{"MID"}, DefaultStrength: "MID"}
	orchestrator := createOrchestrator(funds, &leonardoImageCreateAccountReaderFake{account: createLeonardoAccount()}, &leonardoImageCreateClientFactoryFake{client: client}, &leonardoGenerationRepositoryMock{})
	orchestrator.uploads = NewLeonardoImageUploadService(nil)

	_, err := orchestrator.Create(context.Background(), request)

	require.Error(t, err)
	require.Equal(t, 1, funds.releaseCalls)
	require.Equal(t, "image_upload_failed", funds.release.Reason)
	require.Zero(t, client.calls)
}

func TestLeonardoImageCreateOrchestratorShortCircuits(t *testing.T) {
	tests := []struct {
		name       string
		request    LeonardoImageCreateRequest
		account    *Account
		factoryErr error
		reserveErr error
	}{
		{name: "invalid request", request: LeonardoImageCreateRequest{}},
		{name: "invalid account", request: createImageRequest("0.005"), account: &Account{ID: 3}},
		{name: "adapter failure", request: createImageRequest("0.005"), account: createLeonardoAccount(), factoryErr: errors.New("adapter")},
		{name: "reserve failure", request: createImageRequest("0.005"), account: createLeonardoAccount(), reserveErr: ErrInsufficientBalance},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &leonardoGenerationClientMock{}
			funds := createFundsFake("0.005")
			funds.reserveErr = test.reserveErr
			accounts := &leonardoImageCreateAccountReaderFake{account: test.account}
			factory := &leonardoImageCreateClientFactoryFake{client: client, err: test.factoryErr}
			repository := &leonardoGenerationRepositoryMock{}
			_, err := createOrchestrator(funds, accounts, factory, repository).Create(context.Background(), test.request)
			require.Error(t, err)
			require.Zero(t, client.calls)
			require.Empty(t, repository.created)
			require.Zero(t, funds.releaseCalls)
		})
	}
}

func TestLeonardoImageCreateOrchestratorInvalidReservation(t *testing.T) {
	funds := createFundsFake("0.004")
	_, err := createOrchestrator(funds, &leonardoImageCreateAccountReaderFake{account: createLeonardoAccount()}, &leonardoImageCreateClientFactoryFake{client: &leonardoGenerationClientMock{}}, &leonardoGenerationRepositoryMock{}).Create(context.Background(), createImageRequest("0.005"))
	require.ErrorIs(t, err, ErrLeonardoImageCreateReservationInvalid)
	require.Zero(t, funds.releaseCalls)

	funds = createFundsFake("0.005")
	funds.reservation.PricingVersion = "wrong"
	_, err = createOrchestrator(funds, &leonardoImageCreateAccountReaderFake{account: createLeonardoAccount()}, &leonardoImageCreateClientFactoryFake{client: &leonardoGenerationClientMock{}}, &leonardoGenerationRepositoryMock{}).Create(context.Background(), createImageRequest("0.005"))
	require.ErrorIs(t, err, ErrLeonardoImageCreateReservationInvalid)
	require.Equal(t, 1, funds.releaseCalls)
	require.Equal(t, "invalid_reservation_response", funds.release.Reason)
}

func TestLeonardoImageCreateOrchestratorSubmissionCompensation(t *testing.T) {
	for _, test := range []struct {
		name       string
		repository *leonardoGenerationRepositoryMock
		client     *leonardoGenerationClientMock
		releases   int
		reason     string
		status     GenerationJobStatus
	}{
		{name: "job create", repository: &leonardoGenerationRepositoryMock{createErr: errors.New("create")}, client: &leonardoGenerationClientMock{}, releases: 1, reason: "job_create_failed"},
		{name: "request not written", repository: &leonardoGenerationRepositoryMock{}, client: &leonardoGenerationClientMock{err: ErrLeonardoGenerationRequestNotWritten}, releases: 1, reason: "request_not_written", status: GenerationJobStatusFailed},
		{name: "unknown", repository: &leonardoGenerationRepositoryMock{}, client: &leonardoGenerationClientMock{err: errors.New("unknown")}, releases: 0, status: GenerationJobStatusUnknown},
	} {
		t.Run(test.name, func(t *testing.T) {
			funds := createFundsFake("0.005")
			job, err := createOrchestrator(funds, &leonardoImageCreateAccountReaderFake{account: createLeonardoAccount()}, &leonardoImageCreateClientFactoryFake{client: test.client}, test.repository).Create(context.Background(), createImageRequest("0.005"))
			if test.name == "unknown" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			require.Equal(t, test.releases, funds.releaseCalls)
			if test.reason != "" {
				require.Equal(t, test.reason, funds.release.Reason)
			}
			if job != nil && test.status != "" {
				require.Equal(t, test.status, job.Status)
			}
			if test.name == "unknown" {
				require.Equal(t, "0.003", job.EstimatedUpstreamCostAmount.String())
				require.Equal(t, "USD", *job.EstimatedUpstreamCostUnit)
				require.Equal(t, "2026-08-01", *job.PricingSnapshotVersion)
				require.Equal(t, "leonardo_authenticated_pricing_calculator", *job.PricingSource)
				require.Equal(t, "exact", *job.PricingMatchType)
				require.Nil(t, job.ActualUpstreamCostAmount)
				require.Nil(t, job.ActualUpstreamCostUnit)
				require.Equal(t, "0.005", job.CustomerCost.String())
			}
		})
	}
}

func TestLeonardoImageCreateOrchestratorReleaseFailureDoesNotMarkRefunded(t *testing.T) {
	releaseErr := errors.New("release failed")
	for _, test := range []struct {
		name       string
		repository *leonardoGenerationRepositoryMock
		client     *leonardoGenerationClientMock
		updates    int
	}{
		{name: "submit gate", repository: &leonardoGenerationRepositoryMock{casErr: ErrGenerationJobConflict}, client: &leonardoGenerationClientMock{}, updates: 1},
		{name: "request not written", repository: &leonardoGenerationRepositoryMock{}, client: &leonardoGenerationClientMock{err: ErrLeonardoGenerationRequestNotWritten}, updates: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			funds := createFundsFake("0.005")
			funds.releaseErr = releaseErr
			job, err := createOrchestrator(funds, &leonardoImageCreateAccountReaderFake{account: createLeonardoAccount()}, &leonardoImageCreateClientFactoryFake{client: test.client}, test.repository).Create(context.Background(), createImageRequest("0.005"))
			require.ErrorIs(t, err, releaseErr)
			require.Equal(t, 1, funds.releaseCalls)
			require.Len(t, test.repository.updates, test.updates)
			if job != nil {
				require.NotEqual(t, GenerationJobBillingStatusRefunded, job.BillingStatus)
			}
			for _, update := range test.repository.updates {
				require.NotEqual(t, GenerationJobBillingStatusRefunded, update.BillingStatus)
			}
		})
	}
}

func createOrchestrator(funds *leonardoImageCreateFundsFake, accounts *leonardoImageCreateAccountReaderFake, factory *leonardoImageCreateClientFactoryFake, jobs GenerationJobRepository) *LeonardoImageCreateOrchestrator {
	resolver := &leonardoImageQuotePriceResolverFake{estimate: quoteEstimate("0.003", "2026-08-01", "leonardo_authenticated_pricing_calculator", "exact")}
	quotes := NewLeonardoImageQuoteGuard(resolver, &leonardoImageQuoteBalanceReaderFake{balance: decimal.RequireFromString("1")})
	return NewLeonardoImageCreateOrchestrator(quotes, funds, accounts, factory, jobs)
}

func createFundsFake(amount string) *leonardoImageCreateFundsFake {
	return &leonardoImageCreateFundsFake{reservation: &LeonardoImageFundsReservation{Reference: "reservation-1", UserID: 1, PublicID: "job-1", AmountUSD: decimal.RequireFromString(amount), PricingVersion: "2026-08-01", PricingSource: "leonardo_authenticated_pricing_calculator", PricingMatchType: "exact"}}
}

func createLeonardoAccount() *Account {
	return &Account{ID: 3, Platform: PlatformLeonardo, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true}
}

func createImageRequest(quote string) LeonardoImageCreateRequest {
	return LeonardoImageCreateRequest{PublicID: "job-1", UserID: 1, APIKeyID: 2, AccountID: 3, RequestHash: "hash", Model: "flux-schnell", Prompt: "cat", Width: 896, Height: 896, Quantity: 1, CustomerQuoteUSD: decimal.RequireFromString(quote)}
}
