package enforcerhttp

import (
	"net"
	"os"
	"strings"
)

// tlsMode returns one of: auto (default), require, disable
func tlsMode() string {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_TLS_MODE")))
	if mode == "require" || mode == "disable" {
		return mode
	}
	return "auto"
}

// isLoopbackAddr returns true if the provided listen address binds only to loopback
func isLoopbackAddr(addr string) bool {
	host := addr
	if strings.HasPrefix(host, ":") {
		// :port means all interfaces
		return false
	}
	if h, _, err := net.SplitHostPort(addr); err == nil {
		host = h
	}
	host = strings.TrimSpace(host)
	if host == "" || host == "0.0.0.0" || host == "::" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip != nil {
		return ip.IsLoopback()
	}
	// Unknown hostname: assume non-loopback to be safe
	return false
}

// findTLSPair returns cert and key file paths if configured or found in default mounts
func findTLSPair() (string, string, bool) {
	certFile := strings.TrimSpace(os.Getenv("PS_ENFORCER_TLS_CERT"))
	keyFile := strings.TrimSpace(os.Getenv("PS_ENFORCER_TLS_KEY"))
	if certFile != "" && keyFile != "" {
		return certFile, keyFile, true
	}
	// Auto-detect common mount locations
	defaults := [][2]string{
		{"/tls/server.crt", "/tls/server.key"},
		{"tls/server.crt", "tls/server.key"},
	}
	for _, p := range defaults {
		if _, err1 := os.Stat(p[0]); err1 == nil {
			if _, err2 := os.Stat(p[1]); err2 == nil {
				return p[0], p[1], true
			}
		}
	}
	return "", "", false
}
