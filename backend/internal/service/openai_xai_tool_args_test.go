package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/stretchr/testify/require"
)

func TestXAIToolArgumentInterceptorRestoresRawArguments(t *testing.T) {
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 36, Platform: PlatformXAI, Type: AccountTypeOAuth}
	sessionKey := "session-a"
	svc.setXAIToolArguments(account, sessionKey, "call_a", `{"cmd":"ls", "cwd":"/root"}`)

	messages := []apicompat.ChatMessage{
		{
			Role: "assistant",
			ToolCalls: []apicompat.ChatToolCall{{
				ID:   "call_a",
				Type: "function",
				Function: apicompat.ChatFunctionCall{
					Name:      "exec",
					Arguments: `{"cmd":"ls"}{"cmd":"ls"}`,
				},
			}},
		},
		{Role: "tool", ToolCallID: "call_a", Content: []byte(`"ok"`)},
	}

	out, err := svc.applyXAIToolArgumentInterceptor(t.Context(), account, sessionKey, messages)
	require.NoError(t, err)
	require.Equal(t, `{"cmd":"ls", "cwd":"/root"}`, out[0].ToolCalls[0].Function.Arguments)
}

func TestXAIToolArgumentInterceptorRejectsOutOfOrderToolMessages(t *testing.T) {
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 36, Platform: PlatformXAI, Type: AccountTypeOAuth}
	sessionKey := "session-a"
	svc.setXAIToolArguments(account, sessionKey, "call_a", `{}`)
	svc.setXAIToolArguments(account, sessionKey, "call_b", `{}`)

	messages := []apicompat.ChatMessage{
		{
			Role: "assistant",
			ToolCalls: []apicompat.ChatToolCall{
				{ID: "call_a", Type: "function", Function: apicompat.ChatFunctionCall{Name: "a", Arguments: `{}`}},
				{ID: "call_b", Type: "function", Function: apicompat.ChatFunctionCall{Name: "b", Arguments: `{}`}},
			},
		},
		{Role: "tool", ToolCallID: "call_b", Content: []byte(`"b"`)},
		{Role: "tool", ToolCallID: "call_a", Content: []byte(`"a"`)},
	}

	_, err := svc.applyXAIToolArgumentInterceptor(t.Context(), account, sessionKey, messages)
	require.Error(t, err)
	require.Contains(t, err.Error(), "XAI_STATE_DESYNC")
}
