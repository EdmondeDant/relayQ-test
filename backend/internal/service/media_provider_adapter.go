package service

import (
	"context"
	"io"
)

type MediaCanonicalRequest struct {
	Operation string
	Model     string
	Modality  string
	Body      []byte
	Fields    map[string]any
}

type MediaSubmissionOutcome struct {
	State      MediaSubmissionState
	UpstreamID string
	AccountID  int64
	Status     string
	Result     map[string]any
}

type MediaContent struct {
	Body        io.ReadCloser
	ContentType string
	Length      int64
}

type MediaProviderAdapter interface {
	Provider() string
	Submit(context.Context, *GenerationJob, MediaCanonicalRequest, MediaCatalogOffer) (MediaSubmissionOutcome, error)
	Poll(context.Context, *GenerationJob) (map[string]any, error)
	Content(context.Context, *GenerationJob, int) (*MediaContent, error)
}
