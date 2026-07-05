package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const xaiToolArgumentsTTL = 2 * time.Hour

type xaiToolArgumentEntry struct {
	Arguments string
	ExpiresAt time.Time
}

type xaiToolArgumentStreamCache struct {
	ids     map[int]string
	buffers map[int]string
}

func newXAIToolArgumentStreamCache() *xaiToolArgumentStreamCache {
	return &xaiToolArgumentStreamCache{
		ids:     make(map[int]string),
		buffers: make(map[int]string),
	}
}

func (s *OpenAIGatewayService) xaiToolArgumentSessionKey(c *gin.Context, body []byte) string {
	if s == nil {
		return ""
	}
	key := strings.TrimSpace(s.GenerateSessionHash(c, body))
	if key != "" {
		return key
	}
	if c != nil {
		for _, header := range []string{"conversation_id", "session_id", "x-session-id", "openai-conversation-id"} {
			if v := strings.TrimSpace(c.GetHeader(header)); v != "" {
				return DeriveSessionHashFromSeed(v)
			}
		}
	}
	return ""
}

func xaiToolArgumentsKey(account *Account, sessionKey, callID string) string {
	if account == nil || strings.TrimSpace(sessionKey) == "" || strings.TrimSpace(callID) == "" {
		return ""
	}
	return fmt.Sprintf("xai_tool_args:%d:%s:%s", account.ID, strings.TrimSpace(sessionKey), strings.TrimSpace(callID))
}

func (s *OpenAIGatewayService) setXAIToolArguments(account *Account, sessionKey, callID, arguments string) {
	key := xaiToolArgumentsKey(account, sessionKey, callID)
	if s == nil || key == "" {
		return
	}
	s.xaiToolArguments.Store(key, xaiToolArgumentEntry{Arguments: arguments, ExpiresAt: time.Now().Add(xaiToolArgumentsTTL)})
}

func (s *OpenAIGatewayService) getXAIToolArguments(account *Account, sessionKey, callID string) (string, bool) {
	key := xaiToolArgumentsKey(account, sessionKey, callID)
	if s == nil || key == "" {
		return "", false
	}
	value, ok := s.xaiToolArguments.Load(key)
	if !ok {
		return "", false
	}
	entry, ok := value.(xaiToolArgumentEntry)
	if !ok {
		s.xaiToolArguments.Delete(key)
		return "", false
	}
	if !entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt) {
		s.xaiToolArguments.Delete(key)
		return "", false
	}
	return entry.Arguments, true
}

func (s *OpenAIGatewayService) recordXAIChatCompletionsResponseToolArguments(account *Account, sessionKey string, resp *apicompat.ChatCompletionsResponse) {
	if !isXAIOAuthAccount(account) || strings.TrimSpace(sessionKey) == "" || resp == nil {
		return
	}
	for _, choice := range resp.Choices {
		for _, toolCall := range choice.Message.ToolCalls {
			if strings.TrimSpace(toolCall.ID) == "" {
				continue
			}
			s.setXAIToolArguments(account, sessionKey, toolCall.ID, toolCall.Function.Arguments)
		}
	}
}

func (s *OpenAIGatewayService) recordXAIChatCompletionsStreamToolArguments(account *Account, sessionKey string, cache *xaiToolArgumentStreamCache, chunk *apicompat.ChatCompletionsChunk) {
	if !isXAIOAuthAccount(account) || strings.TrimSpace(sessionKey) == "" || cache == nil || chunk == nil {
		return
	}
	for _, choice := range chunk.Choices {
		for _, toolCall := range choice.Delta.ToolCalls {
			idx := 0
			if toolCall.Index != nil {
				idx = *toolCall.Index
			}
			if strings.TrimSpace(toolCall.ID) != "" {
				cache.ids[idx] = toolCall.ID
			}
			if toolCall.Function.Arguments != "" {
				cache.buffers[idx] += toolCall.Function.Arguments
			}
			if callID := strings.TrimSpace(cache.ids[idx]); callID != "" {
				s.setXAIToolArguments(account, sessionKey, callID, cache.buffers[idx])
			}
		}
	}
}

func (s *OpenAIGatewayService) finalizeXAIChatCompletionsStreamToolArguments(account *Account, sessionKey string, cache *xaiToolArgumentStreamCache, state *apicompat.ChatCompletionsToResponsesStreamState) {
	if !isXAIOAuthAccount(account) || strings.TrimSpace(sessionKey) == "" {
		return
	}
	if state != nil {
		for _, toolCall := range state.ToolCalls {
			if toolCall == nil || strings.TrimSpace(toolCall.ID) == "" {
				continue
			}
			s.setXAIToolArguments(account, sessionKey, toolCall.ID, toolCall.Function.Arguments)
		}
	}
	if cache != nil {
		for idx, callID := range cache.ids {
			if strings.TrimSpace(callID) == "" {
				continue
			}
			s.setXAIToolArguments(account, sessionKey, callID, cache.buffers[idx])
		}
	}
}

func (s *OpenAIGatewayService) applyXAIToolArgumentInterceptor(ctx context.Context, account *Account, sessionKey string, messages []apicompat.ChatMessage) ([]apicompat.ChatMessage, error) {
	if !isXAIOAuthAccount(account) || strings.TrimSpace(sessionKey) == "" || len(messages) == 0 {
		return messages, nil
	}
	for i := 0; i < len(messages); i++ {
		msg := &messages[i]
		if msg.Role != "assistant" || len(msg.ToolCalls) == 0 {
			continue
		}
		expected := make([]string, 0, len(msg.ToolCalls))
		for j := range msg.ToolCalls {
			callID := strings.TrimSpace(msg.ToolCalls[j].ID)
			if callID == "" {
				return nil, fmt.Errorf("XAI_STATE_DESYNC: assistant tool_call missing id")
			}
			expected = append(expected, callID)
			if raw, ok := s.getXAIToolArguments(account, sessionKey, callID); ok {
				msg.ToolCalls[j].Function.Arguments = raw
			} else if !isValidChatToolArguments(msg.ToolCalls[j].Function.Arguments) {
				logger.L().Warn("xai tool arguments missing and current arguments invalid",
					zap.Int64("account_id", account.ID),
					zap.String("call_id", callID),
				)
				return nil, fmt.Errorf("XAI_TOOL_ARGUMENTS_MISSING_OR_CORRUPT: %s", callID)
			}
		}

		for offset, callID := range expected {
			idx := i + 1 + offset
			if idx >= len(messages) {
				return nil, fmt.Errorf("XAI_STATE_DESYNC: missing tool message for %s", callID)
			}
			next := messages[idx]
			if next.Role != "tool" || strings.TrimSpace(next.ToolCallID) != callID {
				return nil, fmt.Errorf("XAI_STATE_DESYNC: tool message order mismatch for %s", callID)
			}
		}
		i += len(expected)
	}
	return messages, nil
}

func isValidChatToolArguments(arguments string) bool {
	trimmed := strings.TrimSpace(arguments)
	if trimmed == "" {
		return false
	}
	var raw json.RawMessage
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return false
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return false
	}
	return true
}
