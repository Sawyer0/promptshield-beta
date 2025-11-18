package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/promptshield/promptshield/internal/domain"
	"github.com/promptshield/promptshield/internal/util/tracing"
	"go.opentelemetry.io/otel"
)

var tenantTracer = otel.Tracer("promptshield/postgres/tenants")

type TenantRepository interface {
	Create(ctx context.Context, tenant *domain.Tenant) error
	Get(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
	GetByName(ctx context.Context, name string) (*domain.Tenant, error)
	List(ctx context.Context, offset, limit int) ([]*domain.Tenant, int, error)
	Update(ctx context.Context, tenant *domain.Tenant) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgTenantRepo struct{ db *Pool }

func TenantRepo(db *Pool) TenantRepository { return &pgTenantRepo{db: db} }

func (r *pgTenantRepo) Create(ctx context.Context, tenant *domain.Tenant) error {
	ctx, span := tracing.TraceDatabaseQuery(tenantTracer, ctx, "INSERT", "tenants")
	defer span.End()
	if tenant.ID == uuid.Nil {
		tenant.ID = uuid.New()
	}
	if tenant.CreatedAt.IsZero() {
		tenant.CreatedAt = time.Now()
	}
	if tenant.UpdatedAt.IsZero() {
		tenant.UpdatedAt = time.Now()
	}
	if tenant.Status == "" {
		tenant.Status = domain.TenantStatusActive
	}

	q := `INSERT INTO tenants (id, name, status, metadata, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.Raw().Exec(ctx, q,
		tenant.ID, tenant.Name, tenant.Status, tenant.Metadata,
		tenant.CreatedAt, tenant.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create tenant: %w", err)
	}
	return nil
}

func (r *pgTenantRepo) Get(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	ctx, span := tracing.TraceDatabaseQuery(tenantTracer, ctx, "SELECT", "tenants")
	defer span.End()
	var t domain.Tenant
	q := `SELECT id, name, status, metadata, created_at, updated_at 
		FROM tenants WHERE id = $1`

	err := r.db.Raw().QueryRow(ctx, q, id).Scan(
		&t.ID, &t.Name, &t.Status, &t.Metadata, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, fmt.Errorf("get tenant: %w", err)
	}
	return &t, nil
}

func (r *pgTenantRepo) GetByName(ctx context.Context, name string) (*domain.Tenant, error) {
	ctx, span := tracing.TraceDatabaseQuery(tenantTracer, ctx, "SELECT", "tenants")
	defer span.End()
	var t domain.Tenant
	q := `SELECT id, name, status, metadata, created_at, updated_at 
		FROM tenants WHERE name = $1`

	err := r.db.Raw().QueryRow(ctx, q, name).Scan(
		&t.ID, &t.Name, &t.Status, &t.Metadata, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, fmt.Errorf("get tenant by name: %w", err)
	}
	return &t, nil
}

func (r *pgTenantRepo) List(ctx context.Context, offset, limit int) ([]*domain.Tenant, int, error) {
	ctx, span := tracing.TraceDatabaseQuery(tenantTracer, ctx, "SELECT", "tenants")
	defer span.End()
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	// Get total count
	var total int
	countQ := `SELECT COUNT(*) FROM tenants`
	if err := r.db.Raw().QueryRow(ctx, countQ).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tenants: %w", err)
	}

	// Get paginated results
	q := `SELECT id, name, status, metadata, created_at, updated_at 
		FROM tenants ORDER BY name LIMIT $1 OFFSET $2`

	rows, err := r.db.Raw().Query(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*domain.Tenant
	for rows.Next() {
		var t domain.Tenant
		err := rows.Scan(&t.ID, &t.Name, &t.Status, &t.Metadata, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("scan tenant: %w", err)
		}
		tenants = append(tenants, &t)
	}
	return tenants, total, rows.Err()
}

func (r *pgTenantRepo) Update(ctx context.Context, tenant *domain.Tenant) error {
	ctx, span := tracing.TraceDatabaseQuery(tenantTracer, ctx, "UPDATE", "tenants")
	defer span.End()
	tenant.UpdatedAt = time.Now()

	q := `UPDATE tenants SET name = $2, status = $3, metadata = $4, updated_at = $5 
		WHERE id = $1`

	result, err := r.db.Raw().Exec(ctx, q,
		tenant.ID, tenant.Name, tenant.Status, tenant.Metadata, tenant.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update tenant: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("tenant not found")
	}
	return nil
}

func (r *pgTenantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, span := tracing.TraceDatabaseQuery(tenantTracer, ctx, "DELETE", "tenants")
	defer span.End()
	q := `DELETE FROM tenants WHERE id = $1`
	result, err := r.db.Raw().Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete tenant: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("tenant not found")
	}
	return nil
}
