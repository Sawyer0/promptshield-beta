package main

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/interfaces/http/api"
	enforcerhttp "github.com/promptshield/promptshield/internal/interfaces/http/enforcer"
	"github.com/promptshield/promptshield/internal/testutil/mocks"
)

// TestHTTPCheck_SLA enforces a minimal throughput SLA for the /check endpoint.
// Skipped unless PS_ENFORCE_SLA=1 to avoid CI flakiness across machines.
func TestHTTPCheck_SLA(t *testing.T) {
	if os.Getenv("PS_ENFORCE_SLA") != "1" {
		t.Skip("set PS_ENFORCE_SLA=1 to enforce gateway HTTP SLA")
	}
	// Default thresholds (override via env): 10 MB/s minimum
	minMBps := 10.0
	if v := os.Getenv("PS_SLA_HTTP_MBPS_MIN"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			minMBps = f
		}
	}
	// Bypass evaluation rate limit by setting a dummy license token
	payload := `{"org":"SLA","tier":"enterprise","expires_at":"2099-01-01T00:00:00Z","entitlements":{}}`
	enc := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString([]byte("sig"))
	t.Setenv("PROMPTSHIELD_LICENSE_KEY", enc)

	// Setup with mock RulepackService
	mockRepo := &mocks.MockRulepackRepository{}
	rulepackService := services.RulepackServiceCstor(mockRepo, nil)
	
	options := api.Options{
		RulepackService: rulepackService,
	}
	h := enforcerhttp.NewMuxWithOptions(options)
	ts := httptest.NewServer(h)
	defer ts.Close()

	const n = 50
	body := bytes.Repeat([]byte("x"), 64*1024) // 64KB
	totalBytes := float64(len(body) * n)
	start := time.Now()
	for i := 0; i < n; i++ {
		resp, err := http.Post(ts.URL+"/check", "text/plain", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("POST /check error: %v", err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusForbidden {
			t.Fatalf("unexpected status: %d", resp.StatusCode)
		}
	}
	dur := time.Since(start).Seconds()
	mbps := (totalBytes / (1024 * 1024)) / dur
	if mbps < minMBps {
		t.Fatalf("/check throughput %.2f MB/s below SLA (min %.2f MB/s)", mbps, minMBps)
	}
}
