package service

import (
	"encoding/json"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// sanitizeOpenAIResponsesRequestBody removes non-standard item fields that some
// upstream Responses-compatible providers reject. It currently focuses on the
// input payload where Codex-style clients may attach private metadata such as
// namespace that OpenAI-compatible upstreams reject as unknown parameters.
func sanitizeOpenAIResponsesRequestBody(body []byte) ([]byte, bool, error) {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return body, false, nil
	}
	input := gjson.GetBytes(body, "input")
	if !input.Exists() {
		return body, false, nil
	}

	var raw any
	if err := json.Unmarshal([]byte(input.Raw), &raw); err != nil {
		return body, false, err
	}

	changed := false
	cleaned := sanitizeOpenAIResponsesInputValue(raw, &changed)
	if !changed {
		return body, false, nil
	}

	next, err := sjson.SetRawBytes(body, "input", mustMarshalResponsesSanitizedValue(cleaned))
	if err != nil {
		return body, false, err
	}
	return next, true, nil
}

func sanitizeOpenAIResponsesRequestMap(reqBody map[string]any) bool {
	if len(reqBody) == 0 {
		return false
	}
	input, ok := reqBody["input"]
	if !ok {
		return false
	}
	changed := false
	reqBody["input"] = sanitizeOpenAIResponsesInputValue(input, &changed)
	return changed
}

func sanitizeOpenAIResponsesInputValue(v any, changed *bool) any {
	switch item := v.(type) {
	case []any:
		out := make([]any, 0, len(item))
		for _, part := range item {
			out = append(out, sanitizeOpenAIResponsesInputValue(part, changed))
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(item))
		itemType, _ := item["type"].(string)
		for key, val := range item {
			if key == "namespace" {
				*changed = true
				continue
			}
			if key == "id" {
				if id, ok := val.(string); ok && shouldStripOpenAIResponsesInputItemID(itemType, id) {
					*changed = true
					continue
				}
			}
			out[key] = sanitizeOpenAIResponsesInputValue(val, changed)
		}
		return out
	default:
		return v
	}
}

// Invalid replayed IDs are removed rather than rewritten because a fabricated
// msg/fc ID may point at a different upstream object.
func shouldStripOpenAIResponsesInputItemID(itemType, id string) bool {
	if id == "" {
		return false
	}
	if itemType == "message" {
		return !strings.HasPrefix(id, "msg")
	}
	if isCodexToolCallInputType(itemType) {
		return !strings.HasPrefix(id, "fc")
	}
	return false
}

func isCodexToolCallInputType(itemType string) bool {
	switch itemType {
	case "function_call", "tool_call", "local_shell_call", "tool_search_call", "custom_tool_call", "mcp_tool_call":
		return true
	default:
		return false
	}
}

func mustMarshalResponsesSanitizedValue(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
