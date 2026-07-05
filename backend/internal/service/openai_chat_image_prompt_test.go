package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractOpenAIChatImagePromptStringContent(t *testing.T) {
	body := []byte(`{"model":"grok-imagine-image","messages":[{"role":"system","content":"x"},{"role":"user","content":"draw a lobster"}]}`)

	require.Equal(t, "draw a lobster", ExtractOpenAIChatImagePrompt(body))
}

func TestExtractOpenAIChatImagePromptArrayContent(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":[{"type":"text","text":"draw"},{"type":"input_text","text":"a cyber city"},{"type":"image_url","image_url":{"url":"https://example.com/a.png"}}]}]}`)

	require.Equal(t, "draw\na cyber city", ExtractOpenAIChatImagePrompt(body))
}

func TestExtractOpenAIChatImagePromptUsesLastUserMessage(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":"old"},{"role":"assistant","content":"ok"},{"role":"user","content":"new"}]}`)

	require.Equal(t, "new", ExtractOpenAIChatImagePrompt(body))
}
