package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type leonardoGenerationRepositoryMock struct {
	created   []*GenerationJob
	updates   []*GenerationJob
	existing  *GenerationJob
	createErr error
	getErr    error
	casErr    error
}

func (m *leonardoGenerationRepositoryMock) Create(_ context.Context, job *GenerationJob) error {
	m.created = append(m.created, cloneGenerationJob(job))
	return m.createErr
}

func (m *leonardoGenerationRepositoryMock) GetByPublicID(context.Context, string) (*GenerationJob, error) {
	if m.existing != nil || m.getErr != nil {
		return cloneGenerationJob(m.existing), m.getErr
	}
	return nil, ErrGenerationJobNotFound
}

func (m *leonardoGenerationRepositoryMock) GetByUpstreamGenerationID(context.Context, string) (*GenerationJob, error) {
	return nil, ErrGenerationJobNotFound
}

func (m *leonardoGenerationRepositoryMock) CompareAndSwapStatus(_ context.Context, _ string, _ GenerationJobStatus, job *GenerationJob) error {
	m.updates = append(m.updates, cloneGenerationJob(job))
	return m.casErr
}

type leonardoGenerationClientMock struct {
	response      *leonardo.CreateGenerationResponse
	err           error
	calls         int
	request       leonardo.CreateGenerationRequest
	rawRequest    []byte
	initUpload    *leonardo.InitImageUpload
	initUploadErr error
	uploadErr     error
	initCalls     int
	uploadCalls   int
}

func (m *leonardoGenerationClientMock) CreateGeneration(_ context.Context, request leonardo.CreateGenerationRequest) (*leonardo.CreateGenerationResponse, error) {
	m.calls++
	m.request = request
	return m.response, m.err
}

func (m *leonardoGenerationClientMock) CreateGenerationRaw(_ context.Context, request []byte) (*leonardo.CreateGenerationResponse, error) {
	m.calls++
	m.rawRequest = append([]byte(nil), request...)
	_ = json.Unmarshal(request, &m.request)
	return m.response, m.err
}

func (m *leonardoGenerationClientMock) CreateInitImageUpload(context.Context, string) (*leonardo.InitImageUpload, error) {
	m.initCalls++
	return m.initUpload, m.initUploadErr
}

func (m *leonardoGenerationClientMock) UploadInitImage(context.Context, *leonardo.InitImageUpload, string, []byte) error {
	m.uploadCalls++
	return m.uploadErr
}

func TestLeonardoGenerationServiceSuccessSeparatesCosts(t *testing.T) {
	creditCost := 13.5
	estimatedCost := decimal.RequireFromString("0.10")
	customerCost := decimal.RequireFromString("0.20")
	unit := "USD"
	repository := &leonardoGenerationRepositoryMock{}
	client := &leonardoGenerationClientMock{response: &leonardo.CreateGenerationResponse{
		GenerationID:  "1dd50843-d653-4516-a8e3-f0238ee453ff",
		Cost:          &leonardo.GenerationCost{Amount: 0.12, Unit: "USD"},
		APICreditCost: &creditCost,
	}}
	service := NewLeonardoGenerationService(repository, client)

	input := leonardoGenerationJob()
	input.EstimatedUpstreamCostAmount = &estimatedCost
	input.EstimatedUpstreamCostUnit = &unit
	input.CustomerCost = &customerCost
	input.BillingStatus = GenerationJobBillingStatusReserved
	input.BillingReference = stringPointer("reservation-1")
	job, err := service.CreateGeneration(context.Background(), input, leonardoGenerationRequestWithSecrets())

	require.NoError(t, err)
	require.Equal(t, 1, client.calls)
	require.Len(t, repository.created, 1)
	require.Equal(t, GenerationJobStatusCreated, repository.created[0].Status)
	require.Len(t, repository.updates, 2)
	require.Equal(t, GenerationJobStatusSubmitting, repository.updates[0].Status)
	require.Equal(t, GenerationJobStatusQueued, repository.updates[1].Status)
	require.Equal(t, "1dd50843-d653-4516-a8e3-f0238ee453ff", *job.UpstreamGenerationID)
	require.Equal(t, "0.12", job.ActualUpstreamCostAmount.String())
	require.Equal(t, "USD", *job.ActualUpstreamCostUnit)
	require.Equal(t, "0.02", job.CostVariance.String())
	require.Equal(t, "0.08", job.GrossMargin.String())
	require.Equal(t, map[string]any{"amount": 0.12, "unit": "USD"}, job.ResultPayload["cost"])
	require.Equal(t, creditCost, job.ResultPayload["apiCreditCost"])
	requireLeonardoGenerationSecretsAbsent(t, repository)
}

