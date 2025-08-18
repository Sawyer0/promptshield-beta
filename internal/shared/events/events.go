package events

import (
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// Event represents a domain event that can be published and consumed
type Event interface {
	EventType() string
	EventID() string
	Timestamp() time.Time
	TenantID() *uuid.UUID
	Version() int
}

// BaseEvent provides common event fields
type BaseEvent struct {
	ID           string     `json:"id"`
	Type         string     `json:"type"`
	Time         time.Time  `json:"timestamp"`
	Tenant       *uuid.UUID `json:"tenant_id,omitempty"`
	EventVersion int        `json:"version"`
}

func (e BaseEvent) EventType() string    { return e.Type }
func (e BaseEvent) EventID() string      { return e.ID }
func (e BaseEvent) Timestamp() time.Time { return e.Time }
func (e BaseEvent) TenantID() *uuid.UUID { return e.Tenant }
func (e BaseEvent) Version() int         { return e.EventVersion }

// Tenant events
type TenantCreated struct {
	BaseEvent
	TenantData types.Tenant `json:"tenant"`
	CreatedBy  *uuid.UUID   `json:"created_by,omitempty"`
}

type TenantUpdated struct {
	BaseEvent
	TenantID  uuid.UUID              `json:"tenant_id"`
	Before    types.Tenant           `json:"before"`
	After     types.Tenant           `json:"after"`
	Changes   map[string]interface{} `json:"changes"`
	UpdatedBy *uuid.UUID             `json:"updated_by,omitempty"`
}

type TenantSuspended struct {
	BaseEvent
	TenantID    uuid.UUID  `json:"tenant_id"`
	Reason      string     `json:"reason"`
	SuspendedBy *uuid.UUID `json:"suspended_by,omitempty"`
}

type TenantRestored struct {
	BaseEvent
	TenantID   uuid.UUID  `json:"tenant_id"`
	Reason     string     `json:"reason,omitempty"`
	RestoredBy *uuid.UUID `json:"restored_by,omitempty"`
}

type TenantDeleted struct {
	BaseEvent
	TenantID  uuid.UUID  `json:"tenant_id"`
	DeletedBy *uuid.UUID `json:"deleted_by,omitempty"`
}

// Policy events
type PolicyCreated struct {
	BaseEvent
	PolicyData types.Policy `json:"policy"`
	CreatedBy  *uuid.UUID   `json:"created_by,omitempty"`
}

type PolicyUpdated struct {
	BaseEvent
	PolicyID  uuid.UUID              `json:"policy_id"`
	Before    types.Policy           `json:"before"`
	After     types.Policy           `json:"after"`
	Changes   map[string]interface{} `json:"changes"`
	UpdatedBy *uuid.UUID             `json:"updated_by,omitempty"`
}

type PolicyAssigned struct {
	BaseEvent
	Assignment types.PolicyAssignment `json:"assignment"`
	AssignedBy *uuid.UUID             `json:"assigned_by,omitempty"`
}

type PolicyUnassigned struct {
	BaseEvent
	AssignmentID uuid.UUID  `json:"assignment_id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	PolicyID     uuid.UUID  `json:"policy_id"`
	UnassignedBy *uuid.UUID `json:"unassigned_by,omitempty"`
}

type PolicyDeleted struct {
	BaseEvent
	PolicyID  uuid.UUID  `json:"policy_id"`
	DeletedBy *uuid.UUID `json:"deleted_by,omitempty"`
}

// Provider key events
type ProviderKeyAdded struct {
	BaseEvent
	KeyData types.ProviderKey `json:"key_data"`
	AddedBy *uuid.UUID        `json:"added_by,omitempty"`
}

type ProviderKeyRotated struct {
	BaseEvent
	KeyID     uuid.UUID  `json:"key_id"`
	RotatedBy *uuid.UUID `json:"rotated_by,omitempty"`
}

type ProviderKeyRevoked struct {
	BaseEvent
	KeyID     uuid.UUID  `json:"key_id"`
	Reason    string     `json:"reason,omitempty"`
	RevokedBy *uuid.UUID `json:"revoked_by,omitempty"`
}

// API token events
type APITokenCreated struct {
	BaseEvent
	TokenData types.APIToken `json:"token_data"`
	CreatedBy *uuid.UUID     `json:"created_by,omitempty"`
}

type APITokenRevoked struct {
	BaseEvent
	TokenID   uuid.UUID  `json:"token_id"`
	Reason    string     `json:"reason,omitempty"`
	RevokedBy *uuid.UUID `json:"revoked_by,omitempty"`
}

// Quota events
type QuotaUpdated struct {
	BaseEvent
	QuotaData types.Quota            `json:"quota_data"`
	Changes   map[string]interface{} `json:"changes"`
	UpdatedBy *uuid.UUID             `json:"updated_by,omitempty"`
}

type QuotaExceeded struct {
	BaseEvent
	TenantID  uuid.UUID `json:"tenant_id"`
	QuotaType string    `json:"quota_type"`
	Current   int64     `json:"current"`
	Limit     int64     `json:"limit"`
	RequestID string    `json:"request_id,omitempty"`
}

// Security and enforcement events
type ViolationDetected struct {
	BaseEvent
	RequestID  string                  `json:"request_id"`
	Violations []types.PolicyViolation `json:"violations"`
	ScanResult types.ScanResult        `json:"scan_result"`
	Provider   types.Provider          `json:"provider,omitempty"`
	Endpoint   string                  `json:"endpoint,omitempty"`
}

type RequestBlocked struct {
	BaseEvent
	RequestID  string                  `json:"request_id"`
	Reason     string                  `json:"reason"`
	Violations []types.PolicyViolation `json:"violations,omitempty"`
	Provider   types.Provider          `json:"provider,omitempty"`
	Endpoint   string                  `json:"endpoint,omitempty"`
}

type RequestAllowed struct {
	BaseEvent
	RequestID  string           `json:"request_id"`
	ScanResult types.ScanResult `json:"scan_result"`
	Provider   types.Provider   `json:"provider,omitempty"`
	Endpoint   string           `json:"endpoint,omitempty"`
}

// Usage and metrics events
type UsageRecorded struct {
	BaseEvent
	UsageData types.UsageRecord `json:"usage_data"`
}

type UsageAggregated struct {
	BaseEvent
	Metric types.UsageMetric `json:"metric"`
}

// System events
type SystemStarted struct {
	BaseEvent
	Version string                 `json:"version"`
	Config  map[string]interface{} `json:"config,omitempty"`
}

type SystemStopped struct {
	BaseEvent
	Reason string `json:"reason,omitempty"`
}

type HealthCheckFailed struct {
	BaseEvent
	Service string `json:"service"`
	Error   string `json:"error"`
}

type HealthCheckRecovered struct {
	BaseEvent
	Service string `json:"service"`
}

// Semantic analysis events
type SemanticAnalysisCompleted struct {
	BaseEvent
	RequestID  string         `json:"request_id"`
	Provider   types.Provider `json:"provider"`
	Model      string         `json:"model"`
	CacheHit   bool           `json:"cache_hit"`
	LatencyMs  int64          `json:"latency_ms"`
	TokensUsed int            `json:"tokens_used,omitempty"`
}

type SemanticAnalysisFailed struct {
	BaseEvent
	RequestID string         `json:"request_id"`
	Provider  types.Provider `json:"provider"`
	Model     string         `json:"model"`
	Error     string         `json:"error"`
	Retryable bool           `json:"retryable"`
}

// Event type constants
const (
	EventTypeTenantCreated   = "tenant.created"
	EventTypeTenantUpdated   = "tenant.updated"
	EventTypeTenantSuspended = "tenant.suspended"
	EventTypeTenantRestored  = "tenant.restored"
	EventTypeTenantDeleted   = "tenant.deleted"

	EventTypePolicyCreated    = "policy.created"
	EventTypePolicyUpdated    = "policy.updated"
	EventTypePolicyAssigned   = "policy.assigned"
	EventTypePolicyUnassigned = "policy.unassigned"
	EventTypePolicyDeleted    = "policy.deleted"

	EventTypeProviderKeyAdded   = "provider_key.added"
	EventTypeProviderKeyRotated = "provider_key.rotated"
	EventTypeProviderKeyRevoked = "provider_key.revoked"

	EventTypeAPITokenCreated = "api_token.created"
	EventTypeAPITokenRevoked = "api_token.revoked"

	EventTypeQuotaUpdated  = "quota.updated"
	EventTypeQuotaExceeded = "quota.exceeded"

	EventTypeViolationDetected = "violation.detected"
	EventTypeRequestBlocked    = "request.blocked"
	EventTypeRequestAllowed    = "request.allowed"

	EventTypeUsageRecorded   = "usage.recorded"
	EventTypeUsageAggregated = "usage.aggregated"

	EventTypeSystemStarted        = "system.started"
	EventTypeSystemStopped        = "system.stopped"
	EventTypeHealthCheckFailed    = "health_check.failed"
	EventTypeHealthCheckRecovered = "health_check.recovered"

	EventTypeSemanticAnalysisCompleted = "semantic_analysis.completed"
	EventTypeSemanticAnalysisFailed    = "semantic_analysis.failed"

	// Audit action constants
	EventTypeUserLogin         = "user.login"
	EventTypeUserLogout        = "user.logout"
	EventTypeUserLoginFailed   = "user.login.failed"
	EventTypeAPIKeyCreated     = "api_key.created"
	EventTypeAPIKeyRevoked     = "api_key.revoked"
	EventTypeRuleTriggered     = "rule.triggered"
	EventTypeDataAccessed      = "data.accessed"
	EventTypeConfigChanged     = "config.changed"
	EventTypeEnforcementAction = "enforcement.action"
)

// NewEventID generates a new event ID
func NewEventID() string {
	return uuid.New().String()
}

// NewBaseEvent creates a new base event with common fields
func NewBaseEvent(eventType string, tenantID *uuid.UUID) BaseEvent {
	return BaseEvent{
		ID:           NewEventID(),
		Type:         eventType,
		Time:         time.Now().UTC(),
		Tenant:       tenantID,
		EventVersion: 1,
	}
}

// Object type constants for audit events
const (
	ObjectTypeUser     = "user"
	ObjectTypeAPIKey   = "api_key"
	ObjectTypePolicy   = "policy"
	ObjectTypeRule     = "rule"
	ObjectTypeRequest  = "request"
	ObjectTypeResponse = "response"
	ObjectTypeTenant   = "tenant"
	ObjectTypeConfig   = "config"
	ObjectTypeData     = "data"
)
