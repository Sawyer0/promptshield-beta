package contracts

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// SecurityManager defines the interface for security operations
type SecurityManager interface {
	// CreateSecurityContext creates a security context for a request
	CreateSecurityContext(ctx context.Context, requestID string, userID string, tenantID string) (*types.SecurityContext, error)
	
	// ValidateSecurityContext validates a security context
	ValidateSecurityContext(ctx context.Context, securityCtx *types.SecurityContext) error
	
	// EnrichSecurityContext enriches context with additional security information
	EnrichSecurityContext(ctx context.Context, securityCtx *types.SecurityContext) error
	
	// GetPermissions returns permissions for a security context
	GetPermissions(ctx context.Context, securityCtx *types.SecurityContext) ([]string, error)
	
	// CheckPermission checks if a security context has a specific permission
	CheckPermission(ctx context.Context, securityCtx *types.SecurityContext, permission string) (bool, error)
	
	// LogSecurityEvent logs a security-related event
	LogSecurityEvent(ctx context.Context, event *types.SecurityEvent) error
	
	// GetSecurityEvents retrieves security events with filtering
	GetSecurityEvents(ctx context.Context, filter *types.SecurityEventFilter) ([]*types.SecurityEvent, error)
}

// EncryptionService defines the interface for encryption operations
type EncryptionService interface {
	// Encrypt encrypts data using the specified key
	Encrypt(ctx context.Context, data []byte, keyID string) ([]byte, error)
	
	// Decrypt decrypts data using the specified key
	Decrypt(ctx context.Context, encryptedData []byte, keyID string) ([]byte, error)
	
	// GenerateKey generates a new encryption key
	GenerateKey(ctx context.Context, keyType types.KeyType, keySize int) (*types.EncryptionKey, error)
	
	// RotateKey rotates an encryption key
	RotateKey(ctx context.Context, keyID string) (*types.EncryptionKey, error)
	
	// GetKey retrieves an encryption key by ID
	GetKey(ctx context.Context, keyID string) (*types.EncryptionKey, error)
	
	// ListKeys lists all available encryption keys
	ListKeys(ctx context.Context) ([]*types.EncryptionKey, error)
	
	// DeleteKey securely deletes an encryption key
	DeleteKey(ctx context.Context, keyID string) error
	
	// Sign signs data using the specified signing key
	Sign(ctx context.Context, data []byte, keyID string) ([]byte, error)
	
	// Verify verifies a signature using the specified verification key
	Verify(ctx context.Context, data []byte, signature []byte, keyID string) (bool, error)
}

// CredentialManager defines the interface for credential management
type CredentialManager interface {
	// Store stores a credential securely
	Store(ctx context.Context, credential *types.Credential) error
	
	// Retrieve retrieves a credential by ID
	Retrieve(ctx context.Context, credentialID string) (*types.Credential, error)
	
	// Update updates an existing credential
	Update(ctx context.Context, credentialID string, credential *types.Credential) error
	
	// Delete deletes a credential
	Delete(ctx context.Context, credentialID string) error
	
	// List lists all credentials for a user/tenant
	List(ctx context.Context, userID string, tenantID string) ([]*types.Credential, error)
	
	// Rotate rotates a credential (generates new value)
	Rotate(ctx context.Context, credentialID string) (*types.Credential, error)
	
	// ValidateCredential validates a credential format and strength
	ValidateCredential(ctx context.Context, credential *types.Credential) (*types.ValidationResult, error)
	
	// GetCredentialHistory returns credential rotation history
	GetCredentialHistory(ctx context.Context, credentialID string) ([]*types.CredentialHistory, error)
}

// TLSManager defines the interface for TLS configuration management
type TLSManager interface {
	// GetTLSConfig returns TLS configuration for a service
	GetTLSConfig(ctx context.Context, serviceName string) (*tls.Config, error)
	
	// CreateTLSConfig creates a new TLS configuration
	CreateTLSConfig(ctx context.Context, config *types.TLSConfig) (*tls.Config, error)
	
	// ValidateCertificate validates a TLS certificate
	ValidateCertificate(ctx context.Context, certPEM []byte) (*types.CertificateInfo, error)
	
	// GetCertificateInfo returns information about a certificate
	GetCertificateInfo(ctx context.Context, certPEM []byte) (*types.CertificateInfo, error)
	
	// IssueCertificate issues a new TLS certificate
	IssueCertificate(ctx context.Context, request *types.CertificateRequest) (*types.Certificate, error)
	
	// RevokeCertificate revokes a TLS certificate
	RevokeCertificate(ctx context.Context, serialNumber string) error
	
	// GetCRL returns the certificate revocation list
	GetCRL(ctx context.Context) ([]byte, error)
	
	// VerifyCertificateChain verifies a certificate chain
	VerifyCertificateChain(ctx context.Context, certChain [][]byte) error
}

