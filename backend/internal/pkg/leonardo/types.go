package leonardo

import (
	"encoding/json"
	"fmt"
	"time"
)

type Model struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Parameters json.RawMessage `json:"parameters"`
}

type modelsResponse struct {
	Models []Model `json:"productionApiAvailableModels"`
}

type CreateGenerationRequest struct {
	Model      string         `json:"model"`
	Public     bool           `json:"public"`
	Parameters map[string]any `json:"parameters"`
}

type GenerationCost struct {
	Amount float64 `json:"amount"`
	Unit   string  `json:"unit"`
}

type CreateGenerationResponse struct {
	GenerationID  string          `json:"generationId"`
	Cost          *GenerationCost `json:"cost,omitempty"`
	APICreditCost *float64        `json:"apiCreditCost,omitempty"`
}

type GeneratedImage struct {
	ID   string `json:"id"`
	URL  string `json:"url"`
	NSFW bool   `json:"nsfw"`
}

type Generation struct {
	ID              string           `json:"id"`
	Status          string           `json:"status"`
	GeneratedImages []GeneratedImage `json:"generated_images"`
}

type generationResponse struct {
	Generation Generation `json:"generations_by_pk"`
}

type LeonardoError struct {
	StatusCode       int
	Code             string
	Message          string
	Path             string
	RequestID        string
	RetryAfter       time.Duration
	RetryableRead    bool
	SubmissionStatus string
	SideEffectStatus string
	SafeToRetry      bool
}

func (e *LeonardoError) Error() string {
	if e == nil {
		return "leonardo request failed"
	}
	message := e.Message
	if message == "" {
		message = "request failed"
	}
	if e.Code != "" {
		return fmt.Sprintf("leonardo: HTTP %d: %s (code=%s)", e.StatusCode, message, e.Code)
	}
	return fmt.Sprintf("leonardo: HTTP %d: %s", e.StatusCode, message)
}

type errorResponse struct {
	Error string `json:"error"`
	Path  string `json:"path"`
	Code  string `json:"code"`
}
