package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

// registerProxyHandlers registers all LLM proxy endpoints
func registerProxyHandlers(r chi.Router, opt Options) {
	// Universal LLM proxy endpoints
	r.Route("/v1/proxy", func(pr chi.Router) {
		pr.Use(userAuth(opt))
		pr.Use(correlationIDMiddleware)
		pr.Use(tenantContextMiddleware)
		pr.Use(rateLimitMiddleware(opt.QuotaStore))
		
		// Universal endpoints (provider-agnostic)
		pr.Post("/chat/completions", universalChatHandler(opt))
		pr.Post("/completions", universalCompletionHandler(opt))
		pr.Post("/embeddings", universalEmbeddingsHandler(opt))
		
		// Direct provider proxy (more flexible)
		pr.Post("/{provider}/{endpoint:.*}", directProviderProxyHandler(opt))
	})
}

// ProxyRequest represents a normalized request to any LLM provider
type ProxyRequest struct {
	Model       string                 `json:"model"`
	Messages    []ChatMessage         `json:"messages,omitempty"`
	Prompt      string                `json:"prompt,omitempty"`
	MaxTokens   int                   `json:"max_tokens,omitempty"`
	Temperature float64               `json:"temperature,omitempty"`
	Stream      bool                  `json:"stream,omitempty"`
	Extra       map[string]interface{} `json:"-"` // Provider-specific fields
}

// ChatMessage is defined in llm_clients.go

