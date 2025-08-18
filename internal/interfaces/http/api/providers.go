package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
	"github.com/promptshield/promptshield/internal/security/crypto"
)

// registerProviderHandlers registers all provider key management endpoints
func registerProviderHandlers(r chi.Router, opt Options) {
	// Provider key management (admin only)
	r.Route("/v1/admin/providers", func(pr chi.Router) {
		pr.Use(adminAuth(opt))

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
	TenantID    uuid.UUID              `json:"tenant_id"`
	Provider    string                 `json:"provider"`
	Settings    map[string]interface{} `json:"settings"`
	RateLimits  *ProviderRateLimits    `json:"rate_limits,omitempty"`
	Timeouts    *ProviderTimeouts      `json:"timeouts,omitempty"`
	RetryPolicy *RetryPolicy           `json:"retry_policy,omitempty"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ProviderRateLimits represents rate limiting config for a provider
type ProviderRateLimits struct {
	RequestsPerMinute  int `json:"requests_per_minute"`
	TokensPerMinute    int `json:"tokens_per_minute"`
	ConcurrentRequests int `json:"concurrent_requests"`
}

// ProviderTimeouts represents timeout configuration
type ProviderTimeouts struct {
	RequestTimeoutSeconds   int `json:"request_timeout_seconds"`
	ConnectTimeoutSeconds   int `json:"connect_timeout_seconds"`
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
	ID         uuid.UUID              `json:"id"`
	TenantID   uuid.UUID              `json:"tenant_id"`
	Priority   int                    `json:"priority"`
	Name       string                 `json:"name"`
	Conditions map[string]interface{} `json:"conditions"`
	Target     RoutingTarget          `json:"target"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// RoutingTarget represents the target provider for a routing rule
type RoutingTarget struct {
	Provider string `json:"provider"`
	KeyAlias string `json:"key_alias"`
	Endpoint string `json:"endpoint,omitempty"`
	Fallback bool   `json:"fallback,omitempty"`
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
			Provider   string `json:"provider"`
			KeyAlias   string `json:"key_alias"`
			APIKey     string `json:"api_key"`
			Endpoint   string `json:"endpoint,omitempty"`
			Deployment string `json:"deployment,omitempty"`
			IsDefault  bool   `json:"is_default"`
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

		// Encrypt the API key
		encryptedKey, err := encryptAPIKey(req.APIKey)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "ENCRYPTION_ERROR",
				"Failed to encrypt API key", nil, r)
			return
		}

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
		if opt.ProviderKeyStore == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED",
				"Provider key management not configured", nil, r)
			return
		}

		keyID := chi.URLParam(r, "keyId")
		if keyID == "" {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Key ID is required", nil, r)
			return
		}

		keyUUID, err := uuid.Parse(keyID)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Invalid key ID format", nil, r)
			return
		}

		var req struct {
			KeyAlias  string `json:"key_alias,omitempty"`
			APIKey    string `json:"api_key,omitempty"`
			Endpoint  string `json:"endpoint,omitempty"`
			IsDefault bool   `json:"is_default,omitempty"`
			Status    string `json:"status,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST",
				"Invalid request body", map[string]interface{}{"error": err.Error()}, r)
			return
		}

		// Get existing key first
		existingKey, err := opt.ProviderKeyStore.Get(r.Context(), keyUUID)
		if err != nil {
			writeErrorJSON(w, http.StatusNotFound, "KEY_NOT_FOUND",
				"Provider key not found", map[string]interface{}{"key_id": keyID}, r)
			return
		}

		// Update only provided fields
		if req.KeyAlias != "" {
			existingKey.KeyAlias = req.KeyAlias
		}
		if req.APIKey != "" {
			encryptedKey, err := encryptAPIKey(req.APIKey)
			if err != nil {
				writeErrorJSON(w, http.StatusInternalServerError, "ENCRYPTION_ERROR",
					"Failed to encrypt API key", nil, r)
				return
			}
			existingKey.EncryptedKey = encryptedKey
		}
		if req.Endpoint != "" {
			existingKey.Endpoint = req.Endpoint
		}
		if req.Status != "" {
			existingKey.Status = domain.KeyStatus(req.Status)

		}
		existingKey.IsDefault = req.IsDefault
		existingKey.UpdatedAt = time.Now()

		err = opt.ProviderKeyStore.Update(r.Context(), existingKey)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "UPDATE_FAILED",
				"Failed to update provider key", map[string]interface{}{"error": err.Error()}, r)
			return
		}

		// Return updated key (without encrypted key)
		response := &domain.ProviderKey{
			ID:         existingKey.ID,
			TenantID:   existingKey.TenantID,
			Provider:   existingKey.Provider,
			KeyAlias:   existingKey.KeyAlias,
			Endpoint:   existingKey.Endpoint,
			Deployment: existingKey.Deployment,
			IsDefault:  existingKey.IsDefault,
			Status:     existingKey.Status,
			CreatedAt:  existingKey.CreatedAt,
			UpdatedAt:  existingKey.UpdatedAt,
		}

		writeJSON(w, http.StatusOK, response, r)
	}
}

// deleteProviderKeyHandler handles DELETE /v1/admin/providers/keys/{keyId}
func deleteProviderKeyHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.ProviderKeyStore == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED",
				"Provider key management not configured", nil, r)
			return
		}

		keyID := chi.URLParam(r, "keyId")
		if keyID == "" {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Key ID is required", nil, r)
			return
		}

		keyUUID, err := uuid.Parse(keyID)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Invalid key ID format", nil, r)
			return
		}

		// Get existing key to verify it exists and for audit logging
		existingKey, err := opt.ProviderKeyStore.Get(r.Context(), keyUUID)
		if err != nil {
			writeErrorJSON(w, http.StatusNotFound, "KEY_NOT_FOUND",
				"Provider key not found", map[string]interface{}{"key_id": keyID}, r)
			return
		}

		// Delete the key
		err = opt.ProviderKeyStore.Delete(r.Context(), keyUUID)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "DELETE_FAILED",
				"Failed to delete provider key", map[string]interface{}{"error": err.Error()}, r)
			return
		}

		// Audit log the deletion
		if opt.AuditRepository != nil {
			metadata, _ := json.Marshal(map[string]interface{}{
				"provider":   string(existingKey.Provider),
				"key_alias":  existingKey.KeyAlias,
				"deleted_by": "admin", // TODO: get actual user from context
			})
			_ = opt.AuditRepository.Create(r.Context(), &domain.AuditEntry{
				Action:     "provider_key.delete",
				ObjectType: "provider_key",
				ObjectID:   existingKey.ID,
				TenantID:   &existingKey.TenantID,
				Metadata:   metadata,
				CreatedAt:  time.Now(),
			})
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// testProviderConnectivityHandler handles POST /v1/admin/providers/test
func testProviderConnectivityHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Provider   string `json:"provider"`
			APIKey     string `json:"api_key,omitempty"`
			KeyID      string `json:"key_id,omitempty"`
			Endpoint   string `json:"endpoint,omitempty"`
			Deployment string `json:"deployment,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST",
				"Invalid request body", map[string]interface{}{"error": err.Error()}, r)
			return
		}

		if req.Provider == "" {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Provider is required", nil, r)
			return
		}

		if !isValidProvider(req.Provider) {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_PROVIDER",
				"Unsupported provider", map[string]interface{}{"provider": req.Provider}, r)
			return
		}

		var providerKey *domain.ProviderKey
		var err error

		// Either use provided API key or fetch from store
		if req.APIKey != "" {
			// Create temporary key for testing
			encryptedKey, err := encryptAPIKey(req.APIKey)
			if err != nil {
				writeErrorJSON(w, http.StatusInternalServerError, "ENCRYPTION_ERROR",
					"Failed to encrypt API key", nil, r)
				return
			}

			providerKey = &domain.ProviderKey{
				Provider:     domain.Provider(req.Provider),
				EncryptedKey: encryptedKey,
				Endpoint:     req.Endpoint,
				Deployment:   req.Deployment,
			}
		} else if req.KeyID != "" {
			if opt.ProviderKeyStore == nil {
				writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED",
					"Provider key store not configured", nil, r)
				return
			}

			keyUUID, err := uuid.Parse(req.KeyID)
			if err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
					"Invalid key ID format", nil, r)
				return
			}

			providerKey, err = opt.ProviderKeyStore.Get(r.Context(), keyUUID)
			if err != nil {
				writeErrorJSON(w, http.StatusNotFound, "KEY_NOT_FOUND",
					"Provider key not found", map[string]interface{}{"key_id": req.KeyID}, r)
				return
			}
		} else {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Either api_key or key_id must be provided", nil, r)
			return
		}

		// Test basic connectivity with a simple request
		testPayload := []byte(`{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"test"}],"max_tokens":1}`)
		if req.Provider == "anthropic" {
			testPayload = []byte(`{"model":"claude-3-haiku-20240307","messages":[{"role":"user","content":"test"}],"max_tokens":1}`)
		}

		startTime := time.Now()
		_, err = providerClient.ProxyRequest(r.Context(), req.Provider, providerKey, "chat/completions", testPayload)
		duration := time.Since(startTime)

		response := map[string]interface{}{
			"provider":      req.Provider,
			"status":        "success",
			"response_time": duration.Milliseconds(),
			"timestamp":     time.Now().UTC(),
		}

		if err != nil {
			response["status"] = "failed"
			if providerErr, ok := err.(*ProviderError); ok {
				response["error"] = map[string]interface{}{
					"code":      providerErr.ErrorCode,
					"message":   providerErr.Message,
					"retryable": providerErr.Retryable,
				}
				if providerErr.StatusCode == 401 || providerErr.StatusCode == 403 {
					response["status"] = "auth_failed"
				}
			} else {
				response["error"] = map[string]interface{}{
					"message": err.Error(),
				}
			}
		}

		writeJSON(w, http.StatusOK, response, r)
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

		response := map[string]interface{}{
			"providers": providers,
			"count":     len(providers),
		}

		// If provider key store is available, add statistics about configured providers
		if opt.ProviderKeyStore != nil {
			// Note: This could be expanded to show actual configured provider counts per tenant
			response["key_store_available"] = true
		} else {
			response["key_store_available"] = false
		}

		// Log provider info access if audit repository is available
		if opt.AuditRepository != nil {
			metadata, _ := json.Marshal(map[string]interface{}{
				"endpoint":       "providers/supported",
				"provider_count": len(providers),
			})
			_ = opt.AuditRepository.Create(r.Context(), &domain.AuditEntry{
				Action:     "providers.supported_accessed",
				ObjectType: "provider_info",
				ObjectID:   uuid.New(),
				Metadata:   metadata,
				CreatedAt:  time.Now(),
			})
		}

		writeJSON(w, http.StatusOK, response, r)
	}
}

