package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDebugPlaygroundVideoCreateMissingIdempotencyKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Idempotency-Key") == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"idempotency key is required"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"request_id":"video_123"}`))
	}))
	defer server.Close()

	_, _, err := doPlaygroundJSONRequest(context.Background(), server.URL, "test-key", http.MethodPost, "/v1/videos/generations", json.RawMessage(`{"model":"wan-2.7"}`))
	require.EqualError(t, err, "idempotency key is required")
}

func TestPlaygroundVideoCreateSendsStableIdempotencyKey(t *testing.T) {
	var received string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.Header.Get("Idempotency-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"request_id":"video_123"}`))
	}))
	defer server.Close()

	_, _, err := doPlaygroundJSONRequestWithIdempotency(context.Background(), server.URL, "test-key", http.MethodPost, "/v1/videos/generations", json.RawMessage(`{"model":"wan-2.7"}`), "playground-video-42")
	require.NoError(t, err)
	require.Equal(t, "playground-video-42", received)
}

func TestExtractVideoStatusReadsRelayQEnvelope(t *testing.T) {
	status := extractVideoStatus(map[string]any{
		"code": float64(0),
		"data": map[string]any{"id": "gen_rq_0123456789abcdef0123456789abcdef", "status": "queued"},
	})
	require.Equal(t, "gen_rq_0123456789abcdef0123456789abcdef", status.RequestID)
	require.Equal(t, "queued", status.Status)
}

func TestExtractVideoStatusReadsMiniMaxJobID(t *testing.T) {
	status := extractVideoStatus(map[string]any{"job_id": "job-123", "status": "queued"})
	require.Equal(t, "job-123", status.RequestID)
}
