package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadHandler_Index(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handler := NewDownloadHandler()
	router.GET("/downloads", handler.Index)

	req := httptest.NewRequest(http.MethodGet, "/downloads", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "cc-switch-windows-x64-msi")
	assert.Contains(t, recorder.Body.String(), "nodejs-linux-x64-tar-xz")
	assert.Contains(t, recorder.Body.String(), "\"size_bytes\"")
}

func TestDownloadHandler_DownloadsAndCachesFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tempDir := t.TempDir()
	body := []byte("relayq-download-cache")
	var requestCount atomic.Int32

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
		_, _ = w.Write(body)
	}))
	defer upstream.Close()

	handler := newDownloadHandler(tempDir, upstream.Client(), map[string]DownloadAsset{
		"test-download": {
			Slug:          "test-download",
			Product:       "Test",
			Title:         "Fixture",
			Version:       "v1.0.0",
			Platform:      "Windows",
			Architecture:  "x64",
			PackageFormat: "ZIP",
			FileName:      "fixture.zip",
			SizeBytes:     int64(len(body)),
			SourceURL:     upstream.URL + "/fixture.zip",
		},
	}, func(string) error { return nil })

	router := gin.New()
	router.GET("/downloads/:slug", handler.Redirect)

	req := httptest.NewRequest(http.MethodGet, "/downloads/test-download", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "miss", recorder.Header().Get("X-RelayQ-Cache-Status"))
	assert.Equal(t, "attachment; filename=\"fixture.zip\"", recorder.Header().Get("Content-Disposition"))
	assert.Equal(t, string(body), recorder.Body.String())
	assert.Equal(t, int32(1), requestCount.Load())

	cachePath := filepath.Join(tempDir, "test-download", "fixture.zip")
	cachedData, err := os.ReadFile(cachePath)
	require.NoError(t, err)
	assert.Equal(t, body, cachedData)

	req2 := httptest.NewRequest(http.MethodGet, "/downloads/test-download", nil)
	recorder2 := httptest.NewRecorder()
	router.ServeHTTP(recorder2, req2)

	require.Equal(t, http.StatusOK, recorder2.Code)
	assert.Equal(t, "hit", recorder2.Header().Get("X-RelayQ-Cache-Status"))
	assert.Equal(t, string(body), recorder2.Body.String())
	assert.Equal(t, int32(1), requestCount.Load())
}

func TestDownloadHandler_RejectsUnknownSlug(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handler := NewDownloadHandler()
	router.GET("/downloads/:slug", handler.Redirect)

	req := httptest.NewRequest(http.MethodGet, "/downloads/not-exist", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestDownloadHandler_ReturnsBadGatewayWhenUpstreamFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tempDir := t.TempDir()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer upstream.Close()

	handler := newDownloadHandler(tempDir, upstream.Client(), map[string]DownloadAsset{
		"broken-download": {
			Slug:          "broken-download",
			Product:       "Test",
			Title:         "Broken",
			Version:       "v1.0.0",
			Platform:      "Linux",
			Architecture:  "x64",
			PackageFormat: "TAR.XZ",
			FileName:      "broken.tar.xz",
			SizeBytes:     4,
			SourceURL:     upstream.URL + "/broken.tar.xz",
		},
	}, func(string) error { return nil })

	router := gin.New()
	router.GET("/downloads/:slug", handler.Redirect)

	req := httptest.NewRequest(http.MethodGet, "/downloads/broken-download", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "download unavailable")
}

func TestValidateTrustedDownloadURL(t *testing.T) {
	assert.NoError(t, validateTrustedDownloadURL("https://github.com/example/file.zip"))
	assert.NoError(t, validateTrustedDownloadURL("https://release-assets.githubusercontent.com/example/file.zip"))
	assert.NoError(t, validateTrustedDownloadURL("https://nodejs.org/dist/file.pkg"))
	assert.Error(t, validateTrustedDownloadURL("http://github.com/example/file.zip"))
	assert.Error(t, validateTrustedDownloadURL("https://evil.example.com/file.zip"))
}

func TestIsCachedDownloadReady(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "asset.zip")

	assert.False(t, isCachedDownloadReady(filePath, 10))

	err := os.WriteFile(filePath, []byte("12345"), 0o644)
	require.NoError(t, err)

	assert.False(t, isCachedDownloadReady(filePath, 10))
	assert.True(t, isCachedDownloadReady(filePath, 5))

	file, err := os.Open(filePath)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()
	_, _ = io.ReadAll(file)
}
