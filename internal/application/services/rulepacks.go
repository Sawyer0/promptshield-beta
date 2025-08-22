package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/audit"
	"github.com/promptshield/promptshield/internal/contracts"
	nats "github.com/promptshield/promptshield/internal/infrastructure/messaging/nats"
	"github.com/promptshield/promptshield/internal/observability/metrics"
	"github.com/promptshield/promptshield/internal/rules"
	"gopkg.in/yaml.v3"
)

// canaryDelay blocks activation until the configured delay elapses to allow passive monitoring
// of error/latency spikes before rolling out a new rulepack.
// PS_RULEPACK_CANARY_DELAY_SECONDS controls the duration; 0 or unset disables the delay.
func canaryDelay(ctx context.Context) error {
	dVal := os.Getenv("PS_RULEPACK_CANARY_DELAY_SECONDS")
	if dVal == "" {
		return nil
	}
	secs, err := strconv.Atoi(dVal)
	if err != nil || secs <= 0 {
		return nil
	}
	select {
	case <-time.After(time.Duration(secs) * time.Second):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RulepackInfo is an alias for the contract type
type RulepackInfo = contracts.RulepackInfo

// RulepackService contains business logic around rulepacks.
type RulepackService struct {
	repo  contracts.RulepackRepository
	pub   *nats.Publisher
	audit audit.Logger
}

// RulepackServiceCstor creates a RulepackService with auditing configured from environment.
func RulepackServiceCstor(r contracts.RulepackRepository, pub *nats.Publisher) *RulepackService {
	auditLogger, _, _ := audit.NewLoggerFromEnv() // TODO: Handle close func and error properly
	return &RulepackService{repo: r, pub: pub, audit: auditLogger}
}

func checksumJSON(raw json.RawMessage) string {
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}

// CreateVersionActivate stores a new version and immediately activates it.
func (s *RulepackService) CreateVersionActivate(ctx context.Context, tenantID uuid.UUID, packID uuid.UUID, version int, dsl json.RawMessage) error {
	if err := canaryDelay(ctx); err != nil {
		return err
	}
	verID, err := s.repo.CreateVersionActivateTx(ctx, packID, version, dsl, uuid.Nil)
	if err != nil {
		return err
	}
	_ = verID // not used beyond audit/metrics
	metrics.IncRulepackActivations()
	if s.pub != nil {
		u := nats.RuleUpdate{TenantID: tenantID.String(), TargetScope: "global", RulepackID: packID.String(), Version: version, ContentSHA256: checksumJSON(dsl)}
		_ = s.pub.PublishRuleUpdate(ctx, u)
	}

	// GC old versions
	retention := 10
	if v := os.Getenv("PS_RULEPACK_RETENTION"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			retention = n
		}
	}
	_ = s.repo.PurgeOldVersions(ctx, packID, retention)

	return nil
}

// Create creates a new rulepack.
func (s *RulepackService) Create(ctx context.Context, tenantID uuid.UUID, name, description string) (uuid.UUID, error) {
	id, err := s.repo.Create(ctx, tenantID, name, description)
	if err == nil {
		s.emitAudit("rulepack.create", map[string]any{"tenant_id": tenantID.String(), "rulepack_id": id.String(), "name": name})
	}
	return id, err
}

// Upload validates and creates a new rulepack version, optionally activating it.
func (s *RulepackService) Upload(ctx context.Context, tenantID uuid.UUID, packID uuid.UUID, version int, dsl json.RawMessage, activate bool) (uuid.UUID, error) {
	verID, err := s.repo.CreateVersion(ctx, packID, version, dsl, "approved", uuid.Nil)
	if err != nil {
		return uuid.Nil, err
	}

	if activate {
		if err := canaryDelay(ctx); err != nil {
			return verID, err
		}
		if err := s.repo.Activate(ctx, packID, verID); err != nil {
			return verID, err
		}
		metrics.IncRulepackActivations()

		if s.pub != nil {
			u := nats.RuleUpdate{TenantID: tenantID.String(), TargetScope: "global", RulepackID: packID.String(), Version: version, ContentSHA256: checksumJSON(dsl)}
			_ = s.pub.PublishRuleUpdate(ctx, u)
		}
	}

	// Audit upload event (approved version creation)
	s.emitAudit("rulepack.upload", map[string]any{"tenant_id": tenantID.String(), "rulepack_id": packID.String(), "version": version, "checksum": checksumJSON(dsl)})

	return verID, nil
}

// SetActive activates a specific rulepack version.
func (s *RulepackService) SetActive(ctx context.Context, tenantID uuid.UUID, packID, versionID uuid.UUID) error {
	if err := canaryDelay(ctx); err != nil {
		return err
	}
	if err := s.repo.Activate(ctx, packID, versionID); err != nil {
		return err
	}
	metrics.IncRulepackActivations()
	if s.pub != nil {
		// Get version number for NATS message
		if dsl, version, err := s.repo.GetActive(ctx, packID); err == nil {
			u := nats.RuleUpdate{TenantID: tenantID.String(), TargetScope: "global", RulepackID: packID.String(), Version: version, ContentSHA256: checksumJSON(dsl)}
			_ = s.pub.PublishRuleUpdate(ctx, u)
		}
	}

	return nil
}

// ActivateLatest activates the latest version of a rulepack.
func (s *RulepackService) ActivateLatest(ctx context.Context, tenantID uuid.UUID, packID uuid.UUID) error {
	if err := canaryDelay(ctx); err != nil {
		return err
	}
	if err := s.repo.ActivateLatest(ctx, packID); err != nil {
		return err
	}
	metrics.IncRulepackActivations()
	s.emitAudit("rulepack.activate", map[string]any{"tenant_id": tenantID.String(), "rulepack_id": packID.String()})
	if s.pub != nil {
		// Get version number for NATS message
		if dsl, version, err := s.repo.GetActive(ctx, packID); err == nil {
			u := nats.RuleUpdate{TenantID: tenantID.String(), TargetScope: "global", RulepackID: packID.String(), Version: version, ContentSHA256: checksumJSON(dsl)}
			_ = s.pub.PublishRuleUpdate(ctx, u)
		}
	}

	return nil
}

// GetActive returns the active version DSL and version number for a rulepack.
func (s *RulepackService) GetActive(ctx context.Context, packID uuid.UUID) (json.RawMessage, int, error) {
	return s.repo.GetActive(ctx, packID)
}

// GetVersion returns the DSL and status for a specific version.
func (s *RulepackService) GetVersion(ctx context.Context, packID uuid.UUID, version int) (json.RawMessage, string, error) {
	return s.repo.GetVersion(ctx, packID, version)
}

// ValidateDSL validates a rulepack DSL payload (JSON or YAML).
func (s *RulepackService) ValidateDSL(data []byte) (bool, []string, []string) {
	// Enforce hard limits before parsing to avoid resource exhaustion.
	const defaultMaxKB = 1024 // 1 MiB YAML/JSON payload limit
	const defaultMaxRules = 1000

	// Allow overrides via env for operators.
	maxKB := defaultMaxKB
	if v := os.Getenv("PS_MAX_RULEPACK_KB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxKB = n
		}
	}

	maxRules := defaultMaxRules
	if v := os.Getenv("PS_MAX_RULES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxRules = n
		}
	}

	if len(data) > maxKB*1024 {
		metrics.IncRulepackValidationFailures()
		return false, nil, []string{fmt.Sprintf("rulepack exceeds maximum size of %d KB", maxKB)}
	}

	var pack rules.RulePack

	// Try parsing as YAML first (covers JSON too)
	if err := yaml.Unmarshal(data, &pack); err != nil {
		metrics.IncRulepackValidationFailures()
		return false, nil, []string{fmt.Sprintf("invalid format: %v", err)}
	}

	// Validate using the rules package
	errs := rules.ValidatePack(pack)
	if len(pack.Rules) > maxRules {
		errs = append(errs, fmt.Errorf("too many rules: %d (max %d)", len(pack.Rules), maxRules))
	}

	if len(errs) > 0 {
		metrics.IncRulepackValidationFailures()
		var errorStrs []string
		for _, err := range errs {
			errorStrs = append(errorStrs, err.Error())
		}
		return false, nil, errorStrs
	}

	return true, nil, nil
}

