# Repository Factory Pattern

This document describes the repository factory pattern implemented in PromptShield for managing database connections and repository instances.

## Overview

The repository factory pattern provides a centralized way to create and manage repository instances with proper dependency injection, connection management, and environment-specific configurations.

## Factory Types

### ProductionRepositoryFactory
- **Use Case**: Production environments with PostgreSQL and Redis
- **Features**: 
  - PostgreSQL-backed repositories
  - Redis caching for hot-path repositories (tenant, assignment, token)
  - Connection pooling and health checks
  - Comprehensive monitoring and metrics

### DevelopmentRepositoryFactory  
- **Use Case**: Development environments
- **Features**:
  - PostgreSQL-backed repositories without caching
  - Enhanced logging and debugging
  - Graceful fallback when Redis is unavailable
  - Detailed error reporting

### TestRepositoryFactory
- **Use Case**: Unit tests and integration tests
- **Features**:
  - In-memory repository implementations
  - Fast test execution
  - Isolated test environments
  - Mock implementations for all repository types

## Usage Examples

### Creating a Factory

```go
// Production factory
config := &repository.RepositoryConfig{
    DatabaseURL: "postgres://user:pass@localhost:5432/db",
    RedisAddr:   "localhost:6379",
    Environment: "production",
}

cm, err := repository.NewConnectionManager(ctx, config)
if err != nil {
    return err
}

factory, err := repository.NewProductionRepositoryFactory(config, cm)
if err != nil {
    return err
}
defer factory.Close()
```

### Using Factory with Services

```go
// Create services using factory
rulepackService := services.RulepackServiceFromFactory(factory, publisher)
policyService := services.PolicyServiceFromFactory(factory)

// Or create all services at once
allServices := services.NewServicesFromFactory(factory, publisher)
```

### Test Factory Usage

```go
// In tests
factory, err := repository.NewTestRepositoryFactory(nil, nil)
if err != nil {
    t.Fatalf("Failed to create test factory: %v", err)
}

service := services.RulepackServiceFromFactory(factory, nil)
```

## Configuration

### RepositoryConfig

```go
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
```

### Environment Detection

The factory system includes automatic environment detection:

```go
// Auto-detect environment based on configuration
factory, err := repository.BuildWithAutoDetection(config)
```

Detection logic:
- **Production**: Database + Redis available
- **Development**: Database available, Redis optional
- **Test**: No external dependencies required

## Health Checks and Monitoring

### Health Check Endpoints

```go
// HTTP health check handler
http.HandleFunc("/health", repository.HTTPHealthHandler(factory))
http.HandleFunc("/ready", repository.HTTPReadinessHandler(factory))
http.HandleFunc("/live", repository.HTTPLivenessHandler(factory))
```

### Monitoring

```go
// Start background monitoring
factory.StartMonitoring(ctx)

// Get metrics
monitor := factory.GetMonitor()
metrics := monitor.GetMetrics()
```

## Migration from Legacy Patterns

### Before (Deprecated)

```go
// Old pattern - direct repository creation
repo := memory.NewRulepackRepository()
service := services.RulepackServiceCstor(repo, publisher)

// Or direct PostgreSQL
repo := pg.RulepackRepo(pool)
service := services.RulepackServiceCstor(repo, publisher)
```

### After (Recommended)

```go
// New pattern - factory-based
factory, err := repository.NewTestRepositoryFactory(nil, nil)
if err != nil {
    return err
}
service := services.RulepackServiceFromFactory(factory, publisher)
```

## Benefits

1. **Centralized Configuration**: All repository configuration in one place
2. **Environment Flexibility**: Easy switching between production, development, and test environments
3. **Connection Management**: Proper connection pooling and lifecycle management
4. **Health Monitoring**: Built-in health checks and metrics
5. **Error Handling**: Comprehensive error handling with proper error types
6. **Testing**: Simplified test setup with consistent mock implementations
7. **Dependency Injection**: Clean dependency injection without tight coupling

## Best Practices

1. **Use Factory Pattern**: Always use factory methods instead of direct repository creation
2. **Proper Cleanup**: Always call `factory.Close()` to clean up resources
3. **Health Checks**: Implement health check endpoints for production deployments
4. **Monitoring**: Enable monitoring for production environments
5. **Environment Detection**: Use auto-detection for flexible deployments
6. **Error Handling**: Handle factory creation errors appropriately
7. **Testing**: Use TestRepositoryFactory for all unit tests

## Troubleshooting

### Common Issues

1. **Connection Failures**: Check database URL and network connectivity
2. **Redis Unavailable**: Development factory gracefully falls back to no caching
3. **Factory Creation Errors**: Check configuration validation and dependencies
4. **Health Check Failures**: Verify database and Redis connectivity

### Debugging

Enable debug logging:
```go
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
```

Check factory metrics:
```go
monitor := factory.GetMonitor()
monitor.LogMetricsSummary()
```