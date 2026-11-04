package api

import (
	"net/http"
	"os"
	"strings"
)

// userAuth enforces authentication for user-facing endpoints when configured.
// Uses PS_ENFORCER_AUTH_TOKEN for authentication.
func userAuth(_ Options) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only enforce on decision endpoints
			p := r.URL.Path
			if !(p == "/check" || p == "/scan" || strings.HasPrefix(p, "/v1/check") || strings.HasPrefix(p, "/v1/scan")) {
				next.ServeHTTP(w, r)
				return
			}
			reqToken := os.Getenv("PS_ENFORCER_AUTH_TOKEN")
			if reqToken != "" {
				if !httpAuthOK(r, reqToken) {
					writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required", nil)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
