// Package openai provides helpers and types for OpenAI API integration.
package openai

import _ "embed"

// Model represents an OpenAI model
type Model struct {
	ID          string `json:"id"`
	Object      string `json:"object"`
	Created     int64  `json:"created"`
	OwnedBy     string `json:"owned_by"`
	Type        string `json:"type"`
	DisplayName string `json:"display_name"`
}

// DefaultModels OpenAI models list
var DefaultModels = []Model{
	{ID: "gpt-5.5", Object: "model", Created: 1776873600, OwnedBy: "openai", Type: "model", DisplayName: "GPT-5.5"},
	{ID: "gpt-5.4", Object: "model", Created: 1738368000, OwnedBy: "openai", Type: "model", DisplayName: "GPT-5.4"},
	{ID: "gpt-5.4-mini", Object: "model", Created: 1738368000, OwnedBy: "openai", Type: "model", DisplayName: "GPT-5.4 Mini"},
	{ID: "gpt-5.3-codex", Object: "model", Created: 1735689600, OwnedBy: "openai", Type: "model", DisplayName: "GPT-5.3 Codex"},
	{ID: "gpt-5.3-codex-spark", Object: "model", Created: 1735689600, OwnedBy: "openai", Type: "model", DisplayName: "GPT-5.3 Codex Spark"},
	{ID: "gpt-5.2", Object: "model", Created: 1733875200, OwnedBy: "openai", Type: "model", DisplayName: "GPT-5.2"},
	{ID: "mimo-v2.5-pro", Object: "model", Created: 1746144000, OwnedBy: "openai", Type: "model", DisplayName: "MiMo V2.5 Pro"},
	{ID: "mimo-v2.5", Object: "model", Created: 1746144000, OwnedBy: "openai", Type: "model", DisplayName: "MiMo V2.5"},
	{ID: "mimo-v2-pro", Object: "model", Created: 1743465600, OwnedBy: "openai", Type: "model", DisplayName: "MiMo V2 Pro"},
	{ID: "mimo-v2-omni", Object: "model", Created: 1743465600, OwnedBy: "openai", Type: "model", DisplayName: "MiMo V2 Omni"},
	{ID: "mimo-v2.5-asr", Object: "model", Created: 1746144000, OwnedBy: "openai", Type: "model", DisplayName: "MiMo V2.5 ASR"},
	{ID: "mimo-v2.5-tts", Object: "model", Created: 1746144000, OwnedBy: "openai", Type: "model", DisplayName: "MiMo V2.5 TTS"},
	{ID: "mimo-v2.5-tts-voiceclone", Object: "model", Created: 1746144000, OwnedBy: "openai", Type: "model", DisplayName: "MiMo V2.5 TTS Voice Clone"},
	{ID: "mimo-v2.5-tts-voicedesign", Object: "model", Created: 1746144000, OwnedBy: "openai", Type: "model", DisplayName: "MiMo V2.5 TTS Voice Design"},
	{ID: "mimo-v2-tts", Object: "model", Created: 1743465600, OwnedBy: "openai", Type: "model", DisplayName: "MiMo V2 TTS"},
	{ID: "gpt-image-1", Object: "model", Created: 1733875200, OwnedBy: "openai", Type: "model", DisplayName: "GPT Image 1"},
	{ID: "gpt-image-1.5", Object: "model", Created: 1735689600, OwnedBy: "openai", Type: "model", DisplayName: "GPT Image 1.5"},
	{ID: "gpt-image-2", Object: "model", Created: 1738368000, OwnedBy: "openai", Type: "model", DisplayName: "GPT Image 2"},
}

// DefaultModelIDs returns the default model ID list
func DefaultModelIDs() []string {
	ids := make([]string, len(DefaultModels))
	for i, m := range DefaultModels {
		ids[i] = m.ID
	}
	return ids
}

// DefaultTestModel default model for testing OpenAI accounts
const DefaultTestModel = "gpt-5.4"

// DefaultInstructions default instructions for non-Codex CLI requests
// Content loaded from instructions.txt at compile time
//
//go:embed instructions.txt
var DefaultInstructions string
