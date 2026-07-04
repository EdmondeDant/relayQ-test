//go:build unit

package apicompat

import "testing"

func TestIsPotentiallyUnsafeRemoteMediaURL(t *testing.T) {
	cases := []struct {
		url    string
		unsafe bool
	}{
		{"https://example.com/image.png", false},
		{"http://cdn.example.com/video.mp4", false},
		{"file:///tmp/a.png", true},
		{"https://127.0.0.1/a.png", true},
		{"https://[::1]/a.png", true},
		{"https://localhost/a.png", true},
		{"https://169.254.1.1/a.png", true},
		{"data:image/png;base64,abc", true},
	}
	for _, tc := range cases {
		if got := IsPotentiallyUnsafeRemoteMediaURL(tc.url); got != tc.unsafe {
			t.Fatalf("url=%q got unsafe=%v want %v", tc.url, got, tc.unsafe)
		}
	}
}
