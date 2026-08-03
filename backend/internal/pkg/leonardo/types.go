package leonardo

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var ErrGenerationRequestNotWritten = errors.New("leonardo generation request not written")

const (
	GenerationErrorClassRequestNotWritten    = "request_not_written"
	GenerationErrorClassTransportAfterWrite  = "transport_after_write"
	GenerationErrorClassUpstreamNon2xx       = "upstream_non_2xx"
	GenerationErrorClassResponseReadFailed   = "response_read_failed"
	GenerationErrorClassResponseTooLarge     = "response_too_large"
	GenerationErrorClassResponseDecodeFailed = "response_decode_failed"
	GenerationErrorClassGenerationIDMissing  = "generation_id_missing"
	GenerationErrorClassGenerationIDInvalid  = "generation_id_invalid"
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
	Class            string
	StatusCode       int
	Code             string
	Message          string
	Path             string
	RequestID        string
	RetryAfter       time.Duration
	BodySHA256       string
	BodySize         int64
	BodyTruncated    bool
	RetryableRead    bool
	SubmissionStatus string
	SideEffectStatus string
	SafeToRetry      bool
	cause            error
}

func (e *LeonardoError) Error() string {
	if e == nil {
		return "leonardo request failed"
	}
	if e.SubmissionStatus == SubmissionUnknown {
		return "leonardo generation submission status is unknown"
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

func (e *LeonardoError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

type errorResponse struct {
	Error string `json:"error"`
	Path  string `json:"path"`
	Code  string `json:"code"`
}
