package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
	"github.com/promptshield/promptshield/internal/util/tracing"
	redis "github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
)

// RedisAPITokenRepository implements APITokenRepository with Redis write-through cache
// Optimized for token validation (GetByHash) which happens on every request
type RedisAPITokenRepository struct {
	pg    APITokenRepository
	redis *redis.Client
	ttl   time.Duration
}

var redisTokenCacheTracer = otel.Tracer("promptshield/redis/api_tokens")

func NewRedisAPITokenRepository(pg APITokenRepository, redisClient *redis.Client, ttl time.Duration) APITokenRepository {
	if ttl == 0 {
		ttl = 30 * time.Minute // Longer TTL for auth tokens
	}
	return &RedisAPITokenRepository{
		pg:    pg,
		redis: redisClient,
		ttl:   ttl,
	}
}

func (r *RedisAPITokenRepository) tokenKey(id uuid.UUID) string {
	return fmt.Sprintf("token:%s", id.String())
}

func (r *RedisAPITokenRepository) tokenHashKey(hash string) string {
	return fmt.Sprintf("token:hash:%s", hash)
}

// Removed unused function tenantTokensKey

func (r *RedisAPITokenRepository) Create(ctx context.Context, token *domain.APIToken) error {
	// Write to PostgreSQL first
	if err := r.pg.Create(ctx, token); err != nil {
		return err
	}

	// Cache the token
	r.cacheToken(ctx, token)
	return nil
}

func (r *RedisAPITokenRepository) Get(ctx context.Context, id uuid.UUID) (*domain.APIToken, error) {
	// Check Redis cache first
	key := r.tokenKey(id)
	ctx, span := tracing.TraceRedisCommand(redisTokenCacheTracer, ctx, "GET", key)
	cached, err := r.redis.Get(ctx, key).Result()
	span.End()

	if err == nil {
		var token domain.APIToken
		if json.Unmarshal([]byte(cached), &token) == nil {
			return &token, nil
		}
	}

	// Cache miss - get from PostgreSQL
	token, err := r.pg.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache the result
	r.cacheToken(ctx, token)
	return token, nil
}

// GetByHash - CRITICAL HOT PATH - happens on every authenticated request
func (r *RedisAPITokenRepository) GetByHash(ctx context.Context, tokenHash string) (*domain.APIToken, error) {
	// Check hash->token mapping in Redis first
	hashKey := r.tokenHashKey(tokenHash)
	ctx, span := tracing.TraceRedisCommand(redisTokenCacheTracer, ctx, "GET", hashKey)
	cached, err := r.redis.Get(ctx, hashKey).Result()
	span.End()

	if err == nil {
		var token domain.APIToken
		if json.Unmarshal([]byte(cached), &token) == nil {
			// Update last_used asynchronously to avoid blocking the hot path
			go func() {
				_ = r.UpdateLastUsed(context.Background(), token.ID)
			}()
			return &token, nil
		}
	}

	// Cache miss - get from PostgreSQL (this should be rare in production)
	token, err := r.pg.GetByHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}

	// Cache the result for future requests
	r.cacheToken(ctx, token)
	return token, nil
}

func (r *RedisAPITokenRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.APIToken, error) {
	// List operations go to PostgreSQL (admin operations, not hot path)
	return r.pg.ListByTenant(ctx, tenantID)
}

func (r *RedisAPITokenRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	// Update PostgreSQL
	if err := r.pg.UpdateLastUsed(ctx, id); err != nil {
		return err
	}

	// Update cached token's last_used field if it exists
	key := r.tokenKey(id)
	ctx, span := tracing.TraceRedisCommand(redisTokenCacheTracer, ctx, "GET", key)
	cached, err := r.redis.Get(ctx, key).Result()
	span.End()

	if err == nil {
		var token domain.APIToken
		if json.Unmarshal([]byte(cached), &token) == nil {
			now := time.Now()
			token.LastUsed = &now
			r.cacheToken(ctx, &token)
		}
	}

	return nil
}

func (r *RedisAPITokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	// Get token for cache invalidation
	token, err := r.pg.Get(ctx, id)
	if err != nil {
		return err
	}

	// Revoke in PostgreSQL
	if err := r.pg.Revoke(ctx, id); err != nil {
		return err
	}

	// Invalidate cache immediately
	r.invalidateToken(ctx, token)
	return nil
}

func (r *RedisAPITokenRepository) DeleteExpired(ctx context.Context) error {
	// This is a maintenance operation, just delegate to PostgreSQL
	// We could implement cache cleanup here but tokens naturally expire
	return r.pg.DeleteExpired(ctx)
}

func (r *RedisAPITokenRepository) cacheToken(ctx context.Context, token *domain.APIToken) {
	// Only cache non-revoked, non-expired tokens
	if token.RevokedAt != nil {
		return
	}
	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
		return
	}

	data, err := json.Marshal(token)
	if err != nil {
		return // Silent fail on cache operations
	}

	// Cache token by ID
	tokenKey := r.tokenKey(token.ID)
	ctxSetID, spanSetID := tracing.TraceRedisCommand(redisTokenCacheTracer, ctx, "SET", tokenKey)
	r.redis.Set(ctxSetID, tokenKey, data, r.ttl)
	spanSetID.End()

	// Cache hash->token mapping (most important for auth)
	hashKey := r.tokenHashKey(token.TokenHash)
	ctxSetHash, spanSetHash := tracing.TraceRedisCommand(redisTokenCacheTracer, ctx, "SET", hashKey)
	r.redis.Set(ctxSetHash, hashKey, data, r.ttl)
	spanSetHash.End()
}

func (r *RedisAPITokenRepository) invalidateToken(ctx context.Context, token *domain.APIToken) {
	ctxDelID, spanDelID := tracing.TraceRedisCommand(redisTokenCacheTracer, ctx, "DEL", r.tokenKey(token.ID))
	r.redis.Del(ctxDelID, r.tokenKey(token.ID))
	spanDelID.End()
	ctxDelHash, spanDelHash := tracing.TraceRedisCommand(redisTokenCacheTracer, ctx, "DEL", r.tokenHashKey(token.TokenHash))
	r.redis.Del(ctxDelHash, r.tokenHashKey(token.TokenHash))
	spanDelHash.End()
}
