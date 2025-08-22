package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/contracts"
)

// RulepackRepositoryImpl implements RulepackRepository interface using in-memory storage
// This is for development and testing when PostgreSQL is not available
type RulepackRepositoryImpl struct {
	mu            sync.RWMutex
	rulepacks     map[uuid.UUID]*rulepack
	versions      map[uuid.UUID]*rulepackVersion
	activeVersion map[uuid.UUID]uuid.UUID // packID -> versionID
}

type rulepack struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Name        string
	Description string
	CreatedAt   time.Time
}

type rulepackVersion struct {
	ID         uuid.UUID
	RulepackID uuid.UUID
	Version    int
	DSL        json.RawMessage
	Status     string
	CreatedBy  uuid.UUID
	CreatedAt  time.Time
}

// NewRulepackRepository creates a new in-memory rulepack repository
func NewRulepackRepository() contracts.RulepackRepository {
	repo := &RulepackRepositoryImpl{
		rulepacks:     make(map[uuid.UUID]*rulepack),
		versions:      make(map[uuid.UUID]*rulepackVersion),
		activeVersion: make(map[uuid.UUID]uuid.UUID),
	}
	
	// Seed with default rulepack for demo
	defaultPack := &rulepack{
		ID:          uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
		TenantID:    uuid.Nil,
		Name:        "Default Security Rules",
		Description: "Built-in security rules for prompt injection detection",
		CreatedAt:   time.Now().Add(-24 * time.Hour),
	}
	repo.rulepacks[defaultPack.ID] = defaultPack
	
	// Add a default version with rules
	defaultVersion := &rulepackVersion{
		ID:         uuid.New(),
		RulepackID: defaultPack.ID,
		Version:    1,
		DSL:        json.RawMessage(getDefaultRulepackDSL()),
		Status:     "approved",
		CreatedBy:  uuid.Nil,
		CreatedAt:  time.Now().Add(-24 * time.Hour),
	}
	repo.versions[defaultVersion.ID] = defaultVersion
	repo.activeVersion[defaultPack.ID] = defaultVersion.ID
	
	return repo
}

// Create creates a new rulepack
func (r *RulepackRepositoryImpl) Create(ctx context.Context, tenantID uuid.UUID, name, desc string) (uuid.UUID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	pack := &rulepack{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        name,
		Description: desc,
		CreatedAt:   time.Now(),
	}
	
	r.rulepacks[pack.ID] = pack
	return pack.ID, nil
}

// CreateVersion creates a new version of a rulepack
func (r *RulepackRepositoryImpl) CreateVersion(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, status string, createdBy uuid.UUID) (uuid.UUID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.rulepacks[packID]; !exists {
		return uuid.Nil, fmt.Errorf("rulepack %s not found", packID)
	}
	
	ver := &rulepackVersion{
		ID:         uuid.New(),
		RulepackID: packID,
		Version:    version,
		DSL:        dsl,
		Status:     status,
		CreatedBy:  createdBy,
		CreatedAt:  time.Now(),
	}
	
	r.versions[ver.ID] = ver
	return ver.ID, nil
}

// CreateVersionActivateTx atomically creates and activates a new version
func (r *RulepackRepositoryImpl) CreateVersionActivateTx(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, createdBy uuid.UUID) (uuid.UUID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.rulepacks[packID]; !exists {
		return uuid.Nil, fmt.Errorf("rulepack %s not found", packID)
	}
	
	ver := &rulepackVersion{
		ID:         uuid.New(),
		RulepackID: packID,
		Version:    version,
		DSL:        dsl,
		Status:     "approved",
		CreatedBy:  createdBy,
		CreatedAt:  time.Now(),
	}
	
	r.versions[ver.ID] = ver
	r.activeVersion[packID] = ver.ID
	
	return ver.ID, nil
}

// GetActive returns the active version's DSL and version number
func (r *RulepackRepositoryImpl) GetActive(ctx context.Context, packID uuid.UUID) (json.RawMessage, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	versionID, exists := r.activeVersion[packID]
	if !exists {
		return nil, 0, fmt.Errorf("no active version for rulepack %s", packID)
	}
	
	ver, exists := r.versions[versionID]
	if !exists {
		return nil, 0, fmt.Errorf("version %s not found", versionID)
	}
	
	return ver.DSL, ver.Version, nil
}

// Activate sets a specific version as active
func (r *RulepackRepositoryImpl) Activate(ctx context.Context, packID, versionID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.rulepacks[packID]; !exists {
		return fmt.Errorf("rulepack %s not found", packID)
	}
	
	if _, exists := r.versions[versionID]; !exists {
		return fmt.Errorf("version %s not found", versionID)
	}
	
	r.activeVersion[packID] = versionID
	return nil
}

