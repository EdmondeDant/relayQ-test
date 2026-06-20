package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/setup"
	"github.com/gin-gonic/gin"
)

const (
	downloadCacheDirName  = "downloads-cache"
	maxDownloadFileSize   = 1024 * 1024 * 1024
	maxDownloadRedirects  = 10
	downloadClientTimeout = 45 * time.Minute
)

var (
	downloadSlugPattern     = regexp.MustCompile(`^[a-z0-9-]+$`)
	downloadFileNamePattern = regexp.MustCompile(`^[A-Za-z0-9._+-]+$`)
)

type DownloadAsset struct {
	Slug          string `json:"slug"`
	Product       string `json:"product"`
	Title         string `json:"title"`
	Version       string `json:"version"`
	Platform      string `json:"platform"`
	Architecture  string `json:"architecture"`
	PackageFormat string `json:"package_format"`
	FileName      string `json:"file_name"`
	SizeBytes     int64  `json:"size_bytes"`
	SourceURL     string `json:"-"`
}

type downloadURLValidator func(rawURL string) error

type DownloadHandler struct {
	cacheDir     string
	client       *http.Client
	catalog      map[string]DownloadAsset
	urlValidator downloadURLValidator
	assetLocks   sync.Map
}

func NewDownloadHandler() *DownloadHandler {
	return newDownloadHandler(
		filepath.Join(setup.GetDataDir(), downloadCacheDirName),
		nil,
		downloadCatalog,
		validateTrustedDownloadURL,
	)
}

func newDownloadHandler(
	cacheDir string,
	client *http.Client,
	catalog map[string]DownloadAsset,
	urlValidator downloadURLValidator,
) *DownloadHandler {
	validator := urlValidator
	if validator == nil {
		validator = validateTrustedDownloadURL
	}

	if client == nil {
		client = newDownloadHTTPClient(validator)
	}

	return &DownloadHandler{
		cacheDir:     cacheDir,
		client:       client,
		catalog:      catalog,
		urlValidator: validator,
	}
}

func newDownloadHTTPClient(validator downloadURLValidator) *http.Client {
	return &http.Client{
		Timeout: downloadClientTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxDownloadRedirects {
				return errors.New("too many redirects")
			}
			return validator(req.URL.String())
		},
	}
}

func (h *DownloadHandler) Index(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"cache_dir": filepath.ToSlash(h.cacheDir),
		"downloads": availableDownloads(h.catalog),
	})
}

func (h *DownloadHandler) Redirect(c *gin.Context) {
	slug := strings.TrimSpace(c.Param("slug"))
	if !downloadSlugPattern.MatchString(slug) || len(slug) > 96 {
		c.Status(http.StatusNotFound)
		return
	}

	asset, ok := h.catalog[slug]
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}

	filePath, cacheStatus, err := h.ensureCached(c.Request.Context(), asset)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "download unavailable",
			"slug":  slug,
		})
		return
	}

	if contentType := mime.TypeByExtension(filepath.Ext(asset.FileName)); contentType != "" {
		c.Header("Content-Type", contentType)
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("X-RelayQ-Download-Mode", "cached-proxy")
	c.Header("X-RelayQ-Cache-Status", cacheStatus)
	c.FileAttachment(filePath, asset.FileName)
}

