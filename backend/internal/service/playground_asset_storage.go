package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const playgroundMaxStoredAssetBytes = 64 * 1024 * 1024

type PlaygroundAssetStorage struct {
	baseDir    string
	httpClient *http.Client
}

func NewPlaygroundAssetStorage() *PlaygroundAssetStorage {
	return &PlaygroundAssetStorage{
		baseDir: filepath.Join(playgroundDataDir(), "playground-assets"),
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

func (s *PlaygroundAssetStorage) Persist(ctx context.Context, userID int64, input CreatePlaygroundAssetInput) (CreatePlaygroundAssetInput, error) {
	kind := strings.TrimSpace(strings.ToLower(input.Kind))
	// image/audio/video 一律落盘；text 继续存 content。
	if kind != "image" && kind != "audio" && kind != "video" {
		return input, nil
	}
	if strings.TrimSpace(input.StorageKey) != "" {
		// 已有 storage_key 时，列表侧只保留 content URL，避免大 content 回流。
		if strings.TrimSpace(input.URL) == "" {
			input.URL = buildPlaygroundAssetURL(input.StorageKey)
		}
		input.Content = ""
		return input, nil
	}
	if data := strings.TrimSpace(input.Content); strings.HasPrefix(strings.ToLower(data), "data:") {
		return s.persistDataURL(userID, input, data)
	}
	if rawURL := strings.TrimSpace(input.URL); rawURL != "" {
		// 外部 http(s) 直链可直接保存；本机受保护路径再走下载。
		if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
			if !isLocalPlaygroundProtectedURL(rawURL) {
				input.Content = ""
				return input, nil
			}
		}
		return s.persistRemoteURL(ctx, userID, input, rawURL)
	}
	return input, nil
}

func (s *PlaygroundAssetStorage) ResolvePath(storageKey string) (string, bool) {
	key := strings.TrimSpace(storageKey)
	if key == "" || filepath.Base(key) != key {
		return "", false
	}
	return filepath.Join(s.baseDir, key), true
}

func (s *PlaygroundAssetStorage) persistDataURL(userID int64, input CreatePlaygroundAssetInput, dataURL string) (CreatePlaygroundAssetInput, error) {
	contentType, payload, err := decodePlaygroundDataURL(dataURL)
	if err != nil {
		return input, err
	}
	stored, err := s.writeAssetBytes(userID, input, contentType, payload)
	if err != nil {
		return input, err
	}
	// 落盘后 content 清空，统一用 storage_key + url 引用，避免列表接口返回数 MB base64。
	stored.Content = ""
	return stored, nil
}

func (s *PlaygroundAssetStorage) persistRemoteURL(ctx context.Context, userID int64, input CreatePlaygroundAssetInput, rawURL string) (CreatePlaygroundAssetInput, error) {
	resolvedURL := resolvePlaygroundAssetURL(rawURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resolvedURL, nil)
	if err != nil {
		return input, fmt.Errorf("build asset request: %w", err)
	}
	// 本机受保护资源（如 /v1/videos/{id}/content）需要网关 API Key；前端可放在 metadata.auth_token。
	if token := playgroundAssetAuthToken(input); token != "" && isLocalPlaygroundProtectedURL(resolvedURL) {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return input, fmt.Errorf("download asset: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	// 兼容 /content 代理返回 302 到真实 CDN 的场景。
	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusPermanentRedirect {
		location := strings.TrimSpace(resp.Header.Get("Location"))
		_ = resp.Body.Close()
		if location == "" {
			return input, fmt.Errorf("download asset: redirect without location")
		}
		redirectReq, redirectErr := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
		if redirectErr != nil {
			return input, fmt.Errorf("build asset redirect request: %w", redirectErr)
		}
		resp, err = s.httpClient.Do(redirectReq)
		if err != nil {
			return input, fmt.Errorf("download asset redirect: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
	}
	if resp.StatusCode != http.StatusOK {
		return input, fmt.Errorf("download asset: unexpected status %d", resp.StatusCode)
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, playgroundMaxStoredAssetBytes+1))
	if err != nil {
		return input, fmt.Errorf("read asset body: %w", err)
	}
	if len(payload) > playgroundMaxStoredAssetBytes {
		return input, fmt.Errorf("asset too large: %d bytes", len(payload))
	}
	contentType := strings.TrimSpace(input.ContentType)
	if contentType == "" {
		contentType = strings.TrimSpace(resp.Header.Get("Content-Type"))
	}
	stored, err := s.writeAssetBytes(userID, input, contentType, payload)
	if err != nil {
		return input, err
	}
	stored.URL = ""
	stored.Content = ""
	return stored, nil
}

func (s *PlaygroundAssetStorage) writeAssetBytes(userID int64, input CreatePlaygroundAssetInput, contentType string, payload []byte) (CreatePlaygroundAssetInput, error) {
	if len(payload) == 0 {
		return input, fmt.Errorf("empty asset payload")
	}
	if err := os.MkdirAll(s.baseDir, 0o755); err != nil {
		return input, fmt.Errorf("create playground asset dir: %w", err)
	}
	contentType = normalizePlaygroundContentType(contentType, payload, input.Kind)
	ext := playgroundAssetExtension(contentType, input.Kind)
	fileName := fmt.Sprintf("u%d_%s_%s%s", userID, sanitizePlaygroundAssetName(input.Kind), strconv.FormatInt(time.Now().UnixNano(), 10), ext)
	path := filepath.Join(s.baseDir, fileName)
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return input, fmt.Errorf("write asset file: %w", err)
	}
	byteSize := int64(len(payload))
	input.StorageKey = fileName
	input.ContentType = contentType
	input.ByteSize = &byteSize
	input.URL = buildPlaygroundAssetURL(fileName)
	return input, nil
}

func decodePlaygroundDataURL(raw string) (string, []byte, error) {
	parts := strings.SplitN(strings.TrimSpace(raw), ",", 2)
	if len(parts) != 2 || !strings.HasPrefix(strings.ToLower(parts[0]), "data:") {
		return "", nil, fmt.Errorf("invalid data url")
	}
	meta := strings.TrimSpace(parts[0][5:])
	contentType := meta
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = contentType[:idx]
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(parts[1]))
	if err != nil {
		return "", nil, fmt.Errorf("decode data url: %w", err)
	}
	return contentType, decoded, nil
}

func normalizePlaygroundContentType(contentType string, payload []byte, kind string) string {
	contentType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	if contentType != "" {
		return contentType
	}
	if detected := http.DetectContentType(payload); detected != "application/octet-stream" {
		return detected
	}
	if kind == "audio" {
		return "audio/wav"
	}
	if kind == "video" {
		return "video/mp4"
	}
	return "application/octet-stream"
}

func playgroundAssetExtension(contentType string, kind string) string {
	if exts, _ := mime.ExtensionsByType(contentType); len(exts) > 0 {
		return exts[0]
	}
	if kind == "audio" {
		return ".wav"
	}
	if kind == "video" {
		return ".mp4"
	}
	return ".bin"
}

func sanitizePlaygroundAssetName(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "asset"
	}
	var b strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	name := strings.Trim(b.String(), "_")
	if name == "" {
		return "asset"
	}
	return name
}

func buildPlaygroundAssetURL(storageKey string) string {
	return "/api/v1/playground/assets/content/" + url.PathEscape(strings.TrimSpace(storageKey))
}

func resolvePlaygroundAssetURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") || strings.HasPrefix(trimmed, "data:") {
		return trimmed
	}
	if strings.HasPrefix(trimmed, "/") {
		return "http://127.0.0.1:8080" + trimmed
	}
	return trimmed
}

func playgroundAssetAuthToken(input CreatePlaygroundAssetInput) string {
	if len(input.Metadata) == 0 {
		return ""
	}
	var meta map[string]any
	if err := json.Unmarshal(input.Metadata, &meta); err != nil {
		return ""
	}
	for _, key := range []string{"auth_token", "api_key", "bearer_token"} {
		if value, ok := meta[key]; ok {
			if token := strings.TrimSpace(fmt.Sprint(value)); token != "" {
				return token
			}
		}
	}
	return ""
}

func isLocalPlaygroundProtectedURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if host != "127.0.0.1" && host != "localhost" {
		return false
	}
	path := u.EscapedPath()
	return strings.HasPrefix(path, "/v1/videos/") || strings.HasPrefix(path, "/api/v1/playground/assets/content/")
}

func playgroundDataDir() string {
	if dir := strings.TrimSpace(os.Getenv("DATA_DIR")); dir != "" {
		return dir
	}
	if info, err := os.Stat("/app/data"); err == nil && info.IsDir() {
		return "/app/data"
	}
	return "."
}
