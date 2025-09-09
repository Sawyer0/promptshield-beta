package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
	redis "github.com/redis/go-redis/v9"
)

// RedisTenantRepository implements TenantRepository with Redis write-through cache
// Hot path: Check Redis first, fallback to PostgreSQL, cache the result
type RedisTenantRepository struct {
	pg    TenantRepository // PostgreSQL source of truth
	redis *redis.Client
	ttl   time.Duration // Cache TTL
}

func NewRedisTenantRepository(pg TenantRepository, redisClient *redis.Client, ttl time.Duration) TenantRepository {
	if ttl == 0 {
		ttl = 15 * time.Minute // Default cache TTL
	}
	return &RedisTenantRepository{
		pg:    pg,
		redis: redisClient,
		ttl:   ttl,
	}
}

func (r *RedisTenantRepository) tenantKey(id uuid.UUID) string {
	return fmt.Sprintf("tenant:%s", id.String())
}

func (r *RedisTenantRepository) tenantNameKey(name string) string {
	return fmt.Sprintf("tenant:name:%s", name)
}

func (r *RedisTenantRepository) Create(ctx context.Context, tenant *domain.Tenant) error {
	// Write to PostgreSQL first
	if err := r.pg.Create(ctx, tenant); err != nil {
		return err
	}

	// Cache the created tenant
	r.cacheTenant(ctx, tenant)
	return nil
}

func (r *RedisTenantRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	// Check Redis cache first
	key := r.tenantKey(id)
	cached, err := r.redis.Get(ctx, key).Result()
	if err == nil {
		var tenant domain.Tenant
		if json.Unmarshal([]byte(cached), &tenant) == nil {
			return &tenant, nil
		}
	}

	// Cache miss - get from PostgreSQL
	tenant, err := r.pg.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache the result
	r.cacheTenant(ctx, tenant)
	return tenant, nil
}

func (r *RedisTenantRepository) GetByName(ctx context.Context, name string) (*domain.Tenant, error) {
	// Check name->ID mapping in Redis
	nameKey := r.tenantNameKey(name)
	idStr, err := r.redis.Get(ctx, nameKey).Result()
	if err == nil {
		if id, parseErr := uuid.Parse(idStr); parseErr == nil {
			return r.Get(ctx, id) // This will use cache if available
		}
	}

	// Cache miss - get from PostgreSQL
	tenant, err := r.pg.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}

	// Cache the result
	r.cacheTenant(ctx, tenant)
	return tenant, nil
}

func (r *RedisTenantRepository) List(ctx context.Context, offset, limit int) ([]*domain.Tenant, int, error) {
	// List operations always go to PostgreSQL (not frequently cached)
	return r.pg.List(ctx, offset, limit)
}

func (r *RedisTenantRepository) Update(ctx context.Context, tenant *domain.Tenant) error {
	// Write to PostgreSQL first
	if err := r.pg.Update(ctx, tenant); err != nil {
		return err
	}

	// Invalidate cache and re-cache updated tenant
	r.invalidateTenant(ctx, tenant.ID, tenant.Name)
	r.cacheTenant(ctx, tenant)
	return nil
}

func (r *RedisTenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Get tenant name for cache invalidation
	tenant, err := r.pg.Get(ctx, id)
	if err != nil {
		return err
	}

	// Delete from PostgreSQL
	if err := r.pg.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate cache
	r.invalidateTenant(ctx, id, tenant.Name)
	return nil
}

// GetByExternalOrg resolves tenant via external organization mapping, with read-through cache
func (r *RedisTenantRepository) GetByExternalOrg(ctx context.Context, provider string, externalOrgID string) (*domain.Tenant, error) {
	// External mapping is low-volume; delegate to PostgreSQL repository
	t, err := r.pg.GetByExternalOrg(ctx, provider, externalOrgID)
	if err != nil {
		return nil, err
	}
	// Cache resolved tenant
	r.cacheTenant(ctx, t)
	return t, nil
}

// LinkExternalOrg upserts external organization mapping and invalidates cache
func (r *RedisTenantRepository) LinkExternalOrg(ctx context.Context, provider string, externalOrgID string, tenantID uuid.UUID) error {
	if err := r.pg.LinkExternalOrg(ctx, provider, externalOrgID, tenantID); err != nil {
		return err
	}
	// Invalidate cached tenant; fetch to warm cache
	if t, err := r.pg.Get(ctx, tenantID); err == nil {
		r.invalidateTenant(ctx, tenantID, t.Name)
		r.cacheTenant(ctx, t)
	}
	return nil
}

func (r *RedisTenantRepository) cacheTenant(ctx context.Context, tenant *domain.Tenant) {
	data, err := json.Marshal(tenant)
	if err != nil {
		return // Silent fail on cache operations
	}

	// Cache tenant by ID
	tenantKey := r.tenantKey(tenant.ID)
	r.redis.Set(ctx, tenantKey, data, r.ttl)

	// Cache name->ID mapping
	nameKey := r.tenantNameKey(tenant.Name)
	r.redis.Set(ctx, nameKey, tenant.ID.String(), r.ttl)
}

func (r *RedisTenantRepository) invalidateTenant(ctx context.Context, id uuid.UUID, name string) {
	r.redis.Del(ctx, r.tenantKey(id))
	if name != "" {
		r.redis.Del(ctx, r.tenantNameKey(name))
	}
}
