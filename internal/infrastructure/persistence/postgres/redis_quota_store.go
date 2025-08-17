package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/go-redis/redis_rate/v10"
	redis "github.com/redis/go-redis/v9"
	"github.com/promptshield/promptshield/internal/domain"
)

// RedisQuotaRepository implements QuotaRepository with Redis-based rate limiting
// Uses go-redis/redis_rate library for robust rate limiting
type RedisQuotaRepository struct {
	pg      domain.QuotaRepository
	redis   *redis.Client
	limiter *redis_rate.Limiter
	ttl     time.Duration
}

func NewRedisQuotaRepository(pg domain.QuotaRepository, redisClient *redis.Client, ttl time.Duration) domain.QuotaRepository {
	if ttl == 0 {
		ttl = 5 * time.Minute // Quota config cache TTL
	}
	return &RedisQuotaRepository{
		pg:      pg,
		redis:   redisClient,
		limiter: redis_rate.NewLimiter(redisClient),
		ttl:     ttl,
	}
}

func (r *RedisQuotaRepository) quotaKey(tenantID uuid.UUID) string {
	return fmt.Sprintf("quota:config:%s", tenantID.String())
}


func (r *RedisQuotaRepository) Create(ctx context.Context, quota *domain.Quota) error {
	// Write to PostgreSQL first
	if err := r.pg.Create(ctx, quota); err != nil {
		return err
	}

	// Cache the quota config
	r.cacheQuota(ctx, quota)
	return nil
}

func (r *RedisQuotaRepository) Get(ctx context.Context, tenantID uuid.UUID) (*domain.Quota, error) {
	// Check Redis cache first
	key := r.quotaKey(tenantID)
	cached, err := r.redis.Get(ctx, key).Result()
	if err == nil {
		var quota domain.Quota
		if json.Unmarshal([]byte(cached), &quota) == nil {
			return &quota, nil
		}
	}

	// Cache miss - get from PostgreSQL
	quota, err := r.pg.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	r.cacheQuota(ctx, quota)
	return quota, nil
}

func (r *RedisQuotaRepository) Update(ctx context.Context, quota *domain.Quota) error {
	// Write to PostgreSQL first
	if err := r.pg.Update(ctx, quota); err != nil {
		return err
	}

	// Update cache
	r.cacheQuota(ctx, quota)
	return nil
}

func (r *RedisQuotaRepository) Delete(ctx context.Context, tenantID uuid.UUID) error {
	// Delete from PostgreSQL
	if err := r.pg.Delete(ctx, tenantID); err != nil {
		return err
	}

	// Remove from cache
	r.redis.Del(ctx, r.quotaKey(tenantID))
	return nil
}

// CheckRateLimit checks if tenant can make a request within rate limits
// This is the HOT PATH method called on every request - now uses redis_rate library
func (r *RedisQuotaRepository) CheckRateLimit(ctx context.Context, tenantID uuid.UUID) (*domain.RateLimitResult, error) {
	// Get quota config
	quota, err := r.Get(ctx, tenantID)
	if err != nil {
		// If no quota found, allow by default
		return &domain.RateLimitResult{Allowed: true}, nil
	}

	result := &domain.RateLimitResult{Allowed: true}

	// Check requests per minute using redis_rate library
	if quota.RequestsPerMinute != nil && *quota.RequestsPerMinute > 0 {
		key := fmt.Sprintf("tenant:%s:rpm", tenantID.String())
		res, err := r.limiter.Allow(ctx, key, redis_rate.PerMinute(*quota.RequestsPerMinute))
		if err != nil {
			return nil, fmt.Errorf("rate limit check failed: %w", err)
		}
		
		if res.Allowed == 0 {
			result.Allowed = false
			result.RetryAfter = res.RetryAfter
		}
		result.RequestsPerMinuteRemaining = int(res.Remaining)
	}

	// Check requests per hour using redis_rate library
	if quota.RequestsPerHour != nil && *quota.RequestsPerHour > 0 {
		key := fmt.Sprintf("tenant:%s:rph", tenantID.String())
		res, err := r.limiter.Allow(ctx, key, redis_rate.PerHour(*quota.RequestsPerHour))
		if err != nil {
			return nil, fmt.Errorf("rate limit check failed: %w", err)
		}
		
		if res.Allowed == 0 {
			result.Allowed = false
			if result.RetryAfter == 0 || res.RetryAfter < result.RetryAfter {
				result.RetryAfter = res.RetryAfter
			}
		}
		result.RequestsPerHourRemaining = int(res.Remaining)
	}

	return result, nil
}

// IncrementUsage increments token usage for a tenant
func (r *RedisQuotaRepository) IncrementUsage(ctx context.Context, tenantID uuid.UUID, tokens int64) error {
	// Delegate to postgres for persistent usage tracking
	return r.pg.IncrementUsage(ctx, tenantID, tokens)
}

func (r *RedisQuotaRepository) cacheQuota(ctx context.Context, quota *domain.Quota) {
	data, err := json.Marshal(quota)
	if err != nil {
		return // Silent fail on cache operations
	}

	key := r.quotaKey(quota.TenantID)
	r.redis.Set(ctx, key, data, r.ttl)
}

// RateLimitResult represents the result of a rate limit check
type RateLimitResult struct {
	Allowed                     bool
	RequestsPerMinuteRemaining  int
	RequestsPerHourRemaining    int
	RetryAfter                  time.Duration
}