package service

import "context"

type MediaLeonardoExecutor interface {
	SubmitMedia(context.Context, *GenerationJob, MediaCanonicalRequest, MediaCatalogOffer) (MediaSubmissionOutcome, error)
	PollMedia(context.Context, *GenerationJob) (map[string]any, error)
	MediaContent(context.Context, *GenerationJob, int) (*MediaContent, error)
}

type MediaLeonardoAdapter struct{ executor MediaLeonardoExecutor }

func NewMediaLeonardoAdapter(executor *DefaultMediaLeonardoExecutor) *MediaLeonardoAdapter {
	return &MediaLeonardoAdapter{executor: executor}
}

func (a *MediaLeonardoAdapter) Provider() string {
	return PlatformLeonardo
}

func (a *MediaLeonardoAdapter) Submit(ctx context.Context, job *GenerationJob, request MediaCanonicalRequest, offer MediaCatalogOffer) (MediaSubmissionOutcome, error) {
	if a == nil || a.executor == nil {
		return MediaSubmissionOutcome{State: MediaSubmissionNotWritten}, ErrMediaProviderUnavailable
	}
	return a.executor.SubmitMedia(ctx, job, request, offer)
}

func (a *MediaLeonardoAdapter) Poll(ctx context.Context, job *GenerationJob) (map[string]any, error) {
	if a == nil || a.executor == nil {
		return nil, ErrMediaProviderUnavailable
	}
	return a.executor.PollMedia(ctx, job)
}

func (a *MediaLeonardoAdapter) Content(ctx context.Context, job *GenerationJob, index int) (*MediaContent, error) {
	if a == nil || a.executor == nil {
		return nil, ErrMediaProviderUnavailable
	}
	return a.executor.MediaContent(ctx, job, index)
}

var _ MediaProviderAdapter = (*MediaLeonardoAdapter)(nil)
