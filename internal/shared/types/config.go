package types

import "time"

// ConfigSchema represents the configuration schema
type ConfigSchema struct {
	Fields   map[string]*ConfigField `json:"fields"`
	Required []string                `json:"required"`
	Version  string                  `json:"version"`
}

// ConfigField represents a configuration field definition
type ConfigField struct {
	Type         ConfigFieldType `json:"type"`
	Description  string          `json:"description"`
	Default      interface{}     `json:"default,omitempty"`
	Required     bool            `json:"required"`
	Sensitive    bool            `json:"sensitive"`
	Min          interface{}     `json:"min,omitempty"`
	Max          interface{}     `json:"max,omitempty"`
	Options      []interface{}   `json:"options,omitempty"`
	Pattern      string          `json:"pattern,omitempty"`
	Dependencies []string        `json:"dependencies,omitempty"`
}

// ConfigFieldType represents the type of a configuration field
type ConfigFieldType string

const (
	ConfigFieldTypeString   ConfigFieldType = "string"
	ConfigFieldTypeInt      ConfigFieldType = "int"
	ConfigFieldTypeBool     ConfigFieldType = "bool"
	ConfigFieldTypeDuration ConfigFieldType = "duration"
	ConfigFieldTypeList     ConfigFieldType = "list"
	ConfigFieldTypeMap      ConfigFieldType = "map"
)

// ConfigValidationError represents a configuration validation error
type ConfigValidationError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// AuthConfig represents authentication configuration for remote sources
type AuthConfig struct {
	Type     AuthType               `json:"type"`
	Username string                 `json:"username,omitempty"`
	Password string                 `json:"password,omitempty"`
	Token    string                 `json:"token,omitempty"`
	Headers  map[string]string      `json:"headers,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AuthType represents authentication types
type AuthType string

const (
	AuthTypeNone   AuthType = "none"
	AuthTypeBasic  AuthType = "basic"
	AuthTypeBearer AuthType = "bearer"
	AuthTypeAPI    AuthType = "api_key"
	AuthTypeOAuth  AuthType = "oauth"
)

// ParsedTemplate represents a parsed configuration template
type ParsedTemplate struct {
	Template  string                 `json:"template"`
	Variables []string               `json:"variables"`
	Functions []string               `json:"functions"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// DefaultConfig represents default configuration values
type DefaultConfig struct {
	// Server configuration
	ServerAddr    string        `mapstructure:"server_addr" default:":8080"`
	ServerTimeout time.Duration `mapstructure:"server_timeout" default:"30s"`
	GRPCAddr      string        `mapstructure:"grpc_addr" default:":9090"`
	GRPCTimeout   time.Duration `mapstructure:"grpc_timeout" default:"30s"`

	// Database configuration
	DatabaseURL     string        `mapstructure:"database_url"`
	DatabaseMaxConn int           `mapstructure:"database_max_conn" default:"10"`
	DatabaseTimeout time.Duration `mapstructure:"database_timeout" default:"5s"`

	// Redis configuration
	RedisAddr     string        `mapstructure:"redis_addr"`
	RedisPassword string        `mapstructure:"redis_password"`
	RedisDB       int           `mapstructure:"redis_db" default:"0"`
	RedisTimeout  time.Duration `mapstructure:"redis_timeout" default:"5s"`

	// Scanning configuration
	ScanTimeout time.Duration `mapstructure:"scan_timeout" default:"5s"`
	MaxBodySize int64         `mapstructure:"max_body_size" default:"1048576"`
	Workers     int           `mapstructure:"workers" default:"0"`

	// Security configuration
	TLSEnabled    bool   `mapstructure:"tls_enabled" default:"false"`
	TLSCertFile   string `mapstructure:"tls_cert_file"`
	TLSKeyFile    string `mapstructure:"tls_key_file"`
	EncryptionKey string `mapstructure:"encryption_key"`

	// Telemetry configuration
	TelemetryEnabled  bool   `mapstructure:"telemetry_enabled" default:"false"`
	TelemetryEndpoint string `mapstructure:"telemetry_endpoint"`
	MetricsEnabled    bool   `mapstructure:"metrics_enabled" default:"true"`
	MetricsPath       string `mapstructure:"metrics_path" default:"/metrics"`

	// Feature flags
	Features map[string]bool `mapstructure:"features"`
}

// ConfigCallback is called when configuration changes
type ConfigCallback func(key string, oldValue, newValue interface{}) error

// ConfigSourceCallback is called when configuration source changes
type ConfigSourceCallback func(config map[string]interface{}) error

// FileChangeCallback is called when a configuration file changes
type FileChangeCallback func(path string, config map[string]interface{}) error

// RemoteChangeCallback is called when remote configuration changes
type RemoteChangeCallback func(config map[string]interface{}) error

// FeatureFlagCallback is called when a feature flag changes
type FeatureFlagCallback func(feature string, oldValue, newValue interface{}) error

// HTTPClientConfig represents HTTP client configuration
type HTTPClientConfig struct {
	Timeout             time.Duration `json:"timeout"`
	MaxIdleConns        int           `json:"max_idle_conns"`
	MaxIdleConnsPerHost int           `json:"max_idle_conns_per_host"`
	IdleConnTimeout     time.Duration `json:"idle_conn_timeout"`
	DisableKeepAlives   bool          `json:"disable_keep_alives"`
	UserAgent           string        `json:"user_agent"`
	RetryConfig         *RetryPolicy  `json:"retry_config,omitempty"`
}