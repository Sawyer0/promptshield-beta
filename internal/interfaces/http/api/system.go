package api

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
	"github.com/promptshield/promptshield/internal/license"
	"github.com/promptshield/promptshield/internal/version"
)

// registerSystemHandlers registers all system management and diagnostics endpoints
func registerSystemHandlers(r chi.Router, opt Options) {
	// System management endpoints (admin only)
	r.Route("/v1/admin/system", func(sr chi.Router) {
		sr.Use(adminAuth(opt))

		sr.Get("/features", getFeaturesHandler(opt))
		sr.Get("/stats", getSystemStatsHandler(opt))
		sr.Post("/drain", drainSystemHandler(opt))
		sr.Post("/shutdown", shutdownSystemHandler(opt))
		sr.Get("/info", getSystemInfoHandler(opt))
	})

	// Debug and diagnostics endpoints (admin only)
	r.Route("/debug", func(dr chi.Router) {
		dr.Use(adminAuth(opt))

		dr.Get("/status", getSystemStatusHandler(opt))
		dr.Get("/goroutines", getGoroutinesHandler(opt))
		dr.Get("/memory", getMemoryStatsHandler(opt))
		dr.Get("/health", getHealthCheckHandler(opt))
	})
}

// SystemFeatures represents available features and their status
type SystemFeatures struct {
	AsyncJobs        bool `json:"async_jobs"`
	L3Semantic       bool `json:"l3_semantic"`
	TenantManagement bool `json:"tenant_management"`
	AuditLogging     bool `json:"audit_logging"`
	UsageTracking    bool `json:"usage_tracking"`
	QuotaManagement  bool `json:"quota_management"`
	// OIDCAuth removed - Security Gateway uses simple token auth only
}

// SystemStats represents system performance statistics
type SystemStats struct {
	DecisionsTotal map[string]int64 `json:"decisions_total"`
	P95LatencyMs   float64          `json:"p95_latency_ms"`
	RequestsTotal  int64            `json:"requests_total"`
	ErrorsTotal    int64            `json:"errors_total"`
	Uptime         string           `json:"uptime"`
	Version        string           `json:"version"`
}

// SystemInfo represents general system information
type SystemInfo struct {
	Version   string                 `json:"version"`
	Commit    string                 `json:"commit"`
	BuildDate string                 `json:"build_date"`
	GoVersion string                 `json:"go_version"`
	Platform  string                 `json:"platform"`
	StartTime time.Time              `json:"start_time"`
	Uptime    string                 `json:"uptime"`
	License   map[string]interface{} `json:"license"`
	Features  SystemFeatures         `json:"features"`
}

// SystemStatus represents comprehensive system health status
type SystemStatus struct {
	Status      string                     `json:"status"` // healthy, degraded, unhealthy
	Timestamp   time.Time                  `json:"timestamp"`
	Version     string                     `json:"version"`
	Uptime      string                     `json:"uptime"`
	Components  map[string]ComponentStatus `json:"components"`
	Metrics     SystemMetrics              `json:"metrics"`
	Environment map[string]interface{}     `json:"environment"`
}

