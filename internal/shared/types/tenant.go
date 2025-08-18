package types

import (
	"time"

	"github.com/google/uuid"
)

// TenantID represents a tenant identifier
type TenantID uuid.UUID

// String returns the string representation of the tenant ID
func (t TenantID) String() string {
	return uuid.UUID(t).String()
}

// ParseTenantID parses a string into a TenantID
func ParseTenantID(s string) (TenantID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return TenantID{}, err
	}
	return TenantID(id), nil
}

// TenantContext represents tenant-specific context for requests
type TenantContext struct {
	ID       TenantID `json:"id"`
	Name     string   `json:"name,omitempty"`
	Status   string   `json:"status,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TenantQuota represents rate limiting configuration for a tenant
type TenantQuota struct {
	TenantID             TenantID `json:"tenant_id"`
	RequestsPerMinute    *int     `json:"requests_per_minute,omitempty"`
	RequestsPerHour      *int     `json:"requests_per_hour,omitempty"`
	TokensPerHour        *int64   `json:"tokens_per_hour,omitempty"`
	MaxPromptTokens      *int     `json:"max_prompt_tokens,omitempty"`
	MaxCompletionTokens  *int     `json:"max_completion_tokens,omitempty"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// RateLimitResult represents the result of a rate limit check
type RateLimitResult struct {
	Allowed                     bool          `json:"allowed"`
	RequestsPerMinuteRemaining  int           `json:"requests_per_minute_remaining"`
	RequestsPerHourRemaining    int           `json:"requests_per_hour_remaining"`
	RetryAfter                  time.Duration `json:"retry_after"`
}