package service

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

// ErrInvalidOpenAIServiceTier reports a malformed service_tier request value.
// It is intentionally kept in the request layer so all OpenAI-compatible
// endpoints share the same validation semantics without changing provider or
// billing behavior.
type ErrInvalidOpenAIServiceTier struct {
	Value string
}

func (e *ErrInvalidOpenAIServiceTier) Error() string {
	return fmt.Sprintf("invalid service_tier %q: must be one of auto, default, fast, flex, priority, scale", e.Value)
}

// ValidateOpenAIServiceTierField validates the top-level service_tier field.
// Missing and null values are valid and leave the existing policy unchanged.
// The returned value is canonicalized only for fast (priority); callers must
// still apply the returned value if they need the normalized request body.
func ValidateOpenAIServiceTierField(body []byte) (string, error) {
	result := gjson.GetBytes(body, "service_tier")
	if !result.Exists() || result.Type == gjson.Null {
		return "", nil
	}
	if result.Type != gjson.String {
		return "", &ErrInvalidOpenAIServiceTier{Value: "<non-string>"}
	}

	raw := strings.TrimSpace(result.String())
	if raw == "" {
		return "", &ErrInvalidOpenAIServiceTier{Value: raw}
	}
	normalized := normalizedOpenAIServiceTierValue(raw)
	if normalized == "" {
		return "", &ErrInvalidOpenAIServiceTier{Value: raw}
	}
	return normalized, nil
}
