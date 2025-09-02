package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

// ToolRepository defines CRUD operations for tools/actions registry.
type ToolRepository interface {
	Create(ctx context.Context, tool *domain.Tool) error
	Get(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.Tool, error)
	GetByToolID(ctx context.Context, tenantID uuid.UUID, toolID string) (*domain.Tool, error)
	List(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*domain.Tool, int, error)
	Update(ctx context.Context, tool *domain.Tool) error
	Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
}

type pgToolRepository struct{ db DB }

func Tools(db DB) ToolRepository { return &pgToolRepository{db: db} }

func (r *pgToolRepository) Create(ctx context.Context, tool *domain.Tool) error {
	if tool.ID == uuid.Nil {
		tool.ID = uuid.New()
	}
	now := time.Now().UTC()
	tool.CreatedAt = now
	tool.UpdatedAt = now

	caps, _ := json.Marshal(tool.CapabilityTags)
	doms, _ := json.Marshal(tool.DataDomains)

	q := `INSERT INTO tools (
            id, tenant_id, tool_id, name, description, capability_tags, data_domains,
            side_effect, auth_scope, arg_schema, risk_score, created_at, updated_at
         ) VALUES (
            $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13
         )`
	_, err := r.db.ExecContext(ctx, q,
		tool.ID, tool.TenantID, tool.ToolID, tool.Name, tool.Description,
		string(caps), string(doms), tool.SideEffect, tool.AuthScope,
		tool.ArgSchema, sql.NullInt32{Int32: int32(valueOrZero(tool.RiskScore)), Valid: tool.RiskScore != nil},
		tool.CreatedAt, tool.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create tool: %w", err)
	}
	return nil
}

func (r *pgToolRepository) Get(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.Tool, error) {
	q := `SELECT id, tenant_id, tool_id, name, description, capability_tags, data_domains,
                 side_effect, auth_scope, arg_schema, risk_score, created_at, updated_at
          FROM tools WHERE tenant_id=$1 AND id=$2`
	var (
		t          domain.Tool
		caps, doms string
		risk       sql.NullInt32
	)
	if err := r.db.QueryRowContext(ctx, q, tenantID, id).Scan(
		&t.ID, &t.TenantID, &t.ToolID, &t.Name, &t.Description, &caps, &doms,
		&t.SideEffect, &t.AuthScope, &t.ArgSchema, &risk, &t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("get tool: %w", err)
	}
	_ = json.Unmarshal([]byte(caps), &t.CapabilityTags)
	_ = json.Unmarshal([]byte(doms), &t.DataDomains)
	if risk.Valid {
		v := int(risk.Int32)
		t.RiskScore = &v
	}
	return &t, nil
}

func (r *pgToolRepository) GetByToolID(ctx context.Context, tenantID uuid.UUID, toolID string) (*domain.Tool, error) {
	q := `SELECT id, tenant_id, tool_id, name, description, capability_tags, data_domains,
                 side_effect, auth_scope, arg_schema, risk_score, created_at, updated_at
          FROM tools WHERE tenant_id=$1 AND tool_id=$2`
	var (
		t          domain.Tool
		caps, doms string
		risk       sql.NullInt32
	)
	if err := r.db.QueryRowContext(ctx, q, tenantID, toolID).Scan(
		&t.ID, &t.TenantID, &t.ToolID, &t.Name, &t.Description, &caps, &doms,
		&t.SideEffect, &t.AuthScope, &t.ArgSchema, &risk, &t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("get tool by tool_id: %w", err)
	}
	_ = json.Unmarshal([]byte(caps), &t.CapabilityTags)
	_ = json.Unmarshal([]byte(doms), &t.DataDomains)
	if risk.Valid {
		v := int(risk.Int32)
		t.RiskScore = &v
	}
	return &t, nil
}

func (r *pgToolRepository) List(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*domain.Tool, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	q := `SELECT id, tenant_id, tool_id, name, description, capability_tags, data_domains,
                 side_effect, auth_scope, arg_schema, risk_score, created_at, updated_at
          FROM tools WHERE tenant_id=$1
          ORDER BY created_at DESC
          OFFSET $2 LIMIT $3`
	rows, err := r.db.QueryContext(ctx, q, tenantID, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list tools: %w", err)
	}
	defer rows.Close()
	var out []*domain.Tool
	for rows.Next() {
		var (
			t          domain.Tool
			caps, doms string
			risk       sql.NullInt32
		)
		if err := rows.Scan(&t.ID, &t.TenantID, &t.ToolID, &t.Name, &t.Description, &caps, &doms,
			&t.SideEffect, &t.AuthScope, &t.ArgSchema, &risk, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, err
		}
		_ = json.Unmarshal([]byte(caps), &t.CapabilityTags)
		_ = json.Unmarshal([]byte(doms), &t.DataDomains)
		if risk.Valid {
			v := int(risk.Int32)
			t.RiskScore = &v
		}
		out = append(out, &t)
	}

	// Count
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM tools WHERE tenant_id=$1`, tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tools: %w", err)
	}

	return out, total, rows.Err()
}

func (r *pgToolRepository) Update(ctx context.Context, tool *domain.Tool) error {
	tool.UpdatedAt = time.Now().UTC()
	caps, _ := json.Marshal(tool.CapabilityTags)
	doms, _ := json.Marshal(tool.DataDomains)
	q := `UPDATE tools SET
            name=$1, description=$2, capability_tags=$3, data_domains=$4,
            side_effect=$5, auth_scope=$6, arg_schema=$7, risk_score=$8,
            updated_at=$9
          WHERE tenant_id=$10 AND id=$11`
	_, err := r.db.ExecContext(ctx, q,
		tool.Name, tool.Description, string(caps), string(doms),
		tool.SideEffect, tool.AuthScope, tool.ArgSchema,
		sql.NullInt32{Int32: int32(valueOrZero(tool.RiskScore)), Valid: tool.RiskScore != nil},
		tool.UpdatedAt, tool.TenantID, tool.ID,
	)
	if err != nil {
		return fmt.Errorf("update tool: %w", err)
	}
	return nil
}

func (r *pgToolRepository) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	q := `DELETE FROM tools WHERE tenant_id=$1 AND id=$2`
	res, err := r.db.ExecContext(ctx, q, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete tool: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("delete tool: not found")
	}
	return nil
}

func valueOrZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
