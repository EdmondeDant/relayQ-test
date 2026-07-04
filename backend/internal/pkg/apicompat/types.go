// Package apicompat provides type definitions and conversion utilities for
// translating between Anthropic Messages and OpenAI Responses API formats.
// It enables multi-protocol support so that clients using different API
// formats can be served through a unified gateway.
package apicompat

import "encoding/json"

// ---------------------------------------------------------------------------
// Anthropic Messages API types
// ---------------------------------------------------------------------------

// AnthropicRequest is the request body for POST /v1/messages.
type AnthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	System      json.RawMessage    `json:"system,omitempty"` // string or []AnthropicContentBlock
	Messages    []AnthropicMessage `json:"messages"`
	Tools       []AnthropicTool    `json:"tools,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
	Temperature *float64           `json:"temperature,omitempty"`
	TopP        *float64           `json:"top_p,omitempty"`
	StopSeqs    []string           `json:"stop_sequences,omitempty"`
	Thinking    *AnthropicThinking `json:"thinking,omitempty"`
	ToolChoice  json.RawMessage    `json:"tool_choice,omitempty"`
	// Metadata 会被原样透传给上游。OAuth/Claude-Code 路径依赖 metadata.user_id
	// 参与上游的"是否为官方 Claude Code 请求"判定；如果经由本结构体重新序列化
	// 时丢弃该字段，网关侧后续的 metadata 重写(ensureClaudeOAuthMetadataUserID/
	// RewriteUserIDWithMasking) 在 body 里拿不到起点，就无法重建一个合法的
	// user_id，进而导致请求被归类为第三方 app。
	Metadata     json.RawMessage        `json:"metadata,omitempty"`
	OutputConfig *AnthropicOutputConfig `json:"output_config,omitempty"`
}

// AnthropicOutputConfig controls output generation parameters.
type AnthropicOutputConfig struct {
	Effort string `json:"effort,omitempty"` // "low" | "medium" | "high" | "max"
}

// AnthropicThinking configures extended thinking in the Anthropic API.
type AnthropicThinking struct {
	Type         string `json:"type"`                    // "enabled" | "adaptive" | "disabled"
	BudgetTokens int    `json:"budget_tokens,omitempty"` // max thinking tokens
}

// AnthropicMessage is a single message in the Anthropic conversation.
type AnthropicMessage struct {
	Role    string          `json:"role"` // "user" | "assistant"
	Content json.RawMessage `json:"content"`
}

// AnthropicContentBlock is one block inside a message's content array.
type AnthropicContentBlock struct {
	Type string `json:"type"`

	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`

	// type=text
	Text string `json:"text,omitempty"`

	// type=thinking
	Thinking string `json:"thinking,omitempty"`

	// type=image
	Source *AnthropicImageSource `json:"source,omitempty"`

	// type=tool_use
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	// type=tool_result
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"` // string or []AnthropicContentBlock
	IsError   bool            `json:"is_error,omitempty"`
}

func (b AnthropicContentBlock) MarshalJSON() ([]byte, error) {
	type anthropicContentBlock AnthropicContentBlock
	base := struct {
		anthropicContentBlock
	}{anthropicContentBlock: anthropicContentBlock(b)}

	switch b.Type {
	case "text":
		return json.Marshal(struct {
			Text string `json:"text"`
			anthropicContentBlock
		}{Text: b.Text, anthropicContentBlock: anthropicContentBlock(b)})
	case "thinking":
		return json.Marshal(struct {
			Thinking string `json:"thinking"`
			anthropicContentBlock
		}{Thinking: b.Thinking, anthropicContentBlock: anthropicContentBlock(b)})
	default:
		return json.Marshal(base)
	}
}

