package repository

import (
	"context"
	"fmt"
)

// FactoryBuilder provides a fluent interface for building repository factories
type FactoryBuilder struct {
	config             *RepositoryConfig
	customRepositories map[string]interface{}
	explicitEnv        bool // Track if environment was explicitly set
}

// NewFactoryBuilder creates a new factory builder with default configuration
func NewFactoryBuilder() *FactoryBuilder {
	return &FactoryBuilder{
		config:             DefaultConfig(),
		customRepositories: make(map[string]interface{}),
	}
}

// WithConfig sets the repository configuration
func (fb *FactoryBuilder) WithConfig(config *RepositoryConfig) *FactoryBuilder {
	if config != nil {
		fb.config = config
	}
	return fb
}

// WithDatabaseURL sets the database connection URL
func (fb *FactoryBuilder) WithDatabaseURL(url string) *FactoryBuilder {
	fb.config.DatabaseURL = url
	return fb
}

// WithRedis configures Redis connection settings
func (fb *FactoryBuilder) WithRedis(addr, password string, db int) *FactoryBuilder {
	fb.config.RedisAddr = addr
	fb.config.RedisPassword = password
	fb.config.RedisDB = db
	return fb
}

// WithEnvironment sets the environment type
func (fb *FactoryBuilder) WithEnvironment(env string) *FactoryBuilder {
	fb.config.Environment = env
	fb.explicitEnv = true // Mark environment as explicitly set
	return fb
}

// WithTestMode enables test mode
func (fb *FactoryBuilder) WithTestMode(enabled bool) *FactoryBuilder {
	fb.config.TestMode = enabled
	return fb
}

// WithCustomRepository allows injection of custom repository implementations
func (fb *FactoryBuilder) WithCustomRepository(name string, repo interface{}) *FactoryBuilder {
	fb.customRepositories[name] = repo
	return fb
}

// Build creates the appropriate repository factory based on configuration
func (fb *FactoryBuilder) Build(ctx context.Context) (RepositoryFactory, error) {
	// Validate configuration
	if err := fb.config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Use environment detector for enhanced environment detection
	detector := NewEnvironmentDetector(fb.config)
	
	// Auto-detect environment only if not explicitly set
	shouldAutoDetect := !fb.explicitEnv && (fb.config.Environment == "" || fb.config.Environment == "development")
	
	if shouldAutoDetect {
		detectedEnv, err := detector.DetectEnvironment(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to detect environment: %w", err)
		}
		fb.config.Environment = detectedEnv
	}

	// Validate environment configuration
	if err := detector.ValidateEnvironmentConfiguration(ctx, fb.config.Environment); err != nil {
		return nil, fmt.Errorf("environment validation failed: %w", err)
	}

	// Select factory implementation based on environment
	switch {
	case fb.config.IsTest():
		return fb.buildTestFactory(ctx)
	case fb.config.IsProduction():
		return fb.buildProductionFactory(ctx)
	case fb.config.IsDevelopment():
		return fb.buildDevelopmentFactory(ctx)
	default:
		return nil, fmt.Errorf("unknown environment: %s", fb.config.Environment)
	}
}

// buildTestFactory creates a test repository factory
func (fb *FactoryBuilder) buildTestFactory(ctx context.Context) (RepositoryFactory, error) {
	return NewTestRepositoryFactory(fb.config, fb.customRepositories)
}

// buildProductionFactory creates a production repository factory with Redis caching
func (fb *FactoryBuilder) buildProductionFactory(ctx context.Context) (RepositoryFactory, error) {
	if fb.config.DatabaseURL == "" {
		return nil, fmt.Errorf("database URL is required for production environment")
	}

	connectionManager, err := NewConnectionManager(ctx, fb.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	return NewProductionRepositoryFactory(fb.config, connectionManager)
}

// buildDevelopmentFactory creates a development repository factory without caching
func (fb *FactoryBuilder) buildDevelopmentFactory(ctx context.Context) (RepositoryFactory, error) {
	if fb.config.DatabaseURL == "" {
		return nil, fmt.Errorf("database URL is required for development environment")
	}

	connectionManager, err := NewConnectionManager(ctx, fb.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	return NewDevelopmentRepositoryFactory(fb.config, connectionManager)
}

// BuildFromEnvironment creates a factory using environment variables
func BuildFromEnvironment(ctx context.Context) (RepositoryFactory, error) {
	config := ConfigFromEnvironment()
	return NewFactoryBuilder().WithConfig(config).Build(ctx)
}

// BuildWithAutoDetection creates a factory with automatic environment detection and dependency checking
func BuildWithAutoDetection(ctx context.Context) (RepositoryFactory, error) {
	config := ConfigFromEnvironment()
	detector := NewEnvironmentDetector(config)
	
	// Get factory recommendation based on available dependencies
	factoryType, reason, err := detector.GetFactoryRecommendation(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get factory recommendation: %w", err)
	}
	
	// Update config with detected environment
	config.Environment = factoryType
	
	// Log the factory selection reasoning
	fmt.Printf("Repository factory auto-detection: Using %s factory - %s\n", factoryType, reason)
	
	return NewFactoryBuilder().WithConfig(config).Build(ctx)
}

// BuildWithFallback creates a factory with automatic fallback to simpler implementations when dependencies are unavailable
func BuildWithFallback(ctx context.Context) (RepositoryFactory, error) {
	config := ConfigFromEnvironment()
	detector := NewEnvironmentDetector(config)
	
	// Try to detect the best environment
	detectedEnv, err := detector.DetectEnvironment(ctx)
	if err != nil {
		// If detection fails, fallback to test environment
		fmt.Printf("Environment detection failed (%v), falling back to test factory\n", err)
		config.Environment = "test"
		config.TestMode = true
		return NewFactoryBuilder().WithConfig(config).Build(ctx)
	}
	
	config.Environment = detectedEnv
	
	// Try to build the factory
	factory, err := NewFactoryBuilder().WithConfig(config).Build(ctx)
	if err != nil {
		// If factory creation fails, try fallback strategies
		return buildWithFallbackStrategies(ctx, config, detector, err)
	}
	
	return factory, nil
}

// buildWithFallbackStrategies implements fallback strategies when primary factory creation fails
func buildWithFallbackStrategies(ctx context.Context, config *RepositoryConfig, detector *EnvironmentDetector, originalErr error) (RepositoryFactory, error) {
	fmt.Printf("Primary factory creation failed (%v), attempting fallback strategies\n", originalErr)
	
	// Strategy 1: If production failed, try development (without Redis)
	if config.Environment == "production" {
		fmt.Printf("Fallback strategy 1: Trying development factory without Redis caching\n")
		config.Environment = "development"
		config.RedisAddr = "" // Disable Redis
		
		if factory, err := NewFactoryBuilder().WithConfig(config).Build(ctx); err == nil {
			fmt.Printf("Fallback successful: Using development factory\n")
			return factory, nil
		}
	}
	
	// Strategy 2: If development failed, try test factory
	if config.Environment == "development" {
		fmt.Printf("Fallback strategy 2: Trying test factory with in-memory repositories\n")
		config.Environment = "test"
		config.TestMode = true
		config.DatabaseURL = "" // Disable database
		config.RedisAddr = ""   // Disable Redis
		
		if factory, err := NewFactoryBuilder().WithConfig(config).Build(ctx); err == nil {
			fmt.Printf("Fallback successful: Using test factory\n")
			return factory, nil
		}
	}
	
	// All fallback strategies failed
	return nil, fmt.Errorf("all factory creation strategies failed, original error: %w", originalErr)
}

