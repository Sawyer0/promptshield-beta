package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
	"golang.org/x/time/rate"
)

// registerQuotaHandlers registers all quota-related endpoints
func registerQuotaHandlers(r chi.Router, opt Options) {
	r.Route("/v1/admin/tenants/{id}/quota", func(qr chi.Router) {
		qr.Use(adminAuth(opt))

		qr.Get("/", getTenantQuotaHandler(opt))
		qr.Put("/", updateTenantQuotaHandler(opt))
	})

	// General quota/limits endpoints
	r.Route("/v1/limits", func(lr chi.Router) {
		lr.Use(userAuth(opt))

		lr.Get("/", getCurrentLimitsHandler(opt))
		lr.Put("/", updateCurrentLimitsHandler(opt))
	})
}

// QuotaInfo represents quota limits and usage for a tenant
type QuotaInfo struct {
	TenantID               string     `json:"tenant_id"`
	RequestsPerMinute      float64    `json:"requests_per_minute"`
	RequestsPerMinuteBurst int        `json:"requests_per_minute_burst"`
	TokensPerHour          int64      `json:"tokens_per_hour,omitempty"`
	MaxPromptTokens        int        `json:"max_prompt_tokens,omitempty"`
	MaxCompletionTokens    int        `json:"max_completion_tokens,omitempty"`
	CurrentUsage           *UsageInfo `json:"current_usage,omitempty"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// UsageInfo represents current usage statistics
type UsageInfo struct {
	RequestsLastMinute int64     `json:"requests_last_minute"`
	TokensLastHour     int64     `json:"tokens_last_hour"`
	LastReset          time.Time `json:"last_reset"`
}

// getTenantQuotaHandler handles GET /v1/admin/tenants/{id}/quota
func getTenantQuotaHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantIDStr := chi.URLParam(r, "id")
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Invalid tenant ID format", map[string]interface{}{"id": tenantIDStr}, r)
			return
		}

		// Verify tenant exists if repository is available
		if opt.TenantRepository != nil {
			_, err = opt.TenantRepository.Get(r.Context(), tenantID)
			if err != nil {
				writeErrorJSON(w, http.StatusNotFound, "TENANT_NOT_FOUND",
					"Tenant not found", map[string]interface{}{"tenant_id": tenantID.String()}, r)
				return
			}
		}

		// Get current quota settings
		quotaInfo := &QuotaInfo{
			TenantID:               tenantIDStr,
			RequestsPerMinute:      100, // Default values
			RequestsPerMinuteBurst: 10,
			TokensPerHour:          100000,
			MaxPromptTokens:        4000,
			MaxCompletionTokens:    2000,
			UpdatedAt:              time.Now(),
		}

		// If we have a usage store, get current usage
		if opt.UsageStore != nil {
			// TODO: Query current usage from usage store
			quotaInfo.CurrentUsage = &UsageInfo{
				RequestsLastMinute: 0,
				TokensLastHour:     0,
				LastReset:          time.Now().Truncate(time.Minute),
			}
		}

		writeJSON(w, http.StatusOK, quotaInfo, r)
	}
}

// updateTenantQuotaHandler handles PUT /v1/admin/tenants/{id}/quota
func updateTenantQuotaHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantIDStr := chi.URLParam(r, "id")
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Invalid tenant ID format", map[string]interface{}{"id": tenantIDStr}, r)
			return
		}

		var req struct {
			RequestsPerMinute      *float64 `json:"requests_per_minute,omitempty"`
			RequestsPerMinuteBurst *int     `json:"requests_per_minute_burst,omitempty"`
			TokensPerHour          *int64   `json:"tokens_per_hour,omitempty"`
			MaxPromptTokens        *int     `json:"max_prompt_tokens,omitempty"`
			MaxCompletionTokens    *int     `json:"max_completion_tokens,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST",
				"Invalid request body", map[string]interface{}{"error": err.Error()}, r)
			return
		}

		// Verify tenant exists if repository is available
		if opt.TenantRepository != nil {
			_, err = opt.TenantRepository.Get(r.Context(), tenantID)
			if err != nil {
				writeErrorJSON(w, http.StatusNotFound, "TENANT_NOT_FOUND",
					"Tenant not found", map[string]interface{}{"tenant_id": tenantID.String()}, r)
				return
			}
		}

		// Set defaults if not provided
		rps := 100.0
		burst := 10

		if req.RequestsPerMinute != nil {
			if *req.RequestsPerMinute < 0 {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
					"Requests per minute must be non-negative",
					map[string]interface{}{"requests_per_minute": *req.RequestsPerMinute}, r)
				return
			}
			rps = *req.RequestsPerMinute
		}

		if req.RequestsPerMinuteBurst != nil {
			if *req.RequestsPerMinuteBurst < 0 {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
					"Burst must be non-negative",
					map[string]interface{}{"requests_per_minute_burst": *req.RequestsPerMinuteBurst}, r)
				return
			}
			burst = *req.RequestsPerMinuteBurst
		}

		// Update quota store if available
		if opt.QuotaStore != nil {
			opt.QuotaStore.Set(tenantIDStr, rate.Limit(rps), burst)
		}

		// Audit log the quota update
		if opt.AuditRepository != nil {
			metadata, _ := json.Marshal(map[string]interface{}{
				"requests_per_minute":       rps,
				"requests_per_minute_burst": burst,
				"tokens_per_hour":           req.TokensPerHour,
				"max_prompt_tokens":         req.MaxPromptTokens,
				"max_completion_tokens":     req.MaxCompletionTokens,
			})
			_ = opt.AuditRepository.Create(r.Context(), &domain.AuditEntry{
				Action:     "quota.update",
				ObjectType: "tenant",
				ObjectID:   tenantID,
				TenantID:   &tenantID,
				Metadata:   metadata,
				CreatedAt:  time.Now(),
			})
		}

		// Return updated quota info
		quotaInfo := &QuotaInfo{
			TenantID:               tenantIDStr,
			RequestsPerMinute:      rps,
			RequestsPerMinuteBurst: burst,
			UpdatedAt:              time.Now(),
		}

		if req.TokensPerHour != nil {
			quotaInfo.TokensPerHour = *req.TokensPerHour
		}
		if req.MaxPromptTokens != nil {
			quotaInfo.MaxPromptTokens = *req.MaxPromptTokens
		}
		if req.MaxCompletionTokens != nil {
			quotaInfo.MaxCompletionTokens = *req.MaxCompletionTokens
		}

		writeJSON(w, http.StatusOK, quotaInfo, r)
	}
}

