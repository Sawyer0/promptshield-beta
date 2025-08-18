package scanner

import (
	"encoding/json"
	"net/http"
	"time"

	"log/slog"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/shared/contracts"
)

// ExampleGatewayHandler shows how to integrate scanner service into LLM Gateway
type ExampleGatewayHandler struct {
	scannerService *Service
	logger         *slog.Logger
}

// NewExampleGatewayHandler creates a new example gateway handler
func NewExampleGatewayHandler(auditLogger contracts.AuditLogger, logger *slog.Logger) *ExampleGatewayHandler {
	service := NewService(auditLogger, logger)

	// Load example rule packs for LLM Gateway
	service.LoadRulePacks([]rules.RulePack{
		{
			APIVersion: "promptshield.io/v1",
			Kind:       "RulePack",
			Metadata: rules.Metadata{
				Name:        "llm-gateway-security",
				Version:     "1.0.0",
				Description: "Security rules for LLM Gateway",
			},
			Rules: []rules.Rule{
				{
					ID:       "prompt-injection",
					Level:    1,
					Keywords: []string{"ignore previous instructions", "forget your role", "disregard", "override"},
					Severity: "CRITICAL",
					Category: "prompt-injection",
					Response: &rules.Response{
						Action:  "deny",
						Message: "Prompt injection attempt detected",
					},
				},
				{
					ID:    "pii-detection",
					Level: 2,
					Patterns: []rules.Pattern{
						{Regex: `\b\d{3}-\d{2}-\d{4}\b`},                               // SSN
						{Regex: `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`}, // Email
					},
					Severity: "HIGH",
					Category: "pii-detection",
					Response: &rules.Response{
						Action:  "redact",
						Message: "PII detected in request",
					},
				},
				{
					ID:       "malicious-code",
					Level:    1,
					Keywords: []string{"rm -rf", "format c:", "delete system", "drop database"},
					Severity: "CRITICAL",
					Category: "malicious-code",
					Response: &rules.Response{
						Action:  "deny",
						Message: "Malicious code attempt detected",
					},
				},
			},
		},
	})

	return &ExampleGatewayHandler{
		scannerService: service,
		logger:         logger,
	}
}

// HandleLLMRequest processes an LLM request through the gateway
func (h *ExampleGatewayHandler) HandleLLMRequest(w http.ResponseWriter, r *http.Request) {
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

	// Check if request should be blocked
	if !scanResult.Decision.Allow {
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
	h.logger.Info("request allowed",
		"request_id", requestID,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	// TODO: Forward request to actual LLM provider
	// For now, return a mock response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":          "Request processed successfully",
		"request_id":       requestID,
		"scan_duration_ms": time.Since(start).Milliseconds(),
	})
}

// HandleLLMResponse processes an LLM response through the gateway
func (h *ExampleGatewayHandler) HandleLLMResponse(w http.ResponseWriter, r *http.Request) {
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

	// Check if response should be blocked
	if !scanResult.Decision.Allow {
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
func (h *ExampleGatewayHandler) GetScanMetrics(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement metrics collection
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"scanner_status": "healthy",
		"rules_loaded":   3,
		"last_scan":      time.Now().Format(time.RFC3339),
	})
}
