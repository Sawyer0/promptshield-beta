package types

import (
	"time"

	"github.com/google/uuid"
)

// Quota represents rate limiting configuration per tenant
// From domain/models.go
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

// UsageMetric represents aggregated usage data
// From domain/models.go
type UsageMetric struct {
	ID                uuid.UUID  `json:"id"`
	TenantID          uuid.UUID  `json:"tenant_id"`
	Timestamp         time.Time  `json:"timestamp"`
	Window            TimeWindow `json:"window"`
	Provider          Provider   `json:"provider,omitempty"`
	Endpoint          string     `json:"endpoint,omitempty"`
	RequestCount      int64      `json:"request_count"`
	TokenCount        int64      `json:"token_count"`
	PromptTokens      int64      `json:"prompt_tokens"`
	CompletionTokens  int64      `json:"completion_tokens"`
	Violations        int        `json:"violations"`
	BlockedRequests   int        `json:"blocked_requests"`
	LatencyP50        float64    `json:"latency_p50"`
	LatencyP95        float64    `json:"latency_p95"`
	LatencyP99        float64    `json:"latency_p99"`
}

// TimeWindow represents aggregation window
type TimeWindow string

const (
	TimeWindowMinute TimeWindow = "minute"
	TimeWindowHour   TimeWindow = "hour"
	TimeWindowDay    TimeWindow = "day"
)

// UsageRecord represents a single usage record
// From internal/usage/store.go
type UsageRecord struct {
	ID               uuid.UUID `json:"id"`
	TenantID         uuid.UUID `json:"tenant_id"`
	RequestID        string    `json:"request_id"`
	Provider         Provider  `json:"provider"`
	Model            string    `json:"model"`
	Endpoint         string    `json:"endpoint"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	LatencyMs        int64     `json:"latency_ms"`
	Status           string    `json:"status"` // success, error, blocked
	Timestamp        time.Time `json:"timestamp"`
}

// UsageQuery represents query parameters for usage data
// From internal/usage/store.go
type UsageQuery struct {
	TenantID  uuid.UUID  `json:"tenant_id"`
	StartTime time.Time  `json:"start_time"`
	EndTime   time.Time  `json:"end_time"`
	Provider  *Provider  `json:"provider,omitempty"`
	Endpoint  string     `json:"endpoint,omitempty"`
	Window    TimeWindow `json:"window"`
	Limit     int        `json:"limit"`
	Offset    int        `json:"offset"`
}

// UsageResult represents aggregated usage results
// From internal/usage/store.go
type UsageResult struct {
	TotalRequests    int64               `json:"total_requests"`
	TotalTokens      int64               `json:"total_tokens"`
	TotalViolations  int                 `json:"total_violations"`
	AverageLatency   float64             `json:"average_latency"`
	Rows             []UsageAggregateRow `json:"rows"`
}

// UsageAggregateRow represents a single row of aggregated usage data
// From internal/usage/store.go
type UsageAggregateRow struct {
	Timestamp        time.Time `json:"timestamp"`
	Provider         Provider  `json:"provider,omitempty"`
	Endpoint         string    `json:"endpoint,omitempty"`
	RequestCount     int64     `json:"request_count"`
	TokenCount       int64     `json:"token_count"`
	PromptTokens     int64     `json:"prompt_tokens"`
	CompletionTokens int64     `json:"completion_tokens"`
	Violations       int       `json:"violations"`
	BlockedRequests  int       `json:"blocked_requests"`
	AverageLatency   float64   `json:"average_latency"`
}

// QuotaExceeded represents quota limit violations
type QuotaExceeded struct {
	TenantID    uuid.UUID `json:"tenant_id"`
	QuotaType   string    `json:"quota_type"`   // requests_per_minute, tokens_per_hour, etc.
	Current     int64     `json:"current"`      // Current usage
	Limit       int64     `json:"limit"`        // Configured limit
	Window      string    `json:"window"`       // Time window
	ResetsAt    time.Time `json:"resets_at"`    // When quota resets
	RetryAfter  int       `json:"retry_after"`  // Seconds to wait
}

// APIToken represents authentication tokens for API access
// From domain/models.go - moved here as it's related to quota/usage tracking
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

// ProviderKey represents encrypted API credentials for LLM providers
// From domain/models.go - moved here as it's related to provider usage
type ProviderKey struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	Provider     Provider   `json:"provider"`
	KeyAlias     string     `json:"key_alias"`
	EncryptedKey string     `json:"-"` // Never expose in JSON
	Endpoint     string     `json:"endpoint,omitempty"`
	Deployment   string     `json:"deployment,omitempty"` // Azure specific
	IsDefault    bool       `json:"is_default"`
	Status       KeyStatus  `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastUsed     *time.Time `json:"last_used,omitempty"`
	RotatedAt    *time.Time `json:"rotated_at,omitempty"`
}

// KeyStatus represents the state of an API key
type KeyStatus string

const (
	KeyStatusActive   KeyStatus = "active"
	KeyStatusRotating KeyStatus = "rotating"
	KeyStatusExpired  KeyStatus = "expired"
	KeyStatusRevoked  KeyStatus = "revoked"
)