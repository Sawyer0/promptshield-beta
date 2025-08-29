package api

import (
	"context"
	"fmt"
	"time"

	"github.com/promptshield/promptshield/internal/domain"
	"github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
	"github.com/promptshield/promptshield/internal/usage"
	redis "github.com/redis/go-redis/v9"
)

// Repositories creates PostgreSQL repository implementations for Security Gateway
func Repositories(ctx context.Context, databaseURL string) (domain.TenantRepository, domain.RulepackAssignmentRepository, domain.AuditRepository, error) {
	// Connect to PostgreSQL
	pool, err := postgres.NewPool(ctx, databaseURL)
	if err != nil {
		return nil, nil, nil, err
	}

	// Create PostgreSQL repository implementations for security business data
	tenantRepo := postgres.TenantRepo(pool)
	assignmentRepo := postgres.RulepackAssignmentRepo(pool) // rulepack assignments
	auditRepo := postgres.AuditRepo(pool)

	return tenantRepo, assignmentRepo, auditRepo, nil
}

// ProductionRepositories creates Redis-cached repositories with PostgreSQL backing
func ProductionRepositories(ctx context.Context, databaseURL, redisAddr string) (*ProductionRepositoryFactory, error) {
	// Connect to PostgreSQL
	pool, err := postgres.NewPool(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connect to redis: %w", err)
	}

	return &ProductionRepositoryFactory{
		pool:  pool,
		redis: redisClient,
	}, nil
}

// ProductionRepositoryFactory creates Redis-cached repositories for hot path optimization
type ProductionRepositoryFactory struct {
	pool  *postgres.Pool
	redis *redis.Client
}

func (f *ProductionRepositoryFactory) Tenant() domain.TenantRepository {
	pgRepo := postgres.TenantRepo(f.pool)
	return postgres.NewRedisTenantRepository(pgRepo, f.redis, 15*time.Minute)
}

func (f *ProductionRepositoryFactory) RulepackAssignment() domain.RulepackAssignmentRepository {
	pgRepo := postgres.RulepackAssignmentRepo(f.pool)
	return postgres.NewRedisRulepackAssignmentRepository(pgRepo, f.redis, 10*time.Minute)
}

func (f *ProductionRepositoryFactory) Audit() domain.AuditRepository {
	// Audit logs don't need caching - they're write-heavy, read-light
	return postgres.AuditRepo(f.pool)
}

// Security Gateway - no complex quota/provider key management needed

// ProductionOptions creates Options with Redis-cached repositories for maximum performance
func ProductionOptions(ctx context.Context, databaseURL, redisAddr string) (Options, error) {
	// Create optimized repository factory
	factory, err := ProductionRepositories(ctx, databaseURL, redisAddr)
	if err != nil {
		return Options{}, err
	}

	// Create Redis usage store
	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
	redisUsageStore := usage.NewRedisUsageStore(redisClient, "ps", 35*24*time.Hour)

	return Options{
		TenantRepository:     factory.Tenant(),
		AssignmentRepository: factory.RulepackAssignment(),
		AuditRepository:      factory.Audit(),
		// Security Gateway - no complex quota/provider key management needed
		UsageStore: redisUsageStore,
		// Security Gateway uses simple token auth only
		OnDrain:    nil,
		OnShutdown: nil,
	}, nil
}
