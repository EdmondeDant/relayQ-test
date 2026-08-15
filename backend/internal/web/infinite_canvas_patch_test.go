package web

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInfiniteCanvasProductionSourceAndGatewayContract(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))

	dockerfile, err := os.ReadFile(filepath.Join(root, "deploy", "infinite-canvas.Dockerfile"))
	require.NoError(t, err)
	patch, err := os.ReadFile(filepath.Join(root, "deploy", "infinite-canvas-base.patch"))
	require.NoError(t, err)

	require.Contains(t, string(dockerfile), "ARG INFINITE_CANVAS_COMMIT=b66936d891b82c2b51c1ed05e1a6eae3e31d4ca3")
	require.Contains(t, string(dockerfile), "RUN git apply /tmp/infinite-canvas-base.patch")

	contract := string(patch)
	require.Contains(t, contract, `buildApiUrl(config.baseUrl, "/v1/models")`)
	require.Contains(t, contract, `normalizedPath.toLowerCase().startsWith("/v1/")`)
	require.Equal(t, 2, strings.Count(contract, `const idempotencyKey = `+"`canvas-video-${nanoid()}`"))
	require.Equal(t, 2, strings.Count(contract, `"Idempotency-Key": idempotencyKey`))
	require.Contains(t, contract, `[payload.request_id, payload.job_id, payload.id]`)
	require.NotContains(t, contract, `showConfig={!relayq.managed}`)
	require.NotContains(t, contract, `tool.slug !== "config"`)
	require.NotContains(t, contract, `if (window.sessionStorage.getItem("relayq_canvas_bootstrap")) return`)
	require.NotContains(t, contract, `if (managed) return <Navigate to="/" replace />`)
}
