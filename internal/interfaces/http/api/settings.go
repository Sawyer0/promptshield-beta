package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// API response types that map to JSON structures
type PlatformSettingsResponse struct {
	ID        uuid.UUID                  `json:"id"`
	Platform  PlatformConfig             `json:"platform"`
	Defaults  DefaultTenantSettings      `json:"defaults"`
	Email     EmailSettings              `json:"email"`
	Notifications NotificationSettings   `json:"notifications"`
	Security  SecuritySettings           `json:"security"`
	UpdatedAt time.Time                  `json:"updated_at"`
	UpdatedBy string                     `json:"updated_by"`
}

// PlatformConfig contains general platform settings
type PlatformConfig struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	AllowSelfSignup  bool   `json:"allow_self_signup"`
	RequireApproval  bool   `json:"require_approval"`
	DefaultTrialDays int    `json:"default_trial_days"`
	MaintenanceMode  bool   `json:"maintenance_mode"`
	MaintenanceMessage string `json:"maintenance_message,omitempty"`
}

// DefaultTenantSettings contains default limits for new tenants
type DefaultTenantSettings struct {
	TenantLimits TenantLimits            `json:"tenant_limits"`
	Features     DefaultFeatures         `json:"features"`
}

// TenantLimits defines resource limits for tenants
type TenantLimits struct {
	MaxRequestsPerDay    int `json:"max_requests_per_day"`
	MaxRulepacks        int `json:"max_rulepacks"`
	MaxUsers            int `json:"max_users"`
	MaxAPIKeys          int `json:"max_api_keys"`
	MaxRetentionDays    int `json:"max_retention_days"`
}

// DefaultFeatures defines which features are enabled by default for new tenants
type DefaultFeatures struct {
	SemanticAnalysisEnabled bool `json:"semantic_analysis_enabled"`
	AuditLoggingEnabled     bool `json:"audit_logging_enabled"`
	UsageTrackingEnabled    bool `json:"usage_tracking_enabled"`
	QuotaManagementEnabled  bool `json:"quota_management_enabled"`
}

// EmailSettings contains SMTP and email configuration
type EmailSettings struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password,omitempty"` // Omit from responses for security
	FromAddress  string `json:"from_address"`
	FromName     string `json:"from_name"`
	EnableTLS    bool   `json:"enable_tls"`
}

// NotificationSettings contains alerting and notification configuration
type NotificationSettings struct {
	SlackWebhook     string              `json:"slack_webhook,omitempty"`
	AlertThresholds  AlertThresholds     `json:"alert_thresholds"`
	EmailNotifications bool              `json:"email_notifications"`
	NotificationTypes []string          `json:"notification_types"`
}

// AlertThresholds defines when to send alerts
type AlertThresholds struct {
	HighErrorRate     float64 `json:"high_error_rate"`
	HighLatencyMs     float64 `json:"high_latency_ms"`
	HighMemoryUsageMB float64 `json:"high_memory_usage_mb"`
	HighCPUPercent    float64 `json:"high_cpu_percent"`
	DiskUsagePercent  float64 `json:"disk_usage_percent"`
}

// SecuritySettings contains platform security configuration
type SecuritySettings struct {
	SessionTimeoutHours    int      `json:"session_timeout_hours"`
	RequireMFA            bool     `json:"require_mfa"`
	AllowedIPRanges       []string `json:"allowed_ip_ranges"`
	RateLimitRPS          float64  `json:"rate_limit_rps"`
	RateLimitBurst        int      `json:"rate_limit_burst"`
	MaxFailedAttempts     int      `json:"max_failed_attempts"`
	PasswordMinLength     int      `json:"password_min_length"`
	PasswordRequireSpecial bool    `json:"password_require_special"`
}

