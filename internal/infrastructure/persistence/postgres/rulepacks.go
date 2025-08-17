package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/promptshield/promptshield/internal/contracts"
)

// RulepackRepository exposes persistence operations needed by services.
type RulepackRepository interface {
	Create(ctx context.Context, tenantID uuid.UUID, name, desc string) (uuid.UUID, error)
	CreateVersion(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, status string, createdBy uuid.UUID) (uuid.UUID, error)
	CreateVersionActivateTx(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, createdBy uuid.UUID) (uuid.UUID, error)
	GetActive(ctx context.Context, packID uuid.UUID) (json.RawMessage, int, error)
	Activate(ctx context.Context, packID, versionID uuid.UUID) error
	ApproveVersion(ctx context.Context, packID uuid.UUID, version int) error
	GetVersion(ctx context.Context, packID uuid.UUID, version int) (json.RawMessage, string, error)
	GetLatestVersion(ctx context.Context, packID uuid.UUID) (uuid.UUID, int, error)
	ActivateLatest(ctx context.Context, packID uuid.UUID) error
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]contracts.RulepackInfo, error)
	Delete(ctx context.Context, packID uuid.UUID) error
}

type pgRulepackRepo struct{ db *Pool }

func RulepackRepo(db *Pool) *pgRulepackRepo { return &pgRulepackRepo{db: db} }

func (r *pgRulepackRepo) Create(ctx context.Context, tenantID uuid.UUID, name, desc string) (uuid.UUID, error) {
	var id uuid.UUID
	q := `INSERT INTO rulepacks (id, tenant_id, name, description) VALUES (gen_random_uuid(), $1, $2, $3) RETURNING id`
	if err := r.db.Raw().QueryRow(ctx, q, tenantID, name, desc).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("create rulepack: %w", err)
	}
	return id, nil
}

func (r *pgRulepackRepo) CreateVersion(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, status string, createdBy uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	q := `INSERT INTO rulepack_versions (id, rulepack_id, version, dsl, status, created_by)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5) RETURNING id`
	if err := r.db.Raw().QueryRow(ctx, q, packID, version, dsl, status, createdBy).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("create version: %w", err)
	}
	return id, nil
}

func (r *pgRulepackRepo) CreateVersionActivateTx(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, createdBy uuid.UUID) (uuid.UUID, error) {
	// Start transaction
	tx, err := r.db.Raw().Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var verID uuid.UUID
	if err := tx.QueryRow(ctx, `INSERT INTO rulepack_versions (id, rulepack_id, version, dsl, status, created_by) VALUES (gen_random_uuid(), $1, $2, $3, 'approved', $4) RETURNING id`, packID, version, dsl, createdBy).Scan(&verID); err != nil {
		return uuid.Nil, fmt.Errorf("create version: %w", err)
	}

	if _, err := tx.Exec(ctx, `UPDATE rulepacks SET current_version_id=$1 WHERE id=$2`, verID, packID); err != nil {
		return uuid.Nil, fmt.Errorf("activate version: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit: %w", err)
	}
	return verID, nil
}

func (r *pgRulepackRepo) GetActive(ctx context.Context, packID uuid.UUID) (json.RawMessage, int, error) {
	q := `SELECT v.dsl, v.version FROM rulepacks p JOIN rulepack_versions v ON p.current_version_id = v.id WHERE p.id=$1`
	var (
		dsl json.RawMessage
		ver int
	)
	if err := r.db.Raw().QueryRow(ctx, q, packID).Scan(&dsl, &ver); err != nil {
		if err == pgx.ErrNoRows {
			return nil, 0, fmt.Errorf("not found")
		}
		return nil, 0, fmt.Errorf("get active: %w", err)
	}
	return dsl, ver, nil
}

func (r *pgRulepackRepo) Activate(ctx context.Context, packID, versionID uuid.UUID) error {
	q := `UPDATE rulepacks SET current_version_id=$1 WHERE id=$2`
	ct, err := r.db.Raw().Exec(ctx, q, versionID, packID)
	if err != nil {
		return fmt.Errorf("activate: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("activate: no rows")
	}
	return nil
}

// HealthCheck performs a lightweight round-trip to validate DB connectivity.
func (r *pgRulepackRepo) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if _, err := r.db.Raw().Exec(ctx, `SELECT 1`); err != nil {
		return fmt.Errorf("db ping: %w", err)
	}
	return nil
}

