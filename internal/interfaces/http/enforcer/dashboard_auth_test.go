package enforcerhttp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestDashboardOnByDefaultWithDataBehindAdminAuth pins the default: the
// static dashboard is mounted, and every data endpoint it calls answers 401
// without the admin token, including when no admin token is configured.
func TestDashboardOnByDefaultWithDataBehindAdminAuth(t *testing.T) {
	endpoints := []struct{ method, path string }{
		{http.MethodGet, "/v1/decisions"},
		{http.MethodGet, "/v1/decisions/stats"},
		{http.MethodGet, "/v1/decisions/some-id"},
		{http.MethodGet, "/v1/rules"},
		{http.MethodPost, "/v1/rules/reload"},
		{http.MethodGet, "/v1/events"},
	}
	for _, token := range []string{"configured-token", ""} {
		t.Setenv("PS_ENFORCER_ADMIN_TOKEN", token)
		t.Setenv("PS_ENFORCER_ALLOW_INSECURE_ADMIN", "")
		t.Setenv("PS_DASHBOARD_DISABLED", "")
		srv := httptest.NewServer(NewMux())

		resp, err := http.Get(srv.URL + "/dashboard/")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			t.Fatalf("token %q: dashboard not mounted by default", token)
		}

		for _, ep := range endpoints {
			req, _ := http.NewRequest(ep.method, srv.URL+ep.path, nil)
			req.Header.Set("Authorization", "Bearer guess")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("token %q: %s %s without the admin token = %d, want 401", token, ep.method, ep.path, resp.StatusCode)
			}
		}
		srv.Close()
	}
}