// ProxyResponse represents a normalized response from any LLM provider
type ProxyResponse struct {
	ID      string                 `json:"id"`
	Model   string                 `json:"model"`
	Choices []Choice              `json:"choices"`
	Usage   *Usage                `json:"usage,omitempty"`
	Meta    *ProxyMeta            `json:"_meta,omitempty"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int          `json:"index"`
	Message      *ChatMessage `json:"message,omitempty"`
	Text         string       `json:"text,omitempty"`
	FinishReason string       `json:"finish_reason,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ProxyMeta contains metadata about the proxied request
type ProxyMeta struct {
	Provider      string    `json:"provider"`
	RequestID     string    `json:"request_id"`
	PolicyApplied string    `json:"policy_applied,omitempty"`
	CacheStatus   string    `json:"cache_status,omitempty"`
	ProcessingMs  int64     `json:"processing_ms"`
	TokensUsed    int       `json:"tokens_used"`
}

// PolicyViolation represents a policy violation found during scanning
type PolicyViolation struct {
	RuleID      string `json:"rule_id"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Action      string `json:"action"` // allow, deny, quarantine
	Confidence  float64 `json:"confidence,omitempty"`
}

// universalChatHandler handles POST /v1/proxy/chat/completions
func universalChatHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		
		// Parse the request
		var req ProxyRequest
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST", 
				"Failed to read request body", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		if err := json.Unmarshal(body, &req); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST", 
				"Invalid JSON in request body", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		// Validate request
		if len(req.Messages) == 0 {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
				"Messages array is required", nil, r)
			return
		}
		
		// Get tenant context
		tenantID := getTenantID(r)
		if tenantID == "" {
			writeErrorJSON(w, http.StatusBadRequest, "MISSING_TENANT", 
				"Tenant ID required", nil, r)
			return
		}
		
		// Determine provider (from header or routing rules)
		provider := r.Header.Get("X-PS-Provider")
		if provider == "" {
			provider = determineProviderFromModel(req.Model)
		}
		if provider == "" {
			provider = "openai" // Default fallback
		}
		
		// Get provider API key
		providerKey, err := getProviderKey(r.Context(), opt, tenantID, provider)
		if err != nil {
			writeErrorJSON(w, http.StatusUnauthorized, "PROVIDER_KEY_ERROR", 
				"Failed to get provider API key", map[string]interface{}{"provider": provider, "error": err.Error()}, r)
			return
		}
		
		// Pre-request policy enforcement
		violations, err := enforcePreRequestPolicy(r.Context(), opt, &req, tenantID)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "POLICY_ERROR", 
				"Failed to enforce pre-request policy", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		// Check for blocking violations
		for _, violation := range violations {
			if violation.Action == "deny" {
				writeErrorJSON(w, http.StatusForbidden, "POLICY_VIOLATION", 
					"Request blocked by security policy", map[string]interface{}{
						"violation": violation,
						"rule_id": violation.RuleID,
					}, r)
				return
			}
		}
		
		// Convert request to provider-specific format
		providerRequest, err := convertToProviderFormat(provider, &req)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "CONVERSION_ERROR", 
				"Failed to convert request to provider format", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		// Proxy to provider
		providerResponse, err := proxyToProvider(r.Context(), provider, providerKey, "chat/completions", providerRequest)
		if err != nil {
			writeErrorJSON(w, http.StatusBadGateway, "PROVIDER_ERROR", 
				"Failed to proxy request to provider", map[string]interface{}{"provider": provider, "error": err.Error()}, r)
			return
		}
		
		// Convert response to normalized format
		normalizedResponse, err := convertFromProviderFormat(provider, providerResponse)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "CONVERSION_ERROR", 
				"Failed to convert provider response", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		// Post-response policy enforcement
		responseViolations, err := enforcePostResponsePolicy(r.Context(), opt, normalizedResponse, tenantID)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "POLICY_ERROR", 
				"Failed to enforce post-response policy", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		// Check for response blocking violations
		for _, violation := range responseViolations {
			if violation.Action == "deny" {
				writeErrorJSON(w, http.StatusForbidden, "RESPONSE_POLICY_VIOLATION", 
					"Response blocked by security policy", map[string]interface{}{
						"violation": violation,
						"rule_id": violation.RuleID,
					}, r)
				return
			}
		}
		
		// Add metadata
		processingTime := time.Since(startTime).Milliseconds()
		normalizedResponse.Meta = &ProxyMeta{
			Provider:     provider,
			RequestID:    getCorrelationID(r),
			ProcessingMs: processingTime,
		}
		
		if normalizedResponse.Usage != nil {
			normalizedResponse.Meta.TokensUsed = normalizedResponse.Usage.TotalTokens
		}
		
		// Record usage
		if opt.UsageStore != nil {
			_ = opt.UsageStore.Record(r.Context(), tenantID, "chat/completions", "allow", 
				int64(len(body)), time.Now())
		}
		
		// Audit log the request
		if opt.AuditRepository != nil {
			tenantUUID, _ := uuid.Parse(tenantID)
			metadata, _ := json.Marshal(map[string]interface{}{
				"provider":     provider,
				"model":        req.Model,
				"tokens_used":  normalizedResponse.Meta.TokensUsed,
				"violations":   len(violations) + len(responseViolations),
			})
			_ = opt.AuditRepository.Create(r.Context(), &domain.AuditEntry{
				Action:     "llm.chat_completion",
				ObjectType: "request",
				ObjectID:   uuid.New(),
				TenantID:   &tenantUUID,
				Metadata:   metadata,
				CreatedAt:  time.Now(),
			})
		}
		
		// Return response
		writeJSON(w, http.StatusOK, normalizedResponse, r)
	}
}

// universalCompletionHandler handles POST /v1/proxy/completions
func universalCompletionHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
			"Universal completion endpoint not yet implemented", nil, r)
	}
}

// universalEmbeddingsHandler handles POST /v1/proxy/embeddings
func universalEmbeddingsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
			"Universal embeddings endpoint not yet implemented", nil, r)
	}
}

// directProviderProxyHandler handles POST /v1/proxy/{provider}/{endpoint}
func directProviderProxyHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provider := chi.URLParam(r, "provider")
		endpoint := chi.URLParam(r, "endpoint")
		
		if !isValidProvider(provider) {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_PROVIDER", 
				"Unsupported provider", map[string]interface{}{"provider": provider}, r)
			return
		}
		
		tenantID := getTenantID(r)
		if tenantID == "" {
			writeErrorJSON(w, http.StatusBadRequest, "MISSING_TENANT", 
				"Tenant ID required", nil, r)
			return
		}
		
		// Get provider API key
		providerKey, err := getProviderKey(r.Context(), opt, tenantID, provider)
		if err != nil {
			writeErrorJSON(w, http.StatusUnauthorized, "PROVIDER_KEY_ERROR", 
				"Failed to get provider API key", map[string]interface{}{"provider": provider, "error": err.Error()}, r)
			return
		}
		
		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST", 
				"Failed to read request body", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		// Proxy directly to provider (minimal processing)
		response, err := proxyToProvider(r.Context(), provider, providerKey, endpoint, body)
		if err != nil {
			writeErrorJSON(w, http.StatusBadGateway, "PROVIDER_ERROR", 
				"Failed to proxy request to provider", map[string]interface{}{"provider": provider, "error": err.Error()}, r)
			return
		}
		
		// Return raw provider response
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-PS-Provider-Used", provider)
		w.Header().Set("X-PS-Request-ID", getCorrelationID(r))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(response)
	}
}

// Helper functions (placeholders for now)

func determineProviderFromModel(model string) string {
	if strings.HasPrefix(model, "gpt-") {
		return "openai"
	}
	if strings.HasPrefix(model, "claude-") {
		return "anthropic"
	}
	return ""
}

func getProviderKey(ctx context.Context, opt Options, tenantID, provider string) (*domain.ProviderKey, error) {
	if opt.ProviderKeyStore == nil {
		return nil, &ApiError{Message: "Provider key store not configured"}
	}
	
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, err
	}
	
	return opt.ProviderKeyStore.Get(ctx, tenantUUID) // TODO: Fix this to match actual interface
}

func enforcePreRequestPolicy(ctx context.Context, opt Options, req *ProxyRequest, tenantID string) ([]PolicyViolation, error) {
	// Placeholder for policy enforcement
	// In production, this would use the scanner to check the request content
	return []PolicyViolation{}, nil
}

func enforcePostResponsePolicy(ctx context.Context, opt Options, resp *ProxyResponse, tenantID string) ([]PolicyViolation, error) {
	// Placeholder for policy enforcement
	// In production, this would use the scanner to check the response content
	return []PolicyViolation{}, nil
}

func convertToProviderFormat(provider string, req *ProxyRequest) ([]byte, error) {
	// Convert normalized request to provider-specific format
	// For now, just marshal the request as-is
	return json.Marshal(req)
}

func convertFromProviderFormat(provider string, response []byte) (*ProxyResponse, error) {
	// Convert provider response to normalized format
	var resp ProxyResponse
	err := json.Unmarshal(response, &resp)
	return &resp, err
}

func proxyToProvider(ctx context.Context, provider string, key *domain.ProviderKey, endpoint string, body []byte) ([]byte, error) {
	// This would implement the actual HTTP proxy to the provider
	// For now, return a mock response
	mockResponse := ProxyResponse{
		ID:    "mock-" + uuid.New().String(),
		Model: "mock-model",
		Choices: []Choice{
			{
				Index: 0,
				Message: &ChatMessage{
					Role:    "assistant",
					Content: "This is a mock response from the proxy layer.",
				},
				FinishReason: "stop",
			},
		},
		Usage: &Usage{
			PromptTokens:     10,
			CompletionTokens: 15,
			TotalTokens:      25,
		},
	}
	
	return json.Marshal(mockResponse)
}