// getCurrentLimitsHandler handles GET /v1/limits
func getCurrentLimitsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := getTenantID(r)
		if tenantID == "" {
			writeErrorJSON(w, http.StatusBadRequest, "MISSING_TENANT",
				"Tenant ID required", nil, r)
			return
		}

		// Get user-specific limits based on tenant context
		result := map[string]interface{}{
			"tenant": map[string]interface{}{
				"requests_per_minute": 100,
				"tokens_per_hour":     100000,
				"current_usage": map[string]interface{}{
					"requests_last_minute": 0,
					"tokens_last_hour":     0,
				},
			},
			"user": map[string]interface{}{
				"requests_per_minute": 10,
				"tokens_per_hour":     10000,
				"current_usage": map[string]interface{}{
					"requests_last_minute": 0,
					"tokens_last_hour":     0,
				},
			},
		}

		writeJSON(w, http.StatusOK, result, r)
	}
}

// updateCurrentLimitsHandler handles PUT /v1/limits
func updateCurrentLimitsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := getTenantID(r)
		if tenantID == "" {
			writeErrorJSON(w, http.StatusBadRequest, "MISSING_TENANT",
				"Tenant ID required", nil, r)
			return
		}

		var req struct {
			TenantID string `json:"tenant_id"`
			Limits   struct {
				RequestsPerMinute   float64 `json:"requests_per_minute"`
				TokensPerHour       int64   `json:"tokens_per_hour"`
				MaxPromptTokens     int     `json:"max_prompt_tokens"`
				MaxCompletionTokens int     `json:"max_completion_tokens"`
			} `json:"limits"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST",
				"Invalid request body", map[string]interface{}{"error": err.Error()}, r)
			return
		}

		// Validate limits
		if req.Limits.RequestsPerMinute < 0 {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"Requests per minute must be non-negative", nil, r)
			return
		}

		// This endpoint is for user-initiated limit updates
		// In a real system, this would check user permissions and apply limits

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"tenant_id": req.TenantID,
			"limits":    req.Limits,
			"updated":   true,
		}, r)
	}
}
