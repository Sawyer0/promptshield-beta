package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

// registerProviderHandlers registers all provider key management endpoints
func registerProviderHandlers(r chi.Router, opt Options) {
	// Provider key management (admin only)
	r.Route("/v1/admin/providers", func(pr chi.Router) {
		pr.Use(adminAuth(opt))
		pr.Use(correlationIDMiddleware)
		pr.Use(tenantContextMiddleware)
		
		pr.Post("/keys", registerProviderKeyHandler(opt))
		pr.Get("/keys", listProviderKeysHandler(opt))
		pr.Put("/keys/{keyId}", updateProviderKeyHandler(opt))
		pr.Delete("/keys/{keyId}", deleteProviderKeyHandler(opt))
		
		pr.Post("/test", testProviderConnectivityHandler(opt))
		pr.Get("/supported", getSupportedProvidersHandler(opt))
		
		pr.Post("/config", setProviderConfigHandler(opt))
		pr.Get("/config", getProviderConfigHandler(opt))
		pr.Put("/routing", setRoutingRulesHandler(opt))
		pr.Get("/routing", getRoutingRulesHandler(opt))
	})
}


// ProviderConfig represents configuration for a specific provider
type ProviderConfig struct {
	TenantID     uuid.UUID              `json:"tenant_id"`
	Provider     string                 `json:"provider"`
	Settings     map[string]interface{} `json:"settings"`
	RateLimits   *ProviderRateLimits    `json:"rate_limits,omitempty"`
	Timeouts     *ProviderTimeouts      `json:"timeouts,omitempty"`
	RetryPolicy  *RetryPolicy           `json:"retry_policy,omitempty"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// ProviderRateLimits represents rate limiting config for a provider
type ProviderRateLimits struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	TokensPerMinute   int `json:"tokens_per_minute"`
	ConcurrentRequests int `json:"concurrent_requests"`
}

// ProviderTimeouts represents timeout configuration
type ProviderTimeouts struct {
	RequestTimeoutSeconds  int `json:"request_timeout_seconds"`
	ConnectTimeoutSeconds  int `json:"connect_timeout_seconds"`
	StreamingTimeoutSeconds int `json:"streaming_timeout_seconds"`
}

// RetryPolicy represents retry configuration
type RetryPolicy struct {
	MaxRetries      int   `json:"max_retries"`
	BaseDelayMs     int   `json:"base_delay_ms"`
	MaxDelayMs      int   `json:"max_delay_ms"`
	RetryableErrors []int `json:"retryable_errors"` // HTTP status codes
}

// RoutingRule represents a rule for selecting providers
type RoutingRule struct {
	ID         uuid.UUID               `json:"id"`
	TenantID   uuid.UUID               `json:"tenant_id"`
	Priority   int                     `json:"priority"`
	Name       string                  `json:"name"`
	Conditions map[string]interface{}  `json:"conditions"`
	Target     RoutingTarget           `json:"target"`
	CreatedAt  time.Time               `json:"created_at"`
	UpdatedAt  time.Time               `json:"updated_at"`
}

// RoutingTarget represents the target provider for a routing rule
type RoutingTarget struct {
	Provider  string `json:"provider"`
	KeyAlias  string `json:"key_alias"`
	Endpoint  string `json:"endpoint,omitempty"`
	Fallback  bool   `json:"fallback,omitempty"`
}

// SupportedProvider represents information about a supported LLM provider
type SupportedProvider struct {
	Name         string                 `json:"name"`
	DisplayName  string                 `json:"display_name"`
	Description  string                 `json:"description"`
	Endpoints    []string               `json:"endpoints"`
	AuthMethods  []string               `json:"auth_methods"`
	Models       []string               `json:"models,omitempty"`
	Features     []string               `json:"features"`
	ConfigSchema map[string]interface{} `json:"config_schema,omitempty"`
}

// registerProviderKeyHandler handles POST /v1/admin/providers/keys
func registerProviderKeyHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.ProviderKeyStore == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
				"Provider key management not configured", nil, r)
			return
		}
		
		var req struct {
			Provider    string `json:"provider"`
			KeyAlias    string `json:"key_alias"`
			APIKey      string `json:"api_key"`
			Endpoint    string `json:"endpoint,omitempty"`
			Deployment  string `json:"deployment,omitempty"`
			IsDefault   bool   `json:"is_default"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST", 
				"Invalid request body", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		// Validate required fields
		if req.Provider == "" {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
				"Provider is required", nil, r)
			return
		}
		
		if req.APIKey == "" {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", 
				"API key is required", nil, r)
			return
		}
		
		if req.KeyAlias == "" {
			req.KeyAlias = "default"
		}
		
		// Get tenant ID from context
		tenantIDStr := getTenantID(r)
		if tenantIDStr == "" {
			writeErrorJSON(w, http.StatusBadRequest, "MISSING_TENANT", 
				"Tenant ID required", nil, r)
			return
		}
		
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_TENANT", 
				"Invalid tenant ID", nil, r)
			return
		}
		
		// Validate provider
		if !isValidProvider(req.Provider) {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_PROVIDER", 
				"Unsupported provider", map[string]interface{}{"provider": req.Provider}, r)
			return
		}
		
		// Encrypt the API key (in production, use proper encryption)
		encryptedKey := encryptAPIKey(req.APIKey) // TODO: Implement proper encryption
		
		// Create provider key
		providerKey := &domain.ProviderKey{
			ID:           uuid.New(),
			TenantID:     tenantID,
			Provider:     domain.Provider(req.Provider),
			KeyAlias:     req.KeyAlias,
			EncryptedKey: encryptedKey,
			Endpoint:     req.Endpoint,
			Deployment:   req.Deployment,
			IsDefault:    req.IsDefault,
			CreatedAt:    time.Now(),
			Status:       "active",
		}
		
		// Store the key
		err = opt.ProviderKeyStore.Create(r.Context(), providerKey)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", 
				"Failed to store provider key", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		// Audit log the key registration
		if opt.AuditRepository != nil {
			metadata, _ := json.Marshal(map[string]interface{}{
				"provider":   req.Provider,
				"key_alias":  req.KeyAlias,
				"is_default": req.IsDefault,
			})
			_ = opt.AuditRepository.Create(r.Context(), &domain.AuditEntry{
				Action:     "provider_key.create",
				ObjectType: "provider_key",
				ObjectID:   providerKey.ID,
				TenantID:   &tenantID,
				Metadata:   metadata,
				CreatedAt:  time.Now(),
			})
		}
		
		// Return key info (without encrypted key)
		response := &domain.ProviderKey{
			ID:         providerKey.ID,
			TenantID:   providerKey.TenantID,
			Provider:   providerKey.Provider,
			KeyAlias:   providerKey.KeyAlias,
			Endpoint:   providerKey.Endpoint,
			Deployment: providerKey.Deployment,
			IsDefault:  providerKey.IsDefault,
			CreatedAt:  providerKey.CreatedAt,
			Status:     providerKey.Status,
		}
		
		writeJSON(w, http.StatusCreated, response, r)
	}
}

