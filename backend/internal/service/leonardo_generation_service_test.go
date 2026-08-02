package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type leonardoGenerationRepositoryMock struct {
	created   []*GenerationJob
	updates   []*GenerationJob
	createErr error
	casErr    error
}

func (m *leonardoGenerationRepositoryMock) Create(_ context.Context, job *GenerationJob) error {
	m.created = append(m.created, cloneGenerationJob(job))
	return m.createErr
}

func (m *leonardoGenerationRepositoryMock) GetByPublicID(context.Context, string) (*GenerationJob, error) {
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
	response *leonardo.CreateGenerationResponse
	err      error
	calls    int
}

func (m *leonardoGenerationClientMock) CreateGeneration(context.Context, leonardo.CreateGenerationRequest) (*leonardo.CreateGenerationResponse, error) {
	m.calls++
	return m.response, m.err
}

func TestLeonardoGenerationServiceSuccessSeparatesCosts(t *testing.T) {
	creditCost := 13.5
	repository := &leonardoGenerationRepositoryMock{}
	client := &leonardoGenerationClientMock{response: &leonardo.CreateGenerationResponse{
		GenerationID:  "1dd50843-d653-4516-a8e3-f0238ee453ff",
		Cost:          &leonardo.GenerationCost{Amount: 0.12, Unit: "USD"},
		APICreditCost: &creditCost,
	}}
	service := NewLeonardoGenerationService(repository, client)

	job, err := service.CreateGeneration(context.Background(), leonardoGenerationJob(), leonardoGenerationRequestWithSecrets())

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
	require.Equal(t, map[string]any{"amount": 0.12, "unit": "USD"}, job.ResultPayload["cost"])
	require.Equal(t, creditCost, job.ResultPayload["apiCreditCost"])
	requireLeonardoGenerationSecretsAbsent(t, repository)
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
