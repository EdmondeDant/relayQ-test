package handler

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const playgroundSourceHeader = "X-RelayQ-Source"

var playgroundAllowedModels = map[string]struct{}{
	"gpt-image-2":        {},
	"gpt-image-2-pro":    {},
	"deepseek-v4-flash":  {},
	"gpt-5.4":            {},
	"grok-imagine-video": {},
}

// PlaygroundGuard enforces the server-side MVP contract for browser playground
// traffic while keeping normal API-key gateway calls untouched.
func PlaygroundGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.ToLower(strings.TrimSpace(c.GetHeader(playgroundSourceHeader))) != "playground" {
			c.Next()
			return
		}

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		if !isPlaygroundGenerationEndpoint(path) {
			c.Next()
			return
		}

		if strings.Contains(path, "/images/") && strings.TrimSpace(c.GetHeader("Idempotency-Key")) == "" {
			playgroundError(c, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", "Idempotency-Key is required for playground image generation")
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			playgroundError(c, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Failed to read request body")
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		if !gjson.ValidBytes(body) {
			playgroundError(c, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Failed to parse request body")
			return
		}
		model := strings.TrimSpace(gjson.GetBytes(body, "model").String())
		if _, ok := playgroundAllowedModels[model]; !ok {
			playgroundError(c, http.StatusForbidden, "MODEL_NOT_ALLOWED", "This model is not available in online experience")
			return
		}

		c.Next()
	}
}

func isPlaygroundGenerationEndpoint(path string) bool {
	switch path {
	case "/v1/images/generations", "/images/generations", "/v1/images/edits", "/images/edits", "/v1/chat/completions", "/chat/completions", "/v1/videos/generations", "/videos/generations":
		return true
	default:
		return false
	}
}

func playgroundError(c *gin.Context, status int, code string, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}
