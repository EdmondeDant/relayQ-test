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

type LeonardoError struct {
	StatusCode    int
	Code          string
	Message       string
	Path          string
	RequestID     string
	RetryAfter    time.Duration
	RetryableRead bool
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
