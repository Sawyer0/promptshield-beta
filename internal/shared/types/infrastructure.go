package types

import (
	"time"
)

// MessageBrokerConfig represents configuration for message brokers
type MessageBrokerConfig struct {
	Type      MessageBrokerType      `json:"type"`
	Endpoints []string               `json:"endpoints"`
	Username  string                 `json:"username,omitempty"`
	Password  string                 `json:"password,omitempty"`
	TLS       *TLSConfig             `json:"tls,omitempty"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

// MessageBrokerType represents supported message broker types
type MessageBrokerType string

const (
	MessageBrokerTypeKafka MessageBrokerType = "kafka"
	MessageBrokerTypeNATS  MessageBrokerType = "nats"
	MessageBrokerTypeRedis MessageBrokerType = "redis"
)

// KafkaProducerConfig represents Kafka producer configuration
type KafkaProducerConfig struct {
	Topic        string        `json:"topic"`
	Partition    int           `json:"partition,omitempty"`
	BatchSize    int           `json:"batch_size"`
	BatchTimeout time.Duration `json:"batch_timeout"`
	Compression  string        `json:"compression,omitempty"`
	Retries      int           `json:"retries"`
	RetryBackoff time.Duration `json:"retry_backoff"`
	RequiredAcks int           `json:"required_acks"`
}

// KafkaConsumerConfig represents Kafka consumer configuration
type KafkaConsumerConfig struct {
	Topic          string        `json:"topic"`
	GroupID        string        `json:"group_id"`
	AutoCommit     bool          `json:"auto_commit"`
	StartOffset    string        `json:"start_offset"`
	MaxPollRecords int           `json:"max_poll_records"`
	SessionTimeout time.Duration `json:"session_timeout"`
}

// NATSConfig represents NATS-specific configuration
type NATSConfig struct {
	Subject       string        `json:"subject"`
	StreamName    string        `json:"stream_name,omitempty"`
	ConsumerName  string        `json:"consumer_name,omitempty"`
	MaxDeliver    int           `json:"max_deliver"`
	AckWait       time.Duration `json:"ack_wait"`
	ReplayPolicy  string        `json:"replay_policy,omitempty"`
	DeliverPolicy string        `json:"deliver_policy,omitempty"`
	DurableName   string        `json:"durable_name,omitempty"`
}

// MessageHeaders represents message headers
type MessageHeaders map[string]string

// MessageMetadata represents message metadata
type MessageMetadata struct {
	Topic     string         `json:"topic"`
	Partition int            `json:"partition,omitempty"`
	Offset    int64          `json:"offset,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	Headers   MessageHeaders `json:"headers,omitempty"`
	Key       string         `json:"key,omitempty"`
}

// PublishResult represents the result of a message publish operation
type PublishResult struct {
	MessageID string          `json:"message_id"`
	Metadata  MessageMetadata `json:"metadata"`
	Error     error           `json:"error,omitempty"`
}

// ConsumeResult represents the result of message consumption
type ConsumeResult struct {
	Message  []byte          `json:"message"`
	Metadata MessageMetadata `json:"metadata"`
	AckFunc  func() error    `json:"-"`
	NackFunc func() error    `json:"-"`
}

// OIDCProviderConfig represents OIDC provider configuration
type OIDCProviderConfig struct {
	IssuerURL          string            `json:"issuer_url"`
	ClientID           string            `json:"client_id"`
	ClientSecret       string            `json:"client_secret,omitempty"`
	Audience           string            `json:"audience,omitempty"`
	Scopes             []string          `json:"scopes,omitempty"`
	ClaimsMapping      map[string]string `json:"claims_mapping,omitempty"`
	SkipIssuerVerify   bool              `json:"skip_issuer_verify,omitempty"`
	SkipClientIDVerify bool              `json:"skip_client_id_verify,omitempty"`
}

// JWTClaims represents JWT token claims
type JWTClaims struct {
	Subject   string                 `json:"sub,omitempty"`
	Issuer    string                 `json:"iss,omitempty"`
	Audience  string                 `json:"aud,omitempty"`
	ExpiresAt time.Time              `json:"exp,omitempty"`
	IssuedAt  time.Time              `json:"iat,omitempty"`
	NotBefore time.Time              `json:"nbf,omitempty"`
	JTI       string                 `json:"jti,omitempty"`
	Custom    map[string]interface{} `json:"custom,omitempty"`
}

// TokenValidationResult represents the result of token validation
type TokenValidationResult struct {
	Valid    bool      `json:"valid"`
	Claims   JWTClaims `json:"claims,omitempty"`
	Error    error     `json:"error,omitempty"`
	TenantID string    `json:"tenant_id,omitempty"`
	UserID   string    `json:"user_id,omitempty"`
	Scopes   []string  `json:"scopes,omitempty"`
}

// ExternalAPIConfig represents configuration for external API clients
type ExternalAPIConfig struct {
	BaseURL     string            `json:"base_url"`
	APIKey      string            `json:"api_key,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Timeout     time.Duration     `json:"timeout"`
	RetryPolicy RetryPolicy       `json:"retry_policy"`
	TLS         *TLSConfig        `json:"tls,omitempty"`
	RateLimit   *RateLimitConfig  `json:"rate_limit,omitempty"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	RequestsPerSecond int           `json:"requests_per_second"`
	BurstSize         int           `json:"burst_size"`
	BackoffDelay      time.Duration `json:"backoff_delay"`
}

