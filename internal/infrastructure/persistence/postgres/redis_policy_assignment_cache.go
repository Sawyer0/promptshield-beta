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

// RedisPolicyAssignmentRepository implements PolicyAssignmentRepository with Redis write-through cache
// Optimized for ListByScope which happens on every policy enforcement request
type RedisPolicyAssignmentRepository struct {
	pg    domain.PolicyAssignmentRepository
	redis *redis.Client
	ttl   time.Duration
}

var redisPolicyAssignmentTracer = otel.Tracer("promptshield/redis/policy_assignments")

func NewRedisPolicyAssignmentRepository(pg domain.PolicyAssignmentRepository, redisClient *redis.Client, ttl time.Duration) domain.PolicyAssignmentRepository {
	if ttl == 0 {
		ttl = 10 * time.Minute // Policy assignments change infrequently
	}
	return &RedisPolicyAssignmentRepository{
		pg:    pg,
		redis: redisClient,
		ttl:   ttl,
	}
}

func (r *RedisPolicyAssignmentRepository) assignmentKey(id uuid.UUID) string {
	return fmt.Sprintf("assignment:%s", id.String())
}

func (r *RedisPolicyAssignmentRepository) tenantScopeKey(tenantID uuid.UUID, scope string) string {
	return fmt.Sprintf("assignments:tenant:%s:scope:%s", tenantID.String(), scope)
}

func (r *RedisPolicyAssignmentRepository) tenantKey(tenantID uuid.UUID) string {
	return fmt.Sprintf("assignments:tenant:%s", tenantID.String())
}

func (r *RedisPolicyAssignmentRepository) policyKey(policyID uuid.UUID) string {
	return fmt.Sprintf("assignments:policy:%s", policyID.String())
}

func (r *RedisPolicyAssignmentRepository) Create(ctx context.Context, assignment *domain.PolicyAssignment) error {
	// Write to PostgreSQL first
	if err := r.pg.Create(ctx, assignment); err != nil {
		return err
	}

	// Cache the assignment and invalidate affected collections
	r.cacheAssignment(ctx, assignment)
	r.invalidateTenantCollections(ctx, assignment.TenantID)
	return nil
}

