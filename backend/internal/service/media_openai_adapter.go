package service

import (
	"context"
)

type MediaOpenAIExecutor interface {
	SubmitMedia(context.Context, *GenerationJob, MediaCanonicalRequest, MediaCatalogOffer) (MediaSubmissionOutcome, error)
	PollMedia(context.Context, *GenerationJob) (map[string]any, error)
	MediaContent(context.Context, *GenerationJob, int) (*MediaContent, error)
}

type MediaOpenAIAdapter struct{ executor MediaOpenAIExecutor }

func NewMediaOpenAIAdapter(executor *DefaultMediaOpenAIExecutor) *MediaOpenAIAdapter {
	return &MediaOpenAIAdapter{executor: executor}
}

func (a *MediaOpenAIAdapter) Provider() string {
	return PlatformOpenAI
}

func (a *MediaOpenAIAdapter) Submit(ctx context.Context, job *GenerationJob, request MediaCanonicalRequest, offer MediaCatalogOffer) (MediaSubmissionOutcome, error) {
	if a == nil || a.executor == nil {
		return MediaSubmissionOutcome{State: MediaSubmissionNotWritten}, ErrMediaProviderUnavailable
	}
	return a.executor.SubmitMedia(ctx, job, request, offer)
}

func (a *MediaOpenAIAdapter) Poll(ctx context.Context, job *GenerationJob) (map[string]any, error) {
	if a == nil || a.executor == nil {
		return nil, ErrMediaProviderUnavailable
	}
	return a.executor.PollMedia(ctx, job)
}

func (a *MediaOpenAIAdapter) Content(ctx context.Context, job *GenerationJob, index int) (*MediaContent, error) {
	if a == nil || a.executor == nil {
		return nil, ErrMediaProviderUnavailable
	}
	return a.executor.MediaContent(ctx, job, index)
}

var _ MediaProviderAdapter = (*MediaOpenAIAdapter)(nil)
