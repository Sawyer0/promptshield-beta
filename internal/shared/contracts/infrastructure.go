package contracts

import (
	"context"
	"time"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// MessageBroker defines the interface for message broker operations
type MessageBroker interface {
	// Connect establishes connection to the message broker
	Connect(ctx context.Context) error
	
	// Close closes the connection to the message broker
	Close() error
	
	// IsConnected returns true if connected to the broker
	IsConnected() bool
	
	// GetStats returns broker statistics
	GetStats() map[string]interface{}
}

// MessageProducer defines the interface for message publishing
type MessageProducer interface {
	MessageBroker
	
	// Publish publishes a message to a topic/subject
	Publish(ctx context.Context, topic string, message []byte, headers types.MessageHeaders) (*types.PublishResult, error)
	
	// PublishBatch publishes multiple messages in a batch
	PublishBatch(ctx context.Context, messages []types.QueueMessage) ([]*types.PublishResult, error)
	
	// PublishAsync publishes a message asynchronously
	PublishAsync(ctx context.Context, topic string, message []byte, headers types.MessageHeaders) <-chan *types.PublishResult
}

// MessageConsumer defines the interface for message consumption
type MessageConsumer interface {
	MessageBroker
	
	// Subscribe subscribes to a topic/subject for message consumption
	Subscribe(ctx context.Context, topic string, handler MessageHandler) error
	
	// SubscribeBatch subscribes with batch processing
	SubscribeBatch(ctx context.Context, topic string, batchSize int, handler BatchMessageHandler) error
	
	// Consume consumes messages from a topic (pull-based)
	Consume(ctx context.Context, topic string, timeout time.Duration) (*types.ConsumeResult, error)
	
	// ConsumeBatch consumes multiple messages in a batch
	ConsumeBatch(ctx context.Context, topic string, batchSize int, timeout time.Duration) ([]*types.ConsumeResult, error)
	
	// Unsubscribe unsubscribes from a topic
	Unsubscribe(topic string) error
}

// MessageHandler defines the interface for handling individual messages
type MessageHandler interface {
	Handle(ctx context.Context, message *types.ConsumeResult) error
}

// BatchMessageHandler defines the interface for handling message batches
type BatchMessageHandler interface {
	HandleBatch(ctx context.Context, messages []*types.ConsumeResult) error
}

// KafkaProducer defines Kafka-specific producer interface
type KafkaProducer interface {
	MessageProducer
	
	// PublishToPartition publishes to a specific partition
	PublishToPartition(ctx context.Context, topic string, partition int, key string, message []byte, headers types.MessageHeaders) (*types.PublishResult, error)
	
	// GetPartitionCount returns the number of partitions for a topic
	GetPartitionCount(topic string) (int, error)
	
	// Flush flushes pending messages
	Flush(timeout time.Duration) error
}

// KafkaConsumer defines Kafka-specific consumer interface
type KafkaConsumer interface {
	MessageConsumer
	
	// SubscribeWithConfig subscribes with Kafka-specific configuration
	SubscribeWithConfig(ctx context.Context, config types.KafkaConsumerConfig, handler MessageHandler) error
	
	// Seek seeks to a specific offset
	Seek(topic string, partition int, offset int64) error
	
	// CommitOffset commits the current offset
	CommitOffset(topic string, partition int, offset int64) error
}

// NATSPublisher defines NATS-specific publisher interface
type NATSPublisher interface {
	MessageProducer
	
	// PublishWithReply publishes a message and waits for a reply
	PublishWithReply(ctx context.Context, subject string, reply string, message []byte, timeout time.Duration) ([]byte, error)
	
	// Request sends a request and waits for a response
	Request(ctx context.Context, subject string, message []byte, timeout time.Duration) ([]byte, error)
	
	// PublishJetStream publishes to a JetStream stream
	PublishJetStream(ctx context.Context, subject string, message []byte, options map[string]interface{}) (*types.PublishResult, error)
}

// NATSSubscriber defines NATS-specific subscriber interface
type NATSSubscriber interface {
	MessageConsumer
	
	// SubscribeJetStream subscribes to a JetStream consumer
	SubscribeJetStream(ctx context.Context, config types.NATSConfig, handler MessageHandler) error
	
	// QueueSubscribe subscribes as part of a queue group
	QueueSubscribe(ctx context.Context, subject string, queue string, handler MessageHandler) error
}

// OIDCVerifier defines the interface for OIDC token verification
type OIDCVerifier interface {
	// Initialize initializes the verifier with provider configuration
	Initialize(ctx context.Context, config types.OIDCProviderConfig) error
	
	// VerifyToken verifies an ID token or access token
	VerifyToken(ctx context.Context, token string) (*types.TokenValidationResult, error)
	
	// VerifyIDToken verifies an ID token specifically
	VerifyIDToken(ctx context.Context, token string) (*types.TokenValidationResult, error)
	
	// VerifyAccessToken verifies an access token
	VerifyAccessToken(ctx context.Context, token string) (*types.TokenValidationResult, error)
	
	// RefreshProviderKeys refreshes the provider's public keys
	RefreshProviderKeys(ctx context.Context) error
	
	// GetProviderConfig returns the current provider configuration
	GetProviderConfig() types.OIDCProviderConfig
}

// ExternalAPIClient defines the interface for external API clients
type ExternalAPIClient interface {
	// Get performs a GET request
	Get(ctx context.Context, path string, headers map[string]string) (*types.APIResponse, error)
	
	// Post performs a POST request
	Post(ctx context.Context, path string, body []byte, headers map[string]string) (*types.APIResponse, error)
	
	// Put performs a PUT request
	Put(ctx context.Context, path string, body []byte, headers map[string]string) (*types.APIResponse, error)
	
	// Delete performs a DELETE request
	Delete(ctx context.Context, path string, headers map[string]string) (*types.APIResponse, error)
	
	// Request performs a custom request
	Request(ctx context.Context, req *types.APIRequest) (*types.APIResponse, error)
	
	// SetConfig updates the client configuration
	SetConfig(config types.ExternalAPIConfig)
	
	// GetConfig returns the current configuration
	GetConfig() types.ExternalAPIConfig
}

// Cache defines the interface for caching operations
type Cache interface {
	// Get retrieves a value from cache
	Get(ctx context.Context, key string) (interface{}, error)
	
	// Set stores a value in cache with TTL
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	
	// Delete removes a value from cache
	Delete(ctx context.Context, key string) error
	
	// Exists checks if a key exists in cache
	Exists(ctx context.Context, key string) (bool, error)
	
	// Expire sets TTL for an existing key
	Expire(ctx context.Context, key string, ttl time.Duration) error
	
	// GetTTL returns the TTL for a key
	GetTTL(ctx context.Context, key string) (time.Duration, error)
	
	// Clear clears all cache entries
	Clear(ctx context.Context) error
	
	// GetStats returns cache statistics
	GetStats(ctx context.Context) (*types.CacheStats, error)
	
	// Close closes the cache connection
	Close() error
}

// DistributedCache defines the interface for distributed caching
type DistributedCache interface {
	Cache
	
	// GetMultiple retrieves multiple values from cache
	GetMultiple(ctx context.Context, keys []string) (map[string]interface{}, error)
	
	// SetMultiple stores multiple values in cache
	SetMultiple(ctx context.Context, entries map[string]interface{}, ttl time.Duration) error
	
	// DeleteMultiple removes multiple values from cache
	DeleteMultiple(ctx context.Context, keys []string) error
	
	// GetByPattern retrieves keys matching a pattern
	GetByPattern(ctx context.Context, pattern string) ([]string, error)
	
	// Increment atomically increments a numeric value
	Increment(ctx context.Context, key string, delta int64) (int64, error)
	
	// Decrement atomically decrements a numeric value
	Decrement(ctx context.Context, key string, delta int64) (int64, error)
}

// ServiceDiscovery defines the interface for service discovery
type ServiceDiscovery interface {
	// Register registers a service instance
	Register(ctx context.Context, instance *types.ServiceInstance) error
	
	// Deregister removes a service instance
	Deregister(ctx context.Context, instanceID string) error
	
	// Discover finds available service instances
	Discover(ctx context.Context, serviceName string) ([]*types.ServiceInstance, error)
	
	// DiscoverHealthy finds healthy service instances
	DiscoverHealthy(ctx context.Context, serviceName string) ([]*types.ServiceInstance, error)
	
	// Watch watches for service changes
	Watch(ctx context.Context, serviceName string) (<-chan []*types.ServiceInstance, error)
	
	// HealthCheck performs health check on an instance
	HealthCheck(ctx context.Context, instance *types.ServiceInstance) error
	
	// UpdateHealth updates the health status of an instance
	UpdateHealth(ctx context.Context, instanceID string, health types.HealthStatus) error
}

// CircuitBreaker defines the interface for circuit breaker pattern
type CircuitBreaker interface {
	// Execute executes a function with circuit breaker protection
	Execute(ctx context.Context, fn func() (interface{}, error)) (interface{}, error)
	
	// ExecuteWithFallback executes with fallback function
	ExecuteWithFallback(ctx context.Context, fn func() (interface{}, error), fallback func() (interface{}, error)) (interface{}, error)
	
	// GetState returns the current circuit breaker state
	GetState() types.CircuitBreakerState
	
	// GetStats returns circuit breaker statistics
	GetStats() *types.CircuitBreakerStats
	
	// Reset resets the circuit breaker to closed state
	Reset()
	
	// ForceOpen forces the circuit breaker to open state
	ForceOpen()
	
	// ForceClose forces the circuit breaker to closed state
	ForceClose()
}

// LoadBalancer defines the interface for load balancing
type LoadBalancer interface {
	// Next returns the next target based on the load balancing algorithm
	Next() (*types.LoadBalancerTarget, error)
	
	// AddTarget adds a new target to the load balancer
	AddTarget(target *types.LoadBalancerTarget) error
	
	// RemoveTarget removes a target from the load balancer
	RemoveTarget(address string, port int) error
	
	// UpdateTargetHealth updates the health status of a target
	UpdateTargetHealth(address string, port int, health types.HealthStatus) error
	
	// GetTargets returns all targets
	GetTargets() []*types.LoadBalancerTarget
	
	// GetHealthyTargets returns only healthy targets
	GetHealthyTargets() []*types.LoadBalancerTarget
	
	// SetConfig updates the load balancer configuration
	SetConfig(config types.LoadBalancerConfig) error
	
	// GetConfig returns the current configuration
	GetConfig() types.LoadBalancerConfig
}