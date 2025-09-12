package api

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/promptshield/promptshield/internal/observability/metrics"
)

// adminAuth enforces simple admin authentication for ops endpoints only
func adminAuth(opt Options) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prefer JWT-based admin via roles injected by jwtAuthMiddleware
			if roles := r.Header.Get("X-PS-User-Roles"); roles != "" {
				for _, part := range strings.Split(roles, ",") {
					if strings.TrimSpace(strings.ToLower(part)) == "admin" {
						next.ServeHTTP(w, r)
						return
					}
				}
			}
			if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-PS-User-Admin")), "true") {
				next.ServeHTTP(w, r)
				return
			}

			// Fallback: static admin token for ops endpoints
			if opt.AdminToken != "" {
				tok := r.Header.Get("Authorization")
				if strings.HasPrefix(strings.ToLower(tok), "bearer ") {
					tok = tok[7:]
				}
				if tok == "" {
					tok = r.Header.Get("X-PS-Admin-Token")
				}
				if tok == opt.AdminToken {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Development mode (when explicitly enabled)
			if opt.AllowInsecureAdmin {
				slog.Warn("INSECURE: Admin endpoints accessible without authentication")
				next.ServeHTTP(w, r)
				return
			}

			// No valid authentication found
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "admin authentication required", nil)
		})
	}
}

// bytesCounter wraps ResponseWriter to capture bytes written.
type bytesCounter struct {
	http.ResponseWriter
	path       string
	method     string
	wroteBytes int64
}

func (b *bytesCounter) Write(p []byte) (int, error) {
	n, err := b.ResponseWriter.Write(p)
	if n > 0 {
		b.wroteBytes += int64(n)
		metrics.HTTPBytesTotal.WithLabelValues("out", b.method, b.path).Add(float64(n))
	}
	return n, err
}

// Flush proxies flush to the underlying writer when supported.
func (b *bytesCounter) Flush() {
	if f, ok := b.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// captureBytesMiddleware accounts for request and response bytes regardless of Content-Length.
func captureBytesMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wrap body to count bytes read
		var reader io.ReadCloser = r.Body
		if reader != nil {
			r.Body = struct {
				io.Reader
				io.Closer
			}{Reader: io.TeeReader(reader, countWriterFunc(func(n int) {
				if n > 0 {
					metrics.HTTPBytesTotal.WithLabelValues("in", r.Method, normalizePath(r.URL.Path)).Add(float64(n))
				}
			})), Closer: reader}
		}
		bw := &bytesCounter{ResponseWriter: w, path: normalizePath(r.URL.Path), method: r.Method}
		next.ServeHTTP(bw, r)
	})
}

// Path normalization to reduce label cardinality.
// Replaces UUIDs, purely numeric IDs, and long opaque segments with placeholders.
var (
	uuidRe    = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	numericRe = regexp.MustCompile(`/[0-9]+`)
	longSegRe = regexp.MustCompile(`/[A-Za-z0-9_-]{16,}`)
)

func normalizePath(p string) string {
	if p == "" || p == "/" {
		return "/"
	}
	// Opt-out: allow raw paths (could increase cardinality significantly)
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("PS_GATEWAY_RAW_PATHS"))); v == "1" || v == "true" || v == "yes" {
		return p
	}
	// Opt-out for specific prefixes: comma-separated list
	if pref := strings.TrimSpace(os.Getenv("PS_GATEWAY_RAW_PREFIXES")); pref != "" {
		for _, one := range strings.Split(pref, ",") {
			one = strings.TrimSpace(one)
			if one == "" {
				continue
			}
			if strings.HasPrefix(p, one) {
				return p
			}
		}
	}
	// Do not touch metrics or health endpoints
	if p == "/metrics" || p == "/healthz" || p == "/readyz" {
		return p
	}
	s := uuidRe.ReplaceAllString(p, ":uuid")
	s = numericRe.ReplaceAllString(s, "/:id")
	s = longSegRe.ReplaceAllString(s, "/:token")
	// Collapse duplicate slashes if any replacements created them
	for strings.Contains(s, "//") {
		s = strings.ReplaceAll(s, "//", "/")
	}
	return s
}

// countWriterFunc adapts a callback to io.Writer for counting purposes.
type countWriterFunc func(n int)

func (f countWriterFunc) Write(p []byte) (int, error) { f(len(p)); return len(p), nil }

// tenantQuota enforces per-tenant rate limits using Options.QuotaStore when available.
func tenantQuota(opt Options) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if opt.QuotaStore == nil {
				next.ServeHTTP(w, r)
				return
			}
			tenant := tenantFromRequest(r)
			if tenant == "" {
				tenant = "_default"
			}
			if !opt.QuotaStore.Allow(tenant) {
				writeError(w, http.StatusTooManyRequests, "RESOURCE_EXHAUSTED", "rate limit exceeded", map[string]any{"tenant": tenant})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
