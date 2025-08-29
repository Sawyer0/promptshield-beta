package redact

import (
	"regexp"
	"unicode"
)

type rule struct {
	rx       *regexp.Regexp
	verifier func(string) bool
}

var rules = []rule{
	// Credit card candidate (13-19 digits allowing spaces/dashes); verify with Luhn
	{rx: regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`), verifier: luhnValid},
	// OpenAI keys (sk- prefix, typically >= 32 chars)
	{rx: regexp.MustCompile(`(?i)\b(sk-[a-z0-9]{32,})\b`)},
	// OpenAI project-scoped keys (sk-proj- prefix)
	{rx: regexp.MustCompile(`(?i)\b(sk-proj-[a-z0-9]{24,})\b`)},
	// Generic 'key-' tokens (>= 32 chars)
	{rx: regexp.MustCompile(`(?i)\b(key-[A-Za-z0-9_-]{32,})\b`)},
	// Generic 'key=' patterns
	{rx: regexp.MustCompile(`(?i)api[_-]?key\s*[:=]\s*"?([A-Za-z0-9_\-]{16,})"?`)},
	// Common 40-char tokens (e.g., GitHub)
	{rx: regexp.MustCompile(`\b[a-zA-Z0-9]{40}\b`)},
	// AWS Access Key ID (AKIA or ASIA + 16 chars)
	{rx: regexp.MustCompile(`\b(AKIA|ASIA)[A-Z0-9]{16}\b`)},
	// AWS Secret Access Key (40 base64-like chars)
	{rx: regexp.MustCompile(`\b(?i)[A-Za-z0-9\/+]{40}\b`)},
	// AWS Session Token (X-Amz-Security-Token or AWS_SESSION_TOKEN env-like values)
	{rx: regexp.MustCompile(`(?i)(X-Amz-Security-Token|AWS_SESSION_TOKEN)\s*[:=]\s*"?([A-Za-z0-9\/+=]{16,})"?`)},
	// Google API Key (commonly 39 chars total; allow a range to reduce false negatives)
	{rx: regexp.MustCompile(`\bAIza[0-9A-Za-z\-_]{24,48}\b`)},
	// GCP Service Account private key id
	{rx: regexp.MustCompile(`(?i)"private_key_id"\s*:\s*"([a-f0-9]{16,})"`)},
	// GCP Service Account private key block
	{rx: regexp.MustCompile(`-----BEGIN PRIVATE KEY-----[\s\S]*?-----END PRIVATE KEY-----`)},
	// Azure Connection String style SharedAccessKey
	{rx: regexp.MustCompile(`(?i)SharedAccessKey=([A-Za-z0-9\/+]{40,})`)},
	// Azure SAS tokens
	{rx: regexp.MustCompile(`(?i)se=\d{10,}&sp=[rwdlacup]+&spr=https?&sv=20\d{2}-\d{2}-\d{2}&sig=[A-Za-z0-9%\/+=]{10,}`)},
	// Azure Client Secret
	{rx: regexp.MustCompile(`(?i)(client_secret|password)\s*[:=]\s*"?([A-Za-z0-9-_~.]{16,})"?`)},
	// Generic Bearer tokens
	{rx: regexp.MustCompile(`(?i)\b(bearer)\s+([A-Za-z0-9\-\._~\+\/]+=*)`)},
	// JWT (header.payload.signature)
	{rx: regexp.MustCompile(`\beyJ[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+\b`)},
	// SSH private key headers
	{rx: regexp.MustCompile(`-----BEGIN (?:RSA|DSA|EC|OPENSSH) PRIVATE KEY-----[\s\S]*?-----END (?:RSA|DSA|EC|OPENSSH) PRIVATE KEY-----`)},
	// Slack tokens
	{rx: regexp.MustCompile(`(?i)\b(xox[aboprs]-[A-Za-z0-9-]{10,})\b`)},
	// GitHub personal access tokens (ghp_, gho_, ghu_, ghs_, ghr_, or github_pat_)
	{rx: regexp.MustCompile(`\b(ghp_[A-Za-z0-9]{36}|gho_[A-Za-z0-9]{36}|ghu_[A-Za-z0-9]{36}|ghs_[A-Za-z0-9]{36}|ghr_[A-Za-z0-9]{36}|github_pat_[A-Za-z0-9_]{82})\b`)},
	// Stripe keys
	{rx: regexp.MustCompile(`\b(sk_live_[A-Za-z0-9]{24}|sk_test_[A-Za-z0-9]{24})\b`)},
}

// Redact replaces sensitive substrings with [REDACTED].
func Redact(s string) string {
	if !IsEnabled() {
		return s
	}
	out := s
	for _, r := range rules {
		out = r.rx.ReplaceAllStringFunc(out, func(m string) string {
			if r.verifier != nil && !r.verifier(m) {
				return m
			}
			return "[REDACTED]"
		})
	}
	return out
}

// RedactAndTruncate applies Redact and then truncates the string to maxBytes,
// preserving byte boundaries.
func RedactAndTruncate(s string, maxBytes int) string {
	if IsEnabled() {
		s = Redact(s)
	}
	b := []byte(s)
	if maxBytes > 0 && len(b) > maxBytes {
		return string(b[:maxBytes])
	}
	return s
}

// Global toggle for redaction in logs/audit. Defaults to enabled.
var enabled = true

// SetEnabled sets the global redaction toggle.
func SetEnabled(on bool) { enabled = on }

// IsEnabled returns current redaction toggle state.
func IsEnabled() bool { return enabled }

func luhnValid(s string) bool {
	// strip non-digits
	digits := make([]int, 0, len(s))
	for _, r := range s {
		if unicode.IsDigit(r) {
			digits = append(digits, int(r-'0'))
		}
	}
	if len(digits) < 13 || len(digits) > 19 {
		return false
	}
	// Luhn algorithm
	sum := 0
	alt := false
	for i := len(digits) - 1; i >= 0; i-- {
		d := digits[i]
		if alt {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		alt = !alt
	}
	return sum%10 == 0
}
