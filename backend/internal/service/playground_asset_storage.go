package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
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
	kind := normalizePlaygroundAssetKind(input.Kind)
	if kind != "image" && kind != "audio" && kind != "video" {
		return input, nil
	}
	input.Kind = kind
	if storageKey := strings.TrimSpace(input.StorageKey); storageKey != "" {
		if strings.TrimSpace(input.URL) == "" {
			input.URL = buildPlaygroundAssetURL(storageKey)
		}
		input.Content = ""
		return input, nil
	}
	if data := strings.TrimSpace(input.Content); strings.HasPrefix(strings.ToLower(data), "data:") {
		return s.persistDataURL(userID, input, data)
	}
	if rawURL := strings.TrimSpace(input.URL); rawURL != "" {
		return s.persistRemoteURL(ctx, userID, input, rawURL)
	}
	return input, nil
}

func (s *PlaygroundAssetStorage) ResolvePath(storageKey string) (string, bool) {
	meta := parseStorageKey(storageKey)
	if meta == nil {
		logger.L().Warn("playground.asset.resolve_path.invalid_key",
			zap.String("storage_key", strings.TrimSpace(storageKey)),
		)
		return "", false
	}
	resolved := filepath.Join(s.baseDir, meta.Kind, meta.UserDir, meta.FileName)
	logger.L().Info("playground.asset.resolve_path",
		zap.String("storage_key", strings.TrimSpace(storageKey)),
		zap.String("kind", meta.Kind),
		zap.String("user_dir", meta.UserDir),
		zap.String("file_name", meta.FileName),
		zap.String("resolved_path", resolved),
	)
	return resolved, true
}

func (s *PlaygroundAssetStorage) Remove(storageKey string) error {
	path, ok := s.ResolvePath(storageKey)
	if !ok {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
		_ = cleanupEmptyParentDirs(path, s.baseDir)
	return nil
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
	stored.Content = ""
	return stored, nil
}

func (s *PlaygroundAssetStorage) persistRemoteURL(ctx context.Context, userID int64, input CreatePlaygroundAssetInput, rawURL string) (CreatePlaygroundAssetInput, error) {
	resolvedURL := resolvePlaygroundAssetURL(rawURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resolvedURL, nil)
	if err != nil {
		return input, fmt.Errorf("build asset request: %w", err)
	}
	if token := playgroundAssetAuthToken(input); token != "" && isLocalPlaygroundProtectedURL(resolvedURL) {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return input, fmt.Errorf("download asset: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
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
	contentType = normalizePlaygroundContentType(contentType, payload, input.Kind)
	ext := playgroundAssetExtension(contentType, input.Kind)
	hash := sha256.Sum256(payload)
	hashHex := hex.EncodeToString(hash[:])
	userDir := fmt.Sprintf("u%d", userID)
	kindDir := normalizePlaygroundAssetKind(input.Kind)
	fileName := fmt.Sprintf("%d_%s%s", time.Now().UnixMilli(), hashHex[:12], ext)
	storageKey := strings.Join([]string{kindDir, userDir, fileName}, "/")
	dir := filepath.Join(s.baseDir, kindDir, userDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return input, fmt.Errorf("create playground asset dir: %w", err)
	}
	path := filepath.Join(dir, fileName)
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return input, fmt.Errorf("write asset file: %w", err)
	}
	byteSize := int64(len(payload))
	input.StorageKey = storageKey
	input.ContentType = contentType
	input.ByteSize = &byteSize
	input.URL = buildPlaygroundAssetURL(storageKey)
	input.Content = ""
	input.Metadata = mergePlaygroundAssetMetadata(input.Metadata, map[string]any{
		"storage_source": "managed_file",
		"storage_sha256": hashHex,
		"storage_path":   storageKey,
	})
	return input, nil
}

type playgroundStorageMeta struct {
	Kind     string
	UserDir  string
	FileName string
}

func parseStorageKey(storageKey string) *playgroundStorageMeta {
	cleaned := strings.Trim(strings.TrimSpace(storageKey), "/")
	if cleaned == "" {
		return nil
	}
	parts := strings.Split(cleaned, "/")
	if len(parts) != 3 {
		return nil
	}
	kind := sanitizePlaygroundAssetName(parts[0])
	userDir := sanitizePlaygroundAssetName(parts[1])
	fileName := filepath.Base(parts[2])
	if kind == "" || userDir == "" || fileName == "." || fileName == "" || fileName != parts[2] {
		return nil
	}
	return &playgroundStorageMeta{Kind: kind, UserDir: userDir, FileName: fileName}
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
	if kind == "image" {
		return "image/png"
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
	if kind == "image" {
		return ".png"
	}
	return ".bin"
}

func normalizePlaygroundAssetKind(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "image", "audio", "video", "text":
		return value
	default:
		return sanitizePlaygroundAssetName(value)
	}
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

func mergePlaygroundAssetMetadata(raw json.RawMessage, extra map[string]any) json.RawMessage {
	payload := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &payload)
	}
	for key, value := range extra {
		payload[key] = value
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return raw
	}
	return encoded
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

func cleanupEmptyParentDirs(path string, stopDir string) error {
	current := filepath.Dir(path)
	stop := filepath.Clean(stopDir)
	for {
		if current == stop || current == "." || current == string(filepath.Separator) {
			return nil
		}
		entries, err := os.ReadDir(current)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			return nil
		}
		if err := os.Remove(current); err != nil && !os.IsNotExist(err) {
			return err
		}
		current = filepath.Dir(current)
	}
}
