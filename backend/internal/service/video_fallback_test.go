//go:build unit

package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildVideoFallbackPlan(t *testing.T) {
	plan, err := BuildVideoFallbackPlan("https://example.com/video.mp4", VideoFallbackOptions{EnableASR: true})
	require.NoError(t, err)
	require.Equal(t, "https://example.com/video.mp4", plan.VideoURL)
	require.Equal(t, DefaultVideoFallbackMaxFrames, plan.MaxFrames)
	require.Equal(t, DefaultVideoFallbackFrameRate, plan.SampleFPS)
	require.True(t, plan.EnableASR)
	require.True(t, plan.SafetyChecked)
	require.Equal(t, "video_frames", plan.FallbackMethod)
}

func TestBuildVideoFallbackPlanRejectsUnsafeURL(t *testing.T) {
	_, err := BuildVideoFallbackPlan("http://127.0.0.1/video.mp4", VideoFallbackOptions{})
	require.Error(t, err)
}

func TestDownloadRemoteVideoForFallbackRejectsUnsafeURL(t *testing.T) {
	_, err := DownloadRemoteVideoForFallback(context.Background(), "http://127.0.0.1/video.mp4", VideoDownloadOptions{})
	require.Error(t, err)
}

func TestIsAllowedVideoFallbackContentType(t *testing.T) {
	require.True(t, isAllowedVideoFallbackContentType("video/mp4"))
	require.True(t, isAllowedVideoFallbackContentType("video/webm"))
	require.True(t, isAllowedVideoFallbackContentType("application/octet-stream"))
	require.False(t, isAllowedVideoFallbackContentType("image/png"))
	require.False(t, isAllowedVideoFallbackContentType("text/html"))
}

func TestVideoFrameFilesToChatImageParts(t *testing.T) {
	dir := t.TempDir()
	frame := filepath.Join(dir, "frame_001.jpg")
	require.NoError(t, os.WriteFile(frame, []byte{0xff, 0xd8, 0xff, 0xd9}, 0o644))

	parts, err := VideoFrameFilesToChatImageParts([]string{frame})
	require.NoError(t, err)
	require.Len(t, parts, 1)
	require.Equal(t, "image_url", parts[0].Type)
	require.NotNil(t, parts[0].ImageURL)
	require.Equal(t, "data:image/jpeg;base64,/9j/2Q==", parts[0].ImageURL.URL)
}

func TestParseFFprobeDurationOutput(t *testing.T) {
	duration, err := parseFFprobeDurationOutput("12.345\n")
	require.NoError(t, err)
	require.Equal(t, 12.345, duration)

	_, err = parseFFprobeDurationOutput("N/A")
	require.Error(t, err)
}

func TestVideoFrameFilesToFallbackChatParts(t *testing.T) {
	dir := t.TempDir()
	frame := filepath.Join(dir, "frame_001.jpg")
	require.NoError(t, os.WriteFile(frame, []byte{0xff, 0xd8, 0xff, 0xd9}, 0o644))

	parts, err := VideoFrameFilesToFallbackChatParts([]string{frame}, "hello audio")
	require.NoError(t, err)
	require.Len(t, parts, 3)
	require.Equal(t, "text", parts[0].Type)
	require.Contains(t, parts[0].Text, "关键帧")
	require.Equal(t, "text", parts[1].Type)
	require.Contains(t, parts[1].Text, "hello audio")
	require.Equal(t, "image_url", parts[2].Type)
}