func (h *DownloadHandler) ensureCached(ctx context.Context, asset DownloadAsset) (string, string, error) {
	if err := h.validateAsset(asset); err != nil {
		return "", "", err
	}

	cachePath := h.cacheFilePath(asset)
	if isCachedDownloadReady(cachePath, asset.SizeBytes) {
		return cachePath, "hit", nil
	}

	lock := h.getAssetLock(asset.Slug)
	lock.Lock()
	defer lock.Unlock()

	if isCachedDownloadReady(cachePath, asset.SizeBytes) {
		return cachePath, "hit", nil
	}

	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return "", "", fmt.Errorf("create download cache dir: %w", err)
	}

	tempPath := cachePath + ".part"
	_ = os.Remove(tempPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.SourceURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("build download request: %w", err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("request upstream asset: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected upstream status: %d", resp.StatusCode)
	}

	if resp.ContentLength > maxDownloadFileSize {
		return "", "", fmt.Errorf("download exceeds limit: %d", resp.ContentLength)
	}

	file, err := os.Create(tempPath)
	if err != nil {
		return "", "", fmt.Errorf("create cache file: %w", err)
	}

	written, copyErr := io.Copy(file, io.LimitReader(resp.Body, maxDownloadFileSize+1))
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(tempPath)
		return "", "", fmt.Errorf("cache upstream asset: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tempPath)
		return "", "", fmt.Errorf("close cache file: %w", closeErr)
	}
	if written > maxDownloadFileSize {
		_ = os.Remove(tempPath)
		return "", "", fmt.Errorf("download exceeds limit after copy: %d", written)
	}
	if asset.SizeBytes > 0 && written != asset.SizeBytes {
		_ = os.Remove(tempPath)
		return "", "", fmt.Errorf("download size mismatch: expected %d got %d", asset.SizeBytes, written)
	}

	if err := os.Rename(tempPath, cachePath); err != nil {
		_ = os.Remove(tempPath)
		return "", "", fmt.Errorf("finalize cache file: %w", err)
	}

	return cachePath, "miss", nil
}

func (h *DownloadHandler) validateAsset(asset DownloadAsset) error {
	if !downloadSlugPattern.MatchString(asset.Slug) || len(asset.Slug) > 96 {
		return fmt.Errorf("invalid download slug")
	}
	if filepath.Base(asset.FileName) != asset.FileName || !downloadFileNamePattern.MatchString(asset.FileName) {
		return fmt.Errorf("invalid download filename")
	}
	if asset.SizeBytes <= 0 || asset.SizeBytes > maxDownloadFileSize {
		return fmt.Errorf("invalid download size")
	}
	if err := h.urlValidator(asset.SourceURL); err != nil {
		return err
	}
	return nil
}

func (h *DownloadHandler) cacheFilePath(asset DownloadAsset) string {
	return filepath.Join(h.cacheDir, asset.Slug, asset.FileName)
}