// AnthropicImageSource describes the source data for an image content block.
type AnthropicImageSource struct {
	Type      string `json:"type"` // "base64"
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// AnthropicTool describes a tool available to the model.
type AnthropicTool struct {
	Type         string                 `json:"type,omitempty"` // e.g. "web_search_20250305" for server tools
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	InputSchema  json.RawMessage        `json:"input_schema"` // JSON Schema object
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

// AnthropicCacheControl 对应 Anthropic API 的 cache_control 字段。
// ttl 默认由调用方决定；本项目策略见 claude.DefaultCacheControlTTL。
type AnthropicCacheControl struct {
	Type string `json:"type"`          // "ephemeral"
	TTL  string `json:"ttl,omitempty"` // "5m" / "1h" / 省略=默认 5m（由 Anthropic 判定）
}

// AnthropicResponse is the non-streaming response from POST /v1/messages.
type AnthropicResponse struct {
	ID           string                  `json:"id"`
	Type         string                  `json:"type"` // "message"
	Role         string                  `json:"role"` // "assistant"
	Content      []AnthropicContentBlock `json:"content"`
	Model        string                  `json:"model"`
	StopReason   string                  `json:"stop_reason"`
	StopSequence *string                 `json:"stop_sequence,omitempty"`
	Usage        AnthropicUsage          `json:"usage"`
}

// AnthropicUsage holds token counts in Anthropic format.
type AnthropicUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// ---------------------------------------------------------------------------
// Anthropic SSE event types
// ---------------------------------------------------------------------------

// AnthropicStreamEvent is a single SSE event in the Anthropic streaming protocol.
type AnthropicStreamEvent struct {
	Type string `json:"type"`

	// message_start
	Message *AnthropicResponse `json:"message,omitempty"`

	// content_block_start
	Index        *int                   `json:"index,omitempty"`
	ContentBlock *AnthropicContentBlock `json:"content_block,omitempty"`

	// content_block_delta
	Delta *AnthropicDelta `json:"delta,omitempty"`

	// message_delta
	Usage *AnthropicUsage `json:"usage,omitempty"`
}

// AnthropicDelta carries incremental content in streaming events.
type AnthropicDelta struct {
	Type string `json:"type,omitempty"` // "text_delta" | "input_json_delta" | "thinking_delta" | "signature_delta"

	// text_delta
	Text string `json:"text,omitempty"`

	// input_json_delta
	PartialJSON string `json:"partial_json,omitempty"`

	// thinking_delta
	Thinking string `json:"thinking,omitempty"`

	// signature_delta
	Signature string `json:"signature,omitempty"`

	// message_delta fields
	StopReason   string  `json:"stop_reason,omitempty"`
	StopSequence *string `json:"stop_sequence,omitempty"`
}

// ---------------------------------------------------------------------------
// OpenAI Responses API types
// ---------------------------------------------------------------------------

// ResponsesRequest is the request body for POST /v1/responses.
type ResponsesRequest struct {
	Model              string              `json:"model"`
	Instructions       string              `json:"instructions,omitempty"`
	Input              json.RawMessage     `json:"input"` // string or []ResponsesInputItem
	MaxOutputTokens    *int                `json:"max_output_tokens,omitempty"`
	Temperature        *float64            `json:"temperature,omitempty"`
	TopP               *float64            `json:"top_p,omitempty"`
	Stream             bool                `json:"stream,omitempty"`
	Tools              []ResponsesTool     `json:"tools,omitempty"`
	Include            []string            `json:"include,omitempty"`
	Store              *bool               `json:"store,omitempty"`
	ParallelToolCalls  *bool               `json:"parallel_tool_calls,omitempty"`
	Reasoning          *ResponsesReasoning `json:"reasoning,omitempty"`
	Text               *ResponsesText      `json:"text,omitempty"`
	ToolChoice         json.RawMessage     `json:"tool_choice,omitempty"`
	ServiceTier        string              `json:"service_tier,omitempty"`
	PromptCacheKey     string              `json:"prompt_cache_key,omitempty"`
	PreviousResponseID string              `json:"previous_response_id,omitempty"`
}

// ResponsesReasoning configures reasoning effort in the Responses API.
type ResponsesReasoning struct {
	Effort  string `json:"effort"`            // "low" | "medium" | "high" | "xhigh"
	Summary string `json:"summary,omitempty"` // "auto" | "concise" | "detailed"
}

// ResponsesText configures text output options in the Responses API.
type ResponsesText struct {
	Verbosity string `json:"verbosity,omitempty"` // "low" | "medium" | "high"
}

// ResponsesInputItem is one item in the Responses API input array.
// The Type field determines which other fields are populated.
type ResponsesInputItem struct {
	// Common
	Type string `json:"type,omitempty"` // "" for role-based messages

	// Role-based messages (developer/system/user/assistant)
	Role    string          `json:"role,omitempty"`
	Content json.RawMessage `json:"content,omitempty"` // string or []ResponsesContentPart

	// type=function_call
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	ID        string `json:"id,omitempty"`

	// type=function_call_output
	Output string `json:"output,omitempty"`
}

// ResponsesContentPart is a typed content part in a Responses message.
type ResponsesContentPart struct {
	Type     string `json:"type"` // "input_text" | "output_text" | "input_image"
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"` // data URI for input_image
}

// ResponsesTool describes a tool in the Responses API.
type ResponsesTool struct {
	Type        string          `json:"type"` // "function" | "web_search" | "local_shell" etc.
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Strict      *bool           `json:"strict,omitempty"`
}

// ResponsesResponse is the non-streaming response from POST /v1/responses.
type ResponsesResponse struct {
	ID     string            `json:"id"`
	Object string            `json:"object"` // "response"
	Model  string            `json:"model"`
	Status string            `json:"status"` // "completed" | "incomplete" | "failed"
	Output []ResponsesOutput `json:"output"`
	Usage  *ResponsesUsage   `json:"usage,omitempty"`

	// incomplete_details is present when status="incomplete"
	IncompleteDetails *ResponsesIncompleteDetails `json:"incomplete_details,omitempty"`

	// Error is present when status="failed"
	Error *ResponsesError `json:"error,omitempty"`
}

// ResponsesError describes an error in a failed response.
type ResponsesError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ResponsesIncompleteDetails explains why a response is incomplete.
type ResponsesIncompleteDetails struct {
	Reason string `json:"reason"` // "max_output_tokens" | "content_filter"
}

// ResponsesOutput is one output item in a Responses API response.
type ResponsesOutput struct {
	Type string `json:"type"` // "message" | "reasoning" | "function_call" | "web_search_call"

	// type=message
	ID      string                 `json:"id,omitempty"`
	Role    string                 `json:"role,omitempty"`
	Content []ResponsesContentPart `json:"content,omitempty"`
	Status  string                 `json:"status,omitempty"`

	// type=reasoning
	EncryptedContent string             `json:"encrypted_content,omitempty"`
	Summary          []ResponsesSummary `json:"summary,omitempty"`

	// type=function_call
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`

	// type=web_search_call
	Action *WebSearchAction `json:"action,omitempty"`
}

// WebSearchAction describes the search action in a web_search_call output item.
type WebSearchAction struct {
	Type  string `json:"type,omitempty"`  // "search"
	Query string `json:"query,omitempty"` // primary search query
}

// ResponsesSummary is a summary text block inside a reasoning output.
type ResponsesSummary struct {
	Type string `json:"type"` // "summary_text"
	Text string `json:"text"`
}

// ResponsesUsage holds token counts in Responses API format.
type ResponsesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`

	// Optional detailed breakdown
	InputTokensDetails  *ResponsesInputTokensDetails  `json:"input_tokens_details,omitempty"`
	OutputTokensDetails *ResponsesOutputTokensDetails `json:"output_tokens_details,omitempty"`
}

func (u *ResponsesUsage) UnmarshalJSON(data []byte) error {
	type responsesUsageAlias ResponsesUsage
	var aux struct {
		responsesUsageAlias
		PromptTokens            int                           `json:"prompt_tokens"`
		CompletionTokens        int                           `json:"completion_tokens"`
		PromptTokensDetails     *ResponsesInputTokensDetails  `json:"prompt_tokens_details,omitempty"`
		CompletionTokensDetails *ResponsesOutputTokensDetails `json:"completion_tokens_details,omitempty"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*u = ResponsesUsage(aux.responsesUsageAlias)
	if u.InputTokens == 0 && aux.PromptTokens != 0 {
		u.InputTokens = aux.PromptTokens
	}
	if u.OutputTokens == 0 && aux.CompletionTokens != 0 {
		u.OutputTokens = aux.CompletionTokens
	}
	if u.InputTokensDetails == nil && aux.PromptTokensDetails != nil {
		u.InputTokensDetails = aux.PromptTokensDetails
	}
	if u.OutputTokensDetails == nil && aux.CompletionTokensDetails != nil {
		u.OutputTokensDetails = aux.CompletionTokensDetails
	}
	if u.TotalTokens == 0 && (u.InputTokens != 0 || u.OutputTokens != 0) {
		u.TotalTokens = u.InputTokens + u.OutputTokens
	}
	return nil
}

// ResponsesInputTokensDetails breaks down input token usage.
type ResponsesInputTokensDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"`
	AudioTokens  int `json:"audio_tokens,omitempty"`
}

// ResponsesOutputTokensDetails breaks down output token usage.
type ResponsesOutputTokensDetails struct {
	ReasoningTokens          int `json:"reasoning_tokens,omitempty"`
	AudioTokens              int `json:"audio_tokens,omitempty"`
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens,omitempty"`
	RejectedPredictionTokens int `json:"rejected_prediction_tokens,omitempty"`
}

// ---------------------------------------------------------------------------
// Responses SSE event types
// ---------------------------------------------------------------------------

// ResponsesStreamEvent is a single SSE event in the Responses streaming protocol.
// The Type field corresponds to the "type" in the JSON payload.
type ResponsesStreamEvent struct {
	Type string `json:"type"`

	// response.created / response.completed / response.done / response.failed / response.incomplete
	Response *ResponsesResponse `json:"response,omitempty"`
	// 部分 OpenAI 兼容上游会把 usage 放在终止事件顶层，而不是 response.usage。
	Usage *ResponsesUsage `json:"usage,omitempty"`

	// response.output_item.added / response.output_item.done
	Item *ResponsesOutput `json:"item,omitempty"`

	// response.output_text.delta / response.output_text.done
	OutputIndex  int    `json:"output_index,omitempty"`
	ContentIndex int    `json:"content_index,omitempty"`
	Delta        string `json:"delta,omitempty"`
	Text         string `json:"text,omitempty"`
	ItemID       string `json:"item_id,omitempty"`

	// response.function_call_arguments.delta / done
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`

	// response.reasoning_summary_text.delta / done
	// Reuses Text/Delta fields above, SummaryIndex identifies which summary part
	SummaryIndex int `json:"summary_index,omitempty"`

	// response.content_part.added / done and
	// response.reasoning_summary_part.added / done
	Part *ResponsesContentPart `json:"part,omitempty"`

	// error event fields
	Code  string `json:"code,omitempty"`
	Param string `json:"param,omitempty"`

	// Sequence number for ordering events
	SequenceNumber int `json:"sequence_number,omitempty"`
}

// ---------------------------------------------------------------------------
// OpenAI Chat Completions API types
// ---------------------------------------------------------------------------

// ChatCompletionsRequest is the request body for POST /v1/chat/completions.
type ChatCompletionsRequest struct {
	Model               string             `json:"model"`
	Messages            []ChatMessage      `json:"messages"`
	Instructions        string             `json:"instructions,omitempty"` // OpenAI Responses API compat
	MaxTokens           *int               `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int               `json:"max_completion_tokens,omitempty"`
	Temperature         *float64           `json:"temperature,omitempty"`
	TopP                *float64           `json:"top_p,omitempty"`
	Stream              bool               `json:"stream,omitempty"`
	StreamOptions       *ChatStreamOptions `json:"stream_options,omitempty"`
	Tools               []ChatTool         `json:"tools,omitempty"`
	ToolChoice          json.RawMessage    `json:"tool_choice,omitempty"`
	ReasoningEffort     string             `json:"reasoning_effort,omitempty"` // "low" | "medium" | "high" | "xhigh"
	ServiceTier         string             `json:"service_tier,omitempty"`
	Stop                json.RawMessage    `json:"stop,omitempty"` // string or []string

	// Legacy function calling (deprecated but still supported)
	Functions    []ChatFunction  `json:"functions,omitempty"`
	FunctionCall json.RawMessage `json:"function_call,omitempty"`
}

// ChatStreamOptions configures streaming behavior.
type ChatStreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// ChatMessage is a single message in the Chat Completions conversation.
type ChatMessage struct {
	Role             string          `json:"role"` // "system" | "user" | "assistant" | "tool" | "function"
	Content          json.RawMessage `json:"content,omitempty"`
	ReasoningContent string          `json:"reasoning_content,omitempty"`
	Name             string          `json:"name,omitempty"`
	ToolCalls        []ChatToolCall  `json:"tool_calls,omitempty"`
	ToolCallID       string          `json:"tool_call_id,omitempty"`

	// Legacy function calling
	FunctionCall *ChatFunctionCall `json:"function_call,omitempty"`
}

// ChatContentPart is a typed content part in a multi-modal message.
type ChatContentPart struct {
	Type       string          `json:"type"` // "text" | "image_url" | "video_url" | "audio_url" | "input_audio"
	Text       string          `json:"text,omitempty"`
	ImageURL   *ChatImageURL   `json:"image_url,omitempty"`
	VideoURL   *ChatMediaURL   `json:"video_url,omitempty"`
	AudioURL   *ChatMediaURL   `json:"audio_url,omitempty"`
	InputAudio *ChatInputAudio `json:"input_audio,omitempty"`
}

// UnmarshalJSON accepts the strict OpenAI Chat Completions image part shape
// (`{"type":"image_url","image_url":{"url":"..."}}`) plus the common
// OpenAI-compatible variants seen in the wild:
//   - `image_url` as a plain string
//   - Responses-style `input_image` parts inside a chat-completions message
//   - Responses-style `input_text` parts inside a chat-completions message
//
// Without this tolerant parser, clients that send string media URLs or
// Responses-style parts to `/v1/chat/completions` fail to parse as typed
// content, which makes the modality disappear (or turns the whole request
// into a parsing error) before the upstream multimodal model ever sees it.
func (p *ChatContentPart) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	var typ string
	if rawType, ok := raw["type"]; ok {
		_ = json.Unmarshal(rawType, &typ)
	}
	var text string
	if rawText, ok := raw["text"]; ok {
		_ = json.Unmarshal(rawText, &text)
	}

	switch typ {
	case "input_text":
		typ = "text"
	case "input_image":
		typ = "image_url"
	}

	*p = ChatContentPart{Type: typ, Text: text}

	if rawImageURL, ok := raw["image_url"]; ok {
		image, err := unmarshalChatImageURL(rawImageURL)
		if err != nil {
			return err
		}
		p.ImageURL = image
	}
	if rawVideoURL, ok := raw["video_url"]; ok {
		video, err := unmarshalChatMediaURL(rawVideoURL)
		if err != nil {
			return err
		}
		p.VideoURL = video
	}
	if rawAudioURL, ok := raw["audio_url"]; ok {
		audio, err := unmarshalChatMediaURL(rawAudioURL)
		if err != nil {
			return err
		}
		p.AudioURL = audio
	}
	if rawInputAudio, ok := raw["input_audio"]; ok {
		var inputAudio ChatInputAudio
		if err := json.Unmarshal(rawInputAudio, &inputAudio); err != nil {
			return err
		}
		if inputAudio.Data != "" || inputAudio.Format != "" {
			p.InputAudio = &inputAudio
		}
	}

	return nil
}

func unmarshalChatImageURL(raw json.RawMessage) (*ChatImageURL, error) {
	var imageURL string
	if err := json.Unmarshal(raw, &imageURL); err == nil {
		if imageURL == "" {
			return nil, nil
		}
		return &ChatImageURL{URL: imageURL}, nil
	}

	var image ChatImageURL
	if err := json.Unmarshal(raw, &image); err != nil {
		return nil, err
	}
	if image.URL == "" && image.Detail == "" {
		return nil, nil
	}
	return &image, nil
}

func unmarshalChatMediaURL(raw json.RawMessage) (*ChatMediaURL, error) {
	var mediaURL string
	if err := json.Unmarshal(raw, &mediaURL); err == nil {
		if mediaURL == "" {
			return nil, nil
		}
		return &ChatMediaURL{URL: mediaURL}, nil
	}

	var media ChatMediaURL
	if err := json.Unmarshal(raw, &media); err != nil {
		return nil, err
	}
	if media.URL == "" {
		return nil, nil
	}
	return &media, nil
}

// ChatImageURL contains the URL for an image content part.
type ChatImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "auto" | "low" | "high"
}

// ChatMediaURL contains a URL for non-image media parts such as video_url or
// audio_url.
type ChatMediaURL struct {
	URL string `json:"url"`
}

// ChatInputAudio contains inline audio bytes in OpenAI-compatible chat format.
type ChatInputAudio struct {
	Data   string `json:"data,omitempty"`
	Format string `json:"format,omitempty"`
}

// ChatTool describes a tool available to the model.
type ChatTool struct {
	Type     string        `json:"type"` // "function"
	Function *ChatFunction `json:"function,omitempty"`
}

// ChatFunction describes a function tool definition.
type ChatFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Strict      *bool           `json:"strict,omitempty"`
}

