package main

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/interfaces/http/api"
	enforcerhttp "github.com/promptshield/promptshield/internal/interfaces/http/enforcer"
	"github.com/promptshield/promptshield/internal/testutil/mocks"
)

// TestHTTP_SustainedRPS measures sustained RPS of /check under concurrency.
// To enforce an assertion for 12k RPS, set PS_ENFORCE_RPS=1.
func TestHTTP_SustainedRPS(t *testing.T) {
	// Bypass evaluation limiter with a dummy enterprise license
	payload := `{"org":"Perf","tier":"enterprise","expires_at":"2099-01-01T00:00:00Z","entitlements":{}}`
	enc := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString([]byte("sig"))
	os.Setenv("PROMPTSHIELD_LICENSE_KEY", enc)

	// Start in-memory HTTP server with mock RulepackService
	mockRepo := &mocks.MockRulepackRepository{}
	rulepackService := services.NewRulepackService(mockRepo, nil)
	
	options := api.Options{
		RulepackService: rulepackService,
	}
	h := enforcerhttp.NewMuxWithOptions(options)
	ts := httptest.NewServer(h)
	defer ts.Close()

	// High-throughput HTTP client with connection reuse
	tr := &http.Transport{
		MaxIdleConns:        32768,
		MaxIdleConnsPerHost: 32768,
		IdleConnTimeout:     30 * time.Second,
	}
	client := &http.Client{Transport: tr}

	// Load generator settings (overridable via env)
	concurrency := 256
	if v := os.Getenv("PS_RPS_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			concurrency = n
		}
	}
	duration := 3 * time.Second
	if v := os.Getenv("PS_RPS_DURATION"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			duration = time.Duration(n) * time.Second
		}
	}
	body := bytes.NewBufferString("x\n") // small body to stress handler path

	var total uint64
	var wg sync.WaitGroup
	stop := time.Now().Add(duration)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(stop) {
				resp, err := client.Post(ts.URL+"/check", "text/plain", bytes.NewReader(body.Bytes()))
				if err == nil {
					_ = resp.Body.Close()
					if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusForbidden {
						atomic.AddUint64(&total, 1)
					}
				}
			}
		}()
	}
	wg.Wait()

	// Compute RPS
	rps := float64(total) / duration.Seconds()
	t.Logf("/check sustained throughput: %.0f RPS (concurrency=%d, duration=%s)", rps, concurrency, duration)
	if os.Getenv("PS_ENFORCE_RPS") == "1" {
		if rps < 12000 {
			t.Fatalf("throughput %.0f RPS below target (12000)", rps)
		}
	}
}

// TestHTTP_P95_Sub300ms measures P95 latency on 64KB bodies to exercise streaming path.
// To enforce an assertion, set PS_ENFORCE_P95=1.
func TestHTTP_P95_Sub300ms(t *testing.T) {
	// Bypass evaluation limiter
	payload := `{"org":"SLA","tier":"enterprise","expires_at":"2099-01-01T00:00:00Z","entitlements":{}}`
	enc := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString([]byte("sig"))
	os.Setenv("PROMPTSHIELD_LICENSE_KEY", enc)

	// Setup with mock RulepackService
	mockRepo := &mocks.MockRulepackRepository{}
	rulepackService := services.NewRulepackService(mockRepo, nil)
	
	options := api.Options{
		RulepackService: rulepackService,
	}
	h := enforcerhttp.NewMuxWithOptions(options)
	ts := httptest.NewServer(h)
	defer ts.Close()

	tr := &http.Transport{
		MaxIdleConns:        8192,
		MaxIdleConnsPerHost: 8192,
		IdleConnTimeout:     30 * time.Second,
	}
	client := &http.Client{Transport: tr}

	samples := 200
	if v := os.Getenv("PS_P95_SAMPLES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			samples = n
		}
	}
	latencies := make([]time.Duration, 0, samples)
	body := bytes.Repeat([]byte("x"), 64*1024)
	for i := 0; i < samples; i++ {
		start := time.Now()
		resp, err := client.Post(ts.URL+"/check", "text/plain", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("POST error: %v", err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusForbidden {
			t.Fatalf("unexpected status: %d", resp.StatusCode)
		}
		latencies = append(latencies, time.Since(start))
	}
	// Compute P95
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p95 := latencies[int(0.95*float64(len(latencies)))-1]
	t.Logf("/check P95 latency on 64KB: %s (samples=%d)", p95, samples)
	if os.Getenv("PS_ENFORCE_P95") == "1" {
		if p95 > 300*time.Millisecond {
			t.Fatalf("P95 latency %s exceeds 300ms", p95)
		}
	}
}
