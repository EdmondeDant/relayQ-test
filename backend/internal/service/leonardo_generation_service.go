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
		class := leonardo.GenerationErrorClassGenerationIDInvalid
		if response == nil || response.GenerationID == "" {
			class = leonardo.GenerationErrorClassGenerationIDMissing
		}
		return s.storeSubmissionUnknown(ctx, &submitting, map[string]any{"class": class})
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
		return s.storeSubmissionUnknown(ctx, submitting, leonardoSubmissionDiagnostic(submissionErr))
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

func (s *LeonardoGenerationService) storeSubmissionUnknown(ctx context.Context, submitting *GenerationJob, diagnostic map[string]any) (*GenerationJob, error) {
	unknown := *submitting
	unknown.Status = GenerationJobStatusUnknown
	unknown.ErrorMessage = stringPointer("leonardo generation submission status is unknown")
	unknown.UpstreamGenerationID = nil
	unknown.UpstreamStatus = nil
	unknown.ResultPayload = map[string]any{"submission_diagnostic": diagnostic}
	unknown.OutputCount = 0
	unknown.ActualUpstreamCostAmount = nil
	unknown.ActualUpstreamCostUnit = nil
	unknown.CustomerCost = nil
	unknown.NextPollAt = nil
	unknown.LastPolledAt = nil
	unknown.SubmittedAt = nil
	unknown.StartedAt = nil
	unknown.CompletedAt = nil
	unknown.FailedAt = nil
	unknown.PollAttempts = 0
	NormalizeGenerationJob(&unknown)
	if err := s.repository.CompareAndSwapStatus(ctx, submitting.PublicID, GenerationJobStatusSubmitting, &unknown); err != nil {
		return submitting, err
	}
	return &unknown, nil
}

func leonardoSubmissionDiagnostic(err error) map[string]any {
	diagnostic := map[string]any{"class": "unclassified_after_submit"}
	var apiErr *leonardo.LeonardoError
	if !errors.As(err, &apiErr) {
		return diagnostic
	}
	if isLeonardoSubmissionUnknownClass(apiErr.Class) {
		diagnostic["class"] = apiErr.Class
	}
	if apiErr.StatusCode != 0 {
		diagnostic["http_status"] = apiErr.StatusCode
	}
	if apiErr.Code != "" {
		diagnostic["provider_code"] = logredact.RedactText(apiErr.Code, "api_key", "authorization", "cookie", "signature", "x-api-key")
	}
	if apiErr.Path != "" {
		diagnostic["provider_path"] = logredact.RedactText(apiErr.Path, "api_key", "authorization", "cookie", "signature", "x-api-key")
	}
	if requestID := sanitizeLeonardoDiagnosticHeader(apiErr.RequestID); requestID != "" {
		diagnostic["request_id"] = requestID
	}
	if apiErr.RetryAfter > 0 {
		diagnostic["retry_after_seconds"] = int64(apiErr.RetryAfter / time.Second)
	}
	if apiErr.BodySHA256 != "" {
		diagnostic["body_sha256"] = apiErr.BodySHA256
	}
	if apiErr.BodySize > 0 {
		diagnostic["body_size"] = apiErr.BodySize
	}
	if apiErr.BodyTruncated {
		diagnostic["body_truncated"] = true
	}
	return diagnostic
}

func isLeonardoSubmissionUnknownClass(class string) bool {
	switch class {
	case leonardo.GenerationErrorClassTransportAfterWrite,
		leonardo.GenerationErrorClassUpstreamNon2xx,
		leonardo.GenerationErrorClassResponseReadFailed,
		leonardo.GenerationErrorClassResponseTooLarge,
		leonardo.GenerationErrorClassResponseDecodeFailed,
		leonardo.GenerationErrorClassGenerationIDMissing,
		leonardo.GenerationErrorClassGenerationIDInvalid:
		return true
	default:
		return false
	}
}

func sanitizeLeonardoDiagnosticHeader(value string) string {
	value = strings.TrimSpace(logredact.RedactText(value, "api_key", "authorization", "cookie", "signature", "x-api-key"))
	if len(value) > 256 {
		return ""
	}
	for _, r := range value {
		if r < 0x21 || r > 0x7e {
			return ""
		}
	}
	return value
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
