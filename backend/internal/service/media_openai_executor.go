package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type DefaultMediaOpenAIExecutor struct {
	gateway  *OpenAIGatewayService
	accounts AccountRepository
}

func NewDefaultMediaOpenAIExecutor(gateway *OpenAIGatewayService, accounts AccountRepository) *DefaultMediaOpenAIExecutor {
	return &DefaultMediaOpenAIExecutor{gateway: gateway, accounts: accounts}
}

func (e *DefaultMediaOpenAIExecutor) SubmitMedia(ctx context.Context, job *GenerationJob, request MediaCanonicalRequest, offer MediaCatalogOffer) (MediaSubmissionOutcome, error) {
	account, err := e.account(ctx, job)
	if err != nil {
		return MediaSubmissionOutcome{State: MediaSubmissionNotWritten}, err
	}
	body := request.Body
	if !strings.Contains(strings.ToLower(request.ContentType), "multipart/form-data") {
		body, err = mediaRequestBody(request.Body, offer.UpstreamModel)
		if err != nil {
			return MediaSubmissionOutcome{State: MediaSubmissionNotWritten}, err
		}
	}
	c, recorder := mediaGinContext(ctx, http.MethodPost, mediaOperationPath(request.Modality, request.Operation), body)
	if request.ContentType != "" {
		c.Request.Header.Set("Content-Type", request.ContentType)
	}
	if request.Modality == "image" {
		parsed, parseErr := e.gateway.ParseOpenAIImagesRequest(c, body)
		if parseErr != nil {
			return MediaSubmissionOutcome{State: MediaSubmissionNotWritten}, parseErr
		}
		_, err = e.gateway.ForwardImages(ctx, c, account, body, parsed, offer.UpstreamModel)
	} else {
		switch request.Operation {
		case "edits":
			err = e.gateway.ForwardXAIVideoEdit(ctx, c, account, body)
		case "extensions":
			err = e.gateway.ForwardXAIVideoExtension(ctx, c, account, body)
		default:
			err = e.gateway.ForwardXAIVideoGeneration(ctx, c, account, body)
		}
	}
	if err != nil {
		return MediaSubmissionOutcome{State: mediaOpenAIErrorState(err), AccountID: account.ID}, err
	}
	result := map[string]any{}
	if len(recorder.Body.Bytes()) > 0 {
		if err = json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
			return MediaSubmissionOutcome{State: MediaSubmissionSideEffectUnknown, AccountID: account.ID}, err
		}
	}
	upstreamID := mediaResultID(result)
	status := fmt.Sprint(result["status"])
	if request.Modality == "image" && upstreamID == "" {
		upstreamID, status = job.PublicID, "completed"
	}
	if upstreamID == "" {
		return MediaSubmissionOutcome{State: MediaSubmissionSideEffectUnknown, AccountID: account.ID, Result: result}, errorsNewMediaUpstreamID()
	}
	return MediaSubmissionOutcome{State: MediaSubmissionSubmitted, UpstreamID: upstreamID, AccountID: account.ID, Status: status, Result: result}, nil
}

func (e *DefaultMediaOpenAIExecutor) PollMedia(ctx context.Context, job *GenerationJob) (map[string]any, error) {
	if job.Modality == "image" || IsTerminalMediaSuccessStatus(string(job.Status)) {
		return job.ResultPayload, nil
	}
	account, err := e.account(ctx, job)
	if err != nil {
		return nil, err
	}
	c, recorder := mediaGinContext(ctx, http.MethodGet, "/v1/videos/"+mediaStringValue(job.UpstreamGenerationID), nil)
	if err = e.gateway.ForwardXAIVideoStatus(ctx, c, account, mediaStringValue(job.UpstreamGenerationID)); err != nil {
		return nil, err
	}
	result := map[string]any{}
	err = json.Unmarshal(recorder.Body.Bytes(), &result)
	return result, err
}

func (e *DefaultMediaOpenAIExecutor) MediaContent(ctx context.Context, job *GenerationJob, index int) (*MediaContent, error) {
	if index != 0 || job.Modality != "video" {
		return nil, ErrGenerationJobNotFound
	}
	account, err := e.account(ctx, job)
	if err != nil {
		return nil, err
	}
	c, recorder := mediaGinContext(ctx, http.MethodGet, "/v1/videos/"+mediaStringValue(job.UpstreamGenerationID)+"/content", nil)
	if err = e.gateway.ForwardXAIVideoContent(ctx, c, account, mediaStringValue(job.UpstreamGenerationID)); err != nil {
		return nil, err
	}
	return &MediaContent{Body: io.NopCloser(bytes.NewReader(recorder.Body.Bytes())), ContentType: recorder.Header().Get("Content-Type"), Length: int64(recorder.Body.Len())}, nil
}

func (e *DefaultMediaOpenAIExecutor) account(ctx context.Context, job *GenerationJob) (*Account, error) {
	if e == nil || e.gateway == nil || e.accounts == nil || job == nil || job.AccountID <= 0 {
		return nil, ErrMediaProviderUnavailable
	}
	return e.accounts.GetByID(ctx, job.AccountID)
}

func mediaGinContext(ctx context.Context, method, path string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, path, bytes.NewReader(body)).WithContext(ctx)
	c.Request.Header.Set("Content-Type", "application/json")
	return c, recorder
}

func mediaRequestBody(body []byte, model string) ([]byte, error) {
	value := map[string]any{}
	if err := json.Unmarshal(body, &value); err != nil {
		return nil, err
	}
	value["model"] = model
	return json.Marshal(value)
}

func mediaOperationPath(modality, operation string) string {
	if modality == "image" {
		return "/v1/images/" + operation
	}
	if operation == "generations" {
		return "/v1/videos"
	}
	return "/v1/videos/" + operation
}

func mediaResultID(result map[string]any) string {
	for _, key := range []string{"id", "request_id", "job_id", "task_id"} {
		if value := strings.TrimSpace(fmt.Sprint(result[key])); value != "" && value != "<nil>" {
			return value
		}
	}
	if data, ok := result["data"].(map[string]any); ok {
		return mediaResultID(data)
	}
	return ""
}

func errorsNewMediaUpstreamID() error {
	return fmt.Errorf("media upstream response has no task id: %s", strconv.Itoa(http.StatusBadGateway))
}

func mediaOpenAIErrorState(err error) MediaSubmissionState {
	var failover *UpstreamFailoverError
	if errors.As(err, &failover) && failover.StatusCode > 0 {
		return MediaSubmissionNotWritten
	}
	return MediaSubmissionSideEffectUnknown
}
