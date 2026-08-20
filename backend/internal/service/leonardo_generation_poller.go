package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
)

const (
	leonardoPollBaseBackoff  = 2 * time.Second
	leonardoPollPendingDelay = 5 * time.Second
	leonardoPollMaxBackoff   = 5 * time.Minute
	leonardoPollErrorLimit   = 512
)

type LeonardoGenerationPollClient interface {
	GetGeneration(context.Context, string) (*leonardo.Generation, error)
}

type LeonardoGenerationPollClock interface {
	Now() time.Time
}

type LeonardoOutputModerator interface {
	Check(context.Context, ContentModerationCheckInput) (*ContentModerationDecision, error)
}

type LeonardoGenerationPoller struct {
	repository GenerationJobPollRepository
	client     LeonardoGenerationPollClient
	clock      LeonardoGenerationPollClock
	moderator  LeonardoOutputModerator
}

var ErrLeonardoVideoSchemaUnverified = errors.New("leonardo video response schema is not verified")

var ErrLeonardoAudioSchemaUnverified = errors.New("leonardo audio response schema is not verified")

var ErrLeonardo3DSchemaUnverified = errors.New("leonardo 3d response schema is not verified")

var ErrLeonardoMediaModalityUnverified = errors.New("leonardo media modality is not verified")