func TestLeonardoGenerationServiceStoresAPICreditCostWhenCurrencyCostMissing(t *testing.T) {
	creditCost := 13.5
	repository := &leonardoGenerationRepositoryMock{}
	client := &leonardoGenerationClientMock{response: &leonardo.CreateGenerationResponse{
		GenerationID:  "1dd50843-d653-4516-a8e3-f0238ee453ff",
		APICreditCost: &creditCost,
	}}

	job, err := NewLeonardoGenerationService(repository, client).CreateGeneration(context.Background(), leonardoGenerationJob(), leonardoGenerationRequestWithSecrets())

	require.NoError(t, err)
	require.Equal(t, "13.5", job.ActualUpstreamCostAmount.String())
	require.Equal(t, "API_CREDIT", *job.ActualUpstreamCostUnit)
	require.Equal(t, creditCost, job.ResultPayload["apiCreditCost"])
}

func TestLeonardoGenerationServiceDoesNotCompareDifferentCostUnits(t *testing.T) {
	estimatedCost := decimal.RequireFromString("0.10")
	customerCost := decimal.RequireFromString("0.20")
	unit := "USD"
	input := leonardoGenerationJob()
	input.EstimatedUpstreamCostAmount = &estimatedCost
	input.EstimatedUpstreamCostUnit = &unit
	input.CustomerCost = &customerCost
	input.BillingStatus = GenerationJobBillingStatusReserved
	input.BillingReference = stringPointer("reservation-1")
	client := &leonardoGenerationClientMock{response: &leonardo.CreateGenerationResponse{
		GenerationID: "1dd50843-d653-4516-a8e3-f0238ee453ff",
		Cost:         &leonardo.GenerationCost{Amount: 12, Unit: "API_CREDIT"},
	}}

	job, err := NewLeonardoGenerationService(&leonardoGenerationRepositoryMock{}, client).CreateGeneration(context.Background(), input, leonardoGenerationRequestWithSecrets())

	require.NoError(t, err)
	require.Nil(t, job.CostVariance)
	require.Nil(t, job.GrossMargin)
}

func TestLeonardoGenerationServiceCASConflictDoesNotPost(t *testing.T) {
	repository := &leonardoGenerationRepositoryMock{casErr: ErrGenerationJobConflict}
	client := &leonardoGenerationClientMock{}
	service := NewLeonardoGenerationService(repository, client)

	job, err := service.CreateGeneration(context.Background(), leonardoGenerationJob(), leonardoGenerationRequestWithSecrets())

	require.ErrorIs(t, err, ErrGenerationJobConflict)
	require.Equal(t, GenerationJobStatusCreated, job.Status)
	require.Zero(t, client.calls)
	requireLeonardoGenerationSecretsAbsent(t, repository)
}

func TestLeonardoGenerationServiceUnmarkedErrorsAreUnknown(t *testing.T) {
	tests := map[string]error{
		"ordinary":          errors.New("api_key=api-secret Authorization=auth-secret Cookie=cookie-secret signature=signature-secret"),
		"wrapped unknown":   fmt.Errorf("wrapped: %w", errors.New("api_key=api-secret")),
		"unmarked leonardo": &leonardo.LeonardoError{Message: "Authorization=auth-secret"},
		"marked unknown": &leonardo.LeonardoError{
			Message:          "Cookie=cookie-secret",
			SubmissionStatus: leonardo.SubmissionUnknown,
		},
	}
	for name, clientErr := range tests {
		t.Run(name, func(t *testing.T) {
			repository := &leonardoGenerationRepositoryMock{}
			client := &leonardoGenerationClientMock{err: clientErr}
			service := NewLeonardoGenerationService(repository, client)

			job, err := service.CreateGeneration(context.Background(), leonardoGenerationJob(), leonardoGenerationRequestWithSecrets())

			require.NoError(t, err)
			require.Equal(t, 1, client.calls)
			require.Equal(t, GenerationJobStatusUnknown, job.Status)
			require.Equal(t, GenerationJobBillingStatusManualReview, job.BillingStatus)
			require.Equal(t, "submission_unknown", *job.ErrorCode)
			require.Nil(t, job.ActualUpstreamCostAmount)
			require.Nil(t, job.ActualUpstreamCostUnit)
			require.Nil(t, job.CustomerCost)
			requireLeonardoGenerationSecretsAbsent(t, repository)
		})
	}
}

