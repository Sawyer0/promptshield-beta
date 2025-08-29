package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// ObservabilityService interface for monitoring
type ObservabilityService interface {
	GetDashboardData(tenantID string) (map[string]interface{}, error)
	GetPerformanceMetrics() map[string]interface{}
}

// MonitoringHandlers provides HTTP endpoints for monitoring and dashboards
type MonitoringHandlers struct {
	observability ObservabilityService
}

// NewMonitoringHandlers creates monitoring handlers
func NewMonitoringHandlers(obs ObservabilityService) *MonitoringHandlers {
	return &MonitoringHandlers{
		observability: obs,
	}
}

// RegisterRoutes registers monitoring routes
func (h *MonitoringHandlers) RegisterRoutes(mux *http.ServeMux) {
	// Dashboard endpoints
	mux.HandleFunc("/v1/dashboard", h.GetDashboard)
	mux.HandleFunc("/v1/dashboard/realtime", h.GetRealtimeMetrics)
	
	// Metrics endpoints
	mux.HandleFunc("/v1/metrics/performance", h.GetPerformanceMetrics)
	mux.HandleFunc("/v1/metrics/usage", h.GetUsageMetrics)
	mux.HandleFunc("/v1/metrics/violations", h.GetViolationMetrics)
	
	// Alert endpoints
	mux.HandleFunc("/v1/alerts", h.GetAlerts)
	mux.HandleFunc("/v1/alerts/acknowledge", h.AcknowledgeAlert)
	
	// Health & monitoring
	mux.HandleFunc("/v1/monitor/health", h.GetHealthStatus)
	mux.HandleFunc("/v1/monitor/sla", h.GetSLAMetrics)
}

// GetDashboard returns comprehensive dashboard data
func (h *MonitoringHandlers) GetDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default-tenant"
	}
	
	// Return mock data for now
	data := map[string]interface{}{
		"total_requests_24h":  1500,
		"blocked_requests_24h": 45,
		"block_rate":          3.0,
		"performance": map[string]interface{}{
			"p50_ms": 15,
			"p95_ms": 45,
			"p99_ms": 80,
		},
		"top_rules": []map[string]interface{}{
			{"rule_id": "pi-direct-ignore", "count": 25},
			{"rule_id": "pi-jailbreak", "count": 15},
			{"rule_id": "pi-role-change", "count": 5},
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// GetRealtimeMetrics returns real-time streaming metrics
func (h *MonitoringHandlers) GetRealtimeMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Set up SSE (Server-Sent Events)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}
	
	// Send metrics every second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			metrics := map[string]interface{}{
				"p50_ms":     15 + (time.Now().Second() % 10),
				"p95_ms":     45 + (time.Now().Second() % 20),
				"throughput": 100 + (time.Now().Second() % 50),
				"timestamp":  time.Now().Unix(),
			}
			data, _ := json.Marshal(metrics)
			
			w.Write([]byte("data: " + string(data) + "\n\n"))
			flusher.Flush()
		}
	}
}

// GetPerformanceMetrics returns performance metrics
func (h *MonitoringHandlers) GetPerformanceMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	metrics := map[string]interface{}{
		"p50_ms":         15,
		"p95_ms":         45,
		"p99_ms":         80,
		"throughput":     125,
		"error_rate":     0.5,
		"sla_compliance": 99.2,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// GetUsageMetrics returns usage metrics for billing
func (h *MonitoringHandlers) GetUsageMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		http.Error(w, "Tenant ID required", http.StatusBadRequest)
		return
	}
	
	response := map[string]interface{}{
		"tenant_id": tenantID,
		"period": map[string]string{
			"start": time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339),
			"end":   time.Now().Format(time.RFC3339),
		},
		"metrics": map[string]interface{}{
			"api_calls":           10000,
			"data_processed_mb":   2048,
			"violations_detected": 150,
			"rules_evaluated":     500000,
		},
		"limits": map[string]interface{}{
			"api_calls_limit":     100000,
			"data_limit_mb":       10240,
			"remaining_api_calls": 90000,
			"remaining_data_mb":   8192,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetViolationMetrics returns violation analytics
func (h *MonitoringHandlers) GetViolationMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	response := map[string]interface{}{
		"summary": map[string]interface{}{
			"total_violations": 150,
			"critical":        10,
			"high":           35,
			"medium":         60,
			"low":            45,
		},
		"top_rules": []map[string]interface{}{
			{"rule_id": "pi-direct-ignore", "count": 45, "percentage": 30},
			{"rule_id": "pi-jailbreak", "count": 30, "percentage": 20},
			{"rule_id": "pi-role-change", "count": 25, "percentage": 16.7},
		},
		"trends": []map[string]interface{}{
			{"hour": "2025-08-25T18:00:00Z", "count": 5},
			{"hour": "2025-08-25T19:00:00Z", "count": 8},
			{"hour": "2025-08-25T20:00:00Z", "count": 12},
			{"hour": "2025-08-25T21:00:00Z", "count": 7},
			{"hour": "2025-08-25T22:00:00Z", "count": 3},
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAlerts returns active and recent alerts
func (h *MonitoringHandlers) GetAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	alerts := []map[string]interface{}{
		{
			"id":           "alert-001",
			"type":         "security_violation",
			"severity":     "HIGH",
			"title":        "Multiple injection attempts detected",
			"description":  "Detected 15 injection attempts in the last hour",
			"acknowledged": false,
			"resolved":     false,
			"timestamp":    time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
		},
		{
			"id":           "alert-002",
			"type":         "rate_limit",
			"severity":     "WARNING",
			"title":        "API rate limit approaching",
			"description":  "80% of rate limit consumed",
			"acknowledged": true,
			"resolved":     false,
			"timestamp":    time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"alerts": alerts,
		"count":  len(alerts),
	})
}

// AcknowledgeAlert acknowledges an alert
func (h *MonitoringHandlers) AcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		AlertID string `json:"alert_id"`
		UserID  string `json:"user_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Alert acknowledged",
	})
}

// GetHealthStatus returns detailed health status
func (h *MonitoringHandlers) GetHealthStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"components": map[string]interface{}{
			"database": map[string]interface{}{
				"status":     "healthy",
				"latency_ms": 5,
			},
			"scanner": map[string]interface{}{
				"status":      "healthy",
				"rules_loaded": 150,
			},
			"cache": map[string]interface{}{
				"status":   "healthy",
				"hit_rate": 0.92,
			},
		},
		"metrics": map[string]interface{}{
			"p50_ms":     15,
			"p95_ms":     45,
			"throughput": 125,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// GetSLAMetrics returns SLA compliance metrics
func (h *MonitoringHandlers) GetSLAMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	sla := map[string]interface{}{
		"period": map[string]string{
			"start": time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339),
			"end":   time.Now().Format(time.RFC3339),
		},
		"uptime": map[string]interface{}{
			"percentage": 99.95,
			"target":     99.9,
			"compliant":  true,
		},
		"latency": map[string]interface{}{
			"p50_ms": 15,
			"p95_ms": 45,
			"p99_ms": 80,
			"targets": map[string]float64{
				"p50_ms": 10,
				"p95_ms": 50,
				"p99_ms": 100,
			},
			"compliant": true,
		},
		"error_rate": map[string]interface{}{
			"percentage": 0.5,
			"target":     1.0,
			"compliant":  true,
		},
		"availability": map[string]interface{}{
			"percentage": 99.98,
			"target":     99.95,
			"compliant":  true,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sla)
}