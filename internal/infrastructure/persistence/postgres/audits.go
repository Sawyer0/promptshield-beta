package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

// Ensure pgAuditRepo implements domain.AuditRepository
var _ domain.AuditRepository = (*pgAuditRepo)(nil)

type pgAuditRepo struct{ db *Pool }

func AuditRepo(db *Pool) domain.AuditRepository { return &pgAuditRepo{db: db} }

func (r *pgAuditRepo) Create(ctx context.Context, entry *domain.AuditEntry) error {
	q := `INSERT INTO audits (id, tenant_id, actor_id, actor_type, actor_email, action, object_type, object_id, before_data, after_data, metadata, hash, prev_hash) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	
	_, err := r.db.Raw().Exec(ctx, q, 
		entry.ID,
		entry.TenantID, 
		entry.ActorID,
		entry.ActorType,
		entry.ActorEmail, 
		entry.Action, 
		entry.ObjectType, 
		entry.ObjectID, 
		entry.Before, 
		entry.After,
		entry.Metadata,
		entry.Hash,
		entry.PrevHash,
	)
	if err != nil {
		return fmt.Errorf("create audit: %w", err)
	}
	return nil
}

func (r *pgAuditRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*domain.AuditEntry, int, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	
	// Count total for pagination
	countQuery := `SELECT COUNT(*) FROM audits WHERE tenant_id = $1`
	var total int
	if err := r.db.Raw().QueryRow(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audits: %w", err)
	}

	q := `SELECT id, tenant_id, actor_id, actor_type, actor_email, action, object_type, object_id, before_data, after_data, metadata, created_at, hash, prev_hash
		FROM audits 
		WHERE tenant_id = $1 
		ORDER BY created_at DESC 
		OFFSET $2 LIMIT $3`
	
	rows, err := r.db.Raw().Query(ctx, q, tenantID, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list audits: %w", err)
	}
	defer rows.Close()

	var audits []*domain.AuditEntry
	for rows.Next() {
		var a domain.AuditEntry
		if err := rows.Scan(&a.ID, &a.TenantID, &a.ActorID, &a.ActorType, &a.ActorEmail, 
			&a.Action, &a.ObjectType, &a.ObjectID, &a.Before, &a.After, &a.Metadata, &a.CreatedAt, &a.Hash, &a.PrevHash); err != nil {
			return nil, 0, fmt.Errorf("scan audit: %w", err)
		}
		audits = append(audits, &a)
	}
	
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	
	return audits, total, nil
}

func (r *pgAuditRepo) Get(ctx context.Context, id uuid.UUID) (*domain.AuditEntry, error) {
	q := `SELECT id, tenant_id, actor_id, actor_type, actor_email, action, object_type, object_id, before_data, after_data, metadata, created_at, hash, prev_hash
		FROM audits WHERE id = $1`
	
	var a domain.AuditEntry
	err := r.db.Raw().QueryRow(ctx, q, id).Scan(&a.ID, &a.TenantID, &a.ActorID, &a.ActorType, &a.ActorEmail, 
		&a.Action, &a.ObjectType, &a.ObjectID, &a.Before, &a.After, &a.Metadata, &a.CreatedAt, &a.Hash, &a.PrevHash)
	if err != nil {
		return nil, fmt.Errorf("get audit: %w", err)
	}
	return &a, nil
}

func (r *pgAuditRepo) ListByObject(ctx context.Context, objectType string, objectID uuid.UUID, offset, limit int) ([]*domain.AuditEntry, int, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	
	// Count total for pagination
	countQuery := `SELECT COUNT(*) FROM audits WHERE object_type = $1 AND object_id = $2`
	var total int
	if err := r.db.Raw().QueryRow(ctx, countQuery, objectType, objectID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audits by object: %w", err)
	}

	q := `SELECT id, tenant_id, actor_id, actor_type, actor_email, action, object_type, object_id, before_data, after_data, metadata, created_at, hash, prev_hash
		FROM audits 
		WHERE object_type = $1 AND object_id = $2
		ORDER BY created_at DESC 
		OFFSET $3 LIMIT $4`
	
	rows, err := r.db.Raw().Query(ctx, q, objectType, objectID, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list audits by object: %w", err)
	}
	defer rows.Close()

	var audits []*domain.AuditEntry
	for rows.Next() {
		var a domain.AuditEntry
		if err := rows.Scan(&a.ID, &a.TenantID, &a.ActorID, &a.ActorType, &a.ActorEmail, 
			&a.Action, &a.ObjectType, &a.ObjectID, &a.Before, &a.After, &a.Metadata, &a.CreatedAt, &a.Hash, &a.PrevHash); err != nil {
			return nil, 0, fmt.Errorf("scan audit: %w", err)
		}
		audits = append(audits, &a)
	}
	
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	
	return audits, total, nil
}

func (r *pgAuditRepo) ListByAction(ctx context.Context, action string, offset, limit int) ([]*domain.AuditEntry, int, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	
	// Count total for pagination
	countQuery := `SELECT COUNT(*) FROM audits WHERE action = $1`
	var total int
	if err := r.db.Raw().QueryRow(ctx, countQuery, action).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audits by action: %w", err)
	}

	q := `SELECT id, tenant_id, actor_id, actor_type, actor_email, action, object_type, object_id, before_data, after_data, metadata, created_at, hash, prev_hash
		FROM audits 
		WHERE action = $1
		ORDER BY created_at DESC 
		OFFSET $2 LIMIT $3`
	
	rows, err := r.db.Raw().Query(ctx, q, action, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list audits by action: %w", err)
	}
	defer rows.Close()

	var audits []*domain.AuditEntry
	for rows.Next() {
		var a domain.AuditEntry
		if err := rows.Scan(&a.ID, &a.TenantID, &a.ActorID, &a.ActorType, &a.ActorEmail, 
			&a.Action, &a.ObjectType, &a.ObjectID, &a.Before, &a.After, &a.Metadata, &a.CreatedAt, &a.Hash, &a.PrevHash); err != nil {
			return nil, 0, fmt.Errorf("scan audit: %w", err)
		}
		audits = append(audits, &a)
	}
	
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	
	return audits, total, nil
}