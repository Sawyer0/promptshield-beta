package types

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditEntry represents an immutable audit log entry
// From domain/models.go
type AuditEntry struct {
	ID         uuid.UUID       `json:"id"`
	TenantID   *uuid.UUID      `json:"tenant_id,omitempty"`
	ActorID    *uuid.UUID      `json:"actor_id,omitempty"`
	ActorType  ActorType       `json:"actor_type"`
	ActorEmail string          `json:"actor_email,omitempty"`
	Action     string          `json:"action"`      // tenant.create, key.rotate, etc.
	ObjectType string          `json:"object_type"` // tenant, key, policy, etc.
	ObjectID   uuid.UUID       `json:"object_id"`
	Before     json.RawMessage `json:"before,omitempty"`
	After      json.RawMessage `json:"after,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	Hash       string          `json:"hash"`      // SHA-256 of entry
	PrevHash   string          `json:"prev_hash"` // Chain to previous
}

// ActorType represents who performed an action
type ActorType string

const (
	ActorTypeUser   ActorType = "user"
	ActorTypeSystem ActorType = "system"
	ActorTypeAPIKey ActorType = "api_key"
)

// AuditEvent represents a structured audit event before persistence
// Enhanced from internal/audit/logger.go Event
type AuditEvent struct {
	TenantID   *uuid.UUID             `json:"tenant_id,omitempty"`
	ActorID    *uuid.UUID             `json:"actor_id,omitempty"`
	ActorType  ActorType              `json:"actor_type"`
	ActorEmail string                 `json:"actor_email,omitempty"`
	Action     string                 `json:"action"`
	ObjectType string                 `json:"object_type"`
	ObjectID   uuid.UUID              `json:"object_id"`
	Before     map[string]interface{} `json:"before,omitempty"`
	After      map[string]interface{} `json:"after,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	RequestID  string                 `json:"request_id,omitempty"`
}

// AuditAction represents standardized audit actions
type AuditAction string

const (
	// Tenant actions
	AuditActionTenantCreate    AuditAction = "tenant.create"
	AuditActionTenantUpdate    AuditAction = "tenant.update"
	AuditActionTenantSuspend   AuditAction = "tenant.suspend"
	AuditActionTenantRestore   AuditAction = "tenant.restore"
	AuditActionTenantDelete    AuditAction = "tenant.delete"

	// Policy actions
	AuditActionPolicyCreate    AuditAction = "policy.create"
	AuditActionPolicyUpdate    AuditAction = "policy.update"
	AuditActionPolicyAssign    AuditAction = "policy.assign"
	AuditActionPolicyUnassign  AuditAction = "policy.unassign"
	AuditActionPolicyDelete    AuditAction = "policy.delete"

	// Provider key actions
	AuditActionProviderKeyAdd    AuditAction = "provider_key.add"
	AuditActionProviderKeyRotate AuditAction = "provider_key.rotate"
	AuditActionProviderKeyRevoke AuditAction = "provider_key.revoke"

	// API token actions
	AuditActionAPITokenCreate AuditAction = "api_token.create"
	AuditActionAPITokenRevoke AuditAction = "api_token.revoke"

	// Quota actions
	AuditActionQuotaUpdate AuditAction = "quota.update"
	AuditActionQuotaExceed AuditAction = "quota.exceed"

	// Security actions
	AuditActionViolationDetected AuditAction = "violation.detected"
	AuditActionRequestBlocked    AuditAction = "request.blocked"
	AuditActionRequestAllowed    AuditAction = "request.allowed"
)

// ObjectType represents the type of object being audited
type ObjectType string

const (
	ObjectTypeTenant      ObjectType = "tenant"
	ObjectTypePolicy      ObjectType = "policy"
	ObjectTypeProviderKey ObjectType = "provider_key"
	ObjectTypeAPIToken    ObjectType = "api_token"
	ObjectTypeQuota       ObjectType = "quota"
	ObjectTypeRequest     ObjectType = "request"
	ObjectTypeViolation   ObjectType = "violation"
)

// AuditMetadata provides structured metadata for audit entries
type AuditMetadata struct {
	RequestID   string            `json:"request_id,omitempty"`
	UserAgent   string            `json:"user_agent,omitempty"`
	IPAddress   string            `json:"ip_address,omitempty"`
	Provider    Provider          `json:"provider,omitempty"`
	Endpoint    string            `json:"endpoint,omitempty"`
	HTTPMethod  string            `json:"http_method,omitempty"`
	StatusCode  int               `json:"status_code,omitempty"`
	ProcessingMs int64            `json:"processing_ms,omitempty"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}