package types

import (
	"time"
)

// SecurityContext represents a security context for a request
type SecurityContext struct {
	RequestID   string                 `json:"request_id"`
	UserID      string                 `json:"user_id"`
	TenantID    string                 `json:"tenant_id"`
	SessionID   string                 `json:"session_id,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	Permissions []string               `json:"permissions,omitempty"`
	Roles       []string               `json:"roles,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
}

// SecurityEvent represents a security-related event
type SecurityEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	UserID    string                 `json:"user_id,omitempty"`
	TenantID  string                 `json:"tenant_id,omitempty"`
	Resource  string                 `json:"resource,omitempty"`
	Action    string                 `json:"action,omitempty"`
	Result    bool                   `json:"result"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Severity  string                 `json:"severity,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SecurityEventFilter represents a filter for security events
type SecurityEventFilter struct {
	UserID    string                 `json:"user_id,omitempty"`
	TenantID  string                 `json:"tenant_id,omitempty"`
	EventType string                 `json:"event_type,omitempty"`
	Resource  string                 `json:"resource,omitempty"`
	Action    string                 `json:"action,omitempty"`
	Result    *bool                  `json:"result,omitempty"`
	Severity  string                 `json:"severity,omitempty"`
	StartTime *time.Time             `json:"start_time,omitempty"`
	EndTime   *time.Time             `json:"end_time,omitempty"`
	Limit     int                    `json:"limit,omitempty"`
	Offset    int                    `json:"offset,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// KeyType represents the type of encryption key
type KeyType string

const (
	KeyTypeAES      KeyType = "aes"
	KeyTypeRSA      KeyType = "rsa"
	KeyTypeEC       KeyType = "ec"
	KeyTypeChaCha20 KeyType = "chacha20"
)

// EncryptionKey represents an encryption key
type EncryptionKey struct {
	ID          string                 `json:"id"`
	Type        KeyType                `json:"type"`
	Size        int                    `json:"size"`
	Algorithm   string                 `json:"algorithm"`
	KeyMaterial []byte                 `json:"key_material,omitempty"`
	PublicKey   []byte                 `json:"public_key,omitempty"`
	PrivateKey  []byte                 `json:"private_key,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Credential represents a credential
type Credential struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	TenantID  string                 `json:"tenant_id"`
	Type      string                 `json:"type"` // "password", "api_key", "certificate"
	Name      string                 `json:"name"`
	Value     []byte                 `json:"value,omitempty"`
	Hash      []byte                 `json:"hash,omitempty"`
	Salt      []byte                 `json:"salt,omitempty"`
	Algorithm string                 `json:"algorithm,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	ExpiresAt *time.Time             `json:"expires_at,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// CredentialHistory represents credential rotation history
type CredentialHistory struct {
	ID           string                 `json:"id"`
	CredentialID string                 `json:"credential_id"`
	UserID       string                 `json:"user_id"`
	TenantID     string                 `json:"tenant_id"`
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	Hash         []byte                 `json:"hash,omitempty"`
	Salt         []byte                 `json:"salt,omitempty"`
	Algorithm    string                 `json:"algorithm,omitempty"`
	RotatedAt    time.Time              `json:"rotated_at"`
	RotatedBy    string                 `json:"rotated_by"`
	Reason       string                 `json:"reason,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	CertFile           string                 `json:"cert_file,omitempty"`
	KeyFile            string                 `json:"key_file,omitempty"`
	CAFile             string                 `json:"ca_file,omitempty"`
	MinVersion         string                 `json:"min_version,omitempty"`
	MaxVersion         string                 `json:"max_version,omitempty"`
	CipherSuites       []string               `json:"cipher_suites,omitempty"`
	InsecureSkipVerify bool                   `json:"insecure_skip_verify,omitempty"`
	ServerName         string                 `json:"server_name,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// CertificateInfo represents certificate information
type CertificateInfo struct {
	Subject          string                 `json:"subject"`
	Issuer           string                 `json:"issuer"`
	SerialNumber     string                 `json:"serial_number"`
	NotBefore        time.Time              `json:"not_before"`
	NotAfter         time.Time              `json:"not_after"`
	DNSNames         []string               `json:"dns_names,omitempty"`
	IPAddresses      []string               `json:"ip_addresses,omitempty"`
	KeyUsage         []string               `json:"key_usage,omitempty"`
	ExtKeyUsage      []string               `json:"ext_key_usage,omitempty"`
	IsValid          bool                   `json:"is_valid"`
	ValidationErrors []string               `json:"validation_errors,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// CertificateRequest represents a certificate signing request
type CertificateRequest struct {
	CommonName   string                 `json:"common_name"`
	Organization string                 `json:"organization,omitempty"`
	Country      string                 `json:"country,omitempty"`
	State        string                 `json:"state,omitempty"`
	Locality     string                 `json:"locality,omitempty"`
	DNSNames     []string               `json:"dns_names,omitempty"`
	IPAddresses  []string               `json:"ip_addresses,omitempty"`
	KeyType      string                 `json:"key_type"`
	KeySize      int                    `json:"key_size"`
	ValidityDays int                    `json:"validity_days"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Certificate represents a TLS certificate
type Certificate struct {
	ID           string                 `json:"id"`
	CommonName   string                 `json:"common_name"`
	Subject      string                 `json:"subject"`
	Issuer       string                 `json:"issuer"`
	SerialNumber string                 `json:"serial_number"`
	NotBefore    time.Time              `json:"not_before"`
	NotAfter     time.Time              `json:"not_after"`
	CertPEM      []byte                 `json:"cert_pem"`
	KeyPEM       []byte                 `json:"key_pem,omitempty"`
	CAChain      [][]byte               `json:"ca_chain,omitempty"`
	Status       string                 `json:"status"` // "active", "expired", "revoked"
	CreatedAt    time.Time              `json:"created_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Role represents a role in the access control system
type Role struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Permissions []string               `json:"permissions"`
	TenantID    string                 `json:"tenant_id,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AuditFilter represents a filter for audit events
type AuditFilter struct {
	UserID    string                 `json:"user_id,omitempty"`
	TenantID  string                 `json:"tenant_id,omitempty"`
	Resource  string                 `json:"resource,omitempty"`
	Action    string                 `json:"action,omitempty"`
	EventType string                 `json:"event_type,omitempty"`
	StartTime *time.Time             `json:"start_time,omitempty"`
	EndTime   *time.Time             `json:"end_time,omitempty"`
	Limit     int                    `json:"limit,omitempty"`
	Offset    int                    `json:"offset,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Secret represents a secret
type Secret struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Namespace string                 `json:"namespace"`
	Type      string                 `json:"type"` // "password", "api_key", "certificate", "token"
	Value     []byte                 `json:"value,omitempty"`
	Version   int                    `json:"version"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	ExpiresAt *time.Time             `json:"expires_at,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SecretVersion represents a version of a secret
type SecretVersion struct {
	ID        string                 `json:"id"`
	SecretID  string                 `json:"secret_id"`
	Version   int                    `json:"version"`
	Value     []byte                 `json:"value,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	CreatedBy string                 `json:"created_by"`
	Reason    string                 `json:"reason,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// HashAlgorithm represents a hashing algorithm
type HashAlgorithm string

const (
	HashAlgorithmSHA256 HashAlgorithm = "sha256"
	HashAlgorithmSHA512 HashAlgorithm = "sha512"
	HashAlgorithmBcrypt HashAlgorithm = "bcrypt"
	HashAlgorithmArgon2 HashAlgorithm = "argon2"
	HashAlgorithmScrypt HashAlgorithm = "scrypt"
)

// TokenType represents a token type
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
	TokenTypeAPI     TokenType = "api"
	TokenTypeService TokenType = "service"
)

// Token represents a token
type Token struct {
	ID           string                 `json:"id"`
	Type         TokenType              `json:"type"`
	UserID       string                 `json:"user_id"`
	Value        string                 `json:"value"`
	RefreshToken string                 `json:"refresh_token,omitempty"`
	Scopes       []string               `json:"scopes,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	ExpiresAt    time.Time              `json:"expires_at"`
	RevokedAt    *time.Time             `json:"revoked_at,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// TokenInfo represents token information
type TokenInfo struct {
	ID         string                 `json:"id"`
	Type       TokenType              `json:"type"`
	UserID     string                 `json:"user_id"`
	Scopes     []string               `json:"scopes,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	ExpiresAt  time.Time              `json:"expires_at"`
	RevokedAt  *time.Time             `json:"revoked_at,omitempty"`
	LastUsedAt *time.Time             `json:"last_used_at,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}
