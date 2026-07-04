package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
)

const (
	DefaultVideoFallbackMaxFrames = 8
	DefaultVideoFallbackFrameRate = 0.5
	DefaultVideoFallbackMaxBytes  = 50 << 20
	DefaultVideoFallbackTimeout   = 30 * time.Second
	DefaultVideoFallbackMaxSecs   = 60
)

type VideoFallbackPlan struct {
	VideoURL       string
	MaxFrames      int
	SampleFPS      float64
	MaxDurationSec float64
	EnableASR      bool
	SafetyChecked  bool
	FallbackMethod string
}

type VideoFallbackOptions struct {
	MaxFrames  int
	SampleFPS  float64
	MaxSeconds float64
	EnableASR  bool
}

type VideoDownloadOptions struct {
	MaxBytes int64
	Timeout  time.Duration
	TempDir  string
}

type VideoFrameExtractor struct {
	FFmpegPath  string
	FFprobePath string
	WorkDir     string
}

// BuildVideoFallbackPlan prepares the deterministic plan for video understanding
// fallback: video -> sampled frames (+ optional ASR) -> multimodal chat. It does
// not fetch the remote URL yet; fetchers must repeat DNS/IP safety checks at
// download time to defend against DNS rebinding.
func BuildVideoFallbackPlan(videoURL string, opts VideoFallbackOptions) (VideoFallbackPlan, error) {
	videoURL = strings.TrimSpace(videoURL)
	if videoURL == "" {
		return VideoFallbackPlan{}, errors.New("video url is required")
	}
	if apicompat.IsPotentiallyUnsafeRemoteMediaURL(videoURL) {
		return VideoFallbackPlan{}, errors.New("unsafe video url")
	}
	maxFrames := opts.MaxFrames
	if maxFrames <= 0 {
		maxFrames = DefaultVideoFallbackMaxFrames
	}
	sampleFPS := opts.SampleFPS
	if sampleFPS <= 0 {
		sampleFPS = DefaultVideoFallbackFrameRate
	}
	maxSeconds := opts.MaxSeconds
	if maxSeconds <= 0 {
		maxSeconds = DefaultVideoFallbackMaxSecs
	}
	return VideoFallbackPlan{
		VideoURL:       videoURL,
		MaxFrames:      maxFrames,
		SampleFPS:      sampleFPS,
		MaxDurationSec: maxSeconds,
		EnableASR:      opts.EnableASR,
		SafetyChecked:  true,
		FallbackMethod: "video_frames",
	}, nil
}

// DownloadRemoteVideoForFallback downloads a remote video into a local temp
// file using an SSRF-safe HTTP client. It intentionally rejects redirects: a
// follow-up implementation can support controlled same-domain redirects, but
// the initial safe default is no redirects.
func DownloadRemoteVideoForFallback(ctx context.Context, videoURL string, opts VideoDownloadOptions) (string, error) {
	videoURL = strings.TrimSpace(videoURL)
	if videoURL == "" {
		return "", errors.New("video url is required")
	}
	if apicompat.IsPotentiallyUnsafeRemoteMediaURL(videoURL) {
		return "", errors.New("unsafe video url")
	}
	maxBytes := opts.MaxBytes
	if maxBytes <= 0 {
		maxBytes = DefaultVideoFallbackMaxBytes
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultVideoFallbackTimeout
	}
	client := newSSRFSafeHTTPClient(timeout)
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, videoURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return "", fmt.Errorf("video download redirect rejected: status %d", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("video download failed: status %d", resp.StatusCode)
	}
	contentType := strings.ToLower(strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0]))
	if contentType != "" && !isAllowedVideoFallbackContentType(contentType) {
		return "", fmt.Errorf("unsupported video content-type: %s", contentType)
	}
	if resp.ContentLength > maxBytes {
		return "", fmt.Errorf("video too large: %d > %d", resp.ContentLength, maxBytes)
	}
	tmp, err := os.CreateTemp(opts.TempDir, "relayq-video-fallback-*.mp4")
	if err != nil {
		return "", err
	}
	path := tmp.Name()
	defer func() {
		_ = tmp.Close()
	}()
	written, err := io.Copy(tmp, io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		_ = os.Remove(path)
		return "", err
	}
	if written > maxBytes {
		_ = os.Remove(path)
		return "", fmt.Errorf("video too large: exceeded %d bytes", maxBytes)
	}
	return path, nil
}

func isAllowedVideoFallbackContentType(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	return strings.HasPrefix(contentType, "video/") || contentType == "application/octet-stream"
}

func BuildVideoFallbackChatImageParts(ctx context.Context, videoURL string, opts VideoFallbackOptions, downloadOpts VideoDownloadOptions, extractor VideoFrameExtractor) ([]apicompat.ChatContentPart, error) {
	parts, err := BuildVideoFallbackChatParts(ctx, videoURL, opts, downloadOpts, extractor, "")
	if err != nil {
		return nil, err
	}
	imageParts := make([]apicompat.ChatContentPart, 0, len(parts))
	for _, part := range parts {
		if part.Type == "image_url" {
			imageParts = append(imageParts, part)
		}
	}
	return imageParts, nil
}

