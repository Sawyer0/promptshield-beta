package types

import (
	"time"

	"github.com/google/uuid"
)

// SecurityContext represents security-related context for requests
type SecurityContext struct {
	TenantID      uuid.UUID              `json:"tenant_id"`
	UserID        *uuid.UUID             `json:"user_id,omitempty"`
	APITokenID    *uuid.UUID             `json:"api_token_id,omitempty"`
	Scopes        []string               `json:"scopes,omitempty"`
	IPAddress     string                 `json:"ip_address,omitempty"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	RequestID     string                 `json:"request_id"`
	IssuedAt      time.Time              `json:"issued_at"`
	ExpiresAt     *time.Time             `json:"expires_at,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// EncryptionKey represents a cryptographic key for data encryption
type EncryptionKey struct {
	ID          uuid.UUID     `json:"id"`
	KeyType     KeyType       `json:"key_type"`
	Algorithm   string        `json:"algorithm"`
	KeySize     int           `json:"key_size"`
	EncryptedKey string       `json:"-"` // Never expose
	Checksum    string        `json:"checksum"`
	Status      KeyStatus     `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	RotatedAt   *time.Time    `json:"rotated_at,omitempty"`
	ExpiresAt   *time.Time    `json:"expires_at,omitempty"`
}

// KeyType represents the type of encryption key
type KeyType string

const (
	KeyTypeDataEncryption KeyType = "data_encryption"
	KeyTypeTokenSigning   KeyType = "token_signing"
	KeyTypeAPIKey         KeyType = "api_key"
)

// TLSConfig represents TLS configuration for secure connections
type TLSConfig struct {
	Enabled            bool     `json:"enabled"`
	CertFile           string   `json:"cert_file,omitempty"`
	KeyFile            string   `json:"key_file,omitempty"`
	CAFile             string   `json:"ca_file,omitempty"`
	InsecureSkipVerify bool     `json:"insecure_skip_verify,omitempty"`
	MinVersion         string   `json:"min_version,omitempty"`
	CipherSuites       []string `json:"cipher_suites,omitempty"`
}

// AuthenticationResult represents the result of authentication
type AuthenticationResult struct {
	Success       bool                   `json:"success"`
	TenantID      *uuid.UUID             `json:"tenant_id,omitempty"`
	UserID        *uuid.UUID             `json:"user_id,omitempty"`
	APITokenID    *uuid.UUID             `json:"api_token_id,omitempty"`
	Scopes        []string               `json:"scopes,omitempty"`
	ErrorCode     string                 `json:"error_code,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// AuthorizationResult represents the result of authorization check
type AuthorizationResult struct {
	Allowed      bool     `json:"allowed"`
	RequiredScope string  `json:"required_scope,omitempty"`
	ActualScopes []string `json:"actual_scopes,omitempty"`
	Reason       string   `json:"reason,omitempty"`
}

// SecurityPolicy represents security policy configuration
type SecurityPolicy struct {
	ID                    uuid.UUID `json:"id"`
	TenantID              uuid.UUID `json:"tenant_id"`
	Name                  string    `json:"name"`
	RequireTLS            bool      `json:"require_tls"`
	AllowedIPRanges       []string  `json:"allowed_ip_ranges,omitempty"`
	BlockedIPRanges       []string  `json:"blocked_ip_ranges,omitempty"`
	RequiredScopes        []string  `json:"required_scopes,omitempty"`
	MaxRequestSize        int64     `json:"max_request_size"`
	RateLimitPerMinute    int       `json:"rate_limit_per_minute"`
	SessionTimeoutMinutes int       `json:"session_timeout_minutes"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// SecurityEvent represents a security-related event
type SecurityEvent struct {
	ID            uuid.UUID              `json:"id"`
	Type          SecurityEventType      `json:"type"`
	Severity      Severity               `json:"severity"`
	TenantID      *uuid.UUID             `json:"tenant_id,omitempty"`
	UserID        *uuid.UUID             `json:"user_id,omitempty"`
	IPAddress     string                 `json:"ip_address,omitempty"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	RequestID     string                 `json:"request_id,omitempty"`
	Description   string                 `json:"description"`
	Details       map[string]interface{} `json:"details,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	Resolved      bool                   `json:"resolved"`
	ResolvedAt    *time.Time             `json:"resolved_at,omitempty"`
}

// SecurityEventType represents types of security events
type SecurityEventType string

const (
	SecurityEventTypeAuthFailure       SecurityEventType = "auth_failure"
	SecurityEventTypeAuthSuccess       SecurityEventType = "auth_success"
	SecurityEventTypeUnauthorizedAccess SecurityEventType = "unauthorized_access"
	SecurityEventTypeRateLimitExceeded SecurityEventType = "rate_limit_exceeded"
	SecurityEventTypeSuspiciousActivity SecurityEventType = "suspicious_activity"
	SecurityEventTypeDataBreach        SecurityEventType = "data_breach"
	SecurityEventTypeKeyRotation       SecurityEventType = "key_rotation"
	SecurityEventTypePolicyViolation   SecurityEventType = "policy_violation"
)

// Credential represents generic credential information
type Credential struct {
	Type       CredentialType `json:"type"`
	Value      string         `json:"-"` // Never expose
	ExpiresAt  *time.Time     `json:"expires_at,omitempty"`
	Scopes     []string       `json:"scopes,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// CredentialType represents the type of credential
type CredentialType string

const (
	CredentialTypeAPIKey      CredentialType = "api_key"
	CredentialTypeBearerToken CredentialType = "bearer_token"
	CredentialTypeBasicAuth   CredentialType = "basic_auth"
	CredentialTypeOAuth       CredentialType = "oauth"
	CredentialTypeJWT         CredentialType = "jwt"
)