// registerSettingsHandlers registers platform settings management endpoints
func registerSettingsHandlers(r chi.Router, opt Options) {
	if opt.SettingsRepository == nil {
		// Settings management not configured - skip registration
		println("WARNING: SettingsRepository is nil, skipping settings endpoint registration")
		return
	}

	r.Route("/admin/settings", func(sr chi.Router) {
		sr.Use(adminAuth(opt))
		
		sr.Get("/", getSettingsHandler(opt))
		sr.Put("/", updateSettingsHandler(opt))
		sr.Post("/reset", resetSettingsHandler(opt))
		sr.Get("/export", exportSettingsHandler(opt))
	})
	
	println("INFO: Settings endpoints registered at /admin/settings")
}

// getSettingsHandler handles GET /admin/settings
func getSettingsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		settings, err := opt.SettingsRepository.Get(r.Context())
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Failed to retrieve platform settings", map[string]interface{}{"error": err.Error()}, r)
			return
		}

		// Convert to response format and sanitize
		response, err := convertToResponse(settings)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Failed to convert settings", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		writeJSON(w, http.StatusOK, response, r)
	}
}

// updateSettingsHandler handles PUT /admin/settings
func updateSettingsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var settingsData map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&settingsData); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST",
				"Invalid settings data", map[string]interface{}{"error": err.Error()}, r)
			return
		}

		// Get user context for audit trail
		updatedBy := "admin"
		if userID := getUserFromContext(r.Context()); userID != "" {
			updatedBy = userID
		}
		settingsData["updated_by"] = updatedBy
		settingsData["updated_at"] = time.Now().UTC()

		// Update settings in database
		updatedSettings, err := opt.SettingsRepository.Update(r.Context(), settingsData)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Failed to update platform settings", map[string]interface{}{"error": err.Error()}, r)
			return
		}

		// Log settings change for audit
		if opt.AuditLogger != nil {
			auditEvent := types.AuditEvent{
				Action:     "platform.settings_updated",
				ObjectType: "platform_settings",
				ObjectID:   updatedSettings.ID,
				Metadata: map[string]interface{}{
					"action": "platform.settings_updated",
					"updated_by": updatedBy,
					"changes": "settings_configuration",
				},
				Timestamp: time.Now().UTC(),
			}
			opt.AuditLogger.LogWithContext(r.Context(), auditEvent)
		}

		// Convert and return
		response, err := convertToResponse(updatedSettings)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Failed to convert settings", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		writeJSON(w, http.StatusOK, response, r)
	}
}

// resetSettingsHandler handles POST /admin/settings/reset
func resetSettingsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Reset to default settings
		defaultSettings := getDefaultSettings()
		
		// Get user context
		updatedBy := "admin"
		if userID := getUserFromContext(r.Context()); userID != "" {
			updatedBy = userID
		}
		defaultSettings["updated_by"] = updatedBy
		defaultSettings["updated_at"] = time.Now().UTC()

		updatedSettings, err := opt.SettingsRepository.Update(r.Context(), defaultSettings)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Failed to reset platform settings", map[string]interface{}{"error": err.Error()}, r)
			return
		}

		// Log settings reset for audit
		if opt.AuditLogger != nil {
			auditEvent := types.AuditEvent{
				Action:     "platform.settings_reset",
				ObjectType: "platform_settings",
				ObjectID:   updatedSettings.ID,
				Metadata: map[string]interface{}{
					"action": "platform.settings_reset",
					"reset_by": updatedBy,
				},
				Timestamp: time.Now().UTC(),
			}
			opt.AuditLogger.LogWithContext(r.Context(), auditEvent)
		}

		response, err := convertToResponse(updatedSettings)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Failed to convert settings", map[string]interface{}{"error": err.Error()}, r)
			return
		}
		
		writeJSON(w, http.StatusOK, response, r)
	}
}

