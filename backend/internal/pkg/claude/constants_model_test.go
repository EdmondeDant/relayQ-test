package claude

import "testing"

func TestDefaultModelsContainsClaudeFable51(t *testing.T) {
	t.Parallel()

	for _, model := range DefaultModels {
		if model.ID == "claude-fable-5-1" {
			if model.DisplayName != "Claude Fable 5.1" || model.CreatedAt != "2026-09-01T00:00:00Z" {
				t.Fatalf("unexpected Claude Fable 5.1 metadata: %+v", model)
			}
			return
		}
	}
	t.Fatal("claude-fable-5-1 missing from DefaultModels")
}
