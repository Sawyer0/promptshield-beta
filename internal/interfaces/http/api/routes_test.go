package api

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/promptshield/promptshield/internal/application/services"
)

// Test_AllRoutesRespond ensures that every registered route pattern responds to
// an OPTIONS request with a non-404 status, i.e. the router has a handler path
// for it. A 401/403 is acceptable because auth middleware may block the
// request, but a 404 indicates a missing handler or mis-registered path.
func Test_AllRoutesRespond(t *testing.T) {
	// Provide a dummy RulepackService to satisfy router constructor. The zero
	// value is safe because OPTIONS requests do not invoke business logic.
	dummySvc := &services.RulepackService{}
	h := NewMux(Options{RulepackService: dummySvc})

	// chi.Walk needs a chi.Routes implementation, not just http.Handler.
	mx, ok := h.(chi.Routes)
	if !ok {
		t.Fatalf("router does not implement chi.Routes")
	}

	// Collect all route patterns via chi.Walk.
	var patterns []string
	_ = chi.Walk(mx, func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		// Collect only one instance per path pattern; method is irrelevant for OPTIONS.
		for _, p := range patterns {
			if p == route {
				return nil
			}
		}
		patterns = append(patterns, route)
		return nil
	})

	// Replace any path params like {id} or {jobID:[a-z]+} with dummy value "test".
	re := regexp.MustCompile(`\{[^/]+?\}`) // matches {param} or {param:regex}

	for _, pattern := range patterns {
		path := re.ReplaceAllString(pattern, "test")
		// chi uses * and /*catchall for wildcards – skip those in this check.
		if path == "*" || path == "/*" || regexp.MustCompile(`[*]`).MatchString(path) {
			continue
		}
		req := httptest.NewRequest(http.MethodOptions, path, nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code == http.StatusNotFound {
			t.Errorf("route %s returned 404 to OPTIONS", pattern)
		}
	}

	// Sanity: ensure we checked at least some routes.
	require.Greater(t, len(patterns), 0, "no routes discovered in router walk")
}
