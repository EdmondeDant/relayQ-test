package apicompat

import (
	"net"
	"net/netip"
	"net/url"
	"strings"
)

// IsPotentiallyUnsafeRemoteMediaURL performs a cheap preflight check for media
// URLs before any server-side fetch fallback is implemented. It intentionally
// avoids DNS resolution; callers that fetch remote media must also enforce DNS
// rebinding protections at fetch time.
func IsPotentiallyUnsafeRemoteMediaURL(rawURL string) bool {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u == nil {
		return true
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return true
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return true
	}
	return isLocalOrPrivateMediaHost(host)
}

func isLocalOrPrivateMediaHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(strings.Trim(host, "[]")))
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		parsed, ok := netip.AddrFromSlice(ip)
		if !ok {
			return true
		}
		return parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsLinkLocalUnicast() || parsed.IsLinkLocalMulticast() || parsed.IsUnspecified()
	}
	return false
}