// AccessControlManager defines the interface for access control operations
type AccessControlManager interface {
	// CheckAccess checks if a user has access to a resource
	CheckAccess(ctx context.Context, userID string, resource string, action string) (bool, error)
	
	// GrantAccess grants access to a resource
	GrantAccess(ctx context.Context, userID string, resource string, permissions []string) error
	
	// RevokeAccess revokes access to a resource
	RevokeAccess(ctx context.Context, userID string, resource string) error
	
	// GetUserPermissions returns all permissions for a user
	GetUserPermissions(ctx context.Context, userID string) (map[string][]string, error)
	
	// GetResourcePermissions returns all users with access to a resource
	GetResourcePermissions(ctx context.Context, resource string) (map[string][]string, error)
	
	// CreateRole creates a new role
	CreateRole(ctx context.Context, role *types.Role) error
	
	// AssignRole assigns a role to a user
	AssignRole(ctx context.Context, userID string, roleID string) error
	
	// UnassignRole removes a role from a user
	UnassignRole(ctx context.Context, userID string, roleID string) error
	
	// GetUserRoles returns all roles for a user
	GetUserRoles(ctx context.Context, userID string) ([]*types.Role, error)
}

// AuditLogger defines the interface for security audit logging
type AuditLogger interface {
	// LogAccess logs an access attempt
	LogAccess(ctx context.Context, userID string, resource string, action string, result bool) error
	
	// LogAuthentication logs an authentication attempt
	LogAuthentication(ctx context.Context, userID string, method string, success bool, clientIP string) error
	
	// LogAuthorization logs an authorization check
	LogAuthorization(ctx context.Context, userID string, resource string, permission string, granted bool) error
	
	// LogSecurityEvent logs a general security event
	LogSecurityEvent(ctx context.Context, event *types.SecurityEvent) error
	
	// GetAuditTrail returns audit trail for a user/resource
	GetAuditTrail(ctx context.Context, filter *types.AuditFilter) ([]*types.AuditEvent, error)
	
	// ExportAuditLogs exports audit logs in a specific format
	ExportAuditLogs(ctx context.Context, filter *types.AuditFilter, format string) ([]byte, error)
}

// SecretManager defines the interface for secret management
type SecretManager interface {
	// CreateSecret creates a new secret
	CreateSecret(ctx context.Context, secret *types.Secret) error
	
	// GetSecret retrieves a secret by ID
	GetSecret(ctx context.Context, secretID string) (*types.Secret, error)
	
	// UpdateSecret updates an existing secret
	UpdateSecret(ctx context.Context, secretID string, secret *types.Secret) error
	
	// DeleteSecret deletes a secret
	DeleteSecret(ctx context.Context, secretID string) error
	
	// ListSecrets lists all secrets for a namespace
	ListSecrets(ctx context.Context, namespace string) ([]*types.Secret, error)
	
	// RotateSecret rotates a secret value
	RotateSecret(ctx context.Context, secretID string) error
	
	// GetSecretVersion retrieves a specific version of a secret
	GetSecretVersion(ctx context.Context, secretID string, version int) (*types.Secret, error)
	
	// GetSecretHistory returns version history for a secret
	GetSecretHistory(ctx context.Context, secretID string) ([]*types.SecretVersion, error)
}

// HashService defines the interface for hashing operations
type HashService interface {
	// Hash hashes data using the specified algorithm
	Hash(data []byte, algorithm types.HashAlgorithm) ([]byte, error)
	
	// HashWithSalt hashes data with a salt
	HashWithSalt(data []byte, salt []byte, algorithm types.HashAlgorithm) ([]byte, error)
	
	// GenerateSalt generates a cryptographically secure salt
	GenerateSalt(length int) ([]byte, error)
	
	// VerifyHash verifies that data matches a hash
	VerifyHash(data []byte, hash []byte, algorithm types.HashAlgorithm) (bool, error)
	
	// VerifyHashWithSalt verifies data against a salted hash
	VerifyHashWithSalt(data []byte, hash []byte, salt []byte, algorithm types.HashAlgorithm) (bool, error)
	
	// HMAC creates an HMAC of data using a key
	HMAC(data []byte, key []byte, algorithm types.HashAlgorithm) ([]byte, error)
	
	// VerifyHMAC verifies an HMAC
	VerifyHMAC(data []byte, mac []byte, key []byte, algorithm types.HashAlgorithm) (bool, error)
}

// TokenManager defines the interface for token management
type TokenManager interface {
	// GenerateToken generates a new token
	GenerateToken(ctx context.Context, tokenType types.TokenType, userID string, expiresIn time.Duration) (*types.Token, error)
	
	// ValidateToken validates a token
	ValidateToken(ctx context.Context, tokenValue string) (*types.TokenValidationResult, error)
	
	// RefreshToken refreshes an existing token
	RefreshToken(ctx context.Context, refreshToken string) (*types.Token, error)
	
	// RevokeToken revokes a token
	RevokeToken(ctx context.Context, tokenValue string) error
	
	// RevokeAllUserTokens revokes all tokens for a user
	RevokeAllUserTokens(ctx context.Context, userID string) error
	
	// GetTokenInfo returns information about a token
	GetTokenInfo(ctx context.Context, tokenValue string) (*types.TokenInfo, error)
	
	// ListUserTokens lists all tokens for a user
	ListUserTokens(ctx context.Context, userID string) ([]*types.TokenInfo, error)
	
	// CleanupExpiredTokens removes expired tokens
	CleanupExpiredTokens(ctx context.Context) error
}