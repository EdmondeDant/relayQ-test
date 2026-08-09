package service

import "testing"

func TestIsLocalPlaygroundProtectedURLAcceptsLoopbackHosts(t *testing.T) {
	tests := []string{
		"http://localhost:8080/v1/videos/job/content",
		"http://127.0.0.1:8080/v1/videos/job/content",
		"http://[::1]:8080/v1/videos/job/content",
	}
	for _, rawURL := range tests {
		if !isLocalPlaygroundProtectedURL(rawURL) {
			t.Fatalf("isLocalPlaygroundProtectedURL(%q) = false", rawURL)
		}
	}
}

func TestIsLocalPlaygroundProtectedURLRejectsRemoteAndPublicPaths(t *testing.T) {
	tests := []string{
		"https://example.com/v1/videos/job/content",
		"http://[::1]:8080/public/video.mp4",
	}
	for _, rawURL := range tests {
		if isLocalPlaygroundProtectedURL(rawURL) {
			t.Fatalf("isLocalPlaygroundProtectedURL(%q) = true", rawURL)
		}
	}
}
