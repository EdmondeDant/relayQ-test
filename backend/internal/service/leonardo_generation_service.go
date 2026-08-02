package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type LeonardoGenerationClient interface {
	CreateGeneration(context.Context, leonardo.CreateGenerationRequest) (*leonardo.CreateGenerationResponse, error)
}

var ErrLeonardoGenerationRequestNotWritten = errors.New("leonardo generation request not written")

type LeonardoGenerationService struct {
	repository GenerationJobRepository
	client     LeonardoGenerationClient
}

func NewLeonardoGenerationService(repository GenerationJobRepository, client LeonardoGenerationClient) *LeonardoGenerationService {
	return &LeonardoGenerationService{repository: repository, client: client}
}

func (s *LeonardoGenerationService) CreateGeneration(ctx context.Context, job *GenerationJob, request leonardo.CreateGenerationRequest) (*GenerationJob, error) {
	if s == nil || s.repository == nil || s.client == nil {
		return nil, errors.New("leonardo generation service is not configured")
	}
	if job == nil {
		return nil, errors.New("generation job is required")
	}
	reserved := job.BillingStatus == GenerationJobBillingStatusReserved && job.CustomerCost != nil && job.CustomerCost.Sign() > 0 && job.BillingReference != nil && strings.TrimSpace(*job.BillingReference) != ""
	hasReservationData := job.BillingStatus == GenerationJobBillingStatusReserved || job.CustomerCost != nil || job.BillingReference != nil
	if hasReservationData && !reserved {
		return nil, ErrLeonardoImageCreateReservationInvalid
	}

	created := *job
	created.Status = GenerationJobStatusCreated
	created.UpstreamGenerationID = nil
	created.UpstreamStatus = nil
	created.RequestPayload = sanitizeLeonardoGenerationPayload(request)
	created.ResultPayload = map[string]any{}
	created.ErrorCode = nil
	created.ErrorMessage = nil
	created.ActualUpstreamCostAmount = nil
	created.ActualUpstreamCostUnit = nil
	if !reserved {
		created.CustomerCost = nil
		created.BillingReference = nil
		created.BillingStatus = GenerationJobBillingStatusUnpriced
	}
	created.SubmittedAt = nil
	created.FailedAt = nil
	if err := s.repository.Create(ctx, &created); err != nil {
		return nil, err
	}

	submitting := created
	submitting.Status = GenerationJobStatusSubmitting
	if err := s.repository.CompareAndSwapStatus(ctx, created.PublicID, GenerationJobStatusCreated, &submitting); err != nil {
		return &created, err
	}

	response, err := s.client.CreateGeneration(ctx, request)
	if err != nil {
		return s.storeSubmissionError(ctx, &submitting, err)
	}
	if response == nil || !validLeonardoGenerationUUID(response.GenerationID) {
		return s.storeSubmissionUnknown(ctx, &submitting)
	}

	queued := submitting
	queued.Status = GenerationJobStatusQueued
	queued.UpstreamGenerationID = stringPointer(response.GenerationID)
	queued.UpstreamStatus = stringPointer("PENDING")
	queued.SubmittedAt = timePointerValue(time.Now().UTC())
	queued.BillingStatus = GenerationJobBillingStatusSubmitted
	queued.ResultPayload = leonardoGenerationCostPayload(response)
	if response.Cost != nil {
		amount := decimal.NewFromFloat(response.Cost.Amount)
		queued.ActualUpstreamCostAmount = &amount
		queued.ActualUpstreamCostUnit = stringPointer(response.Cost.Unit)
	}
	if err := s.repository.CompareAndSwapStatus(ctx, submitting.PublicID, GenerationJobStatusSubmitting, &queued); err != nil {
		return &submitting, err
	}
	return &queued, nil
}

func (s *LeonardoGenerationService) storeSubmissionError(ctx context.Context, submitting *GenerationJob, submissionErr error) (*GenerationJob, error) {
	if !errors.Is(submissionErr, ErrLeonardoGenerationRequestNotWritten) {
		return s.storeSubmissionUnknown(ctx, submitting)
	}

	failed := *submitting
	failed.Status = GenerationJobStatusFailed
	failed.ErrorCode = stringPointer("not_submitted")
	failed.ErrorMessage = stringPointer("leonardo generation request was not written")
	failed.FailedAt = timePointerValue(time.Now().UTC())
	if err := s.repository.CompareAndSwapStatus(ctx, submitting.PublicID, GenerationJobStatusSubmitting, &failed); err != nil {
		return submitting, errors.Join(submissionErr, err)
	}
	return &failed, submissionErr
}

func (s *LeonardoGenerationService) storeSubmissionUnknown(ctx context.Context, submitting *GenerationJob) (*GenerationJob, error) {
	unknown := *submitting
	unknown.Status = GenerationJobStatusUnknown
	unknown.ErrorMessage = stringPointer("leonardo generation submission status is unknown")
	NormalizeGenerationJob(&unknown)
	if err := s.repository.CompareAndSwapStatus(ctx, submitting.PublicID, GenerationJobStatusSubmitting, &unknown); err != nil {
		return submitting, err
	}
	return &unknown, nil
}

func sanitizeLeonardoGenerationPayload(request leonardo.CreateGenerationRequest) map[string]any {
	return logredact.RedactMap(map[string]any{
		"model":      request.Model,
		"public":     request.Public,
		"parameters": request.Parameters,
	}, "api_key", "apikey", "authorization", "cookie", "signature", "x-api-key", "x-amz-signature")
}

func leonardoGenerationCostPayload(response *leonardo.CreateGenerationResponse) map[string]any {
	payload := map[string]any{}
	if response.Cost != nil {
		payload["cost"] = map[string]any{"amount": response.Cost.Amount, "unit": response.Cost.Unit}
	}
	if response.APICreditCost != nil {
		payload["apiCreditCost"] = *response.APICreditCost
	}
	return payload
}

func validLeonardoGenerationUUID(value string) bool {
	parsed, err := uuid.Parse(value)
	return err == nil && parsed.String() == value
}

func stringPointer(value string) *string {
	return &value
}

func timePointerValue(value time.Time) *time.Time {
	return &value
}
