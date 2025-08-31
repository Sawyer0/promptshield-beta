package postgres

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
)

type ProviderProfile struct {
    ID        uuid.UUID
    TenantID  uuid.UUID
    Provider  string
    Label     string
    APIKeyEnc string
    BaseURL   *string
    ExtraHdrs []byte // JSON
    CreatedAt time.Time
    UpdatedAt time.Time
}

type ProviderProfileRepo interface {
    Create(ctx context.Context, p *ProviderProfile) error
    Get(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*ProviderProfile, error)
    List(ctx context.Context, tenantID uuid.UUID) ([]*ProviderProfile, error)
    Update(ctx context.Context, p *ProviderProfile) error
    Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
}

type pgProviderProfileRepo struct{ db DB }

func ProviderProfiles(db DB) ProviderProfileRepo { return &pgProviderProfileRepo{db: db} }

func (r *pgProviderProfileRepo) Create(ctx context.Context, p *ProviderProfile) error {
    if p.ID == uuid.Nil { p.ID = uuid.New() }
    q := `INSERT INTO provider_profiles (id, tenant_id, provider, label, api_key_encrypted, base_url, extra_headers)
          VALUES ($1,$2,$3,$4,$5,$6,$7)`
    _, err := r.db.ExecContext(ctx, q, p.ID, p.TenantID, p.Provider, p.Label, p.APIKeyEnc, p.BaseURL, p.ExtraHdrs)
    if err != nil { return fmt.Errorf("create provider profile: %w", err) }
    return nil
}

func (r *pgProviderProfileRepo) Get(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*ProviderProfile, error) {
    q := `SELECT id, tenant_id, provider, label, api_key_encrypted, base_url, extra_headers, created_at, updated_at
          FROM provider_profiles WHERE tenant_id=$1 AND id=$2`
    var p ProviderProfile
    if err := r.db.QueryRowContext(ctx, q, tenantID, id).Scan(&p.ID, &p.TenantID, &p.Provider, &p.Label, &p.APIKeyEnc, &p.BaseURL, &p.ExtraHdrs, &p.CreatedAt, &p.UpdatedAt); err != nil {
        return nil, fmt.Errorf("get provider profile: %w", err)
    }
    return &p, nil
}

func (r *pgProviderProfileRepo) List(ctx context.Context, tenantID uuid.UUID) ([]*ProviderProfile, error) {
    q := `SELECT id, tenant_id, provider, label, api_key_encrypted, base_url, extra_headers, created_at, updated_at
          FROM provider_profiles WHERE tenant_id=$1 ORDER BY created_at DESC`
    rows, err := r.db.QueryContext(ctx, q, tenantID)
    if err != nil { return nil, fmt.Errorf("list provider profiles: %w", err) }
    defer rows.Close()
    var out []*ProviderProfile
    for rows.Next() {
        var p ProviderProfile
        if err := rows.Scan(&p.ID, &p.TenantID, &p.Provider, &p.Label, &p.APIKeyEnc, &p.BaseURL, &p.ExtraHdrs, &p.CreatedAt, &p.UpdatedAt); err != nil {
            return nil, err
        }
        out = append(out, &p)
    }
    return out, rows.Err()
}

func (r *pgProviderProfileRepo) Update(ctx context.Context, p *ProviderProfile) error {
    q := `UPDATE provider_profiles SET provider=$1, label=$2, api_key_encrypted=COALESCE($3, api_key_encrypted), base_url=$4, extra_headers=$5, updated_at=now()
          WHERE tenant_id=$6 AND id=$7`
    _, err := r.db.ExecContext(ctx, q, p.Provider, p.Label, nullIfEmpty(p.APIKeyEnc), p.BaseURL, p.ExtraHdrs, p.TenantID, p.ID)
    if err != nil { return fmt.Errorf("update provider profile: %w", err) }
    return nil
}

func (r *pgProviderProfileRepo) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
    q := `DELETE FROM provider_profiles WHERE tenant_id=$1 AND id=$2`
    _, err := r.db.ExecContext(ctx, q, tenantID, id)
    if err != nil { return fmt.Errorf("delete provider profile: %w", err) }
    return nil
}

func nullIfEmpty(s string) *string { if s == "" { return nil }; return &s }
