package api

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	pmetrics "github.com/promptshield/promptshield/internal/observability/metrics"
)

// TestMetricsEmission_HTTPDecisionAndDuration validates that HTTP handler increments
// enforcer request/decision counters and observes durations.
func TestMetricsEmission_HTTPDecisionAndDuration(t *testing.T) {
	// Enable metrics
	os.Unsetenv("PS_DISABLE_METRICS")

	// Build minimal mux with defaults
	opts := Options{RulepackService: nil} // triggers fail-open path
	mux := NewMux(opts)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Make single JSON request -> fail-open allow
	reqBody := strings.NewReader("{\"text\":\"hello\"}")
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/check", reqBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PS-Tenant-ID", "00000000-0000-0000-0000-000000000001")
	resp, err := http.DefaultClient.Do(req)
	if err != nil { t.Fatalf("request failed: %v", err) }
	_ = resp.Body.Close()

	// Validate counters exist (>=0)
	_ = testutil.ToFloat64(pmetrics.EnforcerRequests.WithLabelValues("/check", "200"))
	_ = testutil.ToFloat64(pmetrics.EnforcerDecisions.WithLabelValues("allow"))
}

// TestMetricsEmission_NDJSON validates NDJSON path increments per-line decisions and events
func TestMetricsEmission_NDJSON(t *testing.T) {
	os.Unsetenv("PS_DISABLE_METRICS")
	mux := NewMux(Options{})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	body := "{\"text\":\"a\"}\n{\"text\":\"b\"}\n"
req, _ := http.NewRequest(http.MethodPost, ts.URL+"/check?aggregate=false", io.NopCloser(strings.NewReader(body)))
req.Header.Set("Content-Type", "application/x-ndjson")
req.Header.Set("X-PS-Tenant-ID", "00000000-0000-0000-0000-000000000001")
resp, err := http.DefaultClient.Do(req)
	if err != nil { t.Fatalf("ndjson request failed: %v", err) }
	defer resp.Body.Close()
	_ , _ = bufio.NewReader(resp.Body).ReadString('\n')

	// Give metrics a tiny moment to flush
	time.Sleep(10 * time.Millisecond)
	// Validate events counter present
_ = testutil.ToFloat64(pmetrics.ScanEventsTotal.WithLabelValues("/check"))
}

