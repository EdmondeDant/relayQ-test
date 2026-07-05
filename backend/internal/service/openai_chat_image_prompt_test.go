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

func TestExtractOpenAIChatImageOptions(t *testing.T) {
	body := []byte(`{"aspectRatio":"9:16","providerOptions":{"xai":{"resolution":"2K","quality":"high"}},"n":12,"user":"u1"}`)

	options := ExtractOpenAIChatImageOptions(body)

	require.Equal(t, "9:16", options["aspect_ratio"])
	require.Equal(t, "2k", options["resolution"])
	require.Equal(t, "high", options["quality"])
	require.Equal(t, 10, options["n"])
	require.Equal(t, "u1", options["user"])
}