// ComponentStatus represents the status of individual system components
type ComponentStatus struct {
	Status    string                 `json:"status"` // healthy, degraded, unhealthy
	Message   string                 `json:"message,omitempty"`
	LastCheck time.Time              `json:"last_check"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// SystemMetrics represents current system metrics
type SystemMetrics struct {
	Goroutines    int     `json:"goroutines"`
	MemoryUsedMB  float64 `json:"memory_used_mb"`
	MemoryTotalMB float64 `json:"memory_total_mb"`
	CPUUsage      float64 `json:"cpu_usage_percent,omitempty"`
	InflightBytes int64   `json:"inflight_bytes"`
	OpenFiles     int     `json:"open_files,omitempty"`
}

var systemStartTime = time.Now()

// getFeaturesHandler handles GET /v1/admin/system/features
func getFeaturesHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Ensure license state is loaded
		_ = license.IsLicensed()

		features := SystemFeatures{
			AsyncJobs:        license.HasFeature("async_jobs"),
			L3Semantic:       license.HasFeature("l3_semantic"),
			TenantManagement: opt.TenantRepository != nil,
			AuditLogging:     opt.AuditRepository != nil,
			UsageTracking:    opt.UsageStore != nil,
			QuotaManagement:  opt.QuotaStore != nil,
			// OIDCAuth removed - Security Gateway uses simple token auth only
		}

		writeJSON(w, http.StatusOK, features, r)
	}
}

// getSystemStatsHandler handles GET /v1/admin/system/stats
func getSystemStatsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// This would normally pull from Prometheus metrics
		// For now, return basic stats
		stats := SystemStats{
			DecisionsTotal: map[string]int64{
				"allow":      0,
				"deny":       0,
				"quarantine": 0,
			},
			P95LatencyMs:  0.0,
			RequestsTotal: 0,
			ErrorsTotal:   0,
			Uptime:        time.Since(systemStartTime).String(),
			Version:       version.Version,
		}

		writeJSON(w, http.StatusOK, stats, r)
	}
}

// drainSystemHandler handles POST /v1/admin/system/drain
func drainSystemHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.OnDrain != nil {
			go func() {
				_ = opt.OnDrain(r.Context())
			}()
		}

		writeJSON(w, http.StatusAccepted, map[string]interface{}{
			"message":   "System drain initiated",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}, r)
	}
}

// shutdownSystemHandler handles POST /v1/admin/system/shutdown
func shutdownSystemHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse optional delay parameter
		delay := 0 * time.Second
		if delayStr := r.URL.Query().Get("delay"); delayStr != "" {
			if seconds, err := strconv.Atoi(delayStr); err == nil && seconds >= 0 {
				delay = time.Duration(seconds) * time.Second
			}
		}

		if opt.OnShutdown != nil {
			go func() {
				_ = opt.OnShutdown(context.Background(), delay)
			}()
		}

		response := map[string]interface{}{
			"message":   "System shutdown initiated",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		if delay > 0 {
			response["delay_seconds"] = int(delay.Seconds())
		}

		writeJSON(w, http.StatusAccepted, response, r)
	}
}

// getSystemInfoHandler handles GET /v1/admin/system/info
func getSystemInfoHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get license information
		licenseInfo := license.Info()
		entitlements, _ := license.Entitlement()

		info := SystemInfo{
			Version:   version.Version,
			Commit:    version.Commit,
			BuildDate: version.BuildDate,
			GoVersion: runtime.Version(),
			Platform:  runtime.GOOS + "/" + runtime.GOARCH,
			StartTime: systemStartTime,
			Uptime:    time.Since(systemStartTime).String(),
			License: map[string]interface{}{
				"org":          licenseInfo.Organization,
				"tier":         licenseInfo.Tier,
				"expires_at":   licenseInfo.ExpiresAt,
				"licensed":     license.IsLicensed(),
				"entitlements": entitlements,
			},
			Features: SystemFeatures{
				AsyncJobs:        license.HasFeature("async_jobs"),
				L3Semantic:       license.HasFeature("l3_semantic"),
				TenantManagement: opt.TenantRepository != nil,
				AuditLogging:     opt.AuditRepository != nil,
				UsageTracking:    opt.UsageStore != nil,
				QuotaManagement:  opt.QuotaStore != nil,
				// OIDCAuth removed - Security Gateway uses simple token auth only
			},
		}

		writeJSON(w, http.StatusOK, info, r)
	}
}

// getSystemStatusHandler handles GET /debug/status
func getSystemStatusHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()

		// Check component health
		components := make(map[string]ComponentStatus)

		// Database health (if repositories are available)
		if opt.TenantRepository != nil {
			// Try a simple query to check database connectivity
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()

			_, _, err := opt.TenantRepository.List(ctx, 0, 1)
			if err != nil {
				components["database"] = ComponentStatus{
					Status:    "unhealthy",
					Message:   "Database query failed",
					LastCheck: now,
					Details:   map[string]interface{}{"error": err.Error()},
				}
			} else {
				components["database"] = ComponentStatus{
					Status:    "healthy",
					Message:   "Database connectivity OK",
					LastCheck: now,
				}
			}
		}

		// License health
		if license.IsLicensed() {
			licenseInfo := license.Info()
			if time.Now().After(licenseInfo.ExpiresAt) {
				components["license"] = ComponentStatus{
					Status:    "unhealthy",
					Message:   "License expired",
					LastCheck: now,
					Details:   map[string]interface{}{"expires_at": licenseInfo.ExpiresAt},
				}
			} else {
				components["license"] = ComponentStatus{
					Status:    "healthy",
					Message:   "License valid",
					LastCheck: now,
					Details:   map[string]interface{}{"expires_at": licenseInfo.ExpiresAt},
				}
			}
		} else {
			components["license"] = ComponentStatus{
				Status:    "degraded",
				Message:   "No license configured",
				LastCheck: now,
			}
		}

		// Memory health
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		memUsedMB := float64(memStats.Alloc) / 1024 / 1024
		memTotalMB := float64(memStats.Sys) / 1024 / 1024

		memStatus := "healthy"
		memMessage := "Memory usage normal"
		if memUsedMB > 1000 { // > 1GB
			memStatus = "degraded"
			memMessage = "High memory usage"
		}
		if memUsedMB > 2000 { // > 2GB
			memStatus = "unhealthy"
			memMessage = "Critical memory usage"
		}

		components["memory"] = ComponentStatus{
			Status:    memStatus,
			Message:   memMessage,
			LastCheck: now,
			Details: map[string]interface{}{
				"used_mb":  memUsedMB,
				"total_mb": memTotalMB,
			},
		}

		// Goroutine health
		goroutines := runtime.NumGoroutine()
		goroutineStatus := "healthy"
		goroutineMessage := "Goroutine count normal"
		if goroutines > 1000 {
			goroutineStatus = "degraded"
			goroutineMessage = "High goroutine count"
		}
		if goroutines > 5000 {
			goroutineStatus = "unhealthy"
			goroutineMessage = "Critical goroutine count"
		}

		components["goroutines"] = ComponentStatus{
			Status:    goroutineStatus,
			Message:   goroutineMessage,
			LastCheck: now,
			Details:   map[string]interface{}{"count": goroutines},
		}

		// Determine overall status
		overallStatus := "healthy"
		for _, component := range components {
			if component.Status == "unhealthy" {
				overallStatus = "unhealthy"
				break
			} else if component.Status == "degraded" && overallStatus == "healthy" {
				overallStatus = "degraded"
			}
		}

		status := SystemStatus{
			Status:     overallStatus,
			Timestamp:  now,
			Version:    version.Version,
			Uptime:     time.Since(systemStartTime).String(),
			Components: components,
			Metrics: SystemMetrics{
				Goroutines:    goroutines,
				MemoryUsedMB:  memUsedMB,
				MemoryTotalMB: memTotalMB,
				InflightBytes: 0, // Would need to track this
			},
			Environment: map[string]interface{}{
				"go_version": runtime.Version(),
				"platform":   runtime.GOOS + "/" + runtime.GOARCH,
				"num_cpu":    runtime.NumCPU(),
			},
		}

		// Set appropriate HTTP status code
		statusCode := http.StatusOK
		if overallStatus == "degraded" {
			statusCode = http.StatusPartialContent
		} else if overallStatus == "unhealthy" {
			statusCode = http.StatusServiceUnavailable
		}

		writeJSON(w, statusCode, status, r)
	}
}

// getGoroutinesHandler handles GET /debug/goroutines
func getGoroutinesHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if audit logging is enabled for debug endpoints
		if opt.AuditRepository != nil {
			// Log debug endpoint access for security monitoring
			metadata, _ := json.Marshal(map[string]interface{}{
				"endpoint":         "debug/goroutines",
				"stacks_requested": r.URL.Query().Get("stacks") == "true",
			})
			_ = opt.AuditRepository.Create(r.Context(), &domain.AuditEntry{
				Action:     "debug.goroutines_accessed",
				ObjectType: "debug_endpoint",
				ObjectID:   uuid.New(),
				Metadata:   metadata,
				CreatedAt:  time.Now(),
			})
		}

		goroutines := runtime.NumGoroutine()

		// Get stack traces if requested
		includeStacks := r.URL.Query().Get("stacks") == "true"

		response := map[string]interface{}{
			"count":     goroutines,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		if includeStacks {
			// Get stack traces (be careful with memory usage)
			buf := make([]byte, 1024*1024) // 1MB buffer
			stackSize := runtime.Stack(buf, true)
			response["stacks"] = string(buf[:stackSize])
		}

		writeJSON(w, http.StatusOK, response, r)
	}
}

// getMemoryStatsHandler handles GET /debug/memory
func getMemoryStatsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if audit logging is enabled for debug endpoints
		if opt.AuditRepository != nil {
			// Log debug endpoint access for security monitoring
			metadata, _ := json.Marshal(map[string]interface{}{
				"endpoint": "debug/memory",
			})
			_ = opt.AuditRepository.Create(r.Context(), &domain.AuditEntry{
				Action:     "debug.memory_accessed",
				ObjectType: "debug_endpoint",
				ObjectID:   uuid.New(),
				Metadata:   metadata,
				CreatedAt:  time.Now(),
			})
		}

		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		response := map[string]interface{}{
			"timestamp":        time.Now().UTC().Format(time.RFC3339),
			"alloc_mb":         float64(memStats.Alloc) / 1024 / 1024,
			"total_alloc_mb":   float64(memStats.TotalAlloc) / 1024 / 1024,
			"sys_mb":           float64(memStats.Sys) / 1024 / 1024,
			"num_gc":           memStats.NumGC,
			"gc_pause_ns":      memStats.PauseTotalNs,
			"heap_objects":     memStats.HeapObjects,
			"heap_in_use_mb":   float64(memStats.HeapInuse) / 1024 / 1024,
			"heap_released_mb": float64(memStats.HeapReleased) / 1024 / 1024,
			"stack_in_use_mb":  float64(memStats.StackInuse) / 1024 / 1024,
		}

		writeJSON(w, http.StatusOK, response, r)
	}
}

// getHealthCheckHandler handles GET /debug/health
func getHealthCheckHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Enhanced health check using available dependencies
		response := map[string]interface{}{
			"status":     "healthy",
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
			"uptime":     time.Since(systemStartTime).String(),
			"version":    version.Version,
			"components": map[string]string{},
		}

		components := response["components"].(map[string]string)

		// Check database connectivity if available
		if opt.TenantRepository != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			_, _, err := opt.TenantRepository.List(ctx, 0, 1)
			if err != nil {
				components["database"] = "unhealthy"
				response["status"] = "degraded"
			} else {
				components["database"] = "healthy"
			}
		}

		// Check usage store if available
		if opt.UsageStore != nil {
			components["usage_store"] = "healthy"
		}

		// Check quota store if available
		if opt.QuotaStore != nil {
			components["quota_store"] = "healthy"
		}

		writeJSON(w, http.StatusOK, response, r)
	}
}