func (h *DownloadHandler) getAssetLock(slug string) *sync.Mutex {
	lock, _ := h.assetLocks.LoadOrStore(slug, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func availableDownloads(catalog map[string]DownloadAsset) []DownloadAsset {
	items := make([]DownloadAsset, 0, len(catalog))
	for _, asset := range catalog {
		items = append(items, asset)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Slug < items[j].Slug
	})

	return items
}

func isCachedDownloadReady(filePath string, expectedSize int64) bool {
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() || info.Size() <= 0 {
		return false
	}
	if expectedSize > 0 && info.Size() != expectedSize {
		return false
	}
	return true
}

func validateTrustedDownloadURL(rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Scheme != "https" {
		return fmt.Errorf("only HTTPS URLs are allowed")
	}

	host := strings.ToLower(strings.TrimSpace(parsedURL.Hostname()))
	switch host {
	case "github.com", "objects.githubusercontent.com", "release-assets.githubusercontent.com", "nodejs.org":
		return nil
	default:
		return fmt.Errorf("download from untrusted host: %s", host)
	}
}

var downloadCatalog = map[string]DownloadAsset{
	"cc-switch-windows-x64-msi": {
		Slug:          "cc-switch-windows-x64-msi",
		Product:       "CC Switch",
		Title:         "Windows x64 MSI",
		Version:       "v3.14.1",
		Platform:      "Windows",
		Architecture:  "x64",
		PackageFormat: "MSI",
		FileName:      "CC-Switch-v3.14.1-Windows.msi",
		SizeBytes:     11726848,
		SourceURL:     "https://github.com/farion1231/cc-switch/releases/download/v3.14.1/CC-Switch-v3.14.1-Windows.msi",
	},
	"cc-switch-windows-x64-portable-zip": {
		Slug:          "cc-switch-windows-x64-portable-zip",
		Product:       "CC Switch",
		Title:         "Windows x64 Portable ZIP",
		Version:       "v3.14.1",
		Platform:      "Windows",
		Architecture:  "x64",
		PackageFormat: "ZIP",
		FileName:      "CC-Switch-v3.14.1-Windows-Portable.zip",
		SizeBytes:     11411054,
		SourceURL:     "https://github.com/farion1231/cc-switch/releases/download/v3.14.1/CC-Switch-v3.14.1-Windows-Portable.zip",
	},
	"cc-switch-macos-universal-dmg": {
		Slug:          "cc-switch-macos-universal-dmg",
		Product:       "CC Switch",
		Title:         "macOS Universal DMG",
		Version:       "v3.14.1",
		Platform:      "macOS",
		Architecture:  "Universal",
		PackageFormat: "DMG",
		FileName:      "CC-Switch-v3.14.1-macOS.dmg",
		SizeBytes:     24515295,
		SourceURL:     "https://github.com/farion1231/cc-switch/releases/download/v3.14.1/CC-Switch-v3.14.1-macOS.dmg",
	},
	"cc-switch-macos-universal-zip": {
		Slug:          "cc-switch-macos-universal-zip",
		Product:       "CC Switch",
		Title:         "macOS Universal ZIP",
		Version:       "v3.14.1",
		Platform:      "macOS",
		Architecture:  "Universal",
		PackageFormat: "ZIP",
		FileName:      "CC-Switch-v3.14.1-macOS.zip",
		SizeBytes:     24534878,
		SourceURL:     "https://github.com/farion1231/cc-switch/releases/download/v3.14.1/CC-Switch-v3.14.1-macOS.zip",
	},
	"cc-switch-macos-universal-tar-gz": {
		Slug:          "cc-switch-macos-universal-tar-gz",
		Product:       "CC Switch",
		Title:         "macOS Universal TAR.GZ",
		Version:       "v3.14.1",
		Platform:      "macOS",
		Architecture:  "Universal",
		PackageFormat: "TAR.GZ",
		FileName:      "CC-Switch-v3.14.1-macOS.tar.gz",
		SizeBytes:     25122972,
		SourceURL:     "https://github.com/farion1231/cc-switch/releases/download/v3.14.1/CC-Switch-v3.14.1-macOS.tar.gz",
	},
	"cc-switch-linux-x64-deb": {
		Slug:          "cc-switch-linux-x64-deb",
		Product:       "CC Switch",
		Title:         "Linux x64 DEB",
		Version:       "v3.14.1",
		Platform:      "Linux",
		Architecture:  "x64",
		PackageFormat: "DEB",
		FileName:      "CC-Switch-v3.14.1-Linux-x86_64.deb",
		SizeBytes:     11895708,
		SourceURL:     "https://github.com/farion1231/cc-switch/releases/download/v3.14.1/CC-Switch-v3.14.1-Linux-x86_64.deb",
	},
	"cc-switch-linux-arm64-deb": {
		Slug:          "cc-switch-linux-arm64-deb",
		Product:       "CC Switch",
		Title:         "Linux ARM64 DEB",
		Version:       "v3.14.1",
		Platform:      "Linux",
		Architecture:  "ARM64",
		PackageFormat: "DEB",
		FileName:      "CC-Switch-v3.14.1-Linux-arm64.deb",
		SizeBytes:     11433276,
		SourceURL:     "https://github.com/farion1231/cc-switch/releases/download/v3.14.1/CC-Switch-v3.14.1-Linux-arm64.deb",
	},
	"cc-switch-linux-x64-rpm": {
		Slug:          "cc-switch-linux-x64-rpm",
		Product:       "CC Switch",
		Title:         "Linux x64 RPM",
		Version:       "v3.14.1",
		Platform:      "Linux",
		Architecture:  "x64",
		PackageFormat: "RPM",
		FileName:      "CC-Switch-v3.14.1-Linux-x86_64.rpm",
		SizeBytes:     11896466,
		SourceURL:     "https://github.com/farion1231/cc-switch/releases/download/v3.14.1/CC-Switch-v3.14.1-Linux-x86_64.rpm",
	},
	"cc-switch-linux-arm64-rpm": {
		Slug:          "cc-switch-linux-arm64-rpm",
		Product:       "CC Switch",
		Title:         "Linux ARM64 RPM",
		Version:       "v3.14.1",
		Platform:      "Linux",
		Architecture:  "ARM64",
		PackageFormat: "RPM",
		FileName:      "CC-Switch-v3.14.1-Linux-arm64.rpm",
		SizeBytes:     11436384,
		SourceURL:     "https://github.com/farion1231/cc-switch/releases/download/v3.14.1/CC-Switch-v3.14.1-Linux-arm64.rpm",
	},
	"cc-switch-linux-x64-appimage": {
		Slug:          "cc-switch-linux-x64-appimage",
		Product:       "CC Switch",
		Title:         "Linux x64 AppImage",
		Version:       "v3.14.1",
		Platform:      "Linux",
		Architecture:  "x64",
		PackageFormat: "AppImage",
		FileName:      "CC-Switch-v3.14.1-Linux-x86_64.AppImage",
		SizeBytes:     90749432,
		SourceURL:     "https://github.com/farion1231/cc-switch/releases/download/v3.14.1/CC-Switch-v3.14.1-Linux-x86_64.AppImage",
	},
	"cc-switch-linux-arm64-appimage": {
		Slug:          "cc-switch-linux-arm64-appimage",
		Product:       "CC Switch",
		Title:         "Linux ARM64 AppImage",
		Version:       "v3.14.1",
		Platform:      "Linux",
		Architecture:  "ARM64",
		PackageFormat: "AppImage",
		FileName:      "CC-Switch-v3.14.1-Linux-arm64.AppImage",
		SizeBytes:     88398344,
		SourceURL:     "https://github.com/farion1231/cc-switch/releases/download/v3.14.1/CC-Switch-v3.14.1-Linux-arm64.AppImage",
	},
	"codex-windows-x64-exe": {
		Slug:          "codex-windows-x64-exe",
		Product:       "Codex",
		Title:         "Windows x64 Desktop EXE",
		Version:       "0.128.0",
		Platform:      "Windows",
		Architecture:  "x64",
		PackageFormat: "EXE",
		FileName:      "codex-x86_64-pc-windows-msvc.exe",
		SizeBytes:     245125936,
		SourceURL:     "https://github.com/openai/codex/releases/download/rust-v0.128.0/codex-x86_64-pc-windows-msvc.exe",
	},
	"codex-windows-arm64-exe": {
		Slug:          "codex-windows-arm64-exe",
		Product:       "Codex",
		Title:         "Windows ARM64 Desktop EXE",
		Version:       "0.128.0",
		Platform:      "Windows",
		Architecture:  "ARM64",
		PackageFormat: "EXE",
		FileName:      "codex-aarch64-pc-windows-msvc.exe",
		SizeBytes:     207270704,
		SourceURL:     "https://github.com/openai/codex/releases/download/rust-v0.128.0/codex-aarch64-pc-windows-msvc.exe",
	},
	"codex-macos-arm64-dmg": {
		Slug:          "codex-macos-arm64-dmg",
		Product:       "Codex",
		Title:         "macOS Apple Silicon Desktop DMG",
		Version:       "0.128.0",
		Platform:      "macOS",
		Architecture:  "ARM64",
		PackageFormat: "DMG",
		FileName:      "codex-aarch64-apple-darwin.dmg",
		SizeBytes:     92905254,
		SourceURL:     "https://github.com/openai/codex/releases/download/rust-v0.128.0/codex-aarch64-apple-darwin.dmg",
	},
	"codex-macos-intel-dmg": {
		Slug:          "codex-macos-intel-dmg",
		Product:       "Codex",
		Title:         "macOS Intel Desktop DMG",
		Version:       "0.128.0",
		Platform:      "macOS",
		Architecture:  "Intel",
		PackageFormat: "DMG",
		FileName:      "codex-x86_64-apple-darwin.dmg",
		SizeBytes:     104808046,
		SourceURL:     "https://github.com/openai/codex/releases/download/rust-v0.128.0/codex-x86_64-apple-darwin.dmg",
	},
	"cherry-studio-windows-x64-setup-exe": {
		Slug:          "cherry-studio-windows-x64-setup-exe",
		Product:       "Cherry Studio",
		Title:         "Windows x64 Setup EXE",
		Version:       "v1.9.2",
		Platform:      "Windows",
		Architecture:  "x64",
		PackageFormat: "EXE",
		FileName:      "Cherry-Studio-1.9.2-x64-setup.exe",
		SizeBytes:     139520901,
		SourceURL:     "https://github.com/CherryHQ/cherry-studio/releases/download/v1.9.2/Cherry-Studio-1.9.2-x64-setup.exe",
	},
	"cherry-studio-windows-x64-portable-exe": {
		Slug:          "cherry-studio-windows-x64-portable-exe",
		Product:       "Cherry Studio",
		Title:         "Windows x64 Portable EXE",
		Version:       "v1.9.2",
		Platform:      "Windows",
		Architecture:  "x64",
		PackageFormat: "Portable",
		FileName:      "Cherry-Studio-1.9.2-x64-portable.exe",
		SizeBytes:     139154633,
		SourceURL:     "https://github.com/CherryHQ/cherry-studio/releases/download/v1.9.2/Cherry-Studio-1.9.2-x64-portable.exe",
	},
	"cherry-studio-windows-arm64-setup-exe": {
		Slug:          "cherry-studio-windows-arm64-setup-exe",
		Product:       "Cherry Studio",
		Title:         "Windows ARM64 Setup EXE",
		Version:       "v1.9.2",
		Platform:      "Windows",
		Architecture:  "ARM64",
		PackageFormat: "EXE",
		FileName:      "Cherry-Studio-1.9.2-arm64-setup.exe",
		SizeBytes:     133145036,
		SourceURL:     "https://github.com/CherryHQ/cherry-studio/releases/download/v1.9.2/Cherry-Studio-1.9.2-arm64-setup.exe",
	},
	"cherry-studio-macos-intel-dmg": {
		Slug:          "cherry-studio-macos-intel-dmg",
		Product:       "Cherry Studio",
		Title:         "macOS Intel DMG",
		Version:       "v1.9.2",
		Platform:      "macOS",
		Architecture:  "Intel",
		PackageFormat: "DMG",
		FileName:      "Cherry-Studio-1.9.2-x64.dmg",
		SizeBytes:     196713587,
		SourceURL:     "https://github.com/CherryHQ/cherry-studio/releases/download/v1.9.2/Cherry-Studio-1.9.2-x64.dmg",
	},
	"cherry-studio-macos-arm64-dmg": {
		Slug:          "cherry-studio-macos-arm64-dmg",
		Product:       "Cherry Studio",
		Title:         "macOS Apple Silicon DMG",
		Version:       "v1.9.2",
		Platform:      "macOS",
		Architecture:  "ARM64",
		PackageFormat: "DMG",
		FileName:      "Cherry-Studio-1.9.2-arm64.dmg",
		SizeBytes:     186294940,
		SourceURL:     "https://github.com/CherryHQ/cherry-studio/releases/download/v1.9.2/Cherry-Studio-1.9.2-arm64.dmg",
	},
	"cherry-studio-linux-x64-appimage": {
		Slug:          "cherry-studio-linux-x64-appimage",
		Product:       "Cherry Studio",
		Title:         "Linux x64 AppImage",
		Version:       "v1.9.2",
		Platform:      "Linux",
		Architecture:  "x64",
		PackageFormat: "AppImage",
		FileName:      "Cherry-Studio-1.9.2-x86_64.AppImage",
		SizeBytes:     220743039,
		SourceURL:     "https://github.com/CherryHQ/cherry-studio/releases/download/v1.9.2/Cherry-Studio-1.9.2-x86_64.AppImage",
	},
	"cherry-studio-linux-x64-deb": {
		Slug:          "cherry-studio-linux-x64-deb",
		Product:       "Cherry Studio",
		Title:         "Linux x64 DEB",
		Version:       "v1.9.2",
		Platform:      "Linux",
		Architecture:  "x64",
		PackageFormat: "DEB",
		FileName:      "Cherry-Studio-1.9.2-amd64.deb",
		SizeBytes:     170308068,
		SourceURL:     "https://github.com/CherryHQ/cherry-studio/releases/download/v1.9.2/Cherry-Studio-1.9.2-amd64.deb",
	},
	"cherry-studio-linux-arm64-appimage": {
		Slug:          "cherry-studio-linux-arm64-appimage",
		Product:       "Cherry Studio",
		Title:         "Linux ARM64 AppImage",
		Version:       "v1.9.2",
		Platform:      "Linux",
		Architecture:  "ARM64",
		PackageFormat: "AppImage",
		FileName:      "Cherry-Studio-1.9.2-arm64.AppImage",
		SizeBytes:     217270143,
		SourceURL:     "https://github.com/CherryHQ/cherry-studio/releases/download/v1.9.2/Cherry-Studio-1.9.2-arm64.AppImage",
	},
	"cherry-studio-linux-arm64-deb": {
		Slug:          "cherry-studio-linux-arm64-deb",
		Product:       "Cherry Studio",
		Title:         "Linux ARM64 DEB",
		Version:       "v1.9.2",
		Platform:      "Linux",
		Architecture:  "ARM64",
		PackageFormat: "DEB",
		FileName:      "Cherry-Studio-1.9.2-arm64.deb",
		SizeBytes:     159616796,
		SourceURL:     "https://github.com/CherryHQ/cherry-studio/releases/download/v1.9.2/Cherry-Studio-1.9.2-arm64.deb",
	},
	"nodejs-windows-x64-msi": {
		Slug:          "nodejs-windows-x64-msi",
		Product:       "Node.js",
		Title:         "Windows x64 MSI",
		Version:       "v24.15.0",
		Platform:      "Windows",
		Architecture:  "x64",
		PackageFormat: "MSI",
		FileName:      "node-v24.15.0-x64.msi",
		SizeBytes:     32497664,
		SourceURL:     "https://nodejs.org/dist/v24.15.0/node-v24.15.0-x64.msi",
	},
	"nodejs-windows-arm64-zip": {
		Slug:          "nodejs-windows-arm64-zip",
		Product:       "Node.js",
		Title:         "Windows ARM64 ZIP",
		Version:       "v24.15.0",
		Platform:      "Windows",
		Architecture:  "ARM64",
		PackageFormat: "ZIP",
		FileName:      "node-v24.15.0-win-arm64.zip",
		SizeBytes:     32728930,
		SourceURL:     "https://nodejs.org/dist/v24.15.0/node-v24.15.0-win-arm64.zip",
	},
	"nodejs-macos-intel-pkg": {
		Slug:          "nodejs-macos-intel-pkg",
		Product:       "Node.js",
		Title:         "macOS Intel PKG",
		Version:       "v24.15.0",
		Platform:      "macOS",
		Architecture:  "Intel",
		PackageFormat: "PKG",
		FileName:      "node-v24.15.0.pkg",
		SizeBytes:     91592363,
		SourceURL:     "https://nodejs.org/dist/v24.15.0/node-v24.15.0.pkg",
	},
	"nodejs-macos-arm64-tar-gz": {
		Slug:          "nodejs-macos-arm64-tar-gz",
		Product:       "Node.js",
		Title:         "macOS Apple Silicon TAR",
		Version:       "v24.15.0",
		Platform:      "macOS",
		Architecture:  "ARM64",
		PackageFormat: "TAR.GZ",
		FileName:      "node-v24.15.0-darwin-arm64.tar.gz",
		SizeBytes:     51450940,
		SourceURL:     "https://nodejs.org/dist/v24.15.0/node-v24.15.0-darwin-arm64.tar.gz",
	},
	"nodejs-linux-x64-tar-xz": {
		Slug:          "nodejs-linux-x64-tar-xz",
		Product:       "Node.js",
		Title:         "Linux x64 TAR",
		Version:       "v24.15.0",
		Platform:      "Linux",
		Architecture:  "x64",
		PackageFormat: "TAR.XZ",
		FileName:      "node-v24.15.0-linux-x64.tar.xz",
		SizeBytes:     31164460,
		SourceURL:     "https://nodejs.org/dist/v24.15.0/node-v24.15.0-linux-x64.tar.xz",
	},
	"nodejs-linux-arm64-tar-xz": {
		Slug:          "nodejs-linux-arm64-tar-xz",
		Product:       "Node.js",
		Title:         "Linux ARM64 TAR",
		Version:       "v24.15.0",
		Platform:      "Linux",
		Architecture:  "ARM64",
		PackageFormat: "TAR.XZ",
		FileName:      "node-v24.15.0-linux-arm64.tar.xz",
		SizeBytes:     30108656,
		SourceURL:     "https://nodejs.org/dist/v24.15.0/node-v24.15.0-linux-arm64.tar.xz",
	},
}
