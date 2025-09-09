package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Tenant represents a customer organization in the system
type Tenant struct {
	ID        uuid.UUID              `json:"id"`
	Name      string                 `json:"name"`
	Status    TenantStatus           `json:"status"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// TenantStatus represents the state of a tenant
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusDeleted   TenantStatus = "deleted"
)

// Security Gateway uses simple env-based API keys for semantic analysis
// No complex provider key management needed

// Policy represents a security policy/rulepack
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

// RulepackAssignment represents assignment of a rulepack to a tenant/route
type RulepackAssignment struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	RulepackID  uuid.UUID `json:"rulepack_id"`
	TargetScope string    `json:"target_scope"`
	Priority    int       `json:"priority"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AuditEntry represents an immutable audit log entry
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

// UsageMetric represents aggregated usage data
type UsageMetric struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	Timestamp time.Time  `json:"timestamp"`
	Window    TimeWindow `json:"window"`
	// Provider removed - Security Gateway doesn't manage providers
	Endpoint         string  `json:"endpoint,omitempty"`
	RequestCount     int64   `json:"request_count"`
	TokenCount       int64   `json:"token_count"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	Violations       int     `json:"violations"`
	BlockedRequests  int     `json:"blocked_requests"`
	LatencyP50       float64 `json:"latency_p50"`
	LatencyP95       float64 `json:"latency_p95"`
	LatencyP99       float64 `json:"latency_p99"`
}

// TimeWindow represents aggregation window
type TimeWindow string

const (
	TimeWindowMinute TimeWindow = "minute"
	TimeWindowHour   TimeWindow = "hour"
	TimeWindowDay    TimeWindow = "day"
)

// Security Gateway uses simple environment-based rate limiting
// No complex per-tenant quota management needed

// APIToken represents authentication tokens for API access
type APIToken struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	TokenHash string     `json:"-"` // Never expose
	Name      string     `json:"name"`
	Scopes    []string   `json:"scopes"` // Permissions
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// EnforcementDecision represents the result of policy enforcement
type EnforcementDecision struct {
	Allow       bool                   `json:"allow"`
	Reason      string                 `json:"reason,omitempty"`
	Violations  []Violation            `json:"violations,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ProcessedAt time.Time              `json:"processed_at"`
	Latency     time.Duration          `json:"latency_ms"`
}

// Violation represents a policy violation
type Violation struct {
	Type     string   `json:"type"`
	Severity Severity `json:"severity"`
	Rule     string   `json:"rule"`
	Message  string   `json:"message"`
	Location string   `json:"location,omitempty"`
	Evidence string   `json:"evidence,omitempty"`
}

// Severity represents violation severity levels
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)
