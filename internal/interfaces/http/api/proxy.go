package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
	"github.com/promptshield/promptshield/internal/observability/metrics"
	"github.com/promptshield/promptshield/internal/usage"
)

// registerProxyHandlers registers all LLM proxy endpoints
func registerProxyHandlers(r chi.Router, opt Options) {
	// Universal LLM proxy endpoints
	r.Route("/v1/proxy", func(pr chi.Router) {
		pr.Use(userAuth(opt))
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
	Messages    []ChatMessage          `json:"messages,omitempty"`
	Prompt      string                 `json:"prompt,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Extra       map[string]interface{} `json:"-"` // Provider-specific fields
}

// ChatMessage is defined in llm_clients.go

// ProxyResponse represents a normalized response from any LLM provider
type ProxyResponse struct {
	ID      string     `json:"id"`
	Model   string     `json:"model"`
	Choices []Choice   `json:"choices"`
	Usage   *Usage     `json:"usage,omitempty"`
	Meta    *ProxyMeta `json:"_meta,omitempty"`
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
	Provider      string `json:"provider"`
	RequestID     string `json:"request_id"`
	PolicyApplied string `json:"policy_applied,omitempty"`
	CacheStatus   string `json:"cache_status,omitempty"`
	ProcessingMs  int64  `json:"processing_ms"`
	TokensUsed    int    `json:"tokens_used"`
}

// PolicyViolation represents a policy violation found during scanning
type PolicyViolation struct {
	RuleID     string  `json:"rule_id"`
	Severity   string  `json:"severity"`
	Message    string  `json:"message"`
	Action     string  `json:"action"` // allow, deny, quarantine
	Confidence float64 `json:"confidence,omitempty"`
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
						"rule_id":   violation.RuleID,
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

		// Proxy to provider with enhanced error handling
		providerResponse, err := proxyToProvider(r.Context(), provider, providerKey, "chat/completions", providerRequest)
		if err != nil {
			// Handle structured provider errors
			if providerErr, ok := err.(*ProviderError); ok {
				statusCode := http.StatusBadGateway
				errorCode := "PROVIDER_ERROR"

				// Map provider errors to appropriate HTTP status codes
				switch {
				case providerErr.RateLimited:
					statusCode = http.StatusTooManyRequests
					errorCode = "RATE_LIMITED"
				case providerErr.QuotaExhausted:
					statusCode = http.StatusPaymentRequired
					errorCode = "QUOTA_EXHAUSTED"
				case providerErr.StatusCode == http.StatusUnauthorized:
					statusCode = http.StatusUnauthorized
					errorCode = "PROVIDER_AUTH_FAILED"
				case providerErr.StatusCode >= 400 && providerErr.StatusCode < 500:
					statusCode = http.StatusBadRequest
					errorCode = "PROVIDER_CLIENT_ERROR"
				}

				errorDetails := map[string]interface{}{
					"provider":         provider,
					"provider_error":   providerErr.ErrorCode,
					"provider_message": providerErr.Message,
					"retryable":        providerErr.Retryable,
				}

				if providerErr.RequestID != "" {
					errorDetails["provider_request_id"] = providerErr.RequestID
				}
				if providerErr.RetryAfter != nil {
					errorDetails["retry_after_seconds"] = providerErr.RetryAfter.Seconds()
				}

				writeErrorJSON(w, statusCode, errorCode, providerErr.Message, errorDetails, r)
			} else {
				// Generic error fallback
				writeErrorJSON(w, http.StatusBadGateway, "PROVIDER_ERROR",
					"Failed to proxy request to provider", map[string]interface{}{"provider": provider, "error": err.Error()}, r)
			}
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
						"rule_id":   violation.RuleID,
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

		// Record usage with token tracking
		if opt.UsageStore != nil {
			usageRecord := usage.Record{
				Tenant:    tenantID,
				Route:     "chat/completions",
				Decision:  "allow",
				Bytes:     int64(len(body)),
				Provider:  provider,
				Model:     req.Model,
				Timestamp: time.Now(),
			}

			// Extract token usage from provider response
			if normalizedResponse.Usage != nil {
				usageRecord.PromptTokens = int64(normalizedResponse.Usage.PromptTokens)
				usageRecord.CompletionTokens = int64(normalizedResponse.Usage.CompletionTokens)
				usageRecord.TotalTokens = int64(normalizedResponse.Usage.TotalTokens)

				// Emit Prometheus metrics for billing/observability
				metrics.TokensTotal.WithLabelValues(provider, req.Model, "prompt", tenantID).Add(float64(normalizedResponse.Usage.PromptTokens))
				metrics.TokensTotal.WithLabelValues(provider, req.Model, "completion", tenantID).Add(float64(normalizedResponse.Usage.CompletionTokens))
				metrics.LLMRequestsTotal.WithLabelValues(provider, req.Model, "allow", tenantID).Inc()
				metrics.LLMLatency.WithLabelValues(provider, req.Model).Observe(float64(processingTime) / 1000.0)
			}

			_ = opt.UsageStore.RecordTokens(r.Context(), usageRecord)
		}

		// Audit log the request
		if opt.AuditRepository != nil {
			tenantUUID, _ := uuid.Parse(tenantID)
			metadata, _ := json.Marshal(map[string]interface{}{
				"provider":    provider,
				"model":       req.Model,
				"tokens_used": normalizedResponse.Meta.TokensUsed,
				"violations":  len(violations) + len(responseViolations),
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
		if req.Prompt == "" {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Prompt is required for completion requests", nil, r)
			return
		}

		// Get tenant context
		tenantID := getTenantID(r)
		if tenantID == "" {
			writeErrorJSON(w, http.StatusBadRequest, "MISSING_TENANT",
				"Tenant ID required", nil, r)
			return
		}

		// Determine provider
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

		// Convert to provider-specific format
		providerRequest, err := convertCompletionToProviderFormat(provider, &req)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "CONVERSION_ERROR",
				"Failed to convert request to provider format", map[string]interface{}{"error": err.Error()}, r)
			return
		}

		// Proxy to provider
		providerResponse, err := proxyToProvider(r.Context(), provider, providerKey, "completions", providerRequest)
		if err != nil {
			if providerErr, ok := err.(*ProviderError); ok {
				writeErrorJSON(w, http.StatusBadGateway, "PROVIDER_ERROR",
					providerErr.Message, map[string]interface{}{"provider": provider}, r)
			} else {
				writeErrorJSON(w, http.StatusBadGateway, "PROVIDER_ERROR",
					"Failed to proxy request to provider", map[string]interface{}{"provider": provider, "error": err.Error()}, r)
			}
			return
		}

		// Return response with metadata
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-PS-Provider-Used", provider)
		w.Header().Set("X-PS-Request-ID", getCorrelationID(r))
		w.Header().Set("X-PS-Processing-Time", fmt.Sprintf("%dms", time.Since(startTime).Milliseconds()))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(providerResponse)
	}
}

// universalEmbeddingsHandler handles POST /v1/proxy/embeddings
func universalEmbeddingsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// Parse the request
		var req struct {
			Input []string `json:"input"`
			Model string   `json:"model"`
		}
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
		if len(req.Input) == 0 {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Input array is required for embeddings", nil, r)
			return
		}

		// Get tenant context
		tenantID := getTenantID(r)
		if tenantID == "" {
			writeErrorJSON(w, http.StatusBadRequest, "MISSING_TENANT",
				"Tenant ID required", nil, r)
			return
		}

		// Determine provider (embeddings typically use OpenAI)
		provider := r.Header.Get("X-PS-Provider")
		if provider == "" {
			if strings.Contains(req.Model, "text-embedding") {
				provider = "openai"
			} else {
				provider = "openai" // Default for embeddings
			}
		}

		// Get provider API key
		providerKey, err := getProviderKey(r.Context(), opt, tenantID, provider)
		if err != nil {
			writeErrorJSON(w, http.StatusUnauthorized, "PROVIDER_KEY_ERROR",
				"Failed to get provider API key", map[string]interface{}{"provider": provider, "error": err.Error()}, r)
			return
		}

		// Proxy to provider (embeddings use the same body format for OpenAI)
		providerResponse, err := proxyToProvider(r.Context(), provider, providerKey, "embeddings", body)
		if err != nil {
			if providerErr, ok := err.(*ProviderError); ok {
				writeErrorJSON(w, http.StatusBadGateway, "PROVIDER_ERROR",
					providerErr.Message, map[string]interface{}{"provider": provider}, r)
			} else {
				writeErrorJSON(w, http.StatusBadGateway, "PROVIDER_ERROR",
					"Failed to proxy request to provider", map[string]interface{}{"provider": provider, "error": err.Error()}, r)
			}
			return
		}

		// Return response with metadata
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-PS-Provider-Used", provider)
		w.Header().Set("X-PS-Request-ID", getCorrelationID(r))
		w.Header().Set("X-PS-Processing-Time", fmt.Sprintf("%dms", time.Since(startTime).Milliseconds()))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(providerResponse)
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
			// Handle structured provider errors
			if providerErr, ok := err.(*ProviderError); ok {
				statusCode := http.StatusBadGateway
				errorCode := "PROVIDER_ERROR"

				// Map provider errors to appropriate HTTP status codes
				switch {
				case providerErr.RateLimited:
					statusCode = http.StatusTooManyRequests
					errorCode = "RATE_LIMITED"
				case providerErr.QuotaExhausted:
					statusCode = http.StatusPaymentRequired
					errorCode = "QUOTA_EXHAUSTED"
				case providerErr.StatusCode == http.StatusUnauthorized:
					statusCode = http.StatusUnauthorized
					errorCode = "PROVIDER_AUTH_FAILED"
				case providerErr.StatusCode >= 400 && providerErr.StatusCode < 500:
					statusCode = http.StatusBadRequest
					errorCode = "PROVIDER_CLIENT_ERROR"
				}

				errorDetails := map[string]interface{}{
					"provider":         provider,
					"provider_error":   providerErr.ErrorCode,
					"provider_message": providerErr.Message,
					"retryable":        providerErr.Retryable,
				}

				if providerErr.RequestID != "" {
					errorDetails["provider_request_id"] = providerErr.RequestID
				}

				writeErrorJSON(w, statusCode, errorCode, providerErr.Message, errorDetails, r)
			} else {
				// Generic error fallback
				writeErrorJSON(w, http.StatusBadGateway, "PROVIDER_ERROR",
					"Failed to proxy request to provider", map[string]interface{}{"provider": provider, "error": err.Error()}, r)
			}
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

	// Fetch all keys for this provider – implementation chooses the default if one is flagged.
	keys, err := opt.ProviderKeyStore.ListByProvider(ctx, tenantUUID, provider)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("no %s API key configured for tenant", provider)
	}

	// Return the default key when marked, otherwise the first entry.
	for _, k := range keys {
		if k.IsDefault {
			return k, nil
		}
	}
	return keys[0], nil
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
	switch provider {
	case "openai", "azure":
		// OpenAI/Azure format is already compatible
		return json.Marshal(req)

	case "anthropic":
		// Convert to Anthropic's format
		anthropicReq := map[string]interface{}{
			"model":      req.Model,
			"max_tokens": req.MaxTokens,
			"stream":     req.Stream,
		}

		if req.Temperature > 0 {
			anthropicReq["temperature"] = req.Temperature
		}

		// Convert messages to Anthropic format
		if len(req.Messages) > 0 {
			messages := make([]map[string]interface{}, 0, len(req.Messages))
			for _, msg := range req.Messages {
				messages = append(messages, map[string]interface{}{
					"role":    msg.Role,
					"content": msg.Content,
				})
			}
			anthropicReq["messages"] = messages
		}

		return json.Marshal(anthropicReq)

	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func convertFromProviderFormat(provider string, response []byte) (*ProxyResponse, error) {
	switch provider {
	case "openai", "azure":
		// OpenAI/Azure response is already in the expected format
		var resp ProxyResponse
		err := json.Unmarshal(response, &resp)
		return &resp, err

	case "anthropic":
		// Parse Anthropic response and convert to normalized format
		var anthropicResp map[string]interface{}
		if err := json.Unmarshal(response, &anthropicResp); err != nil {
			return nil, fmt.Errorf("failed to parse Anthropic response: %w", err)
		}

		resp := &ProxyResponse{
			ID:    getString(anthropicResp, "id"),
			Model: getString(anthropicResp, "model"),
		}

		// Convert Anthropic content to choices format
		if content, ok := anthropicResp["content"].([]interface{}); ok && len(content) > 0 {
			if contentBlock, ok := content[0].(map[string]interface{}); ok {
				if text, ok := contentBlock["text"].(string); ok {
					resp.Choices = []Choice{
						{
							Index: 0,
							Message: &ChatMessage{
								Role:    "assistant",
								Content: text,
							},
							FinishReason: getString(anthropicResp, "stop_reason"),
						},
					}
				}
			}
		}

		// Convert usage information
		if usage, ok := anthropicResp["usage"].(map[string]interface{}); ok {
			resp.Usage = &Usage{
				PromptTokens:     getInt(usage, "input_tokens"),
				CompletionTokens: getInt(usage, "output_tokens"),
			}
			resp.Usage.TotalTokens = resp.Usage.PromptTokens + resp.Usage.CompletionTokens
		}

		return resp, nil

	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// Helper functions for type-safe map access
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	if v, ok := m[key].(int); ok {
		return v
	}
	return 0
}

// Global provider client instance with retry logic and error handling
var providerClient = NewProviderClient()

func proxyToProvider(ctx context.Context, provider string, key *domain.ProviderKey, endpoint string, body []byte) ([]byte, error) {
	return providerClient.ProxyRequest(ctx, provider, key, endpoint, body)
}

// convertCompletionToProviderFormat converts completion requests to provider-specific format
func convertCompletionToProviderFormat(provider string, req *ProxyRequest) ([]byte, error) {
	switch provider {
	case "openai", "azure":
		// OpenAI completion format
		completionReq := map[string]interface{}{
			"model":  req.Model,
			"prompt": req.Prompt,
			"stream": req.Stream,
		}

		if req.MaxTokens > 0 {
			completionReq["max_tokens"] = req.MaxTokens
		}
		if req.Temperature > 0 {
			completionReq["temperature"] = req.Temperature
		}

		return json.Marshal(completionReq)

	case "anthropic":
		// Anthropic doesn't have a direct completion endpoint, convert to messages
		anthropicReq := map[string]interface{}{
			"model":      req.Model,
			"max_tokens": req.MaxTokens,
			"stream":     req.Stream,
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": req.Prompt,
				},
			},
		}

		if req.Temperature > 0 {
			anthropicReq["temperature"] = req.Temperature
		}

		return json.Marshal(anthropicReq)

	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}
