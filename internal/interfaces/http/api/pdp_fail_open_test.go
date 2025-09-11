package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	services "github.com/promptshield/promptshield/internal/application/services"
)

// PDP server that returns 500 to simulate upstream error
func setupPDPServer500(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
}

func TestCheck_PDPError_FailOpen_Allows(t *testing.T) {
	ts := setupPDPServer500(t)
	defer ts.Close()
	t.Setenv("PS_DEV_BYPASS_AUTH", "true")
	t.Setenv("PS_PDP_ENDPOINT", ts.URL)
	t.Setenv("PS_PDP_FAIL_OPEN_CHECK", "true") // fail-open
	mux := newTestMux(t)

	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewReader([]byte(`"hi"`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PS-Tenant-ID", uuid.New().String())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 when fail-open on PDP error, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCheck_PDPError_FailClosed_Denies(t *testing.T) {
	ts := setupPDPServer500(t)
	defer ts.Close()
	t.Setenv("PS_DEV_BYPASS_AUTH", "true")
	t.Setenv("PS_PDP_ENDPOINT", ts.URL)
	t.Setenv("PS_PDP_FAIL_OPEN_CHECK", "") // default fail-closed
	mux := newTestMux(t)

	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewReader([]byte(`"hi"`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PS-Tenant-ID", uuid.New().String())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when fail-closed on PDP error, got %d", rec.Code)
	}
}

func TestToolInvoke_PDPError_FailOpen_Allows(t *testing.T) {
	ts := setupPDPServer500(t)
	defer ts.Close()
	t.Setenv("PS_DEV_BYPASS_AUTH", "true")
	t.Setenv("PS_PDP_ENDPOINT", ts.URL)
	t.Setenv("PS_PDP_FAIL_OPEN_TOOL", "true") // fail-open for tools
	mux := NewMux(Options{RulepackService: &services.RulepackService{}, ToolRunner: &stubToolRunner{}})

	body := []byte(`{"tool_id":"echo","args":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/tools/exec", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PS-Tenant-ID", uuid.New().String())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 when fail-open tool PDP error, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestToolInvoke_PDPError_FailClosed_Denies(t *testing.T) {
	ts := setupPDPServer500(t)
	defer ts.Close()
	t.Setenv("PS_DEV_BYPASS_AUTH", "true")
	t.Setenv("PS_PDP_ENDPOINT", ts.URL)
	t.Setenv("PS_PDP_FAIL_OPEN_TOOL", "") // default fail-closed
	mux := NewMux(Options{RulepackService: &services.RulepackService{}, ToolRunner: &stubToolRunner{}})

	body := []byte(`{"tool_id":"echo","args":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/tools/exec", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PS-Tenant-ID", uuid.New().String())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when fail-closed tool PDP error, got %d body=%s", rec.Code, rec.Body.String())
	}
}