func TestLeonardoGenerationServiceStoresWhitelistedDiagnostics(t *testing.T) {
	classes := []string{
		leonardo.GenerationErrorClassTransportAfterWrite,
		leonardo.GenerationErrorClassUpstreamNon2xx,
		leonardo.GenerationErrorClassResponseReadFailed,
		leonardo.GenerationErrorClassResponseTooLarge,
		leonardo.GenerationErrorClassResponseDecodeFailed,
		leonardo.GenerationErrorClassGenerationIDMissing,
		leonardo.GenerationErrorClassGenerationIDInvalid,
		"",
		"arbitrary",
		leonardo.GenerationErrorClassRequestNotWritten,
	}
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			repository := &leonardoGenerationRepositoryMock{}
			client := &leonardoGenerationClientMock{err: fmt.Errorf("wrapped: %w", &leonardo.LeonardoError{
				Class: class, StatusCode: 429, Code: "provider-code", Path: "provider.path", RequestID: "request-1", RetryAfter: 7 * time.Second,
				BodySHA256: strings.Repeat("a", 64), BodySize: 19, BodyTruncated: true, SubmissionStatus: leonardo.SubmissionUnknown,
			})}
			job, err := NewLeonardoGenerationService(repository, client).CreateGeneration(context.Background(), leonardoGenerationJob(), leonardoGenerationRequestWithSecrets())
			require.NoError(t, err)
			diagnostic := job.ResultPayload["submission_diagnostic"].(map[string]any)
			wantClass := class
			if !isLeonardoSubmissionUnknownClass(class) {
				wantClass = "unclassified_after_submit"
			}
			require.Equal(t, wantClass, diagnostic["class"])
			require.Equal(t, map[string]any{
				"class": wantClass, "http_status": 429, "provider_code": "provider-code", "provider_path": "provider.path",
				"request_id": "request-1", "retry_after_seconds": int64(7), "body_sha256": strings.Repeat("a", 64), "body_size": int64(19), "body_truncated": true,
			}, diagnostic)
			require.Equal(t, GenerationJobStatusUnknown, job.Status)
			require.Equal(t, GenerationJobBillingStatusManualReview, job.BillingStatus)
			require.Equal(t, 1, client.calls)
			require.Nil(t, job.UpstreamGenerationID)
			require.Nil(t, job.ActualUpstreamCostAmount)
			require.Nil(t, job.ActualUpstreamCostUnit)
			require.Nil(t, job.CustomerCost)
			require.Nil(t, job.NextPollAt)
			require.Zero(t, job.PollAttempts)
			requireLeonardoGenerationSecretsAbsent(t, repository)
		})
	}
}

func TestLeonardoGenerationServiceOmitsUnsafeRequestID(t *testing.T) {
	for _, requestID := range []string{"bad\nrequest", strings.Repeat("x", 257), "Authorization=auth-secret"} {
		repository := &leonardoGenerationRepositoryMock{}
		client := &leonardoGenerationClientMock{err: &leonardo.LeonardoError{Class: leonardo.GenerationErrorClassUpstreamNon2xx, RequestID: requestID, SubmissionStatus: leonardo.SubmissionUnknown}}
		job, err := NewLeonardoGenerationService(repository, client).CreateGeneration(context.Background(), leonardoGenerationJob(), leonardoGenerationRequestWithSecrets())
		require.NoError(t, err)
		diagnostic := job.ResultPayload["submission_diagnostic"].(map[string]any)
		if value, ok := diagnostic["request_id"]; ok {
			require.NotContains(t, fmt.Sprint(value), "auth-secret")
		}
		requireLeonardoGenerationSecretsAbsent(t, repository)
	}
}

