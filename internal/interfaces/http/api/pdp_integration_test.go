package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	services "github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/shared/contracts"
)

// stubToolRunner echoes args

type stubToolRunner struct{}

func (t *stubToolRunner) Execute(ctx context.Context, req contracts.ToolExecRequest) (contracts.ToolExecResult, error) {
	return contracts.ToolExecResult{
		Result:      json.RawMessage(`{"ok":true}`),
		ContentType: "application/json",
		StartedAt:   time.Now(),
		CompletedAt: time.Now(),
	}, nil
}

func setupPDPServer(_ *testing.T, handler func(map[string]any) map[string]any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		resp := handler(payload)
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func newTestMux(_ *testing.T) http.Handler {
	// Reset PDP client (package-level) between tests
	pdpClient = nil
	pdpOnce = sync.Once{}
	opt := Options{RulepackService: &services.RulepackService{}, ToolRunner: &stubToolRunner{}}
	return NewMux(opt)
}

func TestPDP_ToolInvoke_Deny(t *testing.T) {
	ts := setupPDPServer(t, func(p map[string]any) map[string]any {
		if p["action"] == "tool.invoke" {
			return map[string]any{"decision": "DENY", "reason": "deny_tool"}
		}
		return map[string]any{"decision": "PERMIT"}
	})
	defer ts.Close()
	t.Setenv("PS_DEV_BYPASS_AUTH", "true")
	t.Setenv("PS_PDP_ENDPOINT", ts.URL)
	t.Setenv("PS_PDP_FAIL_OPEN_TOOL", "false")
	mux := newTestMux(t)

	body := []byte(`{"tool_id":"echo","args":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/tools/exec", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PS-Tenant-ID", uuid.New().String())
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestPDP_ToolInvoke_Permit(t *testing.T) {
	ts := setupPDPServer(t, func(p map[string]any) map[string]any { return map[string]any{"decision": "PERMIT"} })
	defer ts.Close()
	t.Setenv("PS_DEV_BYPASS_AUTH", "true")
	t.Setenv("PS_PDP_ENDPOINT", ts.URL)
	t.Setenv("PS_PDP_FAIL_OPEN_TOOL", "")
	mux := newTestMux(t)

	body := []byte(`{"tool_id":"echo","args":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/tools/exec", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PS-Tenant-ID", uuid.New().String())
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPDP_Check_Deny(t *testing.T) {
	ts := setupPDPServer(t, func(p map[string]any) map[string]any {
		if p["action"] == "message.send" {
			return map[string]any{"decision": "DENY", "reason": "deny_msg"}
		}
		return map[string]any{"decision": "PERMIT"}
	})
	defer ts.Close()
	t.Setenv("PS_DEV_BYPASS_AUTH", "true")
	t.Setenv("PS_PDP_ENDPOINT", ts.URL)
	t.Setenv("PS_PDP_FAIL_OPEN_CHECK", "") // default fail-closed
	mux := newTestMux(t)

	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewReader([]byte(`"hello"`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PS-Tenant-ID", uuid.New().String())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}
