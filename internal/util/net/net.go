package net

import (
	"net/url"
	"strings"

	stringutil "github.com/promptshield/promptshield/internal/util/strings"
)

// SanitizeEndpoint sanitizes an endpoint URL to extract just the host
func SanitizeEndpoint(s string) string {
	if stringutil.IsBlank(s) {
		return ""
	}
	
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		// If parsing fails or no host, try to clean the string
		return strings.TrimPrefix(strings.TrimPrefix(s, "http://"), "https://")
	}
	return u.Host
}