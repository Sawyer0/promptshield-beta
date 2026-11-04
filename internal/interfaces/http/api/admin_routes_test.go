package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/testutil/mocks"
)

// TestAdminRoutesServeUnderV1 mounts NewMux at /v1 the way the enforcer does
// and checks the tenant, assignment, audit and system admin routes answer at
// /v1/admin/... and not at a doubled /v1/v1/admin/... prefix.
func TestAdminRoutesServeUnderV1(t *testing.T) {
	const token = "test-admin-token"
	root := chi.NewRouter()
	root.Mount("/v1", NewMux(Options{
		AdminToken:      token,
		RulepackService: services.RulepackServiceCstor(&mocks.MockRulepackRepository{}, nil),
	}))

	do := func(method, path string) int {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		root.ServeHTTP(rr, req)
		return rr.Code
	}

	if got := do(http.MethodGet, "/v1/admin/system/features"); got != http.StatusOK {
		t.Errorf("GET /v1/admin/system/features = %d, want %d", got, http.StatusOK)
	}

	// No repositories are configured, so these reach their handlers and answer
	// 501 rather than falling through to a 404.
	for _, p := range []string{
		"/v1/admin/tenants",
		"/v1/admin/tenants/00000000-0000-0000-0000-000000000001/assignments",
		"/v1/admin/audits",
	} {
		if got := do(http.MethodGet, p); got != http.StatusNotImplemented {
			t.Errorf("GET %s = %d, want %d", p, got, http.StatusNotImplemented)
		}
	}

	if got := do(http.MethodGet, "/v1/v1/admin/system/features"); got != http.StatusNotFound {
		t.Errorf("GET /v1/v1/admin/system/features = %d, want %d", got, http.StatusNotFound)
	}
}