// setProviderConfigHandler handles POST /v1/admin/providers/config
func setProviderConfigHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ProviderConfig

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST",
				"Invalid request body", map[string]interface{}{"error": err.Error()}, r)
			return
		}

		if req.Provider == "" {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Provider is required", nil, r)
			return
		}

		if !isValidProvider(req.Provider) {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_PROVIDER",
				"Unsupported provider", map[string]interface{}{"provider": req.Provider}, r)
			return
		}

		// TODO: Store provider configuration in database/config store
		// For now, return success with the configuration
		req.UpdatedAt = time.Now()

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "Provider configuration updated successfully",
			"config":  req,
		}, r)
	}
}

func getProviderConfigHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provider := r.URL.Query().Get("provider")
		tenantID := r.URL.Query().Get("tenant_id")

		if provider == "" {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Provider parameter is required", nil, r)
			return
		}

		if !isValidProvider(provider) {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_PROVIDER",
				"Unsupported provider", map[string]interface{}{"provider": provider}, r)
			return
		}

		// TODO: Retrieve actual configuration from database/config store
		// For now, return default configuration
		defaultConfig := ProviderConfig{
			Provider: provider,
			Settings: map[string]interface{}{
				"enabled": true,
			},
			RateLimits: &ProviderRateLimits{
				RequestsPerMinute:  60,
				TokensPerMinute:    100000,
				ConcurrentRequests: 10,
			},
			Timeouts: &ProviderTimeouts{
				RequestTimeoutSeconds:   30,
				ConnectTimeoutSeconds:   5,
				StreamingTimeoutSeconds: 60,
			},
			RetryPolicy: &RetryPolicy{
				MaxRetries:      3,
				BaseDelayMs:     100,
				MaxDelayMs:      5000,
				RetryableErrors: []int{500, 502, 503, 504, 429},
			},
			UpdatedAt: time.Now(),
		}

		if tenantID != "" {
			if tenantUUID, err := uuid.Parse(tenantID); err == nil {
				defaultConfig.TenantID = tenantUUID
			}
		}

		writeJSON(w, http.StatusOK, defaultConfig, r)
	}
}

func setRoutingRulesHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Rules []RoutingRule `json:"rules"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST",
				"Invalid request body", map[string]interface{}{"error": err.Error()}, r)
			return
		}

		if len(req.Rules) == 0 {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"At least one routing rule is required", nil, r)
			return
		}

		// Validate routing rules
		for i, rule := range req.Rules {
			if rule.Name == "" {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
					fmt.Sprintf("Rule %d: name is required", i), nil, r)
				return
			}
			if rule.Target.Provider == "" {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
					fmt.Sprintf("Rule %d: target provider is required", i), nil, r)
				return
			}
			if !isValidProvider(rule.Target.Provider) {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_PROVIDER",
					fmt.Sprintf("Rule %d: unsupported provider", i),
					map[string]interface{}{"provider": rule.Target.Provider}, r)
				return
			}
		}

		// TODO: Store routing rules in database
		// For now, return success
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":    "Routing rules updated successfully",
			"rule_count": len(req.Rules),
			"updated_at": time.Now(),
		}, r)
	}
}

func getRoutingRulesHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")

		// TODO: Retrieve actual routing rules from database
		// For now, return empty rules or default rules
		var rules []RoutingRule

		if tenantID != "" {
			if tenantUUID, err := uuid.Parse(tenantID); err == nil {
				// Return tenant-specific rules if they exist
				rules = []RoutingRule{
					{
						ID:       uuid.New(),
						TenantID: tenantUUID,
						Priority: 1,
						Name:     "Default OpenAI Route",
						Conditions: map[string]interface{}{
							"model": "gpt-*",
						},
						Target: RoutingTarget{
							Provider: "openai",
							KeyAlias: "default",
						},
						CreatedAt: time.Now().Add(-24 * time.Hour),
						UpdatedAt: time.Now(),
					},
				}
			}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"rules": rules,
			"count": len(rules),
		}, r)
	}
}

// Helper functions

// isValidProvider checks if a provider is supported
func isValidProvider(provider string) bool {
	validProviders := map[string]bool{
		"openai":    true,
		"anthropic": true,
		"azure":     true,
		"custom":    true,
	}
	return validProviders[strings.ToLower(provider)]
}

// encryptAPIKey encrypts an API key using AES-256-GCM
func encryptAPIKey(key string) (string, error) {
	return encryptProviderKey(key)
}

// encryptProviderKey encrypts a provider API key
func encryptProviderKey(key string) (string, error) {
	return crypto.EncryptString(key)
}