// ChatToolCall represents a tool call made by the assistant.
// Index is only populated in streaming chunks (omitted in non-streaming responses).
type ChatToolCall struct {
	Index    *int             `json:"index,omitempty"`
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type,omitempty"` // "function"
	Function ChatFunctionCall `json:"function"`
}

// ChatFunctionCall contains the function name and arguments.
type ChatFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ChatCompletionsResponse is the non-streaming response from POST /v1/chat/completions.
type ChatCompletionsResponse struct {
	ID                string       `json:"id"`
	Object            string       `json:"object"` // "chat.completion"
	Created           int64        `json:"created"`
	Model             string       `json:"model"`
	Choices           []ChatChoice `json:"choices"`
	Usage             *ChatUsage   `json:"usage,omitempty"`
	SystemFingerprint string       `json:"system_fingerprint,omitempty"`
	ServiceTier       string       `json:"service_tier,omitempty"`
}

// ChatChoice is a single completion choice.
type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"` // "stop" | "length" | "tool_calls" | "content_filter"
}

// ChatUsage holds token counts in Chat Completions format.
type ChatUsage struct {
	PromptTokens            int               `json:"prompt_tokens"`
	CompletionTokens        int               `json:"completion_tokens"`
	TotalTokens             int               `json:"total_tokens"`
	PromptTokensDetails     *ChatTokenDetails `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *ChatTokenDetails `json:"completion_tokens_details,omitempty"`
}

// ChatTokenDetails provides a breakdown of token usage. The same type is
// reused for both prompt_tokens_details and completion_tokens_details;
// unset fields are omitted so each side only emits the fields that apply.
//
// Field set mirrors OpenAI's official CompletionUsage schema:
//   - prompt_tokens_details: cached_tokens, audio_tokens
//   - completion_tokens_details: reasoning_tokens, audio_tokens,
//     accepted_prediction_tokens, rejected_prediction_tokens
type ChatTokenDetails struct {
	CachedTokens             int `json:"cached_tokens,omitempty"`
	AudioTokens              int `json:"audio_tokens,omitempty"`
	ReasoningTokens          int `json:"reasoning_tokens,omitempty"`
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens,omitempty"`
	RejectedPredictionTokens int `json:"rejected_prediction_tokens,omitempty"`
}

