package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/security/crypto"
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

// ProviderKey represents encrypted API credentials for LLM providers
type ProviderKey struct {
	ID           uuid.UUID     `json:"id"`
	TenantID     uuid.UUID     `json:"tenant_id"`
	Provider     Provider      `json:"provider"`
	KeyAlias     string        `json:"key_alias"`
	EncryptedKey string        `json:"-"` // Never expose in JSON
	Endpoint     string        `json:"endpoint,omitempty"`
	Deployment   string        `json:"deployment,omitempty"` // Azure specific
	IsDefault    bool          `json:"is_default"`
	Status       KeyStatus     `json:"status"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	LastUsed     *time.Time    `json:"last_used,omitempty"`
	RotatedAt    *time.Time    `json:"rotated_at,omitempty"`
}

// DecryptKey decrypts and returns the provider API key
func (pk *ProviderKey) DecryptKey() (string, error) {
	if pk.EncryptedKey == "" {
		return "", fmt.Errorf("no encrypted key available")
	}
	return crypto.DecryptString(pk.EncryptedKey)
}

// Provider represents supported LLM providers
type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderAzure     Provider = "azure"
)

// KeyStatus represents the state of an API key
type KeyStatus string

const (
	KeyStatusActive   KeyStatus = "active"
	KeyStatusRotating KeyStatus = "rotating"
	KeyStatusExpired  KeyStatus = "expired"
	KeyStatusRevoked  KeyStatus = "revoked"
)

// Policy represents a security policy/rulepack
type Policy struct {
	ID        uuid.UUID        `json:"id"`
	Name      string           `json:"name"`
	Version   int              `json:"version"`
	Content   string           `json:"content"` // YAML/JSON rules
	Type      PolicyType       `json:"type"`
	CreatedBy *uuid.UUID       `json:"created_by,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
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
	ID                uuid.UUID     `json:"id"`
	TenantID          uuid.UUID     `json:"tenant_id"`
	Timestamp         time.Time     `json:"timestamp"`
	Window            TimeWindow    `json:"window"`
	Provider          Provider      `json:"provider,omitempty"`
	Endpoint          string        `json:"endpoint,omitempty"`
	RequestCount      int64         `json:"request_count"`
	TokenCount        int64         `json:"token_count"`
	PromptTokens      int64         `json:"prompt_tokens"`
	CompletionTokens  int64         `json:"completion_tokens"`
	Violations        int           `json:"violations"`
	BlockedRequests   int           `json:"blocked_requests"`
	LatencyP50        float64       `json:"latency_p50"`
	LatencyP95        float64       `json:"latency_p95"`
	LatencyP99        float64       `json:"latency_p99"`
}

// TimeWindow represents aggregation window
type TimeWindow string

const (
	TimeWindowMinute TimeWindow = "minute"
	TimeWindowHour   TimeWindow = "hour"
	TimeWindowDay    TimeWindow = "day"
)

// Quota represents rate limiting configuration per tenant
type Quota struct {
	ID                   uuid.UUID `json:"id"`
	TenantID             uuid.UUID `json:"tenant_id"`
	RequestsPerMinute    *int      `json:"requests_per_minute,omitempty"`
	RequestsPerHour      *int      `json:"requests_per_hour,omitempty"`
	TokensPerHour        *int64    `json:"tokens_per_hour,omitempty"`
	MaxPromptTokens      *int      `json:"max_prompt_tokens,omitempty"`
	MaxCompletionTokens  *int      `json:"max_completion_tokens,omitempty"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// APIToken represents authentication tokens for API access
type APIToken struct {
	ID        uuid.UUID    `json:"id"`
	TenantID  uuid.UUID    `json:"tenant_id"`
	TokenHash string       `json:"-"` // Never expose
	Name      string       `json:"name"`
	Scopes    []string     `json:"scopes"` // Permissions
	ExpiresAt *time.Time   `json:"expires_at,omitempty"`
	LastUsed  *time.Time   `json:"last_used,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
	RevokedAt *time.Time   `json:"revoked_at,omitempty"`
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
	Type        string    `json:"type"`
	Severity    Severity  `json:"severity"`
	Rule        string    `json:"rule"`
	Message     string    `json:"message"`
	Location    string    `json:"location,omitempty"`
	Evidence    string    `json:"evidence,omitempty"`
}

// Severity represents violation severity levels
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)