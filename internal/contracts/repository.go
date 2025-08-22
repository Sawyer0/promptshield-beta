package contracts

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

// RulepackInfo represents basic rulepack metadata for listing
type RulepackInfo struct {
	ID          uuid.UUID
	Name        string
	Description string
	Version     int
	Active      bool
}

// RulepackRepository abstracts persistence for rulepacks.
type RulepackRepository interface {
	Create(ctx context.Context, tenantID uuid.UUID, name, desc string) (uuid.UUID, error)
	CreateVersion(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, status string, createdBy uuid.UUID) (uuid.UUID, error)
	// CreateVersionActivateTx atomically creates a new version (status=approved) and sets it active within one transaction.
	CreateVersionActivateTx(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, createdBy uuid.UUID) (uuid.UUID, error)
	GetActive(ctx context.Context, packID uuid.UUID) (json.RawMessage, int, error)
	Activate(ctx context.Context, packID, versionID uuid.UUID) error
	GetVersion(ctx context.Context, packID uuid.UUID, version int) (json.RawMessage, string, error)
	GetLatestVersion(ctx context.Context, packID uuid.UUID) (uuid.UUID, int, error)
	ActivateLatest(ctx context.Context, packID uuid.UUID) error
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]RulepackInfo, error)
	Delete(ctx context.Context, packID uuid.UUID) error
	// PurgeOldVersions deletes versions older than the most recent 'keep' versions (excluding current active).
	PurgeOldVersions(ctx context.Context, packID uuid.UUID, keep int) error
}
