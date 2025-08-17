package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lib/pq"
	"github.com/promptshield/promptshield/internal/domain"
)

type APITokenRepository interface {
	Create(ctx context.Context, token *domain.APIToken) error
	Get(ctx context.Context, id uuid.UUID) (*domain.APIToken, error)
	GetByHash(ctx context.Context, tokenHash string) (*domain.APIToken, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.APIToken, error)
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
	Revoke(ctx context.Context, id uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}

type pgAPITokenRepo struct{ db *Pool }

func APITokenRepo(db *Pool) APITokenRepository { return &pgAPITokenRepo{db: db} }

func (r *pgAPITokenRepo) Create(ctx context.Context, token *domain.APIToken) error {
	q := `INSERT INTO api_tokens (id, tenant_id, token_hash, name, scopes, expires_at, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	
	_, err := r.db.Raw().Exec(ctx, q,
		token.ID,
		token.TenantID,
		token.TokenHash,
		token.Name,
		pq.Array(token.Scopes),
		token.ExpiresAt,
		token.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create api token: %w", err)
	}
	return nil
}

func (r *pgAPITokenRepo) Get(ctx context.Context, id uuid.UUID) (*domain.APIToken, error) {
	var token domain.APIToken
	q := `SELECT id, tenant_id, token_hash, name, scopes, expires_at, last_used, created_at, revoked_at
		FROM api_tokens WHERE id = $1`
	
	err := r.db.Raw().QueryRow(ctx, q, id).Scan(
		&token.ID,
		&token.TenantID,
		&token.TokenHash,
		&token.Name,
		pq.Array(&token.Scopes),
		&token.ExpiresAt,
		&token.LastUsed,
		&token.CreatedAt,
		&token.RevokedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("api token not found")
		}
		return nil, fmt.Errorf("get api token: %w", err)
	}
	return &token, nil
}

func (r *pgAPITokenRepo) GetByHash(ctx context.Context, tokenHash string) (*domain.APIToken, error) {
	var token domain.APIToken
	q := `SELECT id, tenant_id, token_hash, name, scopes, expires_at, last_used, created_at, revoked_at
		FROM api_tokens 
		WHERE token_hash = $1 
		AND revoked_at IS NULL 
		AND (expires_at IS NULL OR expires_at > NOW())`
	
	err := r.db.Raw().QueryRow(ctx, q, tokenHash).Scan(
		&token.ID,
		&token.TenantID,
		&token.TokenHash,
		&token.Name,
		pq.Array(&token.Scopes),
		&token.ExpiresAt,
		&token.LastUsed,
		&token.CreatedAt,
		&token.RevokedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("api token not found or invalid")
		}
		return nil, fmt.Errorf("get api token by hash: %w", err)
	}
	return &token, nil
}

func (r *pgAPITokenRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.APIToken, error) {
	q := `SELECT id, tenant_id, token_hash, name, scopes, expires_at, last_used, created_at, revoked_at
		FROM api_tokens WHERE tenant_id = $1 ORDER BY created_at DESC`
	
	rows, err := r.db.Raw().Query(ctx, q, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list api tokens by tenant: %w", err)
	}
	defer rows.Close()

	var tokens []*domain.APIToken
	for rows.Next() {
		var token domain.APIToken
		err := rows.Scan(
			&token.ID,
			&token.TenantID,
			&token.TokenHash,
			&token.Name,
			pq.Array(&token.Scopes),
			&token.ExpiresAt,
			&token.LastUsed,
			&token.CreatedAt,
			&token.RevokedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan api token: %w", err)
		}
		tokens = append(tokens, &token)
	}
	return tokens, rows.Err()
}

func (r *pgAPITokenRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	q := `UPDATE api_tokens SET last_used = $2 WHERE id = $1`
	_, err := r.db.Raw().Exec(ctx, q, id, time.Now())
	if err != nil {
		return fmt.Errorf("update last used: %w", err)
	}
	return nil
}

func (r *pgAPITokenRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	q := `UPDATE api_tokens SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL`
	result, err := r.db.Raw().Exec(ctx, q, id, time.Now())
	if err != nil {
		return fmt.Errorf("revoke api token: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("api token not found or already revoked")
	}
	return nil
}

// Delete removes an API token by ID
func (r *pgAPITokenRepo) Delete(ctx context.Context, id uuid.UUID) error {
	q := `DELETE FROM api_tokens WHERE id = $1`
	result, err := r.db.Raw().Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete api token: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("api token not found")
	}
	return nil
}


func (r *pgAPITokenRepo) DeleteExpired(ctx context.Context) error {
	q := `DELETE FROM api_tokens WHERE expires_at < NOW()`
	_, err := r.db.Raw().Exec(ctx, q)
	if err != nil {
		return fmt.Errorf("delete expired tokens: %w", err)
	}
	return nil
}