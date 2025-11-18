package scanner

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"log/slog"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/contracts"
)

// GatewayHandler integrates scanner service into LLM Gateway for request enforcement
type GatewayHandler struct {
	scannerService *Service
	logger         *slog.Logger
	metrics        *ScanMetrics
}

// ScanMetrics tracks gateway scanning statistics
type ScanMetrics struct {
	mu              sync.RWMutex
	TotalScans      int64     `json:"total_scans"`
	BlockedRequests int64     `json:"blocked_requests"`
	AllowedRequests int64     `json:"allowed_requests"`
	LastScanTime    time.Time `json:"last_scan_time"`
	RulesLoaded     int       `json:"rules_loaded"`
	ScannerStatus   string    `json:"scanner_status"`
}

// incrementTotal safely increments total scans
func (m *ScanMetrics) incrementTotal() {
	m.mu.Lock()
	m.TotalScans++
	m.LastScanTime = time.Now()
	m.mu.Unlock()
}

// incrementBlocked safely increments blocked requests
func (m *ScanMetrics) incrementBlocked() {
	m.mu.Lock()
	m.BlockedRequests++
	m.mu.Unlock()
}

// incrementAllowed safely increments allowed requests
func (m *ScanMetrics) incrementAllowed() {
	m.mu.Lock()
	m.AllowedRequests++
	m.mu.Unlock()
}

// snapshot returns a thread-safe copy of metrics
func (m *ScanMetrics) snapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return map[string]interface{}{
		"total_scans":      m.TotalScans,
		"blocked_requests": m.BlockedRequests,
		"allowed_requests": m.AllowedRequests,
		"last_scan_time":   m.LastScanTime,
		"rules_loaded":     m.RulesLoaded,
		"scanner_status":   m.ScannerStatus,
	}
}

// NewGatewayHandler creates a new gateway handler with scanning capabilities
func NewGatewayHandler(auditLogger contracts.AuditLogger, logger *slog.Logger) *GatewayHandler {
	service := NewService(auditLogger, logger)

	// Rule packs will be loaded from configuration files at runtime

	return &GatewayHandler{
		scannerService: service,
		logger:         logger,
		metrics: &ScanMetrics{
			ScannerStatus: "healthy",
			RulesLoaded:   0,
		},
	}
}

// HandleLLMRequest processes an LLM request through the gateway with policy enforcement
func (h *GatewayHandler) HandleLLMRequest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Extract tenant and user info from request context
	tenantID := r.Header.Get("X-Tenant-ID")
	userID := r.Header.Get("X-User-ID")
	requestID := uuid.New().String()

	// Parse LLM request
	var llmReq struct {
		Model        string    `json:"model"`
		Provider     string    `json:"provider"`
		SystemPrompt string    `json:"system_prompt,omitempty"`
		Messages     []Message `json:"messages"`
	}

	if err := json.NewDecoder(r.Body).Decode(&llmReq); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Create scanner request
	request := &LLMRequest{
		RequestID:    requestID,
		TenantID:     tenantID,
		UserID:       userID,
		Model:        llmReq.Model,
		Provider:     llmReq.Provider,
		SystemPrompt: llmReq.SystemPrompt,
		Messages:     llmReq.Messages,
	}

	// Scan the request
	scanResult, err := h.scannerService.ScanLLMRequest(r.Context(), request)
	if err != nil {
		h.logger.Error("scan failed", "error", err, "request_id", requestID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Update metrics
	h.metrics.incrementTotal()

	// Check if request should be blocked
	if !scanResult.Decision.Allow {
		h.metrics.incrementBlocked()
		h.logger.Warn("request blocked",
			"request_id", requestID,
			"reason", scanResult.Decision.Reason,
			"violations", len(scanResult.Violations),
		)

		// Return blocked response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":      "Request blocked by security policy",
			"reason":     scanResult.Decision.Reason,
			"violations": scanResult.Violations,
			"request_id": requestID,
		})
		return
	}

	// Request is allowed - forward to LLM provider
	h.metrics.incrementAllowed()
	h.logger.Info("request allowed",
		"request_id", requestID,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	// Forward request to actual LLM provider
	if err := h.forwardToProvider(w, r, requestID); err != nil {
		h.logger.Error("failed to forward request to provider",
			"request_id", requestID,
			"error", err,
		)
		http.Error(w, "Failed to process request", http.StatusBadGateway)
		return
	}
}

// HandleLLMResponse processes an LLM response through the gateway with policy enforcement
func (h *GatewayHandler) HandleLLMResponse(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Extract context
	tenantID := r.Header.Get("X-Tenant-ID")
	userID := r.Header.Get("X-User-ID")
	requestID := r.Header.Get("X-Request-ID")

	// Parse LLM response
	var llmResp struct {
		Model         string         `json:"model"`
		Provider      string         `json:"provider"`
		Content       string         `json:"content"`
		FunctionCalls []FunctionCall `json:"function_calls,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&llmResp); err != nil {
		http.Error(w, "Invalid response format", http.StatusBadRequest)
		return
	}

	// Create scanner response
	response := &LLMResponse{
		RequestID:     requestID,
		TenantID:      tenantID,
		UserID:        userID,
		Model:         llmResp.Model,
		Provider:      llmResp.Provider,
		Content:       llmResp.Content,
		FunctionCalls: llmResp.FunctionCalls,
	}

	// Scan the response
	scanResult, err := h.scannerService.ScanLLMResponse(r.Context(), response)
	if err != nil {
		h.logger.Error("response scan failed", "error", err, "request_id", requestID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Update metrics
	h.metrics.incrementTotal()

	// Check if response should be blocked
	if !scanResult.Decision.Allow {
		h.metrics.incrementBlocked()
		h.logger.Warn("response blocked",
			"request_id", requestID,
			"reason", scanResult.Decision.Reason,
			"violations", len(scanResult.Violations),
		)

		// Return blocked response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":      "Response blocked by security policy",
			"reason":     scanResult.Decision.Reason,
			"violations": scanResult.Violations,
			"request_id": requestID,
		})
		return
	}

	// Response is allowed - return to client
	h.metrics.incrementAllowed()
	h.logger.Info("response allowed",
		"request_id", requestID,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	// Return the original response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"model":            llmResp.Model,
		"content":          llmResp.Content,
		"function_calls":   llmResp.FunctionCalls,
		"scan_duration_ms": time.Since(start).Milliseconds(),
	})
}

// GetScanMetrics returns scan metrics for monitoring
func (h *GatewayHandler) GetScanMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(h.metrics.snapshot())
}

// forwardToProvider forwards the request to the appropriate LLM provider
func (h *GatewayHandler) forwardToProvider(w http.ResponseWriter, r *http.Request, requestID string) error {
	// This is a simplified implementation - in production you would:
	// 1. Determine the appropriate provider based on routing rules
	// 2. Get provider credentials from secure storage
	// 3. Transform the request format if needed
	// 4. Use the ProviderClient to make the actual request

	// For now, return a success response indicating request was processed
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"message":    "Request processed by PromptShield Gateway",
		"request_id": requestID,
		"status":     "success",
		"note":       "LLM provider forwarding will be implemented with OpenAI omni moderation",
	}

	return json.NewEncoder(w).Encode(response)
}