func (r *RedisPolicyAssignmentRepository) Get(ctx context.Context, id uuid.UUID) (*domain.PolicyAssignment, error) {
	// Check Redis cache first
	key := r.assignmentKey(id)
	ctx, span := tracing.TraceRedisCommand(redisPolicyAssignmentTracer, ctx, "GET", key)
	cached, err := r.redis.Get(ctx, key).Result()
	span.End()

	if err == nil {
		var assignment domain.PolicyAssignment
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

func (r *RedisPolicyAssignmentRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.PolicyAssignment, error) {
	// Check Redis cache first
	key := r.tenantKey(tenantID)
	ctx, span := tracing.TraceRedisCommand(redisPolicyAssignmentTracer, ctx, "GET", key)
	cached, err := r.redis.Get(ctx, key).Result()
	span.End()

	if err == nil {
		var assignments []*domain.PolicyAssignment
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

func (r *RedisPolicyAssignmentRepository) ListByPolicy(ctx context.Context, policyID uuid.UUID) ([]*domain.PolicyAssignment, error) {
	// Check Redis cache first
	key := r.policyKey(policyID)
	ctx, span := tracing.TraceRedisCommand(redisPolicyAssignmentTracer, ctx, "GET", key)
	cached, err := r.redis.Get(ctx, key).Result()
	span.End()

	if err == nil {
		var assignments []*domain.PolicyAssignment
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
func (r *RedisPolicyAssignmentRepository) ListByScope(ctx context.Context, tenantID uuid.UUID, scope string) ([]*domain.PolicyAssignment, error) {
	// Check Redis cache first
	key := r.tenantScopeKey(tenantID, scope)
	ctx, span := tracing.TraceRedisCommand(redisPolicyAssignmentTracer, ctx, "GET", key)
	cached, err := r.redis.Get(ctx, key).Result()
	span.End()

	if err == nil {
		var assignments []*domain.PolicyAssignment
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

func (r *RedisPolicyAssignmentRepository) Update(ctx context.Context, assignment *domain.PolicyAssignment) error {
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

func (r *RedisPolicyAssignmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Get assignment for cache invalidation
	assignment, _ := r.pg.Get(ctx, id)

	// Delete from PostgreSQL
	if err := r.pg.Delete(ctx, id); err != nil {
		return err
	}

	// Remove from cache and invalidate collections
	assignKey := r.assignmentKey(id)
	ctxDel, spanDel := tracing.TraceRedisCommand(redisPolicyAssignmentTracer, ctx, "DEL", assignKey)
	r.redis.Del(ctxDel, assignKey)
	spanDel.End()

	if assignment != nil {
		r.invalidateTenantCollections(ctx, assignment.TenantID)
	}

	return nil
}

func (r *RedisPolicyAssignmentRepository) DeleteByTenantAndPolicy(ctx context.Context, tenantID, policyID uuid.UUID) error {
	// Delete from PostgreSQL
	if err := r.pg.DeleteByTenantAndPolicy(ctx, tenantID, policyID); err != nil {
		return err
	}

	// Invalidate collections (we don't know which specific assignments were deleted)
	r.invalidateTenantCollections(ctx, tenantID)
	policyKey := r.policyKey(policyID)
	ctxDel, spanDel := tracing.TraceRedisCommand(redisPolicyAssignmentTracer, ctx, "DEL", policyKey)
	r.redis.Del(ctxDel, policyKey)
	spanDel.End()

	return nil
}

func (r *RedisPolicyAssignmentRepository) cacheAssignment(ctx context.Context, assignment *domain.PolicyAssignment) {
	data, err := json.Marshal(assignment)
	if err != nil {
		return // Silent fail on cache operations
	}

	key := r.assignmentKey(assignment.ID)
	ctxSet, spanSet := tracing.TraceRedisCommand(redisPolicyAssignmentTracer, ctx, "SET", key)
	r.redis.Set(ctxSet, key, data, r.ttl)
	spanSet.End()
}

func (r *RedisPolicyAssignmentRepository) cacheTenantAssignments(ctx context.Context, tenantID uuid.UUID, assignments []*domain.PolicyAssignment) {
	data, err := json.Marshal(assignments)
	if err != nil {
		return
	}

	key := r.tenantKey(tenantID)
	ctxSet, spanSet := tracing.TraceRedisCommand(redisPolicyAssignmentTracer, ctx, "SET", key)
	r.redis.Set(ctxSet, key, data, r.ttl)
	spanSet.End()
}

func (r *RedisPolicyAssignmentRepository) cachePolicyAssignments(ctx context.Context, policyID uuid.UUID, assignments []*domain.PolicyAssignment) {
	data, err := json.Marshal(assignments)
	if err != nil {
		return
	}

	key := r.policyKey(policyID)
	ctxSet, spanSet := tracing.TraceRedisCommand(redisPolicyAssignmentTracer, ctx, "SET", key)
	r.redis.Set(ctxSet, key, data, r.ttl)
	spanSet.End()
}

func (r *RedisPolicyAssignmentRepository) cacheScopeAssignments(ctx context.Context, tenantID uuid.UUID, scope string, assignments []*domain.PolicyAssignment) {
	data, err := json.Marshal(assignments)
	if err != nil {
		return
	}

	key := r.tenantScopeKey(tenantID, scope)
	ctxSet, spanSet := tracing.TraceRedisCommand(redisPolicyAssignmentTracer, ctx, "SET", key)
	r.redis.Set(ctxSet, key, data, r.ttl)
	spanSet.End()
}

func (r *RedisPolicyAssignmentRepository) invalidateTenantCollections(ctx context.Context, tenantID uuid.UUID) {
	// Invalidate all cached collections for this tenant
	pattern := fmt.Sprintf("assignments:tenant:%s*", tenantID.String())

	// Use SCAN to find and delete matching keys
	ctxScan, spanScan := tracing.TraceRedisCommand(redisPolicyAssignmentTracer, ctx, "SCAN", pattern)
	iter := r.redis.Scan(ctxScan, 0, pattern, 0).Iterator()
	for iter.Next(ctxScan) {
		key := iter.Val()
		ctxDel, spanDel := tracing.TraceRedisCommand(redisPolicyAssignmentTracer, ctxScan, "DEL", key)
		r.redis.Del(ctxDel, key)
		spanDel.End()
	}
	spanScan.End()
}
