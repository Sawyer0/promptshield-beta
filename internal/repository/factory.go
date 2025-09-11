package repository

import (
	"context"
	"time"

	"github.com/promptshield/promptshield/internal/contracts"
	"github.com/promptshield/promptshield/internal/domain"
	sharedcontracts "github.com/promptshield/promptshield/internal/shared/contracts"
)

// RepositoryFactory defines the contract for creating all repository types
type RepositoryFactory interface {
	// Core domain repositories
	Tenant() domain.TenantRepository
	Audit() domain.AuditRepository
	RulepackAssignment() domain.RulepackAssignmentRepository
	APIToken() domain.APITokenRepository
	Settings() domain.SettingsRepository

	// Business logic repositories
	Rulepack() contracts.RulepackRepository
	Policy() sharedcontracts.PolicyRepository

	// Lifecycle management
	Close() error
	HealthCheck(ctx context.Context) error
}

// RepositoryConfig holds configuration for repository factory initialization
type RepositoryConfig struct {
	// Database configuration
	DatabaseURL string

	// Redis configuration (optional)
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Cache TTL settings
	TenantCacheTTL     time.Duration
	AssignmentCacheTTL time.Duration
	TokenCacheTTL      time.Duration

	// Environment settings
	Environment string // "production", "development", "test"
	TestMode    bool

	// Connection pool settings
	MaxConnections     int
	MaxIdleConnections int
	ConnectionTimeout  time.Duration
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *RepositoryConfig {
	return &RepositoryConfig{
		// Cache TTL defaults
		TenantCacheTTL:     15 * time.Minute,
		AssignmentCacheTTL: 10 * time.Minute,
		TokenCacheTTL:      30 * time.Minute,

		// Environment defaults
		Environment: "development",
		TestMode:    false,

		// Connection pool defaults
		MaxConnections:     25,
		MaxIdleConnections: 5,
		ConnectionTimeout:  30 * time.Second,

		// Redis defaults
		RedisDB: 0,
	}
}

// Validate checks if the configuration is valid
func (c *RepositoryConfig) Validate() error {
	if c.Environment == "" {
		c.Environment = "development"
	}

	if c.MaxConnections <= 0 {
		c.MaxConnections = 25
	}

	if c.MaxIdleConnections <= 0 {
		c.MaxIdleConnections = 5
	}

	if c.ConnectionTimeout <= 0 {
		c.ConnectionTimeout = 30 * time.Second
	}

	// Set default cache TTLs if not specified
	if c.TenantCacheTTL <= 0 {
		c.TenantCacheTTL = 15 * time.Minute
	}
	if c.AssignmentCacheTTL <= 0 {
		c.AssignmentCacheTTL = 10 * time.Minute
	}
	if c.TokenCacheTTL <= 0 {
		c.TokenCacheTTL = 30 * time.Minute
	}

	return nil
}

// IsProduction returns true if this is a production environment
func (c *RepositoryConfig) IsProduction() bool {
	return c.Environment == "production" && c.RedisAddr != ""
}

// IsDevelopment returns true if this is a development environment
func (c *RepositoryConfig) IsDevelopment() bool {
	if c.IsTest() {
		return false
	}
	return c.Environment == "development" || (c.Environment == "production" && c.RedisAddr == "")
}

// IsTest returns true if this is a test environment
func (c *RepositoryConfig) IsTest() bool {
	return c.Environment == "test" || c.TestMode
}