// ParseDSL parses and validates DSL, returning the RulePack struct.
func (s *RulepackService) ParseDSL(data []byte) (rules.RulePack, error) {
	var pack rules.RulePack

	// Reuse same limit constants & env overrides as ValidateDSL.
	const defaultMaxKB = 1024
	const defaultMaxRules = 1000
	maxKB := defaultMaxKB
	if v := os.Getenv("PS_MAX_RULEPACK_KB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxKB = n
		}
	}
	maxRules := defaultMaxRules
	if v := os.Getenv("PS_MAX_RULES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxRules = n
		}
	}

	if len(data) > maxKB*1024 {
		return pack, fmt.Errorf("rulepack exceeds maximum size of %d KB", maxKB)
	}

	if err := yaml.Unmarshal(data, &pack); err != nil {
		return pack, fmt.Errorf("invalid format: %w", err)
	}

	errs := rules.ValidatePack(pack)
	if len(pack.Rules) > maxRules {
		errs = append(errs, fmt.Errorf("too many rules: %d (max %d)", len(pack.Rules), maxRules))
	}
	if len(errs) > 0 {
		return pack, fmt.Errorf("validation failed: %v", errs[0])
	}

	return pack, nil
}

// List returns all rulepacks for a tenant.
func (s *RulepackService) List(ctx context.Context, tenantID uuid.UUID) ([]RulepackInfo, error) {
	return s.repo.ListByTenant(ctx, tenantID)
}

// Delete removes a rulepack and all its versions.
func (s *RulepackService) Delete(ctx context.Context, tenantID uuid.UUID, packID uuid.UUID) error {
	if err := s.repo.Delete(ctx, packID); err != nil {
		return err
	}

	if s.pub != nil {
		// Notify that rulepack was deleted
		u := nats.RuleUpdate{TenantID: tenantID.String(), TargetScope: "global", RulepackID: packID.String(), Version: -1, ContentSHA256: ""}
		_ = s.pub.PublishRuleUpdate(ctx, u)
	}

	s.emitAudit("rulepack.delete", map[string]any{"tenant_id": tenantID.String(), "rulepack_id": packID.String()})

	return nil
}

// emitAudit writes an audit event if a logger is configured. Errors are ignored
// because audit logging must not disrupt primary operations.
func (s *RulepackService) emitAudit(eventType string, data map[string]any) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Log(audit.Event{Type: eventType, Data: data})
}
