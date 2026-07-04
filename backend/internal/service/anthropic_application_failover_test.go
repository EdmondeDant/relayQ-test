//go:build unit

package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldAnthropicApplicationNotFoundFailover(t *testing.T) {
	account := &Account{Platform: PlatformAnthropic}
	require.True(t, shouldAnthropicApplicationNotFoundFailover(account, http.StatusNotFound, []byte(`{"message":"Application not found"}`)))
	require.True(t, shouldAnthropicApplicationNotFoundFailover(account, http.StatusNotFound, []byte(`{"error":"application unavailable"}`)))
	require.False(t, shouldAnthropicApplicationNotFoundFailover(account, http.StatusNotFound, []byte(`{"message":"model not found"}`)))
	require.False(t, shouldAnthropicApplicationNotFoundFailover(&Account{Platform: PlatformOpenAI}, http.StatusNotFound, []byte(`{"message":"Application not found"}`)))
	require.False(t, shouldAnthropicApplicationNotFoundFailover(account, http.StatusBadGateway, []byte(`{"message":"Application not found"}`)))
}
