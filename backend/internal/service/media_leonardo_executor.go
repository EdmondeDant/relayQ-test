package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
)

type DefaultMediaLeonardoExecutor struct {
	create *LeonardoMediaCreateService
	get    *LeonardoMediaGetService
}

func NewDefaultMediaLeonardoExecutor(create *LeonardoMediaCreateService, get *LeonardoMediaGetService) *DefaultMediaLeonardoExecutor {
	return &DefaultMediaLeonardoExecutor{create: create, get: get}
}

func (e *DefaultMediaLeonardoExecutor) SubmitMedia(ctx context.Context, job *GenerationJob, request MediaCanonicalRequest, offer MediaCatalogOffer) (MediaSubmissionOutcome, error) {
	if e == nil || e.create == nil || job == nil || job.GroupID == nil {
		return MediaSubmissionOutcome{State: MediaSubmissionNotWritten}, ErrMediaProviderUnavailable
	}
	if request.Operation != "generations" || e.create.accounts == nil || e.create.orchestrator == nil || e.create.orchestrator.clients == nil {
		return MediaSubmissionOutcome{State: MediaSubmissionNotWritten, AccountID: job.AccountID}, ErrMediaProviderUnavailable
	}
	account, err := e.create.accounts.GetByID(ctx, job.AccountID)
	if err != nil {
		return MediaSubmissionOutcome{State: MediaSubmissionNotWritten, AccountID: job.AccountID}, err
	}
	if !validLeonardoMediaAccount(account, offer.SourceGroupID, offer.UpstreamModel) {
		return MediaSubmissionOutcome{State: MediaSubmissionNotWritten, AccountID: job.AccountID}, ErrLeonardoMediaNoAvailableAccount
	}
	client, err := e.create.orchestrator.clients.Build(account)
	if err != nil {
		return MediaSubmissionOutcome{State: MediaSubmissionNotWritten, AccountID: job.AccountID}, err
	}
	body, err := leonardoUnifiedMediaBody(request, offer.UpstreamModel)
	if err != nil {
		return MediaSubmissionOutcome{State: MediaSubmissionNotWritten, AccountID: job.AccountID}, err
	}
	response, err := client.CreateGenerationRaw(ctx, body)
	if err != nil {
		if errors.Is(err, ErrLeonardoGenerationRequestNotWritten) {
			return MediaSubmissionOutcome{State: MediaSubmissionNotWritten, AccountID: job.AccountID}, err
		}
		return MediaSubmissionOutcome{State: MediaSubmissionSideEffectUnknown, AccountID: job.AccountID, Result: map[string]any{"submission_diagnostic": leonardoSubmissionDiagnostic(err)}}, err
	}
	if response == nil || !validLeonardoGenerationUUID(response.GenerationID) {
		class := leonardo.GenerationErrorClassGenerationIDInvalid
		if response == nil || response.GenerationID == "" {
			class = leonardo.GenerationErrorClassGenerationIDMissing
		}
		return MediaSubmissionOutcome{State: MediaSubmissionSideEffectUnknown, AccountID: job.AccountID, Result: map[string]any{"submission_diagnostic": map[string]any{"class": class}}}, errors.New("leonardo generation submission status is unknown")
	}
	return MediaSubmissionOutcome{State: MediaSubmissionSubmitted, UpstreamID: response.GenerationID, AccountID: job.AccountID, Status: "PENDING", Result: leonardoGenerationCostPayload(response)}, nil
}

