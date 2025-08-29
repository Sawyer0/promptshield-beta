package api

import (
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/promptshield/promptshield/internal/observability/metrics"
)

// adminAuth enforces simple admin authentication for ops endpoints only
func adminAuth(opt Options) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simple admin token for ops endpoints (/metrics, /health, /admin/*)
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
	wroteBytes int64
}

func (b *bytesCounter) Write(p []byte) (int, error) {
	n, err := b.ResponseWriter.Write(p)
	if n > 0 {
		b.wroteBytes += int64(n)
		metrics.HTTPBytesTotal.WithLabelValues("out", b.path).Add(float64(n))
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
					metrics.HTTPBytesTotal.WithLabelValues("in", r.URL.Path).Add(float64(n))
				}
			})), Closer: reader}
		}
		bw := &bytesCounter{ResponseWriter: w, path: r.URL.Path}
		next.ServeHTTP(bw, r)
	})
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
