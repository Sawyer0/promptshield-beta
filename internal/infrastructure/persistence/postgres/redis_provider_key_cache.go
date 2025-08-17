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

// RedisProviderKeyRepository implements ProviderKeyRepository with Redis write-through cache
// Optimized for GetByAlias which happens when making LLM API calls
type RedisProviderKeyRepository struct {
	pg    domain.ProviderKeyRepository
	redis *redis.Client
	ttl   time.Duration
}

func NewRedisProviderKeyRepository(pg domain.ProviderKeyRepository, redisClient *redis.Client, ttl time.Duration) domain.ProviderKeyRepository {
	if ttl == 0 {
		ttl = 30 * time.Minute // Longer TTL for API keys (change infrequently)
	}
	return &RedisProviderKeyRepository{
		pg:    pg,
		redis: redisClient,
		ttl:   ttl,
	}
}

func (r *RedisProviderKeyRepository) keyKey(id uuid.UUID) string {
	return fmt.Sprintf("providerkey:%s", id.String())
}

func (r *RedisProviderKeyRepository) aliasKey(tenantID uuid.UUID, provider string, alias string) string {
	return fmt.Sprintf("providerkey:alias:%s:%s:%s", tenantID.String(), provider, alias)
}

func (r *RedisProviderKeyRepository) tenantKey(tenantID uuid.UUID) string {
	return fmt.Sprintf("providerkeys:tenant:%s", tenantID.String())
}

func (r *RedisProviderKeyRepository) tenantProviderKey(tenantID uuid.UUID, provider string) string {
	return fmt.Sprintf("providerkeys:tenant:%s:provider:%s", tenantID.String(), provider)
}

func (r *RedisProviderKeyRepository) defaultKeyKey(tenantID uuid.UUID, provider string) string {
	return fmt.Sprintf("providerkey:default:%s:%s", tenantID.String(), provider)
}

func (r *RedisProviderKeyRepository) Create(ctx context.Context, key *domain.ProviderKey) error {
	// Write to PostgreSQL first
	if err := r.pg.Create(ctx, key); err != nil {
		return err
	}

	// Cache the key and invalidate collections
	r.cacheProviderKey(ctx, key)
	r.invalidateTenantCollections(ctx, key.TenantID)
	return nil
}

func (r *RedisProviderKeyRepository) Get(ctx context.Context, id uuid.UUID) (*domain.ProviderKey, error) {
	// Check Redis cache first
	key := r.keyKey(id)
	cached, err := r.redis.Get(ctx, key).Result()
	if err == nil {
		var providerKey domain.ProviderKey
		if json.Unmarshal([]byte(cached), &providerKey) == nil {
			return &providerKey, nil
		}
	}

	// Cache miss - get from PostgreSQL
	providerKey, err := r.pg.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache the result
	r.cacheProviderKey(ctx, providerKey)
	return providerKey, nil
}

// GetByAlias - HOT PATH - called when making LLM API requests
func (r *RedisProviderKeyRepository) GetByAlias(ctx context.Context, tenantID uuid.UUID, provider string, alias string) (*domain.ProviderKey, error) {
	// Check Redis cache first
	key := r.aliasKey(tenantID, provider, alias)
	cached, err := r.redis.Get(ctx, key).Result()
	if err == nil {
		var providerKey domain.ProviderKey
		if json.Unmarshal([]byte(cached), &providerKey) == nil {
			// Update last_used asynchronously
			go r.UpdateLastUsed(context.Background(), providerKey.ID)
			return &providerKey, nil
		}
	}

	// Cache miss - get from PostgreSQL
	providerKey, err := r.pg.GetByAlias(ctx, tenantID, provider, alias)
	if err != nil {
		return nil, err
	}

	// Cache the result
	r.cacheProviderKey(ctx, providerKey)
	return providerKey, nil
}

func (r *RedisProviderKeyRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.ProviderKey, error) {
	// Check Redis cache first
	key := r.tenantKey(tenantID)
	cached, err := r.redis.Get(ctx, key).Result()
	if err == nil {
		var keys []*domain.ProviderKey
		if json.Unmarshal([]byte(cached), &keys) == nil {
			return keys, nil
		}
	}

	// Cache miss - get from PostgreSQL
	keys, err := r.pg.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	r.cacheTenantKeys(ctx, tenantID, keys)
	return keys, nil
}

func (r *RedisProviderKeyRepository) ListByProvider(ctx context.Context, tenantID uuid.UUID, provider string) ([]*domain.ProviderKey, error) {
	// Check Redis cache first
	key := r.tenantProviderKey(tenantID, provider)
	cached, err := r.redis.Get(ctx, key).Result()
	if err == nil {
		var keys []*domain.ProviderKey
		if json.Unmarshal([]byte(cached), &keys) == nil {
			return keys, nil
		}
	}

	// Cache miss - get from PostgreSQL
	keys, err := r.pg.ListByProvider(ctx, tenantID, provider)
	if err != nil {
		return nil, err
	}

	// Cache the result
	r.cacheTenantProviderKeys(ctx, tenantID, provider, keys)
	return keys, nil
}

