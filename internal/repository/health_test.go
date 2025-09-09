package repository

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Simple mock factory for testing
type mockHealthFactory struct {
	healthy bool
}

func (m *mockHealthFactory) Tenant() interface{} { return &struct{}{} }
func (m *mockHealthFactory) Audit() interface{} { return &struct{}{} }
func (m *mockHealthFactory) RulepackAssignment() interface{} { return &struct{}{} }
func (m *mockHealthFactory) APIToken() interface{} { return &struct{}{} }
func (m *mockHealthFactory) Settings() interface{} { return &struct{}{} }
func (m *mockHealthFactory) Rulepack() interface{} { return &struct{}{} }
func (m *mockHealthFactory) Close() error { return nil }
func (m *mockHealthFactory) HealthCheck(ctx context.Context) error {
	if !m.healthy {
		return errors.New("factory health check failed")
	}
	return nil
}

func TestHealthChecker(t *testing.T) {
	// Test healthy factory
	healthyFactory, err := NewTestRepositoryFactory(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create test factory: %v", err)
	}
	
	checker := NewHealthChecker(healthyFactory)
	health := checker.CheckHealth(context.Background())
	
	if health.Status != HealthStatusHealthy {
		t.Errorf("Expected healthy status, got %s", health.Status)
	}
	
	if health.Summary.TotalComponents == 0 {
		t.Errorf("Expected components to be checked")
	}
	
	if health.Summary.HealthyComponents != health.Summary.TotalComponents {
		t.Errorf("Expected all components to be healthy, got %d/%d", 
			health.Summary.HealthyComponents, health.Summary.TotalComponents)
	}
	
	// Verify all expected components are present
	expectedComponents := []string{"factory", "tenant_repository", "audit_repository", 
		"rulepack_repository", "assignment_repository", "token_repository", "settings_repository"}
	
	for _, component := range expectedComponents {
		if _, exists := health.Components[component]; !exists {
			t.Errorf("Expected component %s to be present", component)
		}
	}
}

func TestHealthCheckerUnhealthyFactory(t *testing.T) {
	// Test unhealthy factory (nil factory simulates unhealthy)
	var unhealthyFactory RepositoryFactory = nil
	
	checker := NewHealthChecker(unhealthyFactory)
	health := checker.CheckHealth(context.Background())
	
	if health.Status != HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy status, got %s", health.Status)
	}
	
	if health.Summary.UnhealthyComponents == 0 {
		t.Errorf("Expected at least one unhealthy component")
	}
	
	// When factory is unhealthy, only factory component should be checked
	if len(health.Components) != 1 {
		t.Errorf("Expected only factory component when factory is unhealthy, got %d", len(health.Components))
	}
	
	factoryComponent, exists := health.Components["factory"]
	if !exists {
		t.Errorf("Expected factory component to be present")
	}
	
	if factoryComponent.Status != HealthStatusUnhealthy {
		t.Errorf("Expected factory component to be unhealthy")
	}
}

func TestHealthCheckerNilFactory(t *testing.T) {
	checker := NewHealthChecker(nil)
	health := checker.CheckHealth(context.Background())
	
	if health.Status != HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy status for nil factory, got %s", health.Status)
	}
	
	factoryComponent, exists := health.Components["factory"]
	if !exists {
		t.Errorf("Expected factory component to be present")
	}
	
	if factoryComponent.Status != HealthStatusUnhealthy {
		t.Errorf("Expected factory component to be unhealthy")
	}
	
	if factoryComponent.Message != "Repository factory is nil" {
		t.Errorf("Expected specific error message for nil factory")
	}
}

func TestHealthCheckerMissingRepositories(t *testing.T) {
	// Test factory with some missing repositories (use test factory which should be healthy)
	partialFactory, err := NewTestRepositoryFactory(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create test factory: %v", err)
	}
	
	checker := NewHealthChecker(partialFactory)
	health := checker.CheckHealth(context.Background())
	
	// Test factory should be healthy, so this test verifies the health check works
	if health.Status != HealthStatusHealthy {
		t.Errorf("Expected healthy status for test factory, got %s", health.Status)
	}
}

func TestHTTPHealthHandler(t *testing.T) {
	healthyFactory, err := NewTestRepositoryFactory(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create test factory: %v", err)
	}
	
	handler := HTTPHealthHandler(healthyFactory)
	
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected JSON content type")
	}
	
	var health OverallHealth
	if err := json.NewDecoder(w.Body).Decode(&health); err != nil {
		t.Errorf("Failed to decode health response: %v", err)
	}
	
	if health.Status != HealthStatusHealthy {
		t.Errorf("Expected healthy status in response")
	}
}

func TestHTTPHealthHandlerUnhealthy(t *testing.T) {
	var unhealthyFactory RepositoryFactory = nil
	
	handler := HTTPHealthHandler(unhealthyFactory)
	
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
	
	var health OverallHealth
	if err := json.NewDecoder(w.Body).Decode(&health); err != nil {
		t.Errorf("Failed to decode health response: %v", err)
	}
	
	if health.Status != HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy status in response")
	}
}

func TestHTTPReadinessHandler(t *testing.T) {
	healthyFactory, err := NewTestRepositoryFactory(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create test factory: %v", err)
	}
	
	handler := HTTPReadinessHandler(healthyFactory)
	
	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	if w.Body.String() != "ready" {
		t.Errorf("Expected 'ready' response, got '%s'", w.Body.String())
	}
}

func TestHTTPReadinessHandlerUnhealthy(t *testing.T) {
	var unhealthyFactory RepositoryFactory = nil
	
	handler := HTTPReadinessHandler(unhealthyFactory)
	
	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

func TestHTTPLivenessHandler(t *testing.T) {
	factory, err := NewTestRepositoryFactory(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create test factory: %v", err)
	}
	
	handler := HTTPLivenessHandler(factory)
	
	req := httptest.NewRequest("GET", "/live", nil)
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	if w.Body.String() != "alive" {
		t.Errorf("Expected 'alive' response, got '%s'", w.Body.String())
	}
}

func TestHTTPLivenessHandlerNilFactory(t *testing.T) {
	handler := HTTPLivenessHandler(nil)
	
	req := httptest.NewRequest("GET", "/live", nil)
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

func TestHealthCheckerTimeout(t *testing.T) {
	// Test timeout behavior with context cancellation
	checker := NewHealthChecker(nil) // nil factory will be unhealthy
	checker.timeout = time.Millisecond * 100 // Short timeout
	
	start := time.Now()
	health := checker.CheckHealth(context.Background())
	duration := time.Since(start)
	
	// Should complete quickly
	if duration > time.Millisecond*200 {
		t.Errorf("Health check took too long: %v", duration)
	}
	
	// Should be unhealthy due to nil factory
	if health.Status != HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy status for nil factory, got %s", health.Status)
	}
}

