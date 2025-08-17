package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/promptshield/promptshield/internal/domain"
)


type pgProviderKeyRepo struct{ db *Pool }

func ProviderKeyRepo(db *Pool) domain.ProviderKeyRepository { return &pgProviderKeyRepo{db: db} }

func (r *pgProviderKeyRepo) Create(ctx context.Context, key *domain.ProviderKey) error {
	q := `INSERT INTO provider_keys (id, tenant_id, provider, key_alias, encrypted_key, endpoint, deployment, is_default, status, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	
	_, err := r.db.Raw().Exec(ctx, q,
		key.ID,
		key.TenantID,
		key.Provider,
		key.KeyAlias,
		key.EncryptedKey,
		key.Endpoint,
		key.Deployment,
		key.IsDefault,
		key.Status,
		key.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create provider key: %w", err)
	}
	return nil
}

func (r *pgProviderKeyRepo) Get(ctx context.Context, id uuid.UUID) (*domain.ProviderKey, error) {
	var key domain.ProviderKey
	q := `SELECT id, tenant_id, provider, key_alias, encrypted_key, endpoint, deployment, is_default, status, created_at, last_used, rotated_at
		FROM provider_keys WHERE id = $1`
	
	err := r.db.Raw().QueryRow(ctx, q, id).Scan(
		&key.ID,
		&key.TenantID,
		&key.Provider,
		&key.KeyAlias,
		&key.EncryptedKey,
		&key.Endpoint,
		&key.Deployment,
		&key.IsDefault,
		&key.Status,
		&key.CreatedAt,
		&key.LastUsed,
		&key.RotatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("provider key not found")
		}
		return nil, fmt.Errorf("get provider key: %w", err)
	}
	return &key, nil
}

func (r *pgProviderKeyRepo) GetByAlias(ctx context.Context, tenantID uuid.UUID, provider string, alias string) (*domain.ProviderKey, error) {
	var key domain.ProviderKey
	q := `SELECT id, tenant_id, provider, key_alias, encrypted_key, endpoint, deployment, is_default, status, created_at, last_used, rotated_at
		FROM provider_keys WHERE tenant_id = $1 AND provider = $2 AND key_alias = $3`
	
	err := r.db.Raw().QueryRow(ctx, q, tenantID, provider, alias).Scan(
		&key.ID,
		&key.TenantID,
		&key.Provider,
		&key.KeyAlias,
		&key.EncryptedKey,
		&key.Endpoint,
		&key.Deployment,
		&key.IsDefault,
		&key.Status,
		&key.CreatedAt,
		&key.LastUsed,
		&key.RotatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("provider key not found")
		}
		return nil, fmt.Errorf("get provider key by alias: %w", err)
	}
	return &key, nil
}

func (r *pgProviderKeyRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.ProviderKey, error) {
	q := `SELECT id, tenant_id, provider, key_alias, encrypted_key, endpoint, deployment, is_default, status, created_at, last_used, rotated_at
		FROM provider_keys WHERE tenant_id = $1 ORDER BY provider, key_alias`
	
	rows, err := r.db.Raw().Query(ctx, q, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list provider keys by tenant: %w", err)
	}
	defer rows.Close()

	var keys []*domain.ProviderKey
	for rows.Next() {
		var key domain.ProviderKey
		err := rows.Scan(
			&key.ID,
			&key.TenantID,
			&key.Provider,
			&key.KeyAlias,
			&key.EncryptedKey,
			&key.Endpoint,
			&key.Deployment,
			&key.IsDefault,
			&key.Status,
			&key.CreatedAt,
			&key.LastUsed,
			&key.RotatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan provider key: %w", err)
		}
		keys = append(keys, &key)
	}
	return keys, rows.Err()
}

func (r *pgProviderKeyRepo) ListByProvider(ctx context.Context, tenantID uuid.UUID, provider string) ([]*domain.ProviderKey, error) {
	q := `SELECT id, tenant_id, provider, key_alias, encrypted_key, endpoint, deployment, is_default, status, created_at, last_used, rotated_at
		FROM provider_keys WHERE tenant_id = $1 AND provider = $2 ORDER BY key_alias`
	
	rows, err := r.db.Raw().Query(ctx, q, tenantID, provider)
	if err != nil {
		return nil, fmt.Errorf("list provider keys by provider: %w", err)
	}
	defer rows.Close()

	var keys []*domain.ProviderKey
	for rows.Next() {
		var key domain.ProviderKey
		err := rows.Scan(
			&key.ID,
			&key.TenantID,
			&key.Provider,
			&key.KeyAlias,
			&key.EncryptedKey,
			&key.Endpoint,
			&key.Deployment,
			&key.IsDefault,
			&key.Status,
			&key.CreatedAt,
			&key.LastUsed,
			&key.RotatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan provider key: %w", err)
		}
		keys = append(keys, &key)
	}
	return keys, rows.Err()
}

func (r *pgProviderKeyRepo) Update(ctx context.Context, key *domain.ProviderKey) error {
	q := `UPDATE provider_keys SET 
		key_alias = $2, endpoint = $3, deployment = $4, is_default = $5, status = $6
		WHERE id = $1`
	
	result, err := r.db.Raw().Exec(ctx, q,
		key.ID,
		key.KeyAlias,
		key.Endpoint,
		key.Deployment,
		key.IsDefault,
		key.Status,
	)
	if err != nil {
		return fmt.Errorf("update provider key: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("provider key not found")
	}
	return nil
}

func (r *pgProviderKeyRepo) Delete(ctx context.Context, id uuid.UUID) error {
	q := `DELETE FROM provider_keys WHERE id = $1`
	result, err := r.db.Raw().Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete provider key: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("provider key not found")
	}
	return nil
}

func (r *pgProviderKeyRepo) Rotate(ctx context.Context, id uuid.UUID, newEncryptedKey string) error {
	q := `UPDATE provider_keys SET encrypted_key = $2, rotated_at = $3 WHERE id = $1`
	result, err := r.db.Raw().Exec(ctx, q, id, newEncryptedKey, time.Now())
	if err != nil {
		return fmt.Errorf("rotate provider key: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("provider key not found")
	}
	return nil
}

func (r *pgProviderKeyRepo) SetDefault(ctx context.Context, tenantID uuid.UUID, provider string, keyID uuid.UUID) error {
	tx, err := r.db.Raw().Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Clear existing default for this tenant/provider
	_, err = tx.Exec(ctx, 
		`UPDATE provider_keys SET is_default = false WHERE tenant_id = $1 AND provider = $2`,
		tenantID, provider)
	if err != nil {
		return fmt.Errorf("clear existing default: %w", err)
	}

	// Set new default
	result, err := tx.Exec(ctx,
		`UPDATE provider_keys SET is_default = true WHERE id = $1 AND tenant_id = $2 AND provider = $3`,
		keyID, tenantID, provider)
	if err != nil {
		return fmt.Errorf("set new default: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("provider key not found or tenant/provider mismatch")
	}

	return tx.Commit(ctx)
}

func (r *pgProviderKeyRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	q := `UPDATE provider_keys SET last_used = $2 WHERE id = $1`
	_, err := r.db.Raw().Exec(ctx, q, id, time.Now())
	if err != nil {
		return fmt.Errorf("update last used: %w", err)
	}
	return nil
}