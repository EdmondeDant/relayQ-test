package service

import (
	"encoding/json"
	"testing"
)

func TestSanitizePlaygroundAssetMetadataRemovesCredentials(t *testing.T) {
	raw := json.RawMessage(`{"request_id":"req-1","auth_token":"secret-a","api_key":"secret-b","bearer_token":"secret-c","authorization":"Bearer secret-d"}`)
	cleaned := sanitizePlaygroundAssetMetadata(raw)

	var metadata map[string]any
	if err := json.Unmarshal(cleaned, &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata["request_id"] != "req-1" {
		t.Fatalf("request_id = %v", metadata["request_id"])
	}
	for _, key := range []string{"auth_token", "api_key", "bearer_token", "authorization"} {
		if _, ok := metadata[key]; ok {
			t.Fatalf("credential key %q was persisted", key)
		}
	}
}

func TestIsLocalPlaygroundProtectedURLAcceptsLoopbackHosts(t *testing.T) {
	tests := []string{
		"/v1/videos/job/content",
		"/api/v1/playground/assets/content/video/u1/test.mp4",
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