func TestLeonardoGenerationServiceUnknownClearsLifecycleFields(t *testing.T) {
	job := leonardoGenerationJob()
	upstreamID := "old-id"
	upstreamStatus := "PENDING"
	now := time.Now()
	cost := decimal.NewFromInt(1)
	customerCost := decimal.RequireFromString("0.10")
	billingReference := "bill-unknown"
	job.UpstreamGenerationID = &upstreamID
	job.UpstreamStatus = &upstreamStatus
	job.OutputCount = 2
	job.ActualUpstreamCostAmount = &cost
	job.ActualUpstreamCostUnit = stringPointer("USD")
	job.CustomerCost = &customerCost
	job.BillingReference = &billingReference
	job.BillingStatus = GenerationJobBillingStatusReserved
	job.NextPollAt = &now
	job.LastPolledAt = &now
	job.SubmittedAt = &now
	job.StartedAt = &now
	job.CompletedAt = &now
	job.FailedAt = &now
	job.PollAttempts = 3
	repository := &leonardoGenerationRepositoryMock{}
	client := &leonardoGenerationClientMock{err: errors.New("unknown")}
	result, err := NewLeonardoGenerationService(repository, client).CreateGeneration(context.Background(), job, leonardoGenerationRequest())
	require.NoError(t, err)
	require.Nil(t, result.UpstreamGenerationID)
	require.Nil(t, result.UpstreamStatus)
	require.Zero(t, result.OutputCount)
	require.Nil(t, result.ActualUpstreamCostAmount)
	require.Nil(t, result.ActualUpstreamCostUnit)
	require.Equal(t, customerCost.String(), result.CustomerCost.String())
	require.Equal(t, billingReference, *result.BillingReference)
	require.Nil(t, result.NextPollAt)
	require.Nil(t, result.LastPolledAt)
	require.Nil(t, result.SubmittedAt)
	require.Nil(t, result.StartedAt)
	require.Nil(t, result.CompletedAt)
	require.Nil(t, result.FailedAt)
	require.Zero(t, result.PollAttempts)
}

func TestLeonardoGenerationServiceInvalidSuccessIDIsUnknown(t *testing.T) {
	repository := &leonardoGenerationRepositoryMock{}
	client := &leonardoGenerationClientMock{response: &leonardo.CreateGenerationResponse{GenerationID: "not-a-uuid"}}
	service := NewLeonardoGenerationService(repository, client)

	job, err := service.CreateGeneration(context.Background(), leonardoGenerationJob(), leonardoGenerationRequestWithSecrets())

	require.NoError(t, err)
	require.Equal(t, 1, client.calls)
	require.Equal(t, GenerationJobStatusUnknown, job.Status)
	require.Nil(t, job.UpstreamGenerationID)
	require.Equal(t, GenerationJobBillingStatusManualReview, job.BillingStatus)
	require.Nil(t, job.ActualUpstreamCostAmount)
	require.Nil(t, job.ActualUpstreamCostUnit)
	require.Nil(t, job.CustomerCost)
	requireLeonardoGenerationSecretsAbsent(t, repository)
}

func TestLeonardoGenerationServicePreWriteFailureIsFailedWithoutRetry(t *testing.T) {
	repository := &leonardoGenerationRepositoryMock{}
	clientErr := fmt.Errorf("transport setup: %w", ErrLeonardoGenerationRequestNotWritten)
	client := &leonardoGenerationClientMock{err: clientErr}
	service := NewLeonardoGenerationService(repository, client)

	job, err := service.CreateGeneration(context.Background(), leonardoGenerationJob(), leonardoGenerationRequestWithSecrets())

	require.ErrorIs(t, err, clientErr)
	require.Equal(t, 1, client.calls)
	require.Equal(t, GenerationJobStatusFailed, job.Status)
	require.Equal(t, "not_submitted", *job.ErrorCode)
	require.Equal(t, "leonardo generation request was not written", *job.ErrorMessage)
	requireLeonardoGenerationSecretsAbsent(t, repository)
}