// GetVersion fetches DSL JSON and status for a given version.
func (r *pgRulepackRepo) GetVersion(ctx context.Context, packID uuid.UUID, version int) (json.RawMessage, string, error) {
	var dsl json.RawMessage
	var status string
	q := `SELECT dsl, status FROM rulepack_versions WHERE rulepack_id=$1 AND version=$2`
	err := r.db.Raw().QueryRow(ctx, q, packID, version).Scan(&dsl, &status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, "", fmt.Errorf("not found")
		}
		return nil, "", err
	}
	return dsl, status, nil
}

// ApproveVersion sets status='approved' when currently 'draft'.
func (r *pgRulepackRepo) ApproveVersion(ctx context.Context, packID uuid.UUID, version int) error {
	q := `UPDATE rulepack_versions SET status='approved' WHERE rulepack_id=$1 AND version=$2 AND status='draft'`
	ct, err := r.db.Raw().Exec(ctx, q, packID, version)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("no draft version updated")
	}
	return nil
}

// ListByTenant returns all rulepacks for a tenant with their active version info.
func (r *pgRulepackRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]contracts.RulepackInfo, error) {
	q := `SELECT p.id, p.name, p.description, 
	         COALESCE(v.version, 0) as version,
	         (p.current_version_id IS NOT NULL) as active
	      FROM rulepacks p 
	      LEFT JOIN rulepack_versions v ON p.current_version_id = v.id
	      WHERE p.tenant_id = $1
	      ORDER BY p.name`

	rows, err := r.db.Raw().Query(ctx, q, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list rulepacks: %w", err)
	}
	defer rows.Close()

	var result []contracts.RulepackInfo
	for rows.Next() {
		var info contracts.RulepackInfo
		if err := rows.Scan(&info.ID, &info.Name, &info.Description, &info.Version, &info.Active); err != nil {
			return nil, fmt.Errorf("scan rulepack: %w", err)
		}
		result = append(result, info)
	}

	return result, rows.Err()
}

// GetLatestVersion returns the latest version ID and number for a rulepack.
func (r *pgRulepackRepo) GetLatestVersion(ctx context.Context, packID uuid.UUID) (uuid.UUID, int, error) {
	q := `SELECT id, version FROM rulepack_versions WHERE rulepack_id = $1 ORDER BY version DESC LIMIT 1`
	var versionID uuid.UUID
	var version int

	if err := r.db.Raw().QueryRow(ctx, q, packID).Scan(&versionID, &version); err != nil {
		if err == pgx.ErrNoRows {
			return uuid.Nil, 0, fmt.Errorf("no versions found")
		}
		return uuid.Nil, 0, fmt.Errorf("get latest version: %w", err)
	}

	return versionID, version, nil
}

// ActivateLatest activates the latest version of a rulepack.
func (r *pgRulepackRepo) ActivateLatest(ctx context.Context, packID uuid.UUID) error {
	versionID, _, err := r.GetLatestVersion(ctx, packID)
	if err != nil {
		return err
	}

	return r.Activate(ctx, packID, versionID)
}

// Delete removes a rulepack and all its versions.
func (r *pgRulepackRepo) Delete(ctx context.Context, packID uuid.UUID) error {
	// Use a transaction to ensure consistency
	tx, err := r.db.Raw().Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete all versions first
	if _, err := tx.Exec(ctx, `DELETE FROM rulepack_versions WHERE rulepack_id = $1`, packID); err != nil {
		return fmt.Errorf("delete versions: %w", err)
	}

	// Delete the rulepack
	ct, err := tx.Exec(ctx, `DELETE FROM rulepacks WHERE id = $1`, packID)
	if err != nil {
		return fmt.Errorf("delete rulepack: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return fmt.Errorf("rulepack not found")
	}

	return tx.Commit(ctx)
}

// PurgeOldVersions deletes versions older than the most recent 'keep' versions.
// Active/current version is always preserved regardless of retain count.
func (r *pgRulepackRepo) PurgeOldVersions(ctx context.Context, packID uuid.UUID, keep int) error {
	if keep <= 0 {
		return nil
	}
	// Identify IDs to retain: newest 'keep' versions plus current active.
	// Build CTE for versions to delete.
	q := `WITH retained AS (
            SELECT id FROM rulepack_versions WHERE rulepack_id=$1 ORDER BY version DESC LIMIT $2
        )
        DELETE FROM rulepack_versions v USING rulepacks p
        WHERE v.rulepack_id=$1
          AND v.id NOT IN (SELECT id FROM retained)
          AND (p.current_version_id IS NULL OR v.id <> p.current_version_id)`
	if _, err := r.db.Raw().Exec(ctx, q, packID, keep); err != nil {
		return fmt.Errorf("purge old versions: %w", err)
	}
	return nil
}
