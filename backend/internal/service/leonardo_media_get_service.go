package service

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
)

var (
	ErrLeonardoMediaGetNotConfigured = infraerrors.InternalServer("LEONARDO_MEDIA_GET_NOT_CONFIGURED", "Leonardo media get service is not configured")
	ErrLeonardoMediaGetInputInvalid  = infraerrors.BadRequest("LEONARDO_MEDIA_GET_INPUT_INVALID", "Leonardo media generation ID is invalid")
	ErrLeonardoMediaContentInvalid   = infraerrors.BadRequest("LEONARDO_MEDIA_CONTENT_INPUT_INVALID", "Leonardo media content request is invalid")
	ErrLeonardoMediaContentNotReady  = infraerrors.Conflict("LEONARDO_MEDIA_CONTENT_NOT_READY", "Leonardo media generation has not succeeded")
	ErrLeonardoMediaContentNotFound  = infraerrors.NotFound("LEONARDO_MEDIA_CONTENT_NOT_FOUND", "Leonardo media output not found")
	ErrLeonardoMediaContentFailed    = infraerrors.New(http.StatusBadGateway, "LEONARDO_MEDIA_CONTENT_UNAVAILABLE", "Leonardo media content is unavailable")
	ErrLeonardoMediaVideoUnverified  = infraerrors.Conflict("LEONARDO_VIDEO_SCHEMA_UNVERIFIED", "Leonardo video response schema is not verified")
	leonardoMediaPublicIDPattern     = regexp.MustCompile(`^gen_rq_[0-9a-f]{32}$`)
)

const (
	leonardoImageContentMaxBytes = 50 << 20
	leonardoVideoContentMaxBytes = 500 << 20
	// Keep CDN content downloads short so OpenAI-compatible sync endpoints can
	// fall back to returning the remote URL instead of hanging forever.
	leonardoCDNContentTimeout = 12 * time.Second
)

type LeonardoMediaGetInput struct {
	PublicID string
	UserID   int64
	APIKeyID int64
	GroupID  int64
}

type LeonardoMediaContentInput struct {
	LeonardoMediaGetInput
	Index int
}

type LeonardoMediaContent struct {
	File        *os.File
	ContentType string
	path        string
}

func (c *LeonardoMediaContent) Close() error {
	if c == nil {
		return nil
	}
	var closeErr error
	if c.File != nil {
		closeErr = c.File.Close()
	}
	if c.path != "" {
		if err := os.Remove(c.path); closeErr == nil && !errors.Is(err, os.ErrNotExist) {
			closeErr = err
		}
	}
	return closeErr
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
	Videos      []LeonardoMediaGetVideo `json:"videos,omitempty"`
	// OpenAI/Sora-compatible aliases consumed by Infinite Canvas polling.
	// Canvas only treats top-level video_url/url (or status=completed + /content) as ready.
	VideoURL  string                     `json:"video_url,omitempty"`
	URL       string                     `json:"url,omitempty"`
	ResultURL string                     `json:"result_url,omitempty"`
	Content   *LeonardoMediaGetURLBundle `json:"content,omitempty"`
}

type LeonardoMediaGetURLBundle struct {
	VideoURL string `json:"video_url,omitempty"`
	URL      string `json:"url,omitempty"`
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

type LeonardoMediaGetVideo struct {
	ID       string  `json:"id"`
	URL      string  `json:"url"`
	MIME     string  `json:"mime"`
	Duration float64 `json:"duration,omitempty"`
	Width    int     `json:"width,omitempty"`
	Height   int     `json:"height,omitempty"`
	NSFW     bool    `json:"nsfw"`
}

type LeonardoMediaGetService struct {
	repository GenerationJobPollRepository
	poller     *LeonardoGenerationPollOrchestrator
	content    *http.Client
}

func NewLeonardoMediaGetService(repository GenerationJobPollRepository, poller *LeonardoGenerationPollOrchestrator) *LeonardoMediaGetService {
	client := newSSRFSafeHTTPClient(30 * time.Second)
	proxyURL := strings.TrimSpace(os.Getenv("HTTPS_PROXY"))
	if proxyURL == "" {
		proxyURL = strings.TrimSpace(os.Getenv("https_proxy"))
	}
	if proxyURL != "" {
		proxyClient, err := httpclient.GetClient(httpclient.Options{ProxyURL: proxyURL, Timeout: 30 * time.Second, ResponseHeaderTimeout: leonardoCDNContentTimeout, ValidateResolvedIP: true})
		if err != nil {
			client.Transport = leonardoMediaRoundTripperError{err: err}
		} else {
			local := *proxyClient
			client = &local
		}
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &LeonardoMediaGetService{repository: repository, poller: poller, content: client}
}

type leonardoMediaRoundTripperError struct {
	err error
}

func (t leonardoMediaRoundTripperError) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, t.err
}

func ValidLeonardoMediaPublicID(publicID string) bool {
	return leonardoMediaPublicIDPattern.MatchString(strings.TrimSpace(publicID))
}

func (s *LeonardoMediaGetService) Get(ctx context.Context, input LeonardoMediaGetInput) (*LeonardoMediaGetResult, error) {
	job, err := s.getOwnedJob(ctx, input)
	if err != nil {
		return nil, err
	}
	return newLeonardoMediaGetResult(job), nil
}

func (s *LeonardoMediaGetService) getOwnedJob(ctx context.Context, input LeonardoMediaGetInput) (*GenerationJob, error) {
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
	if err = requireLeonardoSupportedModality(job.Modality); err != nil {
		return nil, err
	}
	job, err = s.poller.Poll(ctx, publicID)
	if job != nil && !ownsLeonardoMediaJob(job, input) {
		return nil, ErrGenerationJobNotFound
	}
	if errors.Is(err, ErrGenerationJobConflict) {
		job, err = s.repository.GetByPublicID(ctx, publicID)
		if err == nil && !ownsLeonardoMediaJob(job, input) {
			return nil, ErrGenerationJobNotFound
		}
	}
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, ErrGenerationJobNotFound
	}
	return job, nil
}

