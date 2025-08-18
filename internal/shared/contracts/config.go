package contracts

import (
	"context"
	"time"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// ConfigManager defines the interface for managing application configuration
type ConfigManager interface {
	// Get retrieves a configuration value by key
	Get(key string) (interface{}, error)

	// GetString retrieves a string configuration value
	GetString(key string) (string, error)

	// GetInt retrieves an integer configuration value
	GetInt(key string) (int, error)

	// GetBool retrieves a boolean configuration value
	GetBool(key string) (bool, error)

	// GetDuration retrieves a duration configuration value
	GetDuration(key string) (time.Duration, error)

	// GetStringSlice retrieves a string slice configuration value
	GetStringSlice(key string) ([]string, error)

	// Set sets a configuration value
	Set(key string, value interface{}) error

	// IsSet returns true if a key is set
	IsSet(key string) bool

	// GetAll returns all configuration values
	GetAll() map[string]interface{}

	// Watch watches for configuration changes
	Watch(key string, callback types.ConfigCallback) error

	// Reload reloads configuration from source
	Reload() error

	// Validate validates the current configuration
	Validate() error
}

// ConfigSource defines the interface for configuration sources
type ConfigSource interface {
	// Load loads configuration from the source
	Load() (map[string]interface{}, error)

	// Watch watches for configuration changes
	Watch(ctx context.Context, callback types.ConfigSourceCallback) error

	// Name returns the source name
	Name() string

	// Priority returns the source priority (higher = more important)
	Priority() int
}

// ConfigValidator defines the interface for configuration validation
type ConfigValidator interface {
	// Validate validates configuration values
	Validate(config map[string]interface{}) error

	// ValidateField validates a specific configuration field
	ValidateField(key string, value interface{}) error

	// GetSchema returns the configuration schema
	GetSchema() *types.ConfigSchema

	// GetValidationErrors returns validation errors
	GetValidationErrors() []types.ConfigValidationError
}

// EnvironmentProvider defines the interface for environment-based configuration
type EnvironmentProvider interface {
	// GetEnv retrieves an environment variable
	GetEnv(key string) string

	// SetEnv sets an environment variable
	SetEnv(key, value string) error

	// GetAllEnv returns all environment variables
	GetAllEnv() map[string]string

	// ExpandEnv expands environment variables in a string
	ExpandEnv(s string) string
}

// FileConfigProvider defines the interface for file-based configuration
type FileConfigProvider interface {
	// LoadFile loads configuration from a file
	LoadFile(path string) (map[string]interface{}, error)

	// SaveFile saves configuration to a file
	SaveFile(path string, config map[string]interface{}) error

	// WatchFile watches a file for changes
	WatchFile(path string, callback types.FileChangeCallback) error

	// GetSupportedFormats returns supported file formats
	GetSupportedFormats() []string
}

// RemoteConfigProvider defines the interface for remote configuration sources
type RemoteConfigProvider interface {
	// LoadRemote loads configuration from a remote source
	LoadRemote(endpoint string, auth types.AuthConfig) (map[string]interface{}, error)

	// WatchRemote watches a remote source for changes
	WatchRemote(endpoint string, auth types.AuthConfig, callback types.RemoteChangeCallback) error

	// GetEndpoints returns available endpoints
	GetEndpoints() []string
}

// ConfigEncryption defines the interface for configuration encryption
type ConfigEncryption interface {
	// Encrypt encrypts a configuration value
	Encrypt(value string) (string, error)

	// Decrypt decrypts a configuration value
	Decrypt(encrypted string) (string, error)

	// IsEncrypted returns true if a value is encrypted
	IsEncrypted(value string) bool

	// GetEncryptionKey returns the encryption key ID
	GetEncryptionKey() string
}

// ConfigMigrator defines the interface for configuration migration
type ConfigMigrator interface {
	// Migrate migrates configuration from one version to another
	Migrate(from, to string, config map[string]interface{}) (map[string]interface{}, error)

	// GetCurrentVersion returns the current configuration version
	GetCurrentVersion(config map[string]interface{}) string

	// GetSupportedVersions returns supported configuration versions
	GetSupportedVersions() []string

	// ValidateMigration validates a configuration migration
	ValidateMigration(from, to string) error
}

// ConfigTemplate defines the interface for configuration templates
type ConfigTemplate interface {
	// Render renders a configuration template
	Render(template string, data map[string]interface{}) (string, error)

	// ParseTemplate parses a configuration template
	ParseTemplate(template string) (*types.ParsedTemplate, error)

	// GetVariables returns variables used in a template
	GetVariables(template string) ([]string, error)

	// ValidateTemplate validates a configuration template
	ValidateTemplate(template string) error
}

// SecretManager defines the interface for managing secrets in configuration
type SecretManager interface {
	// GetSecret retrieves a secret value
	GetSecret(key string) (string, error)

	// SetSecret stores a secret value
	SetSecret(key, value string) error

	// DeleteSecret removes a secret
	DeleteSecret(key string) error

	// ListSecrets returns all secret keys
	ListSecrets() ([]string, error)

	// RotateSecret rotates a secret value
	RotateSecret(key string) (string, error)

	// IsSecret returns true if a key is a secret
	IsSecret(key string) bool
}

// FeatureFlag defines the interface for feature flag management
type FeatureFlag interface {
	// IsEnabled returns true if a feature is enabled
	IsEnabled(feature string, context map[string]interface{}) bool

	// GetVariation returns the variation value for a feature
	GetVariation(feature string, context map[string]interface{}) (interface{}, error)

	// SetFlag sets a feature flag value
	SetFlag(feature string, enabled bool) error

	// GetAllFlags returns all feature flags
	GetAllFlags() map[string]interface{}

	// WatchFlag watches for feature flag changes
	WatchFlag(feature string, callback types.FeatureFlagCallback) error
}