package types

import (
	"time"

	"github.com/google/uuid"
)

// Policy represents a security policy/rulepack
// From domain/models.go
type Policy struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Version   int        `json:"version"`
	Content   string     `json:"content"` // YAML/JSON rules
	Type      PolicyType `json:"type"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// PolicyType represents the type of policy
type PolicyType string

const (
	PolicyTypeBuiltin PolicyType = "builtin"
	PolicyTypeCustom  PolicyType = "custom"
	PolicyTypeManaged PolicyType = "managed"
)

// PolicyAssignment represents assignment of a policy to a tenant/route
// From domain/models.go
type PolicyAssignment struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	PolicyID    uuid.UUID `json:"policy_id"`
	TargetScope string    `json:"target_scope"` // /v1/openai/*, /v1/anthropic/*
	Priority    int       `json:"priority"`     // Higher priority wins
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// EnforcementMode represents different enforcement modes
type EnforcementMode string

const (
	EnforcementModeObserve    EnforcementMode = "observe"    // Log violations but allow
	EnforcementModeRedact     EnforcementMode = "redact"     // Redact sensitive content
	EnforcementModeQuarantine EnforcementMode = "quarantine" // Block with review option
	EnforcementModeEnforce    EnforcementMode = "enforce"    // Block immediately
)

// RuleInfo represents metadata about a rule
type RuleInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Level       int               `json:"level"` // 1=keyword, 2=regex, 3=semantic
	Severity    ViolationSeverity `json:"severity"`
	Action      string            `json:"action"` // allow, deny, quarantine
	Tags        []string          `json:"tags,omitempty"`
}

// PolicyContext provides context for policy evaluation
type PolicyContext struct {
	TenantID  uuid.UUID              `json:"tenant_id"`
	Provider  Provider               `json:"provider"`
	Endpoint  string                 `json:"endpoint"`
	UserID    *uuid.UUID             `json:"user_id,omitempty"`
	RequestID string                 `json:"request_id"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}
