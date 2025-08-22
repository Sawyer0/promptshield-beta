package types

import "time"

// ServerInfo represents server information
type ServerInfo struct {
	Name       string        `json:"name"`
	Version    string        `json:"version"`
	StartTime  time.Time     `json:"start_time"`
	Uptime     time.Duration `json:"uptime"`
	GoVersion  string        `json:"go_version"`
	Platform   string        `json:"platform"`
	CPUCount   int           `json:"cpu_count"`
}

// ExtProcRequest represents an Envoy external processor request
type ExtProcRequest struct {
	RequestID string                 `json:"request_id"`
	Method    string                 `json:"method"`
	Path      string                 `json:"path"`
	Headers   map[string]string      `json:"headers"`
	Body      []byte                 `json:"body,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ExtProcResponseMsg represents an Envoy external processor response message
type ExtProcResponseMsg struct {
	RequestID string                 `json:"request_id"`
	Status    int                    `json:"status"`
	Headers   map[string]string      `json:"headers"`
	Body      []byte                 `json:"body,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ExtProcResponse represents an external processor response
type ExtProcResponse struct {
	Allow          bool              `json:"allow"`
	Status         int               `json:"status,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	Body           []byte            `json:"body,omitempty"`
	BodyMutation   *BodyMutation     `json:"body_mutation,omitempty"`
	HeaderMutation *HeaderMutation   `json:"header_mutation,omitempty"`
}

// BodyMutation represents body modification instructions
type BodyMutation struct {
	Action      MutationAction `json:"action"`
	NewBody     []byte         `json:"new_body,omitempty"`
	Redactions  []Redaction    `json:"redactions,omitempty"`
}

// HeaderMutation represents header modification instructions
type HeaderMutation struct {
	Add    map[string]string `json:"add,omitempty"`
	Remove []string          `json:"remove,omitempty"`
	Set    map[string]string `json:"set,omitempty"`
}

// MutationAction represents the type of mutation to perform
type MutationAction string

const (
	MutationActionAllow   MutationAction = "allow"
	MutationActionDeny    MutationAction = "deny"
	MutationActionRedact  MutationAction = "redact"
	MutationActionReplace MutationAction = "replace"
)

// Redaction represents a content redaction
type Redaction struct {
	Start       int    `json:"start"`
	End         int    `json:"end"`
	Replacement string `json:"replacement,omitempty"`
}

// ExtProcConfig represents external processor configuration
type ExtProcConfig struct {
	Timeout          time.Duration  `json:"timeout"`
	MaxBodySize      int64          `json:"max_body_size"`
	AllowPartialBody bool           `json:"allow_partial_body"`
	ProcessingMode   ProcessingMode `json:"processing_mode"`
}

// ProcessingMode represents the processing mode for ext_proc
type ProcessingMode string

const (
	ProcessingModeBuffered    ProcessingMode = "buffered"
	ProcessingModeStreaming   ProcessingMode = "streaming"
	ProcessingModeHeadersOnly ProcessingMode = "headers_only"
)

// Backend represents a backend service
type Backend struct {
	ID       string            `json:"id"`
	Address  string            `json:"address"`
	Port     int               `json:"port"`
	Weight   int               `json:"weight"`
	Status   BackendStatus     `json:"status"`
	Metadata map[string]string `json:"metadata,omitempty"`
	LastSeen time.Time         `json:"last_seen"`
}

// BackendStatus represents the status of a backend
type BackendStatus string

const (
	BackendStatusHealthy   BackendStatus = "healthy"
	BackendStatusUnhealthy BackendStatus = "unhealthy"
	BackendStatusDraining  BackendStatus = "draining"
)

// ConfigChangeCallback is called when configuration changes
type ConfigChangeCallback func(oldConfig, newConfig map[string]interface{}) error