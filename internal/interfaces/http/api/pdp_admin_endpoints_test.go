package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"sync"

	services "github.com/promptshield/promptshield/internal/application/services"
	memrepo "github.com/promptshield/promptshield/internal/infrastructure/persistence/memory"
)

func newAdminMux(t *testing.T) http.Handler {
	// Reset PDP client and build mux with in-memory RulepackService
	pdpClient = nil
	pdpOnce = sync.Once{}
	svc := services.RulepackServiceCstor(memrepo.NewRulepackRepository(), nil)
	return NewMux(Options{RulepackService: svc})
}

func TestAdminEndpoints_PDP_DenyAndPermit(t *testing.T) {
	// Deny server: denies system.*
	tsDeny := setupPDPServer(t, func(p map[string]any) map[string]any {
		if act, _ := p["action"].(string); act == "system.read" || act == "system.drain" || act == "system.shutdown" {
			return map[string]any{"decision":"DENY","reason":"blocked"}
		}
		return map[string]any{"decision":"PERMIT"}
	})
	defer tsDeny.Close()

	// Permit server: permits all
	tsPermit := setupPDPServer(t, func(p map[string]any) map[string]any { return map[string]any{"decision":"PERMIT"} })
	defer tsPermit.Close()

	// Common env: dev bypass
	t.Setenv("PS_DEV_BYPASS_AUTH", "true")

	// 1) Deny
	t.Setenv("PS_PDP_ENDPOINT", tsDeny.URL)
	mux := newAdminMux(t)
	denyCases := []struct{
		name string
		method string
		url string
		expected int
	}{
		{"features_deny", http.MethodGet, "/admin/system/features", http.StatusForbidden},
		{"stats_deny", http.MethodGet, "/admin/system/stats", http.StatusForbidden},
		{"info_deny", http.MethodGet, "/admin/system/info", http.StatusForbidden},
		{"drain_deny", http.MethodPost, "/admin/system/drain", http.StatusForbidden},
		{"shutdown_deny", http.MethodPost, "/admin/system/shutdown", http.StatusForbidden},
	}
	for _, tc := range denyCases {
		req := httptest.NewRequest(tc.method, tc.url, nil)
		req.Header.Set("X-PS-User-Admin", "true")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != tc.expected {
			t.Fatalf("%s: expected %d, got %d body=%s", tc.name, tc.expected, rec.Code, rec.Body.String())
		}
	}

	// 2) Permit
	t.Setenv("PS_PDP_ENDPOINT", tsPermit.URL)
	// reload PDP client in existing mux
	reqReload := httptest.NewRequest(http.MethodPost, "/admin/pdp/reload", nil)
	reqReload.Header.Set("X-PS-User-Admin", "true")
	recReload := httptest.NewRecorder()
	mux.ServeHTTP(recReload, reqReload)
	if recReload.Code != http.StatusNoContent {
		t.Fatalf("pdp reload: expected 204, got %d body=%s", recReload.Code, recReload.Body.String())
	}

	permitCases := []struct{
		name string
		method string
		url string
		expected int
	}{
		{"features_ok", http.MethodGet, "/admin/system/features", http.StatusOK},
		{"stats_ok", http.MethodGet, "/admin/system/stats", http.StatusOK},
		{"info_ok", http.MethodGet, "/admin/system/info", http.StatusOK},
		{"drain_ok", http.MethodPost, "/admin/system/drain", http.StatusAccepted},
		{"shutdown_ok", http.MethodPost, "/admin/system/shutdown", http.StatusAccepted},
	}
	for _, tc := range permitCases {
		req := httptest.NewRequest(tc.method, tc.url, nil)
		req.Header.Set("X-PS-User-Admin", "true")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != tc.expected {
			t.Fatalf("%s: expected %d, got %d body=%s", tc.name, tc.expected, rec.Code, rec.Body.String())
		}
	}
}
