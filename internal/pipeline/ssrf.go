package pipeline

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidatePublicHTTPURL rejects non-http(s) schemes and private/loopback hosts.
func ValidatePublicHTTPURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return Permanent("SSRF_INVALID_URL", fmt.Sprintf("invalid url: %v", err))
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return Permanent("SSRF_BAD_SCHEME", fmt.Sprintf("unsupported url scheme %q", u.Scheme))
	}
	host := u.Hostname()
	if host == "" {
		return Permanent("SSRF_NO_HOST", "url host is required")
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return Permanent("SSRF_BLOCKED_IP", "url host resolves to a blocked address")
		}
		return nil
	}
	addrs, err := net.LookupIP(host)
	if err != nil {
		return Retryable("SSRF_DNS", fmt.Sprintf("dns lookup failed for %s: %v", host, err))
	}
	if len(addrs) == 0 {
		return Permanent("SSRF_DNS", fmt.Sprintf("dns lookup returned no addresses for %s", host))
	}
	for _, ip := range addrs {
		if isBlockedIP(ip) {
			return Permanent("SSRF_BLOCKED_IP", "url host resolves to a blocked address")
		}
	}
	return nil
}

func isBlockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
	}
	return false
}