// listProviderKeysHandler handles GET /v1/admin/providers/keys
func listProviderKeysHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.ProviderKeyStore == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
				"Provider key management not configured", nil, r)
			return
		}
		
		tenantIDStr := getTenantID(r)
		if tenantIDStr == "" {
			writeErrorJSON(w, http.StatusBadRequest, "MISSING_TENANT", 
				"Tenant ID required", nil, r)
			return
		}
		
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_TENANT", 
				"Invalid tenant ID", nil, r)
			return
		}
		
		// Get provider filter
		provider := r.URL.Query().Get("provider")
		
		keys, err := opt.ProviderKeyStore.ListByProvider(r.Context(), tenantID, provider)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", 
				"Failed to list provider keys", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		if keys == nil {
			keys = []*domain.ProviderKey{}
		}
		
		// Strip encrypted keys from response
		for _, key := range keys {
			key.EncryptedKey = ""
		}
		
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"keys":  keys,
			"count": len(keys),
		}, r)
	}
}

// updateProviderKeyHandler handles PUT /v1/admin/providers/keys/{keyId}
func updateProviderKeyHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
			"Provider key updates not yet implemented", nil, r)
	}
}

// deleteProviderKeyHandler handles DELETE /v1/admin/providers/keys/{keyId}
func deleteProviderKeyHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
			"Provider key deletion not yet implemented", nil, r)
	}
}

// testProviderConnectivityHandler handles POST /v1/admin/providers/test
func testProviderConnectivityHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
			"Provider connectivity testing not yet implemented", nil, r)
	}
}

// getSupportedProvidersHandler handles GET /v1/admin/providers/supported
func getSupportedProvidersHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providers := []SupportedProvider{
			{
				Name:        "openai",
				DisplayName: "OpenAI",
				Description: "OpenAI GPT models and services",
				Endpoints:   []string{"chat/completions", "completions", "embeddings", "images/generations"},
				AuthMethods: []string{"bearer_token"},
				Features:    []string{"chat", "completion", "embeddings", "images", "streaming"},
				Models:      []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo", "text-embedding-3-small"},
			},
			{
				Name:        "anthropic",
				DisplayName: "Anthropic",
				Description: "Anthropic Claude models",
				Endpoints:   []string{"messages", "completions"},
				AuthMethods: []string{"api_key"},
				Features:    []string{"chat", "completion", "streaming"},
				Models:      []string{"claude-3-opus", "claude-3-sonnet", "claude-3-haiku"},
			},
			{
				Name:        "azure_openai",
				DisplayName: "Azure OpenAI",
				Description: "Microsoft Azure OpenAI Service",
				Endpoints:   []string{"chat/completions", "completions", "embeddings"},
				AuthMethods: []string{"api_key"},
				Features:    []string{"chat", "completion", "embeddings", "streaming"},
			},
		}
		
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"providers": providers,
			"count":     len(providers),
		}, r)
	}
}

// Placeholder handlers for configuration management
func setProviderConfigHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
			"Provider configuration not yet implemented", nil, r)
	}
}

func getProviderConfigHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
			"Provider configuration not yet implemented", nil, r)
	}
}

func setRoutingRulesHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
			"Routing rules not yet implemented", nil, r)
	}
}

func getRoutingRulesHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", 
			"Routing rules not yet implemented", nil, r)
	}
}

// Helper functions

// isValidProvider checks if a provider is supported
func isValidProvider(provider string) bool {
	validProviders := map[string]bool{
		"openai":      true,
		"anthropic":   true,
		"azure_openai": true,
		"custom":      true,
	}
	return validProviders[strings.ToLower(provider)]
}

// encryptAPIKey encrypts an API key (placeholder - implement proper encryption)
func encryptAPIKey(key string) string {
	// TODO: Implement proper encryption using AES or similar
	// For now, just return the key (THIS IS NOT SECURE)
	return key
}

// decryptAPIKey decrypts an API key (placeholder - implement proper decryption)
func decryptAPIKey(encryptedKey string) string {
	// TODO: Implement proper decryption
	// For now, just return the encrypted key (THIS IS NOT SECURE)
	return encryptedKey
}