func BuildVideoFallbackChatParts(ctx context.Context, videoURL string, opts VideoFallbackOptions, downloadOpts VideoDownloadOptions, extractor VideoFrameExtractor, asrText string) ([]apicompat.ChatContentPart, error) {
	plan, err := BuildVideoFallbackPlan(videoURL, opts)
	if err != nil {
		return nil, err
	}
	localVideo, err := DownloadRemoteVideoForFallback(ctx, plan.VideoURL, downloadOpts)
	if err != nil {
		return nil, err
	}
	defer os.Remove(localVideo)
	frames, err := extractor.ExtractVideoFrames(ctx, localVideo, plan)
	if err != nil {
		return nil, err
	}
	return VideoFrameFilesToFallbackChatParts(frames, asrText)
}

// ExtractVideoFrames samples JPEG frames from an already-downloaded local video
// file. Remote fetching is intentionally separate so fetch-time DNS/IP checks,
// size limits, and redirect policies can be enforced before this function runs.
func (e VideoFrameExtractor) ExtractVideoFrames(ctx context.Context, localVideoPath string, plan VideoFallbackPlan) ([]string, error) {
	localVideoPath = strings.TrimSpace(localVideoPath)
	if localVideoPath == "" {
		return nil, errors.New("local video path is required")
	}
	if _, err := os.Stat(localVideoPath); err != nil {
		return nil, err
	}
	if err := e.ValidateLocalVideoDuration(ctx, localVideoPath, plan.MaxDurationSec); err != nil {
		return nil, err
	}
	ffmpegPath := strings.TrimSpace(e.FFmpegPath)
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	workDir := strings.TrimSpace(e.WorkDir)
	if workDir == "" {
		var err error
		workDir, err = os.MkdirTemp("", "relayq-video-fallback-*")
		if err != nil {
			return nil, err
		}
	} else if err := os.MkdirAll(workDir, 0o755); err != nil {
		return nil, err
	}

	maxFrames := plan.MaxFrames
	if maxFrames <= 0 {
		maxFrames = DefaultVideoFallbackMaxFrames
	}
	sampleFPS := plan.SampleFPS
	if sampleFPS <= 0 {
		sampleFPS = DefaultVideoFallbackFrameRate
	}
	outPattern := filepath.Join(workDir, "frame_%03d.jpg")
	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-i", localVideoPath,
		"-vf", fmt.Sprintf("fps=%g,scale='min(768,iw)':-2", sampleFPS),
		"-frames:v", fmt.Sprintf("%d", maxFrames),
		"-q:v", "3",
		outPattern,
	}
	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg extract frames failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	paths, err := filepath.Glob(filepath.Join(workDir, "frame_*.jpg"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, errors.New("ffmpeg produced no frames")
	}
	return paths, nil
}

func (e VideoFrameExtractor) ValidateLocalVideoDuration(ctx context.Context, localVideoPath string, maxSeconds float64) error {
	if maxSeconds <= 0 {
		return nil
	}
	duration, err := e.ProbeVideoDuration(ctx, localVideoPath)
	if err != nil {
		return err
	}
	if duration > maxSeconds {
		return fmt.Errorf("video duration %.2fs exceeds %.2fs", duration, maxSeconds)
	}
	return nil
}

func (e VideoFrameExtractor) ProbeVideoDuration(ctx context.Context, localVideoPath string) (float64, error) {
	ffprobePath := strings.TrimSpace(e.FFprobePath)
	if ffprobePath == "" {
		ffprobePath = "ffprobe"
	}
	cmd := exec.CommandContext(ctx, ffprobePath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		localVideoPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("ffprobe duration failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return parseFFprobeDurationOutput(string(output))
}

func parseFFprobeDurationOutput(output string) (float64, error) {
	value := strings.TrimSpace(output)
	if value == "" || strings.EqualFold(value, "N/A") {
		return 0, errors.New("ffprobe returned empty duration")
	}
	duration, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}
	if duration <= 0 {
		return 0, errors.New("ffprobe returned non-positive duration")
	}
	return duration, nil
}

func VideoFrameFilesToChatImageParts(paths []string) ([]apicompat.ChatContentPart, error) {
	parts := make([]apicompat.ChatContentPart, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if len(data) == 0 {
			continue
		}
		parts = append(parts, apicompat.ChatContentPart{
			Type: "image_url",
			ImageURL: &apicompat.ChatImageURL{
				URL: "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(data),
			},
		})
	}
	return parts, nil
}

func VideoFrameFilesToFallbackChatParts(paths []string, asrText string) ([]apicompat.ChatContentPart, error) {
	imageParts, err := VideoFrameFilesToChatImageParts(paths)
	if err != nil {
		return nil, err
	}
	parts := make([]apicompat.ChatContentPart, 0, len(imageParts)+2)
	parts = append(parts, apicompat.ChatContentPart{
		Type: "text",
		Text: "下面是同一个视频按时间顺序抽取的关键帧。请结合帧顺序理解视频内容；如有音频转写，也应一并参考。",
	})
	if strings.TrimSpace(asrText) != "" {
		parts = append(parts, apicompat.ChatContentPart{
			Type: "text",
			Text: "视频音频转写：" + strings.TrimSpace(asrText),
		})
	}
	parts = append(parts, imageParts...)
	return parts, nil
}