// GetVersion returns a specific version's DSL and status
func (r *RulepackRepositoryImpl) GetVersion(ctx context.Context, packID uuid.UUID, version int) (json.RawMessage, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	for _, ver := range r.versions {
		if ver.RulepackID == packID && ver.Version == version {
			return ver.DSL, ver.Status, nil
		}
	}
	
	return nil, "", fmt.Errorf("version %d not found for rulepack %s", version, packID)
}

// GetLatestVersion returns the latest version ID and number
func (r *RulepackRepositoryImpl) GetLatestVersion(ctx context.Context, packID uuid.UUID) (uuid.UUID, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var latestVer *rulepackVersion
	for _, ver := range r.versions {
		if ver.RulepackID == packID {
			if latestVer == nil || ver.Version > latestVer.Version {
				latestVer = ver
			}
		}
	}
	
	if latestVer == nil {
		return uuid.Nil, 0, fmt.Errorf("no versions found for rulepack %s", packID)
	}
	
	return latestVer.ID, latestVer.Version, nil
}

// ActivateLatest activates the latest version
func (r *RulepackRepositoryImpl) ActivateLatest(ctx context.Context, packID uuid.UUID) error {
	versionID, _, err := r.GetLatestVersion(ctx, packID)
	if err != nil {
		return err
	}
	
	return r.Activate(ctx, packID, versionID)
}

// ListByTenant returns all rulepacks for a tenant
func (r *RulepackRepositoryImpl) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]contracts.RulepackInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var result []contracts.RulepackInfo
	for _, pack := range r.rulepacks {
		if pack.TenantID == tenantID || tenantID == uuid.Nil {
			info := contracts.RulepackInfo{
				ID:          pack.ID,
				Name:        pack.Name,
				Description: pack.Description,
			}
			
			// Get active version if exists
			if versionID, exists := r.activeVersion[pack.ID]; exists {
				if ver, exists := r.versions[versionID]; exists {
					info.Version = ver.Version
					info.Active = true
				}
			}
			
			result = append(result, info)
		}
	}
	
	return result, nil
}

// Delete deletes a rulepack
func (r *RulepackRepositoryImpl) Delete(ctx context.Context, packID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.rulepacks[packID]; !exists {
		return fmt.Errorf("rulepack %s not found", packID)
	}
	
	// Delete all versions
	for id, ver := range r.versions {
		if ver.RulepackID == packID {
			delete(r.versions, id)
		}
	}
	
	// Remove active version mapping
	delete(r.activeVersion, packID)
	
	// Delete the rulepack
	delete(r.rulepacks, packID)
	
	return nil
}

// PurgeOldVersions removes old versions beyond retention limit
func (r *RulepackRepositoryImpl) PurgeOldVersions(ctx context.Context, packID uuid.UUID, retention int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Collect all versions for this pack
	var versions []*rulepackVersion
	for _, ver := range r.versions {
		if ver.RulepackID == packID {
			versions = append(versions, ver)
		}
	}
	
	// If we have more than retention, delete oldest ones
	if len(versions) > retention {
		// Sort by version number (simple approach)
		// In production, use proper sorting
		toDelete := len(versions) - retention
		deleted := 0
		for id, ver := range r.versions {
			if ver.RulepackID == packID && deleted < toDelete {
				// Don't delete active version
				if activeID, exists := r.activeVersion[packID]; exists && activeID == id {
					continue
				}
				delete(r.versions, id)
				deleted++
			}
		}
	}
	
	return nil
}

// getDefaultRulepackDSL returns the default rulepack DSL for demo/development
func getDefaultRulepackDSL() string {
	return `{
		"apiVersion": "promptshield.io/v1",
		"kind": "RulePack",
		"metadata": {
			"name": "default-security",
			"description": "Default security rules for prompt injection detection"
		},
		"rules": [
			{
				"id": "pi-001",
				"name": "Direct Instruction Override",
				"level": 1,
				"severity": "CRITICAL",
				"keywords": [
					"ignore previous instructions",
					"ignore all previous",
					"disregard previous"
				],
				"response": {
					"action": "block",
					"message": "Potential prompt injection detected"
				}
			},
			{
				"id": "pi-002",
				"name": "Role Manipulation",
				"level": 1,
				"severity": "HIGH",
				"keywords": [
					"you are now",
					"act as",
					"pretend to be"
				],
				"response": {
					"action": "warn",
					"message": "Potential role manipulation detected"
				}
			}
		]
	}`
}