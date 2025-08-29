package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/promptshield/promptshield/internal/domain"
)

// SettingsRepository handles platform settings persistence
type SettingsRepository struct {
	db *Pool
}

// NewSettingsRepository creates a new settings repository
func NewSettingsRepository(db *Pool) domain.SettingsRepository {
	return &SettingsRepository{db: db}
}

// Get retrieves the current platform settings
func (r *SettingsRepository) Get(ctx context.Context) (*domain.PlatformSettings, error) {
	query := `
		SELECT id, settings, updated_at, updated_by
		FROM platform_settings
		ORDER BY updated_at DESC
		LIMIT 1`

	var settings domain.PlatformSettings
	err := r.db.inner.QueryRow(ctx, query).Scan(
		&settings.ID,
		&settings.Settings,
		&settings.UpdatedAt,
		&settings.UpdatedBy,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			// No settings exist, return default settings
			return r.createDefaultSettings(ctx)
		}
		return nil, fmt.Errorf("failed to get platform settings: %w", err)
	}

	return &settings, nil
}

// Update saves the platform settings
func (r *SettingsRepository) Update(ctx context.Context, settings interface{}) (*domain.PlatformSettings, error) {
	// Convert settings to JSON
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Extract metadata from settings if it's the API type
	var updatedBy string = "system"
	var settingsID uuid.UUID

	// Try to extract metadata from the settings object
	if settingsMap, ok := settings.(map[string]interface{}); ok {
		if id, exists := settingsMap["id"].(string); exists {
			if parsed, err := uuid.Parse(id); err == nil {
				settingsID = parsed
			}
		}
		if by, exists := settingsMap["updated_by"].(string); exists {
			updatedBy = by
		}
	}

	// Generate new ID if not provided
	if settingsID == uuid.Nil {
		settingsID = uuid.New()
	}

	// Insert or update settings
	query := `
		INSERT INTO platform_settings (id, settings, updated_at, updated_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) 
		DO UPDATE SET
			settings = EXCLUDED.settings,
			updated_at = EXCLUDED.updated_at,
			updated_by = EXCLUDED.updated_by
		RETURNING id, settings, updated_at, updated_by`

	var result domain.PlatformSettings
	err = r.db.inner.QueryRow(ctx, query, settingsID, settingsJSON, time.Now().UTC(), updatedBy).Scan(
		&result.ID,
		&result.Settings,
		&result.UpdatedAt,
		&result.UpdatedBy,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update platform settings: %w", err)
	}

	return &result, nil
}

// GetHistory retrieves the history of settings changes
func (r *SettingsRepository) GetHistory(ctx context.Context, limit int, offset int) ([]*domain.PlatformSettings, int, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	// Get total count
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM platform_settings`
	err := r.db.inner.QueryRow(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get settings count: %w", err)
	}

	// Get settings with pagination
	query := `
		SELECT id, settings, updated_at, updated_by
		FROM platform_settings
		ORDER BY updated_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.inner.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get settings history: %w", err)
	}
	defer rows.Close()

	var settings []*domain.PlatformSettings
	for rows.Next() {
		var setting domain.PlatformSettings
		err := rows.Scan(
			&setting.ID,
			&setting.Settings,
			&setting.UpdatedAt,
			&setting.UpdatedBy,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan settings row: %w", err)
		}
		settings = append(settings, &setting)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating settings rows: %w", err)
	}

	return settings, totalCount, nil
}

// Delete removes all platform settings (for testing/reset purposes)
func (r *SettingsRepository) Delete(ctx context.Context) error {
	query := `DELETE FROM platform_settings`
	_, err := r.db.inner.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete platform settings: %w", err)
	}
	return nil
}

// createDefaultSettings creates and saves default settings when none exist
func (r *SettingsRepository) createDefaultSettings(ctx context.Context) (*domain.PlatformSettings, error) {
	defaultSettings := map[string]interface{}{
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
				"max_rulepacks":        10,
				"max_users":            5,
				"max_api_keys":         10,
				"max_retention_days":   30,
			},
			"features": map[string]interface{}{
				"semantic_analysis_enabled": true,
				"audit_logging_enabled":     true,
				"usage_tracking_enabled":    true,
				"quota_management_enabled":  false,
				"async_jobs_enabled":        false,
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
			"session_timeout_hours":    24,
			"require_mfa":              false,
			"rate_limit_rps":           100.0,
			"rate_limit_burst":         200,
			"max_failed_attempts":      5,
			"password_min_length":      8,
			"password_require_special": true,
		},
		"updated_at": time.Now().UTC().Format(time.RFC3339),
		"updated_by": "system",
	}

	// Save default settings
	return r.Update(ctx, defaultSettings)
}

// Backup creates a backup of current settings
func (r *SettingsRepository) Backup(ctx context.Context) ([]byte, error) {
	settings, err := r.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings for backup: %w", err)
	}

	backup := map[string]interface{}{
		"backup_info": map[string]interface{}{
			"created_at": time.Now().UTC().Format(time.RFC3339),
			"version":    "1.0",
		},
		"settings": settings,
	}

	backupJSON, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal backup: %w", err)
	}

	return backupJSON, nil
}

// Restore restores settings from a backup
func (r *SettingsRepository) Restore(ctx context.Context, backupData []byte) error {
	var backup map[string]interface{}
	if err := json.Unmarshal(backupData, &backup); err != nil {
		return fmt.Errorf("failed to unmarshal backup data: %w", err)
	}

	settings, ok := backup["settings"]
	if !ok {
		return fmt.Errorf("backup data missing settings")
	}

	// Add restoration metadata
	if settingsMap, ok := settings.(map[string]interface{}); ok {
		settingsMap["updated_by"] = "system_restore"
		settingsMap["updated_at"] = time.Now().UTC().Format(time.RFC3339)
		settingsMap["id"] = uuid.New().String() // New ID for restored settings
	}

	_, err := r.Update(ctx, settings)
	if err != nil {
		return fmt.Errorf("failed to restore settings: %w", err)
	}

	return nil
}

// ValidateConnection tests the database connection
func (r *SettingsRepository) ValidateConnection(ctx context.Context) error {
	query := `SELECT 1`
	var result int
	err := r.db.inner.QueryRow(ctx, query).Scan(&result)
	if err != nil {
		return fmt.Errorf("database connection validation failed: %w", err)
	}
	return nil
}
