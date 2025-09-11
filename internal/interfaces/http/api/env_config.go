package api

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// EnvironmentConfig holds all environment configuration for the API
type EnvironmentConfig struct {
	// Authentication Configuration
	JWTPublicKey  string
	JWTIssuer     string
	JWTAudience   string
	JWTLeeway     time.Duration
	
	// Development Configuration
	DevBypassAuth bool
	DevUserID     string
	DevUserName   string
	DevTenantID   string
	DevRoles      string
	DevIsAdmin    bool
	
	// Admin Configuration
	AdminToken          string
	AllowInsecureAdmin  bool
	
	// Feature Flags
	EnableDebugEndpoints bool
	
	// Rate Limiting
	EnforcerRPS      float64
	EnforcerRPSBurst int
	
	// NATS Configuration
	NATSUrl string
	
	// Environment
	NodeEnv string
}

// LoadEnvironmentConfig loads and validates environment configuration
func LoadEnvironmentConfig() (*EnvironmentConfig, error) {
	config := &EnvironmentConfig{
		// JWT Configuration
		JWTPublicKey: strings.TrimSpace(os.Getenv("PS_BFF_JWT_PUBLIC_KEY")),
		JWTIssuer:    strings.TrimSpace(os.Getenv("PS_BFF_JWT_ISSUER")),
		JWTAudience:  strings.TrimSpace(os.Getenv("PS_BFF_JWT_AUDIENCE")),
		
		// Development Configuration
		DevBypassAuth: strings.EqualFold(strings.TrimSpace(os.Getenv("PS_DEV_BYPASS_AUTH")), "true"),
		DevUserID:     getEnvDefault("PS_DEV_USER_ID", "dev-user"),
		DevUserName:   getEnvDefault("PS_DEV_USER_NAME", "Dev User"),
		DevTenantID:   strings.TrimSpace(os.Getenv("PS_DEV_TENANT_ID")),
		DevRoles:      strings.TrimSpace(os.Getenv("PS_DEV_ROLES")),
		DevIsAdmin:    strings.EqualFold(strings.TrimSpace(os.Getenv("PS_DEV_IS_ADMIN")), "true"),
		
		// Admin Configuration
		AdminToken:         strings.TrimSpace(os.Getenv("PS_ADMIN_TOKEN")),
		AllowInsecureAdmin: strings.EqualFold(strings.TrimSpace(os.Getenv("PS_ALLOW_INSECURE_ADMIN")), "true"),
		
		// Feature Flags
		EnableDebugEndpoints: strings.EqualFold(strings.TrimSpace(os.Getenv("PS_ENABLE_DEBUG_ENDPOINTS")), "true"),
		
		// NATS Configuration
		NATSUrl: strings.TrimSpace(os.Getenv("PS_NATS_URL")),
		
		// Environment
		NodeEnv: strings.ToLower(strings.TrimSpace(os.Getenv("NODE_ENV"))),
	}
	
	// Parse JWT leeway
	if v := strings.TrimSpace(os.Getenv("PS_BFF_JWT_LEEWAY")); v != "" {
		if n, err := time.ParseDuration(v + "s"); err == nil {
			config.JWTLeeway = n
		} else {
			config.JWTLeeway = 60 * time.Second
		}
	} else {
		config.JWTLeeway = 60 * time.Second
	}
	
	// Parse rate limiting configuration
	if v := strings.TrimSpace(os.Getenv("PS_ENFORCER_RPS")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			config.EnforcerRPS = f
		}
	}
	
	if config.EnforcerRPS > 0 {
		if v := strings.TrimSpace(os.Getenv("PS_ENFORCER_RPS_BURST")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				config.EnforcerRPSBurst = n
			}
		}
	}
	
	return config, nil
}

// ValidateEnvironmentConfig validates the environment configuration
func ValidateEnvironmentConfig(config *EnvironmentConfig) []string {
	var errors []string
	
	isProduction := config.NodeEnv == "production"
	
	// Production validations
	if isProduction && config.DevBypassAuth {
		errors = append(errors, "PS_DEV_BYPASS_AUTH should not be enabled in production")
	}
	
	if isProduction && config.AllowInsecureAdmin {
		errors = append(errors, "PS_ALLOW_INSECURE_ADMIN should not be enabled in production")
	}
	
	// JWT validations
	if !config.DevBypassAuth {
		if config.JWTPublicKey == "" {
			errors = append(errors, "PS_BFF_JWT_PUBLIC_KEY is required when dev bypass is disabled")
		} else {
			// Test public key parsing
			if _, err := parseRSAPublicKeyFromPEM([]byte(config.JWTPublicKey)); err != nil {
				errors = append(errors, fmt.Sprintf("PS_BFF_JWT_PUBLIC_KEY is invalid: %v", err))
			}
		}
		
		if config.JWTIssuer == "" {
			errors = append(errors, "PS_BFF_JWT_ISSUER should be set when JWT auth is enabled")
		}
		
		if config.JWTAudience == "" {
			errors = append(errors, "PS_BFF_JWT_AUDIENCE should be set when JWT auth is enabled")
		}
	}
	
	// JWT leeway validation
	if config.JWTLeeway < 0 || config.JWTLeeway > 5*time.Minute {
		errors = append(errors, "PS_BFF_JWT_LEEWAY must be between 0 and 300 seconds")
	}
	
	// Rate limiting validation
	if config.EnforcerRPS > 0 && config.EnforcerRPSBurst <= 0 {
		errors = append(errors, "PS_ENFORCER_RPS_BURST must be set when PS_ENFORCER_RPS is configured")
	}
	
	return errors
}

// LogEnvironmentConfig logs the current environment configuration
func LogEnvironmentConfig(config *EnvironmentConfig) {
	slog.Info("Environment configuration loaded",
		"node_env", config.NodeEnv,
		"dev_bypass_auth", config.DevBypassAuth,
		"has_jwt_public_key", config.JWTPublicKey != "",
		"jwt_issuer", config.JWTIssuer,
		"jwt_audience", config.JWTAudience,
		"jwt_leeway_seconds", int(config.JWTLeeway.Seconds()),
		"has_admin_token", config.AdminToken != "",
		"allow_insecure_admin", config.AllowInsecureAdmin,
		"enable_debug_endpoints", config.EnableDebugEndpoints,
		"enforcer_rps", config.EnforcerRPS,
		"enforcer_rps_burst", config.EnforcerRPSBurst,
		"has_nats_url", config.NATSUrl != "",
	)
	
	if config.DevBypassAuth {
		slog.Warn("Development bypass mode enabled",
			"dev_user_id", config.DevUserID,
			"dev_user_name", config.DevUserName,
			"dev_tenant_id", config.DevTenantID,
			"dev_is_admin", config.DevIsAdmin,
		)
	}
	
	if config.EnableDebugEndpoints {
		slog.Info("Debug endpoints enabled at /debug/*")
	}
}

// GetValidatedEnvironmentConfig loads and validates environment configuration
func GetValidatedEnvironmentConfig() (*EnvironmentConfig, error) {
	config, err := LoadEnvironmentConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load environment configuration: %w", err)
	}
	
	errors := ValidateEnvironmentConfig(config)
	if len(errors) > 0 {
		slog.Error("Environment configuration validation failed",
			"errors", errors)
		return nil, fmt.Errorf("environment configuration validation failed: %s", strings.Join(errors, ", "))
	}
	
	return config, nil
}

// getEnvDefault returns environment variable value or default if empty
func getEnvDefault(key, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return defaultValue
}