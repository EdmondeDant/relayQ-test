package service

import "testing"

func TestPlaygroundInternalBaseURLPrefersProvidedBase(t *testing.T) {
	t.Setenv("SERVER_HOST", "0.0.0.0")
	t.Setenv("SERVER_PORT", "8080")

	got := playgroundInternalBaseURL("http://0.0.0.0:3000")
	want := "http://127.0.0.1:3000"
	if got != want {
		t.Fatalf("playgroundInternalBaseURL() = %q, want %q", got, want)
	}
}

func TestResolvePlaygroundAssetURLUsesProvidedBase(t *testing.T) {
	got := resolvePlaygroundAssetURL("/api/v1/playground/assets/content/image/u1/test.png", "http://127.0.0.1:3000")
	want := "http://127.0.0.1:3000/api/v1/playground/assets/content/image/u1/test.png"
	if got != want {
		t.Fatalf("resolvePlaygroundAssetURL() = %q, want %q", got, want)
	}
}
