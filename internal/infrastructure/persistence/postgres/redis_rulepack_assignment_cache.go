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

// RedisRulepackAssignmentRepository implements RulepackAssignmentRepository with Redis write-through cache
// Optimized for ListByScope which happens on every policy enforcement request
type RedisRulepackAssignmentRepository struct {
	pg    domain.RulepackAssignmentRepository
	redis *redis.Client
	ttl   time.Duration
}

func NewRedisRulepackAssignmentRepository(pg domain.RulepackAssignmentRepository, redisClient *redis.Client, ttl time.Duration) domain.RulepackAssignmentRepository {
	if ttl == 0 {
		ttl = 10 * time.Minute // Rulepack assignments change infrequently
	}
	return &RedisRulepackAssignmentRepository{
		pg:    pg,
		redis: redisClient,
		ttl:   ttl,
	}
}

func (r *RedisRulepackAssignmentRepository) assignmentKey(id uuid.UUID) string {
	return fmt.Sprintf("assignment:%s", id.String())
}

func (r *RedisRulepackAssignmentRepository) tenantScopeKey(tenantID uuid.UUID, scope string) string {
	return fmt.Sprintf("assignments:tenant:%s:scope:%s", tenantID.String(), scope)
}

func (r *RedisRulepackAssignmentRepository) tenantKey(tenantID uuid.UUID) string {
	return fmt.Sprintf("assignments:tenant:%s", tenantID.String())
}

func (r *RedisRulepackAssignmentRepository) policyKey(policyID uuid.UUID) string {
	return fmt.Sprintf("assignments:policy:%s", policyID.String())
}

func (r *RedisRulepackAssignmentRepository) Create(ctx context.Context, assignment *domain.RulepackAssignment) error {
	// Write to PostgreSQL first
	if err := r.pg.Create(ctx, assignment); err != nil {
		return err
	}

	// Cache the assignment and invalidate affected collections
	r.cacheAssignment(ctx, assignment)
	r.invalidateTenantCollections(ctx, assignment.TenantID)
	return nil
}

func (r *RedisRulepackAssignmentRepository) Get(ctx context.Context, id uuid.UUID) (*domain.RulepackAssignment, error) {
	// Check Redis cache first
	key := r.assignmentKey(id)
	cached, err := r.redis.Get(ctx, key).Result()
	if err == nil {
		var assignment domain.RulepackAssignment
		if json.Unmarshal([]byte(cached), &assignment) == nil {
			return &assignment, nil
		}
	}

	// Cache miss - get from PostgreSQL
	assignment, err := r.pg.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache the result
	r.cacheAssignment(ctx, assignment)
	return assignment, nil
}

func (r *RedisRulepackAssignmentRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.RulepackAssignment, error) {
	// Check Redis cache first
	key := r.tenantKey(tenantID)
	cached, err := r.redis.Get(ctx, key).Result()
	if err == nil {
		var assignments []*domain.RulepackAssignment
		if json.Unmarshal([]byte(cached), &assignments) == nil {
			return assignments, nil
		}
	}

	// Cache miss - get from PostgreSQL
	assignments, err := r.pg.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	r.cacheTenantAssignments(ctx, tenantID, assignments)
	return assignments, nil
}

