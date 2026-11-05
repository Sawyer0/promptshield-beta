package enforcerhttp

import (
	"net/http"
	"strings"
)

// exported for reuse in api
func HttpAuthOK(r *http.Request, want string) bool {
	if want == "" {
		return true
	}
	// Authorization: Bearer <token>
	if v := r.Header.Get("Authorization"); v != "" {
		if len(v) >= 7 && (strings.HasPrefix(v, "Bearer ") || strings.HasPrefix(v, "bearer ")) {
			if v[7:] == want {
				return true
			}
		}
		if v == want {
			return true
		}
	}
	if v := r.Header.Get("X-PS-Token"); v != "" {
		if v == want {
			return true
		}
	}
	return false
}