func NewLeonardoGenerationPoller(repository GenerationJobPollRepository, client LeonardoGenerationPollClient, clock LeonardoGenerationPollClock, moderators ...LeonardoOutputModerator) *LeonardoGenerationPoller {
	var moderator LeonardoOutputModerator
	if len(moderators) > 0 {
		moderator = moderators[0]
	}
	return &LeonardoGenerationPoller{repository: repository, client: client, clock: clock, moderator: moderator}
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
	if err := requireLeonardoSupportedModality(job.Modality); err != nil {
		return job, err
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

	updated, pollErr = applyLeonardoGenerationResult(ctx, &updated, generation, now, p.moderator)
	if pollErr != nil && updated.ErrorCode == nil {
		return job, pollErr
	}
	if err := p.repository.CompareAndSwapPoll(ctx, job.PublicID, job.Status, job.PollAttempts, &updated); err != nil {
		return job, errors.Join(pollErr, err)
	}
	return &updated, pollErr
}

func requireLeonardoSupportedModality(modality string) error {
	switch strings.ToLower(strings.TrimSpace(modality)) {
	case "image", "video":
		return nil
	case "audio":
		return ErrLeonardoAudioSchemaUnverified
	case "3d":
		return ErrLeonardo3DSchemaUnverified
	default:
		return ErrLeonardoMediaModalityUnverified
	}
}

func applyLeonardoGenerationResult(ctx context.Context, job *GenerationJob, generation *leonardo.Generation, now time.Time, moderator LeonardoOutputModerator) (GenerationJob, error) {
	updated := *job
	updated.UpstreamStatus = stringPointer(generation.Status)
	updated.ErrorCode = nil
	updated.ErrorMessage = nil
	var resultErr error
	switch generation.Status {
	case "PENDING":
		updated.NextPollAt = timePointerValue(now.Add(leonardoPollPendingDelay))
	case "COMPLETE":
		if strings.EqualFold(strings.TrimSpace(job.Modality), "video") {
			return applyLeonardoVideoResult(updated, generation, now)
		}
		if len(generation.GeneratedImages) == 0 {
			resultErr = errors.New("leonardo: complete generation returned no image outputs")
			updated.ErrorCode = stringPointer("invalid_upstream_output")
			updated.ErrorMessage = stringPointer(resultErr.Error())
			updated.NextPollAt = timePointerValue(now.Add(leonardoPollBackoff(updated.PollAttempts, resultErr)))
			break
		}
		images := make([]map[string]any, 0, len(generation.GeneratedImages))
		policyBlocked := false
		for _, image := range generation.GeneratedImages {
			if image.NSFW == nil || *image.NSFW {
				policyBlocked = true
				continue
			}
			allowed, err := allowLeonardoOutputImage(ctx, job, image.URL, moderator)
			if err != nil {
				return updated, err
			}
			if !allowed {
				policyBlocked = true
				continue
			}
			images = append(images, map[string]any{"id": image.ID, "url": image.URL, "nsfw": false})
		}
		if policyBlocked {
			updated.Status = GenerationJobStatusFailed
			updated.NextPollAt = nil
			updated.FailedAt = timePointerValue(now)
			updated.OutputCount = 0
			updated.BillingStatus = GenerationJobBillingStatusManualReview
			updated.ErrorCode = stringPointer("content_policy_violation")
			updated.ErrorMessage = stringPointer("leonardo generation output was blocked by content policy")
			break
		}
		updated.Status = GenerationJobStatusSucceeded
		updated.NextPollAt = nil
		updated.CompletedAt = timePointerValue(now)
		updated.OutputCount = len(images)
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
		resultErr = errors.New("leonardo: unsupported generation status")
		updated.ErrorCode = stringPointer("poll_error")
		updated.ErrorMessage = stringPointer(resultErr.Error())
		updated.NextPollAt = timePointerValue(now.Add(leonardoPollBackoff(updated.PollAttempts, resultErr)))
	}
	return updated, resultErr
}

func applyLeonardoVideoResult(updated GenerationJob, generation *leonardo.Generation, now time.Time) (GenerationJob, error) {
	videos := make([]map[string]any, 0, len(generation.GeneratedImages))
	for _, output := range generation.GeneratedImages {
		if output.NSFW == nil || *output.NSFW || !validLeonardoHTTPSURL(output.MotionMP4URL) || strings.TrimSpace(output.ID) == "" {
			continue
		}
		video := map[string]any{"id": strings.TrimSpace(output.ID), "url": strings.TrimSpace(output.MotionMP4URL), "mime": "video/mp4", "nsfw": false}
		for _, key := range []string{"duration", "width", "height"} {
			if value, ok := leonardoVideoRequestParameter(updated.RequestPayload, key); ok {
				video[key] = value
			}
		}
		videos = append(videos, video)
	}
	if len(videos) == 0 {
		err := errors.New("leonardo: complete generation returned no valid video outputs")
		updated.ErrorCode = stringPointer("invalid_upstream_output")
		updated.ErrorMessage = stringPointer(err.Error())
		updated.NextPollAt = timePointerValue(now.Add(leonardoPollBackoff(updated.PollAttempts, err)))
		return updated, err
	}
	updated.Status = GenerationJobStatusSucceeded
	updated.NextPollAt = nil
	updated.CompletedAt = timePointerValue(now)
	updated.OutputCount = len(videos)
	resultPayload := make(map[string]any, len(updated.ResultPayload)+1)
	for key, value := range updated.ResultPayload {
		resultPayload[key] = value
	}
	resultPayload["videos"] = videos
	updated.ResultPayload = resultPayload
	return updated, nil
}

func validLeonardoHTTPSURL(rawURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}

func leonardoVideoRequestParameter(payload map[string]any, key string) (any, bool) {
	parameters, ok := payload["parameters"].(map[string]any)
	if !ok {
		return nil, false
	}
	value, ok := parameters[key]
	return value, ok
}

func allowLeonardoOutputImage(ctx context.Context, job *GenerationJob, imageURL string, moderator LeonardoOutputModerator) (bool, error) {
	if moderator == nil {
		return true, nil
	}
	body, err := json.Marshal(map[string]any{"images": []map[string]string{{"image_url": imageURL}}})
	if err != nil {
		return false, err
	}
	decision, err := moderator.Check(ctx, ContentModerationCheckInput{
		RequestID: job.PublicID,
		UserID:    job.UserID,
		APIKeyID:  job.APIKeyID,
		GroupID:   job.GroupID,
		Endpoint:  "/v1/media/generations/:id",
		Provider:  PlatformLeonardo,
		Model:     job.Model,
		Protocol:  ContentModerationProtocolOpenAIImages,
		Body:      body,
	})
	if err != nil {
		return false, err
	}
	if decision == nil {
		return false, errors.New("leonardo output moderation returned no decision")
	}
	if decision.Action == ContentModerationActionError {
		return false, errors.New("leonardo output moderation unavailable")
	}
	return !decision.Blocked, nil
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