func leonardoUnifiedMediaBody(request MediaCanonicalRequest, model string) ([]byte, error) {
	prompt := strings.TrimSpace(mediaStringValueAny(request.Fields["prompt"]))
	quantity := mediaRequestQuantity(request.Fields)
	public, _ := request.Fields["public"].(bool)
	if prompt == "" || quantity <= 0 {
		return nil, ErrLeonardoMediaCreateInputInvalid
	}
	width, height := mediaUnifiedDimensions(request.Fields)
	var parameters map[string]any
	if request.Modality == "video" {
		duration := int(mediaFloatField(request.Fields, "duration", "duration_seconds", "seconds"))
		var err error
		parameters, err = LeonardoVideoGenerationParameters(model, prompt, duration, width, height, quantity)
		if err != nil {
			return nil, err
		}
	} else if request.Modality == "image" && width > 0 && height > 0 {
		parameters = map[string]any{"prompt": prompt, "width": width, "height": height, "quantity": quantity}
		quality := strings.ToLower(strings.TrimSpace(mediaStringValueAny(request.Fields["quality"])))
		if quality == "" {
			quality = "low"
		}
		if model == "gpt-image-2" {
			parameters["quality"] = strings.ToUpper(quality)
		}
		if model == "kino-xl" || model == "concept-art" || model == "graphic-design" || model == "illustrative-albedo" {
			mode := map[string]string{"low": "FAST", "high": "QUALITY"}[quality]
			if mode == "" {
				return nil, ErrLeonardoMediaCreateInputInvalid
			}
			parameters["mode"] = mode
			parameters["prompt_enhance"] = "OFF"
		}
	} else {
		return nil, ErrLeonardoMediaCreateInputInvalid
	}
	return json.Marshal(leonardo.CreateGenerationRequest{Model: model, Public: public, Parameters: parameters})
}

func mediaUnifiedDimensions(fields map[string]any) (int, int) {
	width := int(mediaFloatField(fields, "width"))
	height := int(mediaFloatField(fields, "height"))
	if width > 0 && height > 0 {
		return width, height
	}
	return parseDimensionPair(mediaStringField(fields, "size", "resolution"))
}

func (e *DefaultMediaLeonardoExecutor) PollMedia(ctx context.Context, job *GenerationJob) (map[string]any, error) {
	if e == nil || e.get == nil || e.get.poller == nil || job == nil || job.GroupID == nil || job.UpstreamGenerationID == nil {
		return nil, ErrMediaProviderUnavailable
	}
	poller := e.get.poller
	if poller.accountLoader == nil || poller.upstream == nil || poller.config == nil || poller.clock == nil || job.AccountID <= 0 {
		return nil, ErrMediaProviderUnavailable
	}
	account, err := poller.accountLoader.GetByID(ctx, job.AccountID)
	if err != nil {
		return nil, err
	}
	adapter, err := NewLeonardoGenerationAdapter(account, poller.upstream, poller.config)
	if err != nil {
		return nil, err
	}
	generation, err := adapter.GetGeneration(ctx, *job.UpstreamGenerationID)
	if err != nil {
		return nil, err
	}
	if generation == nil {
		return nil, errors.New("leonardo: empty generation response")
	}
	updated, err := applyLeonardoGenerationResult(ctx, job, generation, poller.clock.Now().UTC(), poller.moderator)
	result := newLeonardoMediaGetResult(&updated)
	return map[string]any{"id": result.ID, "status": result.Status, "data": result.Data, "videos": result.Videos}, err
}

func (e *DefaultMediaLeonardoExecutor) MediaContent(ctx context.Context, job *GenerationJob, index int) (*MediaContent, error) {
	if e == nil || e.get == nil || job == nil || job.GroupID == nil || index < 0 {
		return nil, ErrMediaProviderUnavailable
	}
	if job.Status != GenerationJobStatusSucceeded {
		return nil, ErrLeonardoMediaContentNotReady
	}
	outputs := leonardoMediaOutputURLs(job)
	if index >= len(outputs) {
		return nil, ErrLeonardoMediaContentNotFound
	}
	content, err := e.get.downloadContent(ctx, outputs[index], job.Modality)
	if err != nil {
		return nil, err
	}
	length := int64(-1)
	if info, statErr := content.File.Stat(); statErr == nil {
		length = info.Size()
	}
	return &MediaContent{Body: structReadCloser{Reader: content.File, close: content.Close}, ContentType: content.ContentType, Length: length}, nil
}

type structReadCloser struct {
	io.Reader
	close func() error
}

func (r structReadCloser) Close() error { return r.close() }
