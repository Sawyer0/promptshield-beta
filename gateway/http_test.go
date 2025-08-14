package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	enforcerhttp "github.com/promptshield/promptshield/internal/interfaces/http/enforcer"
)

func TestHealthzAndMetrics(t *testing.T) {
	// Ensure readiness does not fail due to missing rules by setting a dummy env
	t.Setenv("PS_ENFORCER_RULEPACK", "dummy")

	h := enforcerhttp.NewMux()
	ts := httptest.NewServer(h)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/healthz status = %d", resp.StatusCode)
	}

	resp2, err := http.Get(ts.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics error: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("/metrics status = %d", resp2.StatusCode)
	}

	// Unset to avoid leaking to other tests
	_ = os.Unsetenv("PS_ENFORCER_RULEPACK")
}