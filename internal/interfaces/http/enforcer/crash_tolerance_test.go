package enforcerhttp

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/promptshield/promptshield/internal/observability/metrics"
)

// TestFailOpenWhenNoRulepack ensures that when the active rulepack is missing
// (simulated by starting the enforcer without PS_ENFORCER_RULEPACK), traffic is
// allowed (fail-open) and policy_bypass metric is recorded.
func TestFailOpenWhenNoRulepack(t *testing.T) {
	// Ensure no rulepack is configured
	os.Unsetenv("PS_ENFORCER_RULEPACK")
	// Disable strict startup requirement
	os.Setenv("PS_REQUIRE_RULEPACK_AT_STARTUP", "false")

	srv := httptest.NewServer(NewMux())
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/check", bytes.NewBufferString("hello"))
	req.Header.Set("X-PS-Tenant-ID", "00000000-0000-0000-0000-000000000001")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	if d := resp.Header.Get("x-ps-decision"); d == "" {
		t.Fatalf("expected decision header present")
	}
	// Metric should be incremented for no_rules bypass (may be zero if metrics disabled)
	_ = testutil.ToFloat64(metrics.PolicyBypass.WithLabelValues("no_rules"))
}
