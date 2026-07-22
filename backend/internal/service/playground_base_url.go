package service

import (
	"net"
	"net/url"
	"os"
	"strings"
)

func playgroundInternalBaseURL(preferred string) string {
	if baseURL := normalizePlaygroundBaseURL(preferred); baseURL != "" {
		return baseURL
	}
	host := normalizePlaygroundLoopbackHost(strings.TrimSpace(os.Getenv("SERVER_HOST")))
	port := strings.TrimSpace(os.Getenv("SERVER_PORT"))
	if port == "" {
		port = "8080"
	}
	return normalizePlaygroundBaseURL("http://" + net.JoinHostPort(host, port))
}

func resolvePlaygroundAssetURL(raw, baseURL string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") || strings.HasPrefix(trimmed, "data:") {
		return trimmed
	}
	if strings.HasPrefix(trimmed, "/") {
		return strings.TrimRight(playgroundInternalBaseURL(baseURL), "/") + trimmed
	}
	return trimmed
}

func normalizePlaygroundBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	candidate := raw
	if !strings.Contains(candidate, "://") {
		candidate = "http://" + candidate
	}
	parsed, err := url.Parse(candidate)
	if err != nil {
		return ""
	}
	scheme := strings.TrimSpace(strings.ToLower(parsed.Scheme))
	if scheme == "" {
		scheme = "http"
	}
	host := normalizePlaygroundLoopbackHost(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return ""
	}
	port := strings.TrimSpace(parsed.Port())
	if port == "" {
		return scheme + "://" + host
	}
	return scheme + "://" + net.JoinHostPort(host, port)
}

func normalizePlaygroundLoopbackHost(host string) string {
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	switch host {
	case "", "0.0.0.0", "::":
		return "127.0.0.1"
	default:
		return host
	}
}
