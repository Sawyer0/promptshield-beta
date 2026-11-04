package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/interfaces/http/api"
	enforcerhttp "github.com/promptshield/promptshield/internal/interfaces/http/enforcer"
	"github.com/promptshield/promptshield/internal/testutil/mocks"
)

func TestHealthzAndMetrics(t *testing.T) {
	// Ensure readiness does not fail due to missing rules by setting a dummy env
	t.Setenv("PS_ENFORCER_RULEPACK", "dummy")

	// Setup with mock RulepackService
	mockRepo := &mocks.MockRulepackRepository{}
	rulepackService := services.RulepackServiceCstor(mockRepo, nil)

	options := api.Options{
		RulepackService: rulepackService,
	}
	h := enforcerhttp.NewMuxWithOptions(options)
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

func TestScanAsyncAvailable(t *testing.T) {
	t.Setenv("PS_ENFORCER_RULEPACK", "dummy")
	mockRepo := &mocks.MockRulepackRepository{}
	rulepackService := services.RulepackServiceCstor(mockRepo, nil)
	h := enforcerhttp.NewMuxWithOptions(api.Options{RulepackService: rulepackService})
	ts := httptest.NewServer(h)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/v1/scan:async", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("POST /v1/scan:async error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden {
		t.Fatalf("/v1/scan:async unexpectedly forbidden")
	}
}