func (r *RedisProviderKeyRepository) Update(ctx context.Context, key *domain.ProviderKey) error {
	// Write to PostgreSQL first
	if err := r.pg.Update(ctx, key); err != nil {
		return err
	}

	// Update cache and invalidate collections
	r.cacheProviderKey(ctx, key)
	r.invalidateTenantCollections(ctx, key.TenantID)
	return nil
}

func (r *RedisProviderKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Get key for cache invalidation
	key, _ := r.pg.Get(ctx, id)

	// Delete from PostgreSQL
	if err := r.pg.Delete(ctx, id); err != nil {
		return err
	}

	// Remove from cache and invalidate collections
	r.redis.Del(ctx, r.keyKey(id))
	if key != nil {
		r.invalidateProviderKey(ctx, key)
		r.invalidateTenantCollections(ctx, key.TenantID)
	}
	
	return nil
}

func (r *RedisProviderKeyRepository) Rotate(ctx context.Context, id uuid.UUID, newEncryptedKey string) error {
	// Write to PostgreSQL first
	if err := r.pg.Rotate(ctx, id, newEncryptedKey); err != nil {
		return err
	}

	// Get updated key and refresh cache
	key, err := r.pg.Get(ctx, id)
	if err == nil {
		r.cacheProviderKey(ctx, key)
	}
	
	return nil
}

func (r *RedisProviderKeyRepository) SetDefault(ctx context.Context, tenantID uuid.UUID, provider string, keyID uuid.UUID) error {
	// Write to PostgreSQL first
	if err := r.pg.SetDefault(ctx, tenantID, provider, keyID); err != nil {
		return err
	}

	// Invalidate collections (default status changed)
	r.invalidateTenantCollections(ctx, tenantID)
	
	// Cache the new default key
	if key, err := r.pg.Get(ctx, keyID); err == nil {
		r.cacheDefaultKey(ctx, tenantID, provider, key)
	}
	
	return nil
}

func (r *RedisProviderKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	// Update PostgreSQL
	if err := r.pg.UpdateLastUsed(ctx, id); err != nil {
		return err
	}

	// Update cached key's last_used field if it exists
	key := r.keyKey(id)
	cached, err := r.redis.Get(ctx, key).Result()
	if err == nil {
		var providerKey domain.ProviderKey
		if json.Unmarshal([]byte(cached), &providerKey) == nil {
			now := time.Now()
			providerKey.LastUsed = &now
			r.cacheProviderKey(ctx, &providerKey)
		}
	}

	return nil
}

func (r *RedisProviderKeyRepository) cacheProviderKey(ctx context.Context, key *domain.ProviderKey) {
	data, err := json.Marshal(key)
	if err != nil {
		return // Silent fail on cache operations
	}

	// Cache key by ID
	keyKey := r.keyKey(key.ID)
	r.redis.Set(ctx, keyKey, data, r.ttl)

	// Cache alias->key mapping (most important for API calls)
	aliasKey := r.aliasKey(key.TenantID, string(key.Provider), key.KeyAlias)
	r.redis.Set(ctx, aliasKey, data, r.ttl)

	// Cache default key if this is the default
	if key.IsDefault {
		r.cacheDefaultKey(ctx, key.TenantID, string(key.Provider), key)
	}
}

func (r *RedisProviderKeyRepository) cacheDefaultKey(ctx context.Context, tenantID uuid.UUID, provider string, key *domain.ProviderKey) {
	data, err := json.Marshal(key)
	if err != nil {
		return
	}

	defaultKey := r.defaultKeyKey(tenantID, provider)
	r.redis.Set(ctx, defaultKey, data, r.ttl)
}

func (r *RedisProviderKeyRepository) cacheTenantKeys(ctx context.Context, tenantID uuid.UUID, keys []*domain.ProviderKey) {
	data, err := json.Marshal(keys)
	if err != nil {
		return
	}

	key := r.tenantKey(tenantID)
	r.redis.Set(ctx, key, data, r.ttl)
}

func (r *RedisProviderKeyRepository) cacheTenantProviderKeys(ctx context.Context, tenantID uuid.UUID, provider string, keys []*domain.ProviderKey) {
	data, err := json.Marshal(keys)
	if err != nil {
		return
	}

	key := r.tenantProviderKey(tenantID, provider)
	r.redis.Set(ctx, key, data, r.ttl)
}

func (r *RedisProviderKeyRepository) invalidateProviderKey(ctx context.Context, key *domain.ProviderKey) {
	r.redis.Del(ctx, r.keyKey(key.ID))
	r.redis.Del(ctx, r.aliasKey(key.TenantID, string(key.Provider), key.KeyAlias))
	if key.IsDefault {
		r.redis.Del(ctx, r.defaultKeyKey(key.TenantID, string(key.Provider)))
	}
}

func (r *RedisProviderKeyRepository) invalidateTenantCollections(ctx context.Context, tenantID uuid.UUID) {
	// Invalidate all cached collections for this tenant
	pattern := fmt.Sprintf("providerkeys:tenant:%s*", tenantID.String())
	
	// Use SCAN to find and delete matching keys
	iter := r.redis.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		r.redis.Del(ctx, iter.Val())
	}
}