//go:build unit

package claude

import "testing"

func TestNormalizeModelID_DisplayNameToCanonicalID(t *testing.T) {
	got := NormalizeModelID("Claude Opus 4.8")
	if got != "claude-opus-4-8" {
		t.Fatalf("NormalizeModelID(display name) = %q, want %q", got, "claude-opus-4-8")
	}
}