// exportSettingsHandler handles GET /admin/settings/export
func exportSettingsHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		settings, err := opt.SettingsRepository.Get(r.Context())
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Failed to retrieve platform settings for export", map[string]interface{}{"error": err.Error()}, r)
			return
		}

		// Convert settings
		response, err := convertToResponse(settings)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Failed to convert settings", map[string]interface{}{"error": err.Error()}, r)
			return
		}

		// Create export data with metadata
		exportData := map[string]interface{}{
			"export_info": map[string]interface{}{
				"exported_at": time.Now().UTC().Format(time.RFC3339),
				"exported_by": getUserFromContext(r.Context()),
				"version":     "1.0",
			},
			"settings": response,
		}

		// Set export headers
		filename := "platform_settings_" + time.Now().Format("20060102_150405") + ".json"
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename)

		writeJSON(w, http.StatusOK, exportData, r)
	}
}

// Helper functions

// convertToResponse converts domain.PlatformSettings to API response format
func convertToResponse(settings *domain.PlatformSettings) (*PlatformSettingsResponse, error) {
	if settings == nil {
		return nil, nil
	}

	// Parse the JSON settings into structured format
	var settingsData map[string]interface{}
	if err := json.Unmarshal(settings.Settings, &settingsData); err != nil {
		return nil, err
	}

	// Create response with sanitized data
	response := &PlatformSettingsResponse{
		ID:        settings.ID,
		UpdatedAt: settings.UpdatedAt,
		UpdatedBy: settings.UpdatedBy,
	}

	// Extract structured data - simplified for now
	if platform, ok := settingsData["platform"].(map[string]interface{}); ok {
		response.Platform = PlatformConfig{
			Name:             getString(platform, "name"),
			Description:      getString(platform, "description"),
			AllowSelfSignup:  getBool(platform, "allow_self_signup"),
			RequireApproval:  getBool(platform, "require_approval"),
			DefaultTrialDays: getInt(platform, "default_trial_days"),
			MaintenanceMode:  getBool(platform, "maintenance_mode"),
		}
	}

	// Set other fields with defaults or extracted values
	response.Email.SMTPPassword = "" // Always sanitize password
	
	return response, nil
}

// getDefaultSettings returns the default platform configuration as map
func getDefaultSettings() map[string]interface{} {
	return map[string]interface{}{
		"id": uuid.New().String(),
		"platform": map[string]interface{}{
			"name":               "PromptShield Platform",
			"description":        "Enterprise LLM Security Gateway",
			"allow_self_signup":  false,
			"require_approval":   true,
			"default_trial_days": 14,
			"maintenance_mode":   false,
		},
		"defaults": map[string]interface{}{
			"tenant_limits": map[string]interface{}{
				"max_requests_per_day": 10000,
				"max_rulepacks":       10,
				"max_users":           5,
				"max_api_keys":        10,
				"max_retention_days":  30,
			},
			"features": map[string]interface{}{
				"semantic_analysis_enabled": true,
				"audit_logging_enabled":     true,
				"usage_tracking_enabled":    true,
				"quota_management_enabled":  false,
			},
		},
		"email": map[string]interface{}{
			"smtp_port":  587,
			"from_name":  "PromptShield Platform",
			"enable_tls": true,
		},
		"notifications": map[string]interface{}{
			"alert_thresholds": map[string]interface{}{
				"high_error_rate":      0.05,
				"high_latency_ms":      1000.0,
				"high_memory_usage_mb": 1024.0,
				"high_cpu_percent":     80.0,
				"disk_usage_percent":   85.0,
			},
			"email_notifications": true,
			"notification_types":  []string{"error", "warning", "maintenance"},
		},
		"security": map[string]interface{}{
			"session_timeout_hours":     24,
			"require_mfa":              false,
			"rate_limit_rps":           100.0,
			"rate_limit_burst":         200,
			"max_failed_attempts":      5,
			"password_min_length":      8,
			"password_require_special": true,
		},
	}
}

// getUserFromContext extracts user ID from request context
func getUserFromContext(ctx context.Context) string {
	if userID := ctx.Value("user_id"); userID != nil {
		if userStr, ok := userID.(string); ok {
			return userStr
		}
	}
	return "unknown"
}

// Helper functions for type conversion
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return false
}

func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	if val, ok := m[key].(int); ok {
		return val
	}
	return 0
}