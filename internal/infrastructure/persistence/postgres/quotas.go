package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/promptshield/promptshield/internal/domain"
)


type pgQuotaRepo struct{ db *Pool }

func QuotaRepo(db *Pool) domain.QuotaRepository { return &pgQuotaRepo{db: db} }

func (r *pgQuotaRepo) Create(ctx context.Context, quota *domain.Quota) error {
	q := `INSERT INTO quotas (id, tenant_id, requests_per_minute, requests_per_hour, tokens_per_hour, max_prompt_tokens, max_completion_tokens, updated_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (tenant_id) DO UPDATE SET 
			requests_per_minute = $3,
			requests_per_hour = $4,
			tokens_per_hour = $5,
			max_prompt_tokens = $6,
			max_completion_tokens = $7,
			updated_at = $8`
	
	_, err := r.db.Raw().Exec(ctx, q,
		quota.ID,
		quota.TenantID,
		quota.RequestsPerMinute,
		quota.RequestsPerHour,
		quota.TokensPerHour,
		quota.MaxPromptTokens,
		quota.MaxCompletionTokens,
		quota.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create quota: %w", err)
	}
	return nil
}

func (r *pgQuotaRepo) Get(ctx context.Context, tenantID uuid.UUID) (*domain.Quota, error) {
	var quota domain.Quota
	q := `SELECT id, tenant_id, requests_per_minute, requests_per_hour, tokens_per_hour, max_prompt_tokens, max_completion_tokens, updated_at
		FROM quotas WHERE tenant_id = $1`
	
	err := r.db.Raw().QueryRow(ctx, q, tenantID).Scan(
		&quota.ID,
		&quota.TenantID,
		&quota.RequestsPerMinute,
		&quota.RequestsPerHour,
		&quota.TokensPerHour,
		&quota.MaxPromptTokens,
		&quota.MaxCompletionTokens,
		&quota.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("quota not found for tenant")
		}
		return nil, fmt.Errorf("get quota: %w", err)
	}
	return &quota, nil
}

func (r *pgQuotaRepo) Update(ctx context.Context, quota *domain.Quota) error {
	q := `UPDATE quotas SET 
		requests_per_minute = $2,
		requests_per_hour = $3,
		tokens_per_hour = $4,
		max_prompt_tokens = $5,
		max_completion_tokens = $6,
		updated_at = $7
		WHERE tenant_id = $1`
	
	result, err := r.db.Raw().Exec(ctx, q,
		quota.TenantID,
		quota.RequestsPerMinute,
		quota.RequestsPerHour,
		quota.TokensPerHour,
		quota.MaxPromptTokens,
		quota.MaxCompletionTokens,
		quota.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update quota: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("quota not found")
	}
	return nil
}

func (r *pgQuotaRepo) Delete(ctx context.Context, tenantID uuid.UUID) error {
	q := `DELETE FROM quotas WHERE tenant_id = $1`
	result, err := r.db.Raw().Exec(ctx, q, tenantID)
	if err != nil {
		return fmt.Errorf("delete quota: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("quota not found")
	}
	return nil
}

// CheckRateLimit - stub implementation (Redis layer handles real rate limiting)
func (r *pgQuotaRepo) CheckRateLimit(ctx context.Context, tenantID uuid.UUID) (*domain.RateLimitResult, error) {
	// Basic postgres fallback - just check if quota exists
	_, err := r.Get(ctx, tenantID)
	if err != nil {
		// If no quota found, allow by default
		return &domain.RateLimitResult{Allowed: true}, nil
	}
	// Always allow for postgres fallback (Redis layer does real limiting)
	return &domain.RateLimitResult{Allowed: true}, nil
}

// IncrementUsage - stub implementation  
func (r *pgQuotaRepo) IncrementUsage(ctx context.Context, tenantID uuid.UUID, tokens int64) error {
	// Postgres fallback - no-op (Redis layer handles real usage tracking)
	return nil
}