// ChatCompletionsChunk is a single streaming chunk from POST /v1/chat/completions.
type ChatCompletionsChunk struct {
	ID                string            `json:"id"`
	Object            string            `json:"object"` // "chat.completion.chunk"
	Created           int64             `json:"created"`
	Model             string            `json:"model"`
	Choices           []ChatChunkChoice `json:"choices"`
	Usage             *ChatUsage        `json:"usage,omitempty"`
	SystemFingerprint string            `json:"system_fingerprint,omitempty"`
	ServiceTier       string            `json:"service_tier,omitempty"`
}

// ChatChunkChoice is a single choice in a streaming chunk.
type ChatChunkChoice struct {
	Index        int       `json:"index"`
	Delta        ChatDelta `json:"delta"`
	FinishReason *string   `json:"finish_reason"` // pointer: null when not final
}

// ChatDelta carries incremental content in a streaming chunk.
type ChatDelta struct {
	Role             string         `json:"role,omitempty"`
	Content          *string        `json:"content,omitempty"` // pointer: omit when not present, null vs "" matters
	ReasoningContent *string        `json:"reasoning_content,omitempty"`
	ToolCalls        []ChatToolCall `json:"tool_calls,omitempty"`
}

// ---------------------------------------------------------------------------
// Shared constants
// ---------------------------------------------------------------------------

// minMaxOutputTokens is the floor for max_output_tokens in a Responses request.
// Very small values may cause upstream API errors, so we enforce a minimum.
const minMaxOutputTokens = 128
