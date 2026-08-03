package service

import (
	"context"
	"net/url"
	"regexp"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrLeonardoMediaGetNotConfigured = infraerrors.InternalServer("LEONARDO_MEDIA_GET_NOT_CONFIGURED", "Leonardo media get service is not configured")
	ErrLeonardoMediaGetInputInvalid  = infraerrors.BadRequest("LEONARDO_MEDIA_GET_INPUT_INVALID", "Leonardo media generation ID is invalid")
	leonardoMediaPublicIDPattern     = regexp.MustCompile(`^gen_rq_[0-9a-f]{32}$`)
)

type LeonardoMediaGetInput struct {
	PublicID string
	UserID   int64
	APIKeyID int64
	GroupID  int64
}

type LeonardoMediaGetResult struct {
	ID          string                  `json:"id"`
	Object      string                  `json:"object"`
	Provider    string                  `json:"provider"`
	Model       string                  `json:"model"`
	Modality    string                  `json:"modality"`
	Status      string                  `json:"status"`
	CreatedAt   int64                   `json:"created_at"`
	UpdatedAt   int64                   `json:"updated_at"`
	CompletedAt int64                   `json:"completed_at"`
	Error       *LeonardoMediaGetError  `json:"error"`
	Data        []LeonardoMediaGetImage `json:"data"`
}

type LeonardoMediaGetError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type LeonardoMediaGetImage struct {
	ID   string `json:"id"`
	URL  string `json:"url"`
	NSFW bool   `json:"nsfw"`
}

type LeonardoMediaGetService struct {
	repository GenerationJobPollRepository
	poller     *LeonardoGenerationPollOrchestrator
}

func NewLeonardoMediaGetService(repository GenerationJobPollRepository, poller *LeonardoGenerationPollOrchestrator) *LeonardoMediaGetService {
	return &LeonardoMediaGetService{repository: repository, poller: poller}
}

func ValidLeonardoMediaPublicID(publicID string) bool {
	return leonardoMediaPublicIDPattern.MatchString(strings.TrimSpace(publicID))
}

func (s *LeonardoMediaGetService) Get(ctx context.Context, input LeonardoMediaGetInput) (*LeonardoMediaGetResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	publicID := strings.TrimSpace(input.PublicID)
	if !ValidLeonardoMediaPublicID(publicID) || input.UserID <= 0 || input.APIKeyID <= 0 || input.GroupID <= 0 {
		return nil, ErrLeonardoMediaGetInputInvalid
	}
	if s == nil || s.repository == nil || s.poller == nil {
		return nil, ErrLeonardoMediaGetNotConfigured
	}
	job, err := s.repository.GetByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}
	if !ownsLeonardoMediaJob(job, input) {
		return nil, ErrGenerationJobNotFound
	}
	job, err = s.poller.Poll(ctx, publicID)
	if job != nil && !ownsLeonardoMediaJob(job, input) {
		return nil, ErrGenerationJobNotFound
	}
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, ErrGenerationJobNotFound
	}
	return newLeonardoMediaGetResult(job), nil
}

func ownsLeonardoMediaJob(job *GenerationJob, input LeonardoMediaGetInput) bool {
	return job != nil && job.UserID == input.UserID && job.APIKeyID == input.APIKeyID && job.GroupID != nil && *job.GroupID == input.GroupID && job.Provider == PlatformLeonardo
}

func newLeonardoMediaGetResult(job *GenerationJob) *LeonardoMediaGetResult {
	result := &LeonardoMediaGetResult{ID: job.PublicID, Object: "media.generation", Provider: job.Provider, Model: job.Model, Modality: job.Modality, Status: string(job.Status), Data: []LeonardoMediaGetImage{}}
	if !job.CreatedAt.IsZero() {
		result.CreatedAt = job.CreatedAt.Unix()
	}
	if !job.UpdatedAt.IsZero() {
		result.UpdatedAt = job.UpdatedAt.Unix()
	}
	if job.CompletedAt != nil {
		result.CompletedAt = job.CompletedAt.Unix()
	}
	if job.Status == GenerationJobStatusFailed {
		result.Error = &LeonardoMediaGetError{Code: stringValue(job.ErrorCode), Message: stringValue(job.ErrorMessage)}
	}
	result.Data = leonardoMediaImages(job.ResultPayload)
	return result
}

func leonardoMediaImages(payload map[string]any) []LeonardoMediaGetImage {
	images := []LeonardoMediaGetImage{}
	var values []any
	switch typed := payload["images"].(type) {
	case []any:
		values = typed
	case []map[string]any:
		values = make([]any, len(typed))
		for i := range typed {
			values[i] = typed[i]
		}
	default:
		return images
	}
	for _, value := range values {
		image, ok := value.(map[string]any)
		if !ok {
			continue
		}
		id, idOK := image["id"].(string)
		rawURL, urlOK := image["url"].(string)
		id, rawURL = strings.TrimSpace(id), strings.TrimSpace(rawURL)
		parsed, err := url.Parse(rawURL)
		if !idOK || !urlOK || id == "" || err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			continue
		}
		nsfw, _ := image["nsfw"].(bool)
		images = append(images, LeonardoMediaGetImage{ID: id, URL: rawURL, NSFW: nsfw})
	}
	return images
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
