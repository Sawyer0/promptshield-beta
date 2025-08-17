package api

import (
	"context"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/promptshield/promptshield/internal/domain"
	"github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
	"github.com/promptshield/promptshield/internal/usage"
)

// Repositories creates PostgreSQL repository implementations for production
func Repositories(ctx context.Context, databaseURL string) (domain.TenantRepository, domain.PolicyAssignmentRepository, domain.AuditRepository, domain.ProviderKeyRepository, error) {
	// Connect to PostgreSQL
	pool, err := postgres.NewPool(ctx, databaseURL)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	
	// Create PostgreSQL repository implementations for business data
	tenantRepo := postgres.TenantRepo(pool)
	assignmentRepo := postgres.PolicyAssignmentRepo(pool)
	auditRepo := postgres.AuditRepo(pool)
	
	// Provider key repository uses PostgreSQL storage
	providerKeyRepo := postgres.ProviderKeyRepo(pool)
	
	return tenantRepo, assignmentRepo, auditRepo, providerKeyRepo, nil
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

func (f *ProductionRepositoryFactory) PolicyAssignment() domain.PolicyAssignmentRepository {
	pgRepo := postgres.PolicyAssignmentRepo(f.pool)
	return postgres.NewRedisPolicyAssignmentRepository(pgRepo, f.redis, 10*time.Minute)
}

func (f *ProductionRepositoryFactory) Audit() domain.AuditRepository {
	// Audit logs don't need caching - they're write-heavy, read-light
	return postgres.AuditRepo(f.pool)
}

func (f *ProductionRepositoryFactory) ProviderKey() domain.ProviderKeyRepository {
	pgRepo := postgres.ProviderKeyRepo(f.pool)
	return postgres.NewRedisProviderKeyRepository(pgRepo, f.redis, 30*time.Minute)
}

func (f *ProductionRepositoryFactory) Quota() domain.QuotaRepository {
	pgRepo := postgres.QuotaRepo(f.pool)
	return postgres.NewRedisQuotaRepository(pgRepo, f.redis, 5*time.Minute)
}


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
		AssignmentRepository: factory.PolicyAssignment(),
		AuditRepository:      factory.Audit(),
		ProviderKeyStore:     factory.ProviderKey(),
		QuotaRepository:      factory.Quota(),
		UsageStore:          redisUsageStore,
		OIDC: OIDCConfig{
			Issuer:   "",
			Audience: "",
		},
		OnDrain:    nil,
		OnShutdown: nil,
	}, nil
}