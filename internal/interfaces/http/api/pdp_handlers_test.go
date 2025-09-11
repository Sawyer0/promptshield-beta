package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
)

func TestPDP_System_Info_Deny(t *testing.T) {
	ts := setupPDPServer(t, func(p map[string]any) map[string]any {
		if p["action"] == "system.read" {
			return map[string]any{"decision":"DENY","reason":"nope"}
		}
		return map[string]any{"decision":"PERMIT"}
	})
	defer ts.Close()
	os.Setenv("PS_DEV_BYPASS_AUTH", "true")
	os.Setenv("PS_PDP_ENDPOINT", ts.URL)
	mux := newTestMux(t)

req := httptest.NewRequest(http.MethodGet, "/admin/system/info", nil)
	req.Header.Set("X-PS-User-Admin", "true") // satisfy adminAuth
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPDP_Rulepacks_Validate_Deny(t *testing.T) {
	ts := setupPDPServer(t, func(p map[string]any) map[string]any {
		if p["action"] == "rulepack.validate" {
			return map[string]any{"decision":"DENY","reason":"blocked"}
		}
		return map[string]any{"decision":"PERMIT"}
	})
	defer ts.Close()
	os.Setenv("PS_DEV_BYPASS_AUTH", "true")
	os.Setenv("PS_PDP_ENDPOINT", ts.URL)
	mux := newTestMux(t)

	body := []byte(`{"apiVersion":"promptshield.io/v1","kind":"RulePack","metadata":{"name":"x"}}`)
req := httptest.NewRequest(http.MethodPost, "/rulepacks/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PS-Tenant-ID", uuid.New().String())
	req.Header.Set("X-PS-User-Admin", "true")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

