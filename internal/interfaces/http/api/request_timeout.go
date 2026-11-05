package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// RequestTimeout bounds every request at d, except the decision event stream
// (GET /events, GET /v1/events), which stays open until the client leaves.
func RequestTimeout(d time.Duration) func(http.Handler) http.Handler {
	limit := middleware.Timeout(d)
	return func(next http.Handler) http.Handler {
		bounded := limit(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if IsEventStream(r) {
				next.ServeHTTP(w, r)
				return
			}
			bounded.ServeHTTP(w, r)
		})
	}
}

// IsEventStream reports whether r opens the long-lived SSE decision stream.
func IsEventStream(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}
	return r.URL.Path == "/events" || r.URL.Path == "/v1/events"
}
