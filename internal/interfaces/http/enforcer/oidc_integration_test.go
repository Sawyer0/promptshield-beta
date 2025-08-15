package enforcerhttp

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOIDCIntegration(t *testing.T) {
	// Test that OIDC configuration is properly loaded from environment
	t.Setenv("PS_ENFORCER_OIDC_ISSUER", "https://example.com")
	t.Setenv("PS_ENFORCER_OIDC_AUDIENCE", "test-audience")
	t.Setenv("PS_ENFORCER_AUTH_TOKEN", "") // Disable token auth to test OIDC only

	options := getAPIOptions()

	if options.OIDC.Issuer != "https://example.com" {
		t.Errorf("expected OIDC issuer 'https://example.com', got '%s'", options.OIDC.Issuer)
	}

	if options.OIDC.Audience != "test-audience" {
		t.Errorf("expected OIDC audience 'test-audience', got '%s'", options.OIDC.Audience)
	}

	// Test that handler is created without errors
	handler := NewMuxWithOptions(options)
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

	// Test server creation with OIDC config
	srv := httptest.NewServer(handler)
	defer srv.Close()

	// Test request without authentication (should fail when OIDC is configured)
	req, _ := http.NewRequest("POST", srv.URL+"/v1/check", bytes.NewBufferString("test"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// Should return 401 when OIDC is configured but no valid JWT is provided
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestOIDCDisabled(t *testing.T) {
	// Test with OIDC disabled (no issuer set)
	t.Setenv("PS_ENFORCER_OIDC_ISSUER", "")
	t.Setenv("PS_ENFORCER_AUTH_TOKEN", "") // Also disable token auth

	options := getAPIOptions()

	if options.OIDC.Issuer != "" {
		t.Errorf("expected empty OIDC issuer, got '%s'", options.OIDC.Issuer)
	}

	handler := NewMuxWithOptions(options)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	// Test request without authentication (should succeed when no auth is configured)
	req, _ := http.NewRequest("POST", srv.URL+"/v1/check", bytes.NewBufferString("test"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// Should succeed when no authentication is configured
	if resp.StatusCode == http.StatusUnauthorized {
		t.Errorf("unexpected 401 status when no auth is configured")
	}
}

func TestBackwardCompatibility(t *testing.T) {
	// Test that NewMux() still works for backward compatibility
	t.Setenv("PS_ENFORCER_ADMIN_TOKEN", "test-admin-token")
	t.Setenv("PS_ENFORCER_OIDC_ISSUER", "")

	handler := NewMux()
	if handler == nil {
		t.Fatal("expected non-nil handler from NewMux()")
	}

	// Verify admin token is properly set
	srv := httptest.NewServer(handler)
	defer srv.Close()

	// Test admin endpoint with correct token
	req, _ := http.NewRequest("GET", srv.URL+"/v1/license", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// Admin endpoints require different handling, but shouldn't return auth error with correct token
	if resp.StatusCode == http.StatusUnauthorized {
		t.Errorf("unexpected 401 with correct admin token")
	}
}