// APIRequest represents an API request
type APIRequest struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    []byte            `json:"body,omitempty"`
	Query   map[string]string `json:"query,omitempty"`
	Timeout time.Duration     `json:"timeout,omitempty"`
}

// APIResponse represents an API response
type APIResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       []byte            `json:"body,omitempty"`
	Error      error             `json:"error,omitempty"`
	Duration   time.Duration     `json:"duration"`
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	Type       CacheType     `json:"type"`
	TTL        time.Duration `json:"ttl"`
	MaxSize    int           `json:"max_size,omitempty"`
	MaxMemory  int64         `json:"max_memory,omitempty"`
	Endpoints  []string      `json:"endpoints,omitempty"`
	Prefix     string        `json:"prefix,omitempty"`
	Serializer string        `json:"serializer,omitempty"`
}

// CacheType represents supported cache types
type CacheType string

const (
	CacheTypeMemory CacheType = "memory"
	CacheTypeRedis  CacheType = "redis"
)

// CacheEntry represents a cache entry
type CacheEntry struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	ExpiresAt time.Time   `json:"expires_at"`
	CreatedAt time.Time   `json:"created_at"`
	Size      int64       `json:"size,omitempty"`
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	Evictions   int64   `json:"evictions"`
	Size        int     `json:"size"`
	MemoryUsage int64   `json:"memory_usage,omitempty"`
	HitRatio    float64 `json:"hit_ratio"`
}

// ServiceDiscoveryConfig represents service discovery configuration
type ServiceDiscoveryConfig struct {
	Type        ServiceDiscoveryType `json:"type"`
	Endpoints   []string             `json:"endpoints,omitempty"`
	Namespace   string               `json:"namespace,omitempty"`
	Labels      map[string]string    `json:"labels,omitempty"`
	HealthCheck *HealthCheckConfig   `json:"health_check,omitempty"`
}

// ServiceDiscoveryType represents service discovery types
type ServiceDiscoveryType string

const (
	ServiceDiscoveryTypeStatic     ServiceDiscoveryType = "static"
	ServiceDiscoveryTypeConsul     ServiceDiscoveryType = "consul"
	ServiceDiscoveryTypeKubernetes ServiceDiscoveryType = "kubernetes"
	ServiceDiscoveryTypeEtcd       ServiceDiscoveryType = "etcd"
)

// ServiceInstance represents a service instance
type ServiceInstance struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Address  string            `json:"address"`
	Port     int               `json:"port"`
	Tags     []string          `json:"tags,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Health   HealthStatus      `json:"health"`
	Version  string            `json:"version,omitempty"`
	Region   string            `json:"region,omitempty"`
	Zone     string            `json:"zone,omitempty"`
}

// CircuitBreakerConfig represents circuit breaker configuration
type CircuitBreakerConfig struct {
	FailureThreshold int           `json:"failure_threshold"`
	RecoveryTimeout  time.Duration `json:"recovery_timeout"`
	Timeout          time.Duration `json:"timeout"`
	MaxRequests      int           `json:"max_requests"`
	Interval         time.Duration `json:"interval"`
}

// CircuitBreakerState represents circuit breaker states
type CircuitBreakerState string

const (
	CircuitBreakerStateClosed   CircuitBreakerState = "closed"
	CircuitBreakerStateOpen     CircuitBreakerState = "open"
	CircuitBreakerStateHalfOpen CircuitBreakerState = "half_open"
)

// CircuitBreakerStats represents circuit breaker statistics
type CircuitBreakerStats struct {
	State               CircuitBreakerState `json:"state"`
	Requests            int64               `json:"requests"`
	Successes           int64               `json:"successes"`
	Failures            int64               `json:"failures"`
	ConsecutiveFailures int64               `json:"consecutive_failures"`
	LastStateChange     time.Time           `json:"last_state_change"`
}

// LoadBalancerConfig represents load balancer configuration
type LoadBalancerConfig struct {
	Algorithm   LoadBalancerAlgorithm `json:"algorithm"`
	Targets     []LoadBalancerTarget  `json:"targets"`
	HealthCheck *HealthCheckConfig    `json:"health_check,omitempty"`
}

// LoadBalancerAlgorithm represents load balancing algorithms
type LoadBalancerAlgorithm string

const (
	LoadBalancerAlgorithmRoundRobin LoadBalancerAlgorithm = "round_robin"
	LoadBalancerAlgorithmLeastConn  LoadBalancerAlgorithm = "least_conn"
	LoadBalancerAlgorithmWeighted   LoadBalancerAlgorithm = "weighted"
	LoadBalancerAlgorithmRandom     LoadBalancerAlgorithm = "random"
)

// LoadBalancerTarget represents a load balancer target
type LoadBalancerTarget struct {
	Address string       `json:"address"`
	Port    int          `json:"port"`
	Weight  int          `json:"weight,omitempty"`
	Health  HealthStatus `json:"health"`
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Interval         time.Duration `json:"interval"`
	Timeout          time.Duration `json:"timeout"`
	Path             string        `json:"path,omitempty"`
	Method           string        `json:"method,omitempty"`
	SuccessCodes     []int         `json:"success_codes,omitempty"`
	FailureThreshold int           `json:"failure_threshold"`
	SuccessThreshold int           `json:"success_threshold"`
}
