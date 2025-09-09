package main

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/infrastructure/messaging/nats"
	"github.com/promptshield/promptshield/internal/interfaces/http/api"
	enforcerhttp "github.com/promptshield/promptshield/internal/interfaces/http/enforcer"
	"github.com/promptshield/promptshield/internal/repository"
)

func TestHealthzAndMetrics(t *testing.T) {
	// Ensure readiness does not fail due to missing rules by setting a dummy env
	t.Setenv("PS_ENFORCER_RULEPACK", "dummy")

	// Set test mode to ensure we get a test factory
	t.Setenv("PS_TEST_MODE", "true")

	// Setup with repository factory
	ctx := context.Background()
	repoFactory, err := repository.BuildWithFallback(ctx)
	if err != nil {
		t.Fatalf("Failed to create repository factory: %v", err)
	}
	defer repoFactory.Close()

	// Create NATS publisher
	publisher, err := nats.NewPublisher("")
	if err != nil {
		t.Fatalf("Failed to create NATS publisher: %v", err)
	}
	defer publisher.Close()

	// Create RulepackService using factory
	rulepackService := services.RulepackServiceFromFactory(repoFactory, publisher)
	
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

func TestLicenseEndpointsAndGating(t *testing.T) {
	t.Setenv("PS_ENFORCER_RULEPACK", "dummy")
	// Provide a valid-looking license payload without signature verification by omitting public key
	// Token format: base64url(payload).base64url(signature). We only parse payload here.
	payload := `{"org":"Acme","tier":"enterprise","expires_at":"2099-01-01T00:00:00Z","entitlements":{"max_rps": 1, "features": {"l3_semantic": true, "async_jobs": true}}}`
	enc := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString([]byte("sig"))
	t.Setenv("PROMPTSHIELD_LICENSE_KEY", enc)

	// Set test mode to ensure we get a test factory
	t.Setenv("PS_TEST_MODE", "true")

	// Setup with repository factory
	ctx := context.Background()
	repoFactory, err := repository.BuildWithFallback(ctx)
	if err != nil {
		t.Fatalf("Failed to create repository factory: %v", err)
	}
	defer repoFactory.Close()

	// Create NATS publisher
	publisher, err := nats.NewPublisher("")
	if err != nil {
		t.Fatalf("Failed to create NATS publisher: %v", err)
	}
	defer publisher.Close()

	// Create RulepackService using factory
	rulepackService := services.RulepackServiceFromFactory(repoFactory, publisher)
	
	options := api.Options{
		RulepackService: rulepackService,
	}
	h := enforcerhttp.NewMuxWithOptions(options)
	ts := httptest.NewServer(h)
	defer ts.Close()

	// GET /v1/license should reflect entitlements
	resp, err := http.Get(ts.URL + "/v1/license")
	if err != nil {
		t.Fatalf("GET /v1/license error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/v1/license status = %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// /v1/scan:async should be allowed when async_jobs is true
	resp2, err := http.Post(ts.URL+"/v1/scan:async", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("POST /v1/scan:async error: %v", err)
	}
	if resp2.StatusCode == http.StatusForbidden {
		t.Fatalf("/v1/scan:async unexpectedly forbidden with licensed async_jobs")
	}
	_ = resp2.Body.Close()
}
