package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/promptshield/promptshield/internal/domain"
)

// RedisAPITokenRepository implements APITokenRepository with Redis write-through cache
// Optimized for token validation (GetByHash) which happens on every request
type RedisAPITokenRepository struct {
	pg    domain.APITokenRepository
	redis *redis.Client
	ttl   time.Duration
}

func NewRedisAPITokenRepository(pg domain.APITokenRepository, redisClient *redis.Client, ttl time.Duration) domain.APITokenRepository {
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
	cached, err := r.redis.Get(ctx, key).Result()
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
	cached, err := r.redis.Get(ctx, hashKey).Result()
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
	cached, err := r.redis.Get(ctx, key).Result()
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
	r.redis.Set(ctx, tokenKey, data, r.ttl)

	// Cache hash->token mapping (most important for auth)
	hashKey := r.tokenHashKey(token.TokenHash)
	r.redis.Set(ctx, hashKey, data, r.ttl)
}

func (r *RedisAPITokenRepository) invalidateToken(ctx context.Context, token *domain.APIToken) {
	r.redis.Del(ctx, r.tokenKey(token.ID))
	r.redis.Del(ctx, r.tokenHashKey(token.TokenHash))
}

// Update updates an API token and invalidates cache
func (r *RedisAPITokenRepository) Update(ctx context.Context, token *domain.APIToken) error {
	// Update PostgreSQL first
	if err := r.pg.Update(ctx, token); err != nil {
		return err
	}
	
	// Invalidate cache for this token
	r.invalidateToken(ctx, token)
	
	return nil
}

// Delete removes an API token and invalidates cache
func (r *RedisAPITokenRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Get token first to invalidate cache properly
	token, err := r.pg.Get(ctx, id)
	if err != nil {
		return err
	}
	
	// Delete from PostgreSQL
	if err := r.pg.Delete(ctx, id); err != nil {
		return err
	}
	
	// Invalidate cache
	r.invalidateToken(ctx, token)
	
	return nil
}

// Rotate rotates an API token and invalidates cache
func (r *RedisAPITokenRepository) Rotate(ctx context.Context, id uuid.UUID) (string, error) {
	// Get token first to invalidate cache properly
	token, err := r.pg.Get(ctx, id)
	if err != nil {
		return "", err
	}
	
	// Rotate in PostgreSQL
	newToken, err := r.pg.Rotate(ctx, id)
	if err != nil {
		return "", err
	}
	
	// Invalidate old cache entries
	r.invalidateToken(ctx, token)
	
	return newToken, nil
}

