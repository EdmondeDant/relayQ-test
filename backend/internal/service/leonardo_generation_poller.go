package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
)

const (
	leonardoPollBaseBackoff = 2 * time.Second
	leonardoPollMaxBackoff  = 5 * time.Minute
	leonardoPollErrorLimit  = 512
)

type LeonardoGenerationPollClient interface {
	GetGeneration(context.Context, string) (*leonardo.Generation, error)
}

type LeonardoGenerationPollClock interface {
	Now() time.Time
}

type LeonardoGenerationPoller struct {
	repository GenerationJobPollRepository
	client     LeonardoGenerationPollClient
	clock      LeonardoGenerationPollClock
}

func NewLeonardoGenerationPoller(repository GenerationJobPollRepository, client LeonardoGenerationPollClient, clock LeonardoGenerationPollClock) *LeonardoGenerationPoller {
	return &LeonardoGenerationPoller{repository: repository, client: client, clock: clock}
}

func (p *LeonardoGenerationPoller) Poll(ctx context.Context, publicID string) (*GenerationJob, error) {
	if p == nil || p.repository == nil || p.client == nil || p.clock == nil {
		return nil, errors.New("leonardo generation poller is not configured")
	}
	job, err := p.repository.GetByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, ErrGenerationJobNotFound
	}
	if job.Status == GenerationJobStatusUnknown {
		return job, nil
	}
	now := p.clock.Now().UTC()
	if (job.Status != GenerationJobStatusQueued && job.Status != GenerationJobStatusRunning) || job.UpstreamGenerationID == nil || !validLeonardoGenerationUUID(*job.UpstreamGenerationID) || (job.NextPollAt != nil && job.NextPollAt.After(now)) {
		return job, nil
	}

	generation, pollErr := p.client.GetGeneration(ctx, *job.UpstreamGenerationID)
	updated := *job
	updated.PollAttempts++
	updated.LastPolledAt = timePointerValue(now)
	if pollErr != nil {
		updated.ErrorCode = stringPointer("poll_error")
		updated.ErrorMessage = stringPointer(sanitizeLeonardoPollError(pollErr))
		updated.NextPollAt = timePointerValue(now.Add(leonardoPollBackoff(updated.PollAttempts, pollErr)))
		if err := p.repository.CompareAndSwapPoll(ctx, job.PublicID, job.Status, job.PollAttempts, &updated); err != nil {
			return job, errors.Join(pollErr, err)
		}
		return &updated, pollErr
	}
	if generation == nil {
		pollErr = errors.New("leonardo: empty generation response")
		updated.ErrorCode = stringPointer("poll_error")
		updated.ErrorMessage = stringPointer(pollErr.Error())
		updated.NextPollAt = timePointerValue(now.Add(leonardoPollBackoff(updated.PollAttempts, pollErr)))
		if err := p.repository.CompareAndSwapPoll(ctx, job.PublicID, job.Status, job.PollAttempts, &updated); err != nil {
			return job, errors.Join(pollErr, err)
		}
		return &updated, pollErr
	}

	updated.UpstreamStatus = stringPointer(generation.Status)
	updated.ErrorCode = nil
	updated.ErrorMessage = nil
	switch generation.Status {
	case "PENDING":
		updated.NextPollAt = timePointerValue(now.Add(leonardoPollBackoff(updated.PollAttempts, nil)))
	case "COMPLETE":
		updated.Status = GenerationJobStatusSucceeded
		updated.NextPollAt = nil
		updated.CompletedAt = timePointerValue(now)
		updated.OutputCount = len(generation.GeneratedImages)
		images := make([]map[string]any, len(generation.GeneratedImages))
		for i, image := range generation.GeneratedImages {
			images[i] = map[string]any{"id": image.ID, "url": image.URL, "nsfw": image.NSFW}
		}
		resultPayload := make(map[string]any, len(updated.ResultPayload)+1)
		for key, value := range updated.ResultPayload {
			resultPayload[key] = value
		}
		resultPayload["images"] = images
		updated.ResultPayload = resultPayload
	case "FAILED":
		updated.Status = GenerationJobStatusFailed
		updated.NextPollAt = nil
		updated.FailedAt = timePointerValue(now)
		updated.ErrorCode = stringPointer("upstream_failed")
		updated.ErrorMessage = stringPointer("leonardo generation failed")
	default:
		pollErr = errors.New("leonardo: unsupported generation status")
		updated.ErrorCode = stringPointer("poll_error")
		updated.ErrorMessage = stringPointer(pollErr.Error())
		updated.NextPollAt = timePointerValue(now.Add(leonardoPollBackoff(updated.PollAttempts, pollErr)))
	}
	if err := p.repository.CompareAndSwapPoll(ctx, job.PublicID, job.Status, job.PollAttempts, &updated); err != nil {
		return job, errors.Join(pollErr, err)
	}
	return &updated, pollErr
}

func leonardoPollBackoff(attempt int, err error) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	backoff := leonardoPollBaseBackoff
	for i := 1; i < attempt && backoff < leonardoPollMaxBackoff; i++ {
		backoff *= 2
		if backoff > leonardoPollMaxBackoff {
			backoff = leonardoPollMaxBackoff
		}
	}
	var apiErr *leonardo.LeonardoError
	if errors.As(err, &apiErr) && apiErr != nil && apiErr.RetryAfter > backoff {
		backoff = apiErr.RetryAfter
	}
	return backoff
}

func sanitizeLeonardoPollError(err error) string {
	message := "leonardo generation poll failed"
	if err != nil {
		message = logredact.RedactText(err.Error(), "api_key", "apikey", "authorization", "cookie", "signature", "x-api-key", "x-amz-signature")
	}
	message = strings.TrimSpace(message)
	if len(message) > leonardoPollErrorLimit {
		message = message[:leonardoPollErrorLimit]
	}
	return message
}
