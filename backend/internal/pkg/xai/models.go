package xai

type Model struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
}

var DefaultModels = []Model{
	{ID: "grok-4.3", Type: "model", DisplayName: "Grok 4.3", CreatedAt: ""},
	{ID: "grok-4.20-multi-agent-0309", Type: "model", DisplayName: "Grok 4.20 Multi-Agent 0309", CreatedAt: ""},
	{ID: "grok-4.20-0309-reasoning", Type: "model", DisplayName: "Grok 4.20 0309 Reasoning", CreatedAt: ""},
	{ID: "grok-4.20-0309-non-reasoning", Type: "model", DisplayName: "Grok 4.20 0309 Non-Reasoning", CreatedAt: ""},
	{ID: "grok-4-1-fast-reasoning", Type: "model", DisplayName: "Grok 4.1 Fast Reasoning", CreatedAt: ""},
	{ID: "grok-4-1-fast-non-reasoning", Type: "model", DisplayName: "Grok 4.1 Fast Non-Reasoning", CreatedAt: ""},
	{ID: "grok-imagine-image-quality", Type: "model", DisplayName: "Grok Imagine Image Quality", CreatedAt: ""},
	{ID: "grok-imagine-image", Type: "model", DisplayName: "Grok Imagine Image", CreatedAt: ""},
	{ID: "grok-imagine-video", Type: "model", DisplayName: "Grok Imagine Video", CreatedAt: ""},
}

func DefaultModelIDs() []string {
	ids := make([]string, 0, len(DefaultModels))
	for _, model := range DefaultModels {
		ids = append(ids, model.ID)
	}
	return ids
}