func (r *RedisRulepackAssignmentRepository) ListByPolicy(ctx context.Context, policyID uuid.UUID) ([]*domain.RulepackAssignment, error) {
	// Check Redis cache first
	key := r.policyKey(policyID)
	cached, err := r.redis.Get(ctx, key).Result()
	if err == nil {
		var assignments []*domain.RulepackAssignment
		if json.Unmarshal([]byte(cached), &assignments) == nil {
			return assignments, nil
		}
	}

	// Cache miss - get from PostgreSQL
	assignments, err := r.pg.ListByPolicy(ctx, policyID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	r.cachePolicyAssignments(ctx, policyID, assignments)
	return assignments, nil
}

// ListByScope - CRITICAL HOT PATH - called for every policy enforcement
func (r *RedisRulepackAssignmentRepository) ListByScope(ctx context.Context, tenantID uuid.UUID, scope string) ([]*domain.RulepackAssignment, error) {
	// Check Redis cache first
	key := r.tenantScopeKey(tenantID, scope)
	cached, err := r.redis.Get(ctx, key).Result()
	if err == nil {
		var assignments []*domain.RulepackAssignment
		if json.Unmarshal([]byte(cached), &assignments) == nil {
			return assignments, nil
		}
	}

	// Cache miss - get from PostgreSQL
	assignments, err := r.pg.ListByScope(ctx, tenantID, scope)
	if err != nil {
		return nil, err
	}

	// Cache the result (this is the most important cache for performance)
	r.cacheScopeAssignments(ctx, tenantID, scope, assignments)
	return assignments, nil
}

func (r *RedisRulepackAssignmentRepository) Update(ctx context.Context, assignment *domain.RulepackAssignment) error {
	// Get old assignment for cache invalidation
	oldAssignment, _ := r.pg.Get(ctx, assignment.ID)

	// Write to PostgreSQL first
	if err := r.pg.Update(ctx, assignment); err != nil {
		return err
	}

	// Update cache and invalidate affected collections
	r.cacheAssignment(ctx, assignment)
	r.invalidateTenantCollections(ctx, assignment.TenantID)

	// If tenant changed, invalidate old tenant collections too
	if oldAssignment != nil && oldAssignment.TenantID != assignment.TenantID {
		r.invalidateTenantCollections(ctx, oldAssignment.TenantID)
	}

	return nil
}

func (r *RedisRulepackAssignmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Get assignment for cache invalidation
	assignment, _ := r.pg.Get(ctx, id)

	// Delete from PostgreSQL
	if err := r.pg.Delete(ctx, id); err != nil {
		return err
	}

	// Remove from cache and invalidate collections
	r.redis.Del(ctx, r.assignmentKey(id))
	if assignment != nil {
		r.invalidateTenantCollections(ctx, assignment.TenantID)
	}

	return nil
}

func (r *RedisRulepackAssignmentRepository) DeleteByTenantAndPolicy(ctx context.Context, tenantID, policyID uuid.UUID) error {
	// Delete from PostgreSQL
	if err := r.pg.DeleteByTenantAndPolicy(ctx, tenantID, policyID); err != nil {
		return err
	}

	// Invalidate collections (we don't know which specific assignments were deleted)
	r.invalidateTenantCollections(ctx, tenantID)
	r.redis.Del(ctx, r.policyKey(policyID))

	return nil
}

func (r *RedisRulepackAssignmentRepository) cacheAssignment(ctx context.Context, assignment *domain.RulepackAssignment) {
	data, err := json.Marshal(assignment)
	if err != nil {
		return // Silent fail on cache operations
	}

	key := r.assignmentKey(assignment.ID)
	r.redis.Set(ctx, key, data, r.ttl)
}

func (r *RedisRulepackAssignmentRepository) cacheTenantAssignments(ctx context.Context, tenantID uuid.UUID, assignments []*domain.RulepackAssignment) {
	data, err := json.Marshal(assignments)
	if err != nil {
		return
	}

	key := r.tenantKey(tenantID)
	r.redis.Set(ctx, key, data, r.ttl)
}

func (r *RedisRulepackAssignmentRepository) cachePolicyAssignments(ctx context.Context, policyID uuid.UUID, assignments []*domain.RulepackAssignment) {
	data, err := json.Marshal(assignments)
	if err != nil {
		return
	}

	key := r.policyKey(policyID)
	r.redis.Set(ctx, key, data, r.ttl)
}

func (r *RedisRulepackAssignmentRepository) cacheScopeAssignments(ctx context.Context, tenantID uuid.UUID, scope string, assignments []*domain.RulepackAssignment) {
	data, err := json.Marshal(assignments)
	if err != nil {
		return
	}

	key := r.tenantScopeKey(tenantID, scope)
	r.redis.Set(ctx, key, data, r.ttl)
}

func (r *RedisRulepackAssignmentRepository) invalidateTenantCollections(ctx context.Context, tenantID uuid.UUID) {
	// Invalidate all cached collections for this tenant
	pattern := fmt.Sprintf("assignments:tenant:%s*", tenantID.String())

	// Use SCAN to find and delete matching keys
	iter := r.redis.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		r.redis.Del(ctx, iter.Val())
	}
}
