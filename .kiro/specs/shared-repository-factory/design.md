# Design Document

## Overview

The shared repository factory will provide a centralized, type-safe, and environment-aware system for creating and managing all repository instances across the application. This design eliminates the current scattered repository initialization patterns found in `gateway/main.go`, test files, and service constructors by providing a unified factory interface that handles connection management, caching, and environment-specific configurations.

The factory will support multiple deployment scenarios: production (PostgreSQL with Redis caching), development (PostgreSQL without caching), testing (in-memory or mock repositories), and graceful degradation when external dependencies are unavailable.

## Architecture

### Core Components

1. **RepositoryFactory Interface**: Defines the contract for creating all repository types
2. **ProductionRepositoryFactory**: PostgreSQL-backed repositories with Redis caching
3. **DevelopmentRepositoryFactory**: PostgreSQL-backed repositories without caching
4. **TestRepositoryFactory**: In-memory and mock repositories for testing
5. **RepositoryConfig**: Configuration structure for factory initialization
6. **ConnectionManager**: Manages database connections and Redis clients

### Factory Pattern

The factory uses a builder pattern with environment detection to automatically select the appropriate implementation:

```go
type RepositoryFactory interface {
    // Core domain repositories
    Tenant() domain.TenantRepository
    Audit() domain.AuditRepository
    RulepackAssignment() domain.RulepackAssignmentRepository
    APIToken() domain.APITokenRepository
    Settings() domain.SettingsRepository
    
    // Business logic repositories
    Rulepack() contracts.RulepackRepository
    
    // Lifecycle management
    Close() error
    HealthCheck(ctx context.Context) error
}
```

### Environment Detection

The factory automatically detects the runtime environment and selects appropriate implementations:

- **Production**: Detected by presence of Redis configuration
- **Development**: Detected by PostgreSQL configuration without Redis
- **Testing**: Detected by test environment variables or explicit test mode
- **Fallback**: Graceful degradation when dependencies are unavailable

## Components and Interfaces

### RepositoryConfig

```go
type RepositoryConfig struct {
    // Database configuration
    DatabaseURL string
    
    // Redis configuration (optional)
    RedisAddr string
    RedisPassword string
    RedisDB int
    
    // Cache TTL settings
    TenantCacheTTL time.Duration
    AssignmentCacheTTL time.Duration
    TokenCacheTTL time.Duration
    
    // Environment settings
    Environment string // "production", "development", "test"
    TestMode bool
    
    // Connection pool settings
    MaxConnections int
    MaxIdleConnections int
    ConnectionTimeout time.Duration
}
```

### ConnectionManager

```go
type ConnectionManager struct {
    pgPool *postgres.Pool
    redisClient *redis.Client
    config *RepositoryConfig
}

func (cm *ConnectionManager) PostgresPool() *postgres.Pool
func (cm *ConnectionManager) RedisClient() *redis.Client
func (cm *ConnectionManager) Close() error
func (cm *ConnectionManager) HealthCheck(ctx context.Context) error
```

### Factory Implementations

#### ProductionRepositoryFactory
- Uses PostgreSQL with Redis caching for hot-path optimization
- Implements connection pooling and retry logic
- Provides metrics and monitoring hooks
- Handles graceful degradation when Redis is unavailable

#### DevelopmentRepositoryFactory
- Uses PostgreSQL without caching for simpler debugging
- Provides detailed logging and error reporting
- Supports database migrations and schema validation

#### TestRepositoryFactory
- Provides in-memory repositories for unit tests
- Supports mock repositories for integration tests
- Enables test isolation and cleanup
- Allows custom test data seeding

## Data Models

### Repository Registry

The factory maintains a registry of all repository types and their implementations:

```go
type RepositoryRegistry struct {
    repositories map[string]interface{}
    connections *ConnectionManager
    config *RepositoryConfig
}
```

### Factory Builder

```go
type FactoryBuilder struct {
    config *RepositoryConfig
    customRepositories map[string]interface{}
}

func NewFactoryBuilder() *FactoryBuilder
func (fb *FactoryBuilder) WithConfig(config *RepositoryConfig) *FactoryBuilder
func (fb *FactoryBuilder) WithCustomRepository(name string, repo interface{}) *FactoryBuilder
func (fb *FactoryBuilder) Build(ctx context.Context) (RepositoryFactory, error)
```

## Error Handling

### Connection Failures
- PostgreSQL connection failures result in clear error messages with retry suggestions
- Redis connection failures trigger fallback to direct PostgreSQL access
- Network timeouts are handled with exponential backoff

### Repository Creation Errors
- Type mismatches are caught at compile time through interface enforcement
- Configuration validation occurs at factory creation time
- Missing dependencies are reported with actionable error messages

### Graceful Degradation
- Redis unavailability falls back to PostgreSQL-only mode
- Database connection issues trigger in-memory fallback for tests
- Partial failures are logged with detailed context

## Testing Strategy

### Unit Testing
- Each factory implementation has comprehensive unit tests
- Mock dependencies are injected through interfaces
- Configuration validation is thoroughly tested
- Error scenarios are explicitly tested

### Integration Testing
- Real database connections are tested in CI/CD
- Redis caching behavior is validated
- Connection pooling and lifecycle management are tested
- Performance characteristics are benchmarked

### Test Utilities
```go
// Test factory creation helpers
func NewTestFactory(t *testing.T) RepositoryFactory
func NewTestFactoryWithMocks(t *testing.T, mocks map[string]interface{}) RepositoryFactory
func NewIntegrationTestFactory(t *testing.T, dbURL string) RepositoryFactory

// Test cleanup helpers
func CleanupTestFactory(t *testing.T, factory RepositoryFactory)
func ResetTestData(t *testing.T, factory RepositoryFactory)
```

### Mock Repository Support
- Standardized mock implementations for all repository interfaces
- Test data builders for common scenarios
- Assertion helpers for repository interactions
- Automatic cleanup and isolation between tests

## Implementation Phases

### Phase 1: Core Factory Structure
- Create base interfaces and configuration structures
- Implement connection management
- Create factory builder pattern
- Add basic error handling

### Phase 2: Production Implementation
- Implement PostgreSQL-backed repositories
- Add Redis caching layer
- Implement connection pooling
- Add health checks and monitoring

### Phase 3: Test Support
- Create test factory implementations
- Add mock repository support
- Implement test utilities and helpers
- Add integration test support

### Phase 4: Migration and Cleanup
- Replace existing repository initialization patterns
- Update all service constructors
- Remove duplicate initialization code
- Add comprehensive documentation

## Migration Strategy

### Backward Compatibility
- Existing repository interfaces remain unchanged
- Current initialization patterns continue to work during transition
- Gradual migration path with feature flags

### Rollout Plan
1. Deploy factory alongside existing code
2. Migrate test code first to validate functionality
3. Migrate service constructors one by one
4. Update main application entry points
5. Remove deprecated initialization patterns

### Risk Mitigation
- Feature flags allow quick rollback
- Comprehensive test coverage prevents regressions
- Gradual rollout minimizes blast radius
- Monitoring and alerting detect issues early