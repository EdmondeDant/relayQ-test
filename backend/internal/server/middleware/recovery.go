package middleware

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

func reportPlayground500DebugEvent(hypothesisID, location, msg string, data map[string]any) {
	apiURL := "http://127.0.0.1:7777/event"
	sessionID := "playground-500-blockers"
	if envBytes, err := os.ReadFile(".dbg/playground-500-blockers.env"); err == nil {
		for _, line := range strings.Split(string(envBytes), "\n") {
			if strings.HasPrefix(line, "DEBUG_SERVER_URL=") {
				apiURL = strings.TrimSpace(strings.TrimPrefix(line, "DEBUG_SERVER_URL="))
			} else if strings.HasPrefix(line, "DEBUG_SESSION_ID=") {
				sessionID = strings.TrimSpace(strings.TrimPrefix(line, "DEBUG_SESSION_ID="))
			}
		}
	}
	body, err := json.Marshal(map[string]any{
		"sessionId":    sessionID,
		"runId":        "pre-fix",
		"hypothesisId": hypothesisID,
		"location":     location,
		"msg":          msg,
		"data":         data,
		"ts":           time.Now().UnixMilli(),
	})
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err == nil && resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
}

// Recovery converts panics into the project's standard JSON error envelope.
//
// It preserves Gin's broken-pipe handling by not attempting to write a response
// when the client connection is already gone.
func Recovery() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(gin.DefaultErrorWriter, func(c *gin.Context, recovered any) {
		recoveredErr, _ := recovered.(error)

		if isBrokenPipe(recoveredErr) {
			if recoveredErr != nil {
				_ = c.Error(recoveredErr)
			}
			c.Abort()
			return
		}

		if c.Writer.Written() {
			c.Abort()
			return
		}

		// #region debug-point A:recovery-panic
		reportPlayground500DebugEvent("A", "recovery.go:Recovery", "[DEBUG] recovered panic in middleware", map[string]any{
			"path":        c.Request.URL.Path,
			"method":      c.Request.Method,
			"query":       c.Request.URL.RawQuery,
			"recovered":   fmt.Sprint(recovered),
			"remote_addr": c.Request.RemoteAddr,
		})
		// #endregion

		response.ErrorWithDetails(
			c,
			http.StatusInternalServerError,
			infraerrors.UnknownMessage,
			infraerrors.UnknownReason,
			nil,
		)
		c.Abort()
	})
}

func isBrokenPipe(err error) bool {
	if err == nil {
		return false
	}

	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		return false
	}

	var syscallErr *os.SyscallError
	if !errors.As(opErr.Err, &syscallErr) {
		return false
	}

	msg := strings.ToLower(syscallErr.Error())
	return strings.Contains(msg, "broken pipe") || strings.Contains(msg, "connection reset by peer")
}