func (s *LeonardoMediaGetService) Content(ctx context.Context, input LeonardoMediaContentInput) (*LeonardoMediaContent, error) {
	if input.Index < 0 {
		return nil, ErrLeonardoMediaContentInvalid
	}
	job, err := s.getOwnedJob(ctx, input.LeonardoMediaGetInput)
	if err != nil {
		return nil, err
	}
	if job.Status != GenerationJobStatusSucceeded {
		return nil, ErrLeonardoMediaContentNotReady
	}
	outputs := leonardoMediaOutputURLs(job)
	if input.Index >= len(outputs) {
		return nil, ErrLeonardoMediaContentNotFound
	}
	return s.downloadContent(ctx, outputs[input.Index], job.Modality)
}

func (s *LeonardoMediaGetService) ContentBase64(ctx context.Context, input LeonardoMediaContentInput) (string, error) {
	// Bound CDN fetch tightly; callers should fall back to URL on timeout/failure.
	downloadCtx, cancel := context.WithTimeout(ctx, leonardoCDNContentTimeout)
	defer cancel()
	content, err := s.Content(downloadCtx, input)
	if err != nil {
		return "", err
	}
	defer func() { _ = content.Close() }()
	data, err := io.ReadAll(content.File)
	if err != nil {
		return "", ErrLeonardoMediaContentFailed
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func (s *LeonardoMediaGetService) downloadContent(ctx context.Context, rawURL, modality string) (*LeonardoMediaContent, error) {
	if s == nil || s.content == nil || apicompat.IsPotentiallyUnsafeRemoteMediaURL(rawURL) || !strings.HasPrefix(rawURL, "https://") {
		return nil, ErrLeonardoMediaContentFailed
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || !strings.EqualFold(parsed.Hostname(), "cdn.leonardo.ai") {
		return nil, ErrLeonardoMediaContentFailed
	}
	// Always enforce a short download timeout even if the parent context is long-lived.
	downloadCtx, cancel := context.WithTimeout(ctx, leonardoCDNContentTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, ErrLeonardoMediaContentFailed
	}
	video := strings.EqualFold(strings.TrimSpace(modality), "video")
	maxBytes, accept := int64(leonardoImageContentMaxBytes), "image/*"
	if video {
		maxBytes, accept = leonardoVideoContentMaxBytes, "video/mp4"
	}
	req.Header.Set("Accept", accept)
	resp, err := s.content.Do(req)
	if err != nil {
		return nil, ErrLeonardoMediaContentFailed
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK || resp.ContentLength > maxBytes {
		return nil, ErrLeonardoMediaContentFailed
	}
	file, err := os.CreateTemp("", "relayq-leonardo-media-*")
	if err != nil {
		return nil, ErrLeonardoMediaContentFailed
	}
	content := &LeonardoMediaContent{File: file, path: file.Name()}
	failed := true
	defer func() {
		if failed {
			_ = content.Close()
		}
	}()
	written, err := io.Copy(file, io.LimitReader(resp.Body, maxBytes+1))
	if err != nil || written == 0 || written > maxBytes {
		return nil, ErrLeonardoMediaContentFailed
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return nil, ErrLeonardoMediaContentFailed
	}
	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, ErrLeonardoMediaContentFailed
	}
	detected := http.DetectContentType(header[:n])
	if strings.EqualFold(strings.TrimSpace(modality), "video") && isMP4Header(header[:n]) {
		detected = "video/mp4"
	}
	if !allowedLeonardoMediaContentType(detected, modality) {
		return nil, ErrLeonardoMediaContentFailed
	}
	if declared := strings.TrimSpace(resp.Header.Get("Content-Type")); declared != "" {
		mediaType, _, parseErr := mime.ParseMediaType(declared)
		if parseErr != nil || !allowedLeonardoMediaContentType(mediaType, modality) {
			return nil, ErrLeonardoMediaContentFailed
		}
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return nil, ErrLeonardoMediaContentFailed
	}
	content.ContentType = detected
	failed = false
	return content, nil
}

func isMP4Header(header []byte) bool {
	return len(header) >= 12 && string(header[4:8]) == "ftyp"
}

func allowedLeonardoMediaContentType(value, modality string) bool {
	if strings.EqualFold(strings.TrimSpace(modality), "video") {
		return value == "video/mp4"
	}
	switch value {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

func ownsLeonardoMediaJob(job *GenerationJob, input LeonardoMediaGetInput) bool {
	return job != nil && job.UserID == input.UserID && job.APIKeyID == input.APIKeyID && job.GroupID != nil && *job.GroupID == input.GroupID && job.Provider == PlatformLeonardo
}

func newLeonardoMediaGetResult(job *GenerationJob) *LeonardoMediaGetResult {
	result := &LeonardoMediaGetResult{ID: job.PublicID, Object: "media.generation", Provider: job.Provider, Model: job.Model, Modality: job.Modality, Status: openAICompatibleMediaStatus(job.Status), Data: []LeonardoMediaGetImage{}, Videos: []LeonardoMediaGetVideo{}}
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
	result.Videos = leonardoMediaVideos(job.ResultPayload)
	if len(result.Videos) > 0 {
		videoURL := strings.TrimSpace(result.Videos[0].URL)
		if videoURL != "" {
			result.VideoURL = videoURL
			result.URL = videoURL
			result.ResultURL = videoURL
			result.Content = &LeonardoMediaGetURLBundle{VideoURL: videoURL, URL: videoURL}
		}
	}
	return result
}

// openAICompatibleMediaStatus maps internal generation statuses to the
// OpenAI Videos / Infinite Canvas contract. Canvas only leaves the poll loop
// when status is "completed" (and/or a top-level video URL is present).
func openAICompatibleMediaStatus(status GenerationJobStatus) string {
	switch status {
	case GenerationJobStatusSucceeded:
		return "completed"
	case GenerationJobStatusFailed:
		return "failed"
	case GenerationJobStatusCancelled:
		return "cancelled"
	case GenerationJobStatusQueued, GenerationJobStatusCreated, GenerationJobStatusSubmitting:
		return "queued"
	case GenerationJobStatusRunning:
		return "in_progress"
	default:
		return string(status)
	}
}

func leonardoMediaOutputURLs(job *GenerationJob) []string {
	if strings.EqualFold(strings.TrimSpace(job.Modality), "video") {
		videos := leonardoMediaVideos(job.ResultPayload)
		urls := make([]string, len(videos))
		for i := range videos {
			urls[i] = videos[i].URL
		}
		return urls
	}
	images := leonardoMediaImages(job.ResultPayload)
	urls := make([]string, len(images))
	for i := range images {
		urls[i] = images[i].URL
	}
	return urls
}

func leonardoMediaVideos(payload map[string]any) []LeonardoMediaGetVideo {
	videos := []LeonardoMediaGetVideo{}
	values, ok := payload["videos"].([]any)
	if !ok {
		if typed, typedOK := payload["videos"].([]map[string]any); typedOK {
			values = make([]any, len(typed))
			for i := range typed {
				values[i] = typed[i]
			}
		} else {
			return videos
		}
	}
	for _, value := range values {
		video, ok := value.(map[string]any)
		if !ok {
			continue
		}
		id, _ := video["id"].(string)
		rawURL, _ := video["url"].(string)
		mimeType, _ := video["mime"].(string)
		nsfw, nsfwOK := video["nsfw"].(bool)
		if strings.TrimSpace(id) == "" || mimeType != "video/mp4" || !nsfwOK || nsfw || !validLeonardoHTTPSURL(rawURL) {
			continue
		}
		result := LeonardoMediaGetVideo{ID: strings.TrimSpace(id), URL: strings.TrimSpace(rawURL), MIME: mimeType}
		result.Duration, _ = numberAsFloat64(video["duration"])
		result.Width, _ = numberAsInt(video["width"])
		result.Height, _ = numberAsInt(video["height"])
		videos = append(videos, result)
	}
	return videos
}

func numberAsFloat64(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	default:
		return 0, false
	}
}
func numberAsInt(value any) (int, bool) {
	valueFloat, ok := numberAsFloat64(value)
	return int(valueFloat), ok
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
		nsfw, nsfwOK := image["nsfw"].(bool)
		if !nsfwOK || nsfw {
			continue
		}
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
