package types

import "time"

// RulepackInfo represents metadata about a rulepack
// From internal/contracts/repository.go
type RulepackInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	Author      string `json:"author,omitempty"`
	RuleCount   int    `json:"rule_count"`
	LoadedAt    string `json:"loaded_at,omitempty"`
}

// Event represents a server-sent event
// From internal/interfaces/http/api/events.go
type Event struct {
	ID    string                 `json:"id"`
	Type  string                 `json:"type"`
	Data  map[string]interface{} `json:"data"`
	Retry int                    `json:"retry,omitempty"`
}

// EventHub configuration and stats
type EventHubStats struct {
	ActiveConnections int           `json:"active_connections"`
	TotalEvents      int64         `json:"total_events"`
	EventsByType     map[string]int64 `json:"events_by_type"`
	Uptime           time.Duration `json:"uptime"`
	LastEvent        *time.Time    `json:"last_event,omitempty"`
}

// APIError represents a structured API error response
// From internal/interfaces/http/api/errors.go
type APIError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	TraceID TraceID                `json:"trace_id,omitempty"`
}

// ProviderConfig represents provider configuration
// From internal/interfaces/http/api/providers.go
type ProviderConfig struct {
	Name         string                 `json:"name"`
	Endpoint     string                 `json:"endpoint"`
	APIKey       string                 `json:"api_key"`
	Models       []string               `json:"models"`
	RateLimits   ProviderRateLimits     `json:"rate_limits"`
	Timeouts     ProviderTimeouts       `json:"timeouts"`
	RetryPolicy  RetryPolicy            `json:"retry_policy"`
	Headers      map[string]string      `json:"headers,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ProviderRateLimits defines rate limiting configuration for providers
type ProviderRateLimits struct {
	RequestsPerSecond int `json:"requests_per_second"`
	RequestsPerMinute int `json:"requests_per_minute"`
	RequestsPerHour   int `json:"requests_per_hour"`
	BurstSize         int `json:"burst_size"`
}

// ProviderTimeouts defines timeout configuration for providers
type ProviderTimeouts struct {
	Connect time.Duration `json:"connect"`
	Request time.Duration `json:"request"`
	Stream  time.Duration `json:"stream"`
}

// RetryPolicy defines retry behavior for provider requests
type RetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	RetryableErrors []int         `json:"retryable_errors"`
}

// RoutingRule defines routing rules for requests
type RoutingRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Priority    int                    `json:"priority"`
	Conditions  map[string]interface{} `json:"conditions"`
	Targets     []RoutingTarget        `json:"targets"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// RoutingTarget represents a routing target
type RoutingTarget struct {
	Provider Provider `json:"provider"`
	Weight   int      `json:"weight"`
	Endpoint string   `json:"endpoint,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
}

// SupportedProvider represents a supported LLM provider
type SupportedProvider struct {
	Name         Provider  `json:"name"`
	DisplayName  string    `json:"display_name"`
	Description  string    `json:"description,omitempty"`
	Models       []string  `json:"models"`
	Capabilities []string  `json:"capabilities"`
	Status       string    `json:"status"`
}

// RuntimeConfigStore represents runtime configuration state
type RuntimeConfigStore struct {
	Version     string                 `json:"version"`
	LastUpdated time.Time              `json:"last_updated"`
	Config      map[string]interface{} `json:"config"`
	Source      string                 `json:"source"`
	Checksum    string                 `json:"checksum"`
}

// ServiceMetadata represents metadata about an application service
type ServiceMetadata struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description,omitempty"`
	StartTime   time.Time `json:"start_time"`
	Status      string    `json:"status"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// RequestContext represents context information for a request
type RequestContext struct {
	RequestID   string                 `json:"request_id"`
	TenantID    string                 `json:"tenant_id,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	TraceID     TraceID                `json:"trace_id,omitempty"`
	SpanID      SpanID                 `json:"span_id,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ValidationResult represents the result of validation
type ValidationResult struct {
	Valid   bool                   `json:"valid"`
	Errors  []ValidationError      `json:"errors,omitempty"`
	Warnings []ValidationWarning   `json:"warnings,omitempty"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}