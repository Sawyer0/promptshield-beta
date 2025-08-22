package enforcerhttp

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
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

	resp, err := http.Post(srv.URL+"/check", "text/plain", bytes.NewBufferString("hello"))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	if d := resp.Header.Get("x-ps-decision"); d != "allow" {
		t.Fatalf("expected decision=allow, got %s", d)
	}
	// Metric should be incremented for no_rules bypass
	if v := testutil.ToFloat64(policyBypass.WithLabelValues("no_rules")); v < 1 {
		t.Fatalf("expected policyBypass no_rules counter >=1, got %f", v)
	}
}
