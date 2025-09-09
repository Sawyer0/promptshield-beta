package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// ComponentHealth represents the health of a single component
type ComponentHealth struct {
	Name        string                 `json:"name"`
	Status      HealthStatus           `json:"status"`
	Message     string                 `json:"message,omitempty"`
	LastChecked time.Time              `json:"last_checked"`
	Duration    time.Duration          `json:"duration"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// OverallHealth represents the overall health of the repository system
type OverallHealth struct {
	Status     HealthStatus               `json:"status"`
	Timestamp  time.Time                  `json:"timestamp"`
	Duration   time.Duration              `json:"duration"`
	Components map[string]ComponentHealth `json:"components"`
	Summary    HealthSummary              `json:"summary"`
}

// HealthSummary provides a summary of health metrics
type HealthSummary struct {
	TotalComponents   int `json:"total_components"`
	HealthyComponents int `json:"healthy_components"`
	UnhealthyComponents int `json:"unhealthy_components"`
	DegradedComponents  int `json:"degraded_components"`
}

// HealthChecker provides comprehensive health checking for repository components
type HealthChecker struct {
	factory RepositoryFactory
	timeout time.Duration
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(factory RepositoryFactory) *HealthChecker {
	return &HealthChecker{
		factory: factory,
		timeout: 5 * time.Second,
	}
}

// CheckHealth performs a comprehensive health check of all repository components
func (hc *HealthChecker) CheckHealth(ctx context.Context) *OverallHealth {
	start := time.Now()
	
	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, hc.timeout)
	defer cancel()
	
	components := make(map[string]ComponentHealth)
	
	// Check factory health
	factoryHealth := hc.checkFactoryHealth(checkCtx)
	components["factory"] = factoryHealth
	
	// Check individual repository health if factory is healthy
	if factoryHealth.Status == HealthStatusHealthy {
		components["tenant_repository"] = hc.checkRepositoryHealth(checkCtx, "tenant", func() error {
			// Simple health check - just verify the repository exists
			if hc.factory.Tenant() == nil {
				return fmt.Errorf("tenant repository is nil")
			}
			return nil
		})
		
		components["audit_repository"] = hc.checkRepositoryHealth(checkCtx, "audit", func() error {
			if hc.factory.Audit() == nil {
				return fmt.Errorf("audit repository is nil")
			}
			return nil
		})
		
		components["rulepack_repository"] = hc.checkRepositoryHealth(checkCtx, "rulepack", func() error {
			if hc.factory.Rulepack() == nil {
				return fmt.Errorf("rulepack repository is nil")
			}
			return nil
		})
		
		components["assignment_repository"] = hc.checkRepositoryHealth(checkCtx, "assignment", func() error {
			if hc.factory.RulepackAssignment() == nil {
				return fmt.Errorf("assignment repository is nil")
			}
			return nil
		})
		
		components["token_repository"] = hc.checkRepositoryHealth(checkCtx, "token", func() error {
			if hc.factory.APIToken() == nil {
				return fmt.Errorf("token repository is nil")
			}
			return nil
		})
		
		components["settings_repository"] = hc.checkRepositoryHealth(checkCtx, "settings", func() error {
			if hc.factory.Settings() == nil {
				return fmt.Errorf("settings repository is nil")
			}
			return nil
		})
	}
	
	// Calculate overall status and summary
	overallStatus := hc.calculateOverallStatus(components)
	summary := hc.calculateSummary(components)
	
	return &OverallHealth{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Duration:   time.Since(start),
		Components: components,
		Summary:    summary,
	}
}

// checkFactoryHealth checks the health of the repository factory
func (hc *HealthChecker) checkFactoryHealth(ctx context.Context) ComponentHealth {
	start := time.Now()
	
	if hc.factory == nil {
		return ComponentHealth{
			Name:        "factory",
			Status:      HealthStatusUnhealthy,
			Message:     "Repository factory is nil",
			LastChecked: time.Now(),
			Duration:    time.Since(start),
		}
	}
	
	// Check factory health
	if err := hc.factory.HealthCheck(ctx); err != nil {
		return ComponentHealth{
			Name:        "factory",
			Status:      HealthStatusUnhealthy,
			Message:     fmt.Sprintf("Factory health check failed: %v", err),
			LastChecked: time.Now(),
			Duration:    time.Since(start),
		}
	}
	
	return ComponentHealth{
		Name:        "factory",
		Status:      HealthStatusHealthy,
		Message:     "Factory is healthy",
		LastChecked: time.Now(),
		Duration:    time.Since(start),
	}
}

// checkRepositoryHealth checks the health of an individual repository
func (hc *HealthChecker) checkRepositoryHealth(ctx context.Context, name string, checkFn func() error) ComponentHealth {
	start := time.Now()
	
	if err := checkFn(); err != nil {
		return ComponentHealth{
			Name:        name,
			Status:      HealthStatusUnhealthy,
			Message:     fmt.Sprintf("Repository check failed: %v", err),
			LastChecked: time.Now(),
			Duration:    time.Since(start),
		}
	}
	
	return ComponentHealth{
		Name:        name,
		Status:      HealthStatusHealthy,
		Message:     "Repository is healthy",
		LastChecked: time.Now(),
		Duration:    time.Since(start),
	}
}

// calculateOverallStatus determines the overall health status based on component statuses
func (hc *HealthChecker) calculateOverallStatus(components map[string]ComponentHealth) HealthStatus {
	if len(components) == 0 {
		return HealthStatusUnknown
	}
	
	hasUnhealthy := false
	hasDegraded := false
	
	for _, component := range components {
		switch component.Status {
		case HealthStatusUnhealthy:
			hasUnhealthy = true
		case HealthStatusDegraded:
			hasDegraded = true
		}
	}
	
	if hasUnhealthy {
		return HealthStatusUnhealthy
	}
	if hasDegraded {
		return HealthStatusDegraded
	}
	
	return HealthStatusHealthy
}

// calculateSummary calculates summary statistics for the health check
func (hc *HealthChecker) calculateSummary(components map[string]ComponentHealth) HealthSummary {
	summary := HealthSummary{
		TotalComponents: len(components),
	}
	
	for _, component := range components {
		switch component.Status {
		case HealthStatusHealthy:
			summary.HealthyComponents++
		case HealthStatusUnhealthy:
			summary.UnhealthyComponents++
		case HealthStatusDegraded:
			summary.DegradedComponents++
		}
	}
	
	return summary
}

// HTTPHealthHandler creates an HTTP handler for health checks
func HTTPHealthHandler(factory RepositoryFactory) http.HandlerFunc {
	checker := NewHealthChecker(factory)
	
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		health := checker.CheckHealth(ctx)
		
		// Set appropriate HTTP status code
		var statusCode int
		switch health.Status {
		case HealthStatusHealthy:
			statusCode = http.StatusOK
		case HealthStatusDegraded:
			statusCode = http.StatusOK // Still OK, but with warnings
		case HealthStatusUnhealthy:
			statusCode = http.StatusServiceUnavailable
		default:
			statusCode = http.StatusInternalServerError
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		
		if err := json.NewEncoder(w).Encode(health); err != nil {
			http.Error(w, "Failed to encode health response", http.StatusInternalServerError)
		}
	}
}

// HTTPReadinessHandler creates a simple readiness check handler
func HTTPReadinessHandler(factory RepositoryFactory) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		if factory == nil {
			http.Error(w, "Repository factory not initialized", http.StatusServiceUnavailable)
			return
		}
		
		// Quick health check with short timeout
		checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		
		if err := factory.HealthCheck(checkCtx); err != nil {
			http.Error(w, "Repository health check failed", http.StatusServiceUnavailable)
			return
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	}
}

// HTTPLivenessHandler creates a simple liveness check handler
func HTTPLivenessHandler(factory RepositoryFactory) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Liveness check is simpler - just verify factory exists
		if factory == nil {
			http.Error(w, "Repository factory not initialized", http.StatusServiceUnavailable)
			return
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("alive"))
	}
}