func TestLeonardoGenerationServiceSanitizesStoredPayload(t *testing.T) {
	repository := &leonardoGenerationRepositoryMock{}
	client := &leonardoGenerationClientMock{response: &leonardo.CreateGenerationResponse{GenerationID: "1dd50843-d653-4516-a8e3-f0238ee453ff"}}
	service := NewLeonardoGenerationService(repository, client)

	_, err := service.CreateGeneration(context.Background(), leonardoGenerationJob(), leonardoGenerationRequestWithSecrets())

	require.NoError(t, err)
	require.Equal(t, 1, client.calls)
	requireLeonardoGenerationSecretsAbsent(t, repository)
}

func TestLeonardoGenerationServiceReservedBilling(t *testing.T) {
	cost := decimal.RequireFromString("0.005000000000000001")
	reference := "reservation-1"
	job := leonardoGenerationJob()
	job.CustomerCost = &cost
	job.BillingReference = &reference
	job.BillingStatus = GenerationJobBillingStatusReserved
	repository := &leonardoGenerationRepositoryMock{}
	client := &leonardoGenerationClientMock{response: &leonardo.CreateGenerationResponse{GenerationID: "1dd50843-d653-4516-a8e3-f0238ee453ff"}}
	result, err := NewLeonardoGenerationService(repository, client).CreateGeneration(context.Background(), job, leonardoGenerationRequest())
	require.NoError(t, err)
	require.Equal(t, "0.005000000000000001", result.CustomerCost.String())
	require.Equal(t, reference, *result.BillingReference)
	require.Equal(t, GenerationJobBillingStatusSubmitted, result.BillingStatus)
	require.Equal(t, GenerationJobBillingStatusReserved, repository.created[0].BillingStatus)
}

func TestLeonardoGenerationServiceRejectsPartialReservation(t *testing.T) {
	job := leonardoGenerationJob()
	job.BillingStatus = GenerationJobBillingStatusReserved
	repository := &leonardoGenerationRepositoryMock{}
	client := &leonardoGenerationClientMock{}
	result, err := NewLeonardoGenerationService(repository, client).CreateGeneration(context.Background(), job, leonardoGenerationRequest())
	require.Nil(t, result)
	require.ErrorIs(t, err, ErrLeonardoImageCreateReservationInvalid)
	require.Empty(t, repository.created)
	require.Zero(t, client.calls)
}

func leonardoGenerationJob() *GenerationJob {
	return &GenerationJob{
		PublicID:      "generation-job-1",
		Provider:      "leonardo",
		Modality:      "image",
		Model:         "flux-schnell",
		UpstreamModel: "flux-schnell",
		UserID:        1,
		APIKeyID:      2,
		AccountID:     3,
		RequestHash:   "request-hash",
	}
}

func leonardoGenerationRequest() leonardo.CreateGenerationRequest {
	return leonardo.CreateGenerationRequest{Model: "flux-schnell", Parameters: map[string]any{"prompt": "cat"}}
}

func leonardoGenerationRequestWithSecrets() leonardo.CreateGenerationRequest {
	request := leonardoGenerationRequest()
	request.Parameters["api_key"] = "api-secret"
	request.Parameters["Authorization"] = "auth-secret"
	request.Parameters["Cookie"] = "cookie-secret"
	request.Parameters["signature"] = "signature-secret"
	request.Parameters["nested"] = map[string]any{"x-amz-signature": "nested-secret"}
	return request
}

func requireLeonardoGenerationSecretsAbsent(t *testing.T, repository *leonardoGenerationRepositoryMock) {
	t.Helper()
	jobs := append(append([]*GenerationJob{}, repository.created...), repository.updates...)
	payload := strings.ToLower(string(mustMarshalJSON(t, jobs)))
	for _, secret := range []string{"api-secret", "auth-secret", "cookie-secret", "signature-secret", "nested-secret"} {
		require.NotContains(t, payload, secret)
	}
}

func cloneGenerationJob(job *GenerationJob) *GenerationJob {
	if job == nil {
		return nil
	}
	clone := *job
	return &clone
}

func mustMarshalJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	require.NoError(t, err)
	return data
}
