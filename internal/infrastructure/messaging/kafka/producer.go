package kafka

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
	"github.com/segmentio/kafka-go/sasl/plain"
	
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/errors"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// Producer implements contracts.MessageProducer for Kafka
type Producer struct {
	writer   *kafka.Writer
	brokers  []string
	config   *ProducerConfig
	mu       sync.RWMutex
	stats    map[string]interface{}
	closed   bool
}

// ProducerConfig contains Kafka producer configuration
type ProducerConfig struct {
	Brokers      []string
	Topic        string
	BatchSize    int
	BatchTimeout time.Duration
	Compression  string // "none", "gzip", "snappy", "lz4", "zstd"
	RequiredAcks int    // 0=none, 1=leader, -1=all
	MaxAttempts  int
	Async        bool
	TLS          *TLSConfig
	SASL         *SASLConfig
}

// TLSConfig contains TLS configuration
type TLSConfig struct {
	Enabled            bool
	InsecureSkipVerify bool
	CertFile           string
	KeyFile            string
	CAFile             string
}

// SASLConfig contains SASL authentication configuration
type SASLConfig struct {
	Enabled   bool
	Username  string
	Password  string
	Mechanism string // "plain", "scram-sha-256", "scram-sha-512"
}

// NewProducer creates a new Kafka producer
func NewProducer(config *ProducerConfig) (contracts.MessageProducer, error) {
	if config == nil {
		return nil, errors.ConfigurationError("kafka", "producer config is required")
	}
	
	if len(config.Brokers) == 0 {
		// Try to get from environment
		brokers := os.Getenv("PS_KAFKA_BROKERS")
		if brokers == "" {
			return nil, errors.ConfigurationError("kafka", "no brokers configured")
		}
		config.Brokers = []string{brokers}
	}
	
	// Set defaults
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.BatchTimeout == 0 {
		config.BatchTimeout = 100 * time.Millisecond
	}
	if config.RequiredAcks == 0 {
		config.RequiredAcks = 1 // Default to leader acknowledgment
	}
	if config.MaxAttempts == 0 {
		config.MaxAttempts = 3
	}
	
	p := &Producer{
		brokers: config.Brokers,
		config:  config,
		stats:   make(map[string]interface{}),
	}
	
	// Configure writer
	writerConfig := kafka.WriterConfig{
		Brokers:      config.Brokers,
		Topic:        config.Topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: config.RequiredAcks,
		MaxAttempts:  config.MaxAttempts,
		BatchSize:    config.BatchSize,
		BatchTimeout: config.BatchTimeout,
		Async:        config.Async,
	}
	
	// Configure compression using newer API
	switch config.Compression {
	case "gzip":
		writerConfig.CompressionCodec = &compress.GzipCodec
	case "snappy":
		writerConfig.CompressionCodec = &compress.SnappyCodec
	case "lz4":
		writerConfig.CompressionCodec = &compress.Lz4Codec
	case "zstd":
		writerConfig.CompressionCodec = &compress.ZstdCodec
	}
	
	// Configure TLS
	if config.TLS != nil && config.TLS.Enabled {
		dialer := &kafka.Dialer{
			Timeout:   10 * time.Second,
			DualStack: true,
			TLS: &tls.Config{
				InsecureSkipVerify: config.TLS.InsecureSkipVerify,
			},
		}
		
		// Configure SASL if enabled
		if config.SASL != nil && config.SASL.Enabled {
			mechanism := plain.Mechanism{
				Username: config.SASL.Username,
				Password: config.SASL.Password,
			}
			dialer.SASLMechanism = mechanism
		}
		
		writerConfig.Dialer = dialer
	}
	
	// Error logger
	writerConfig.ErrorLogger = kafka.LoggerFunc(func(msg string, args ...interface{}) {
		// Update error stats
		p.mu.Lock()
		p.stats["errors"] = p.stats["errors"].(int) + 1
		p.stats["last_error"] = fmt.Sprintf(msg, args...)
		p.stats["last_error_time"] = time.Now()
		p.mu.Unlock()
	})
	
	p.writer = kafka.NewWriter(writerConfig)
	
	// Initialize stats
	p.stats["messages_sent"] = 0
	p.stats["bytes_sent"] = 0
	p.stats["errors"] = 0
	p.stats["connected"] = true
	p.stats["started_at"] = time.Now()
	
	return p, nil
}

// Connect establishes connection to Kafka (no-op as connection is lazy)
func (p *Producer) Connect(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.closed {
		return errors.ServiceUnavailable("kafka producer is closed")
	}
	
	p.stats["connected"] = true
	p.stats["connected_at"] = time.Now()
	return nil
}

// Close closes the Kafka writer
func (p *Producer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.closed {
		return nil
	}
	
	p.closed = true
	p.stats["connected"] = false
	p.stats["closed_at"] = time.Now()
	
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

// IsConnected returns true if the producer is connected
func (p *Producer) IsConnected() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return !p.closed && p.stats["connected"].(bool)
}

// GetStats returns producer statistics
func (p *Producer) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	// Create a copy of stats
	stats := make(map[string]interface{})
	for k, v := range p.stats {
		stats[k] = v
	}
	
	// Add writer stats if available
	if p.writer != nil {
		writerStats := p.writer.Stats()
		stats["writer_messages"] = writerStats.Messages
		stats["writer_bytes"] = writerStats.Bytes
		stats["writer_errors"] = writerStats.Errors
		stats["writer_retries"] = writerStats.Retries
		stats["writer_batch_count"] = writerStats.Messages
		stats["writer_batch_bytes"] = writerStats.BatchBytes
	}
	
	return stats
}

// Publish publishes a message to the configured topic
func (p *Producer) Publish(ctx context.Context, topic string, message []byte, headers types.MessageHeaders) (*types.PublishResult, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, errors.ServiceUnavailable("kafka producer is closed")
	}
	p.mu.RUnlock()
	
	// Convert headers
	var kafkaHeaders []kafka.Header
	if headers != nil {
		for k, v := range headers {
			kafkaHeaders = append(kafkaHeaders, kafka.Header{
				Key:   k,
				Value: []byte(v),
			})
		}
	}
	
	// Create message
	msg := kafka.Message{
		Topic:   topic,
		Value:   message,
		Headers: kafkaHeaders,
		Time:    time.Now(),
	}
	
	// Write message
	err := p.writer.WriteMessages(ctx, msg)
	if err != nil {
		return nil, errors.ServiceUnavailable(fmt.Sprintf("failed to publish to Kafka: %v", err))
	}
	
	// Update stats
	p.mu.Lock()
	p.stats["messages_sent"] = p.stats["messages_sent"].(int) + 1
	p.stats["bytes_sent"] = p.stats["bytes_sent"].(int) + len(message)
	p.stats["last_publish_time"] = time.Now()
	p.mu.Unlock()
	
	return &types.PublishResult{
		MessageID: fmt.Sprintf("%s-%d", topic, time.Now().UnixNano()),
		Metadata: types.MessageMetadata{
			Topic:     topic,
			Partition: 0,
			Offset:    0,
			Timestamp: time.Now(),
		},
	}, nil
}

// PublishBatch publishes multiple messages in a batch
func (p *Producer) PublishBatch(ctx context.Context, messages []types.QueueMessage) ([]*types.PublishResult, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, errors.ServiceUnavailable("kafka producer is closed")
	}
	p.mu.RUnlock()
	
	// Convert to Kafka messages
	kafkaMessages := make([]kafka.Message, 0, len(messages))
	for _, msg := range messages {
		var headers []kafka.Header
		for k, v := range msg.Headers {
			headers = append(headers, kafka.Header{
				Key:   k,
				Value: []byte(v),
			})
		}
		
		kafkaMessages = append(kafkaMessages, kafka.Message{
			Topic:   msg.Type, // Use Type as topic
			Key:     []byte(msg.ID),
			Value:   msg.Payload,
			Headers: headers,
			Time:    msg.CreatedAt,
		})
	}
	
	// Write batch
	err := p.writer.WriteMessages(ctx, kafkaMessages...)
	if err != nil {
		return nil, errors.ServiceUnavailable(fmt.Sprintf("failed to publish batch to Kafka: %v", err))
	}
	
	// Create results
	results := make([]*types.PublishResult, len(messages))
	now := time.Now()
	totalBytes := 0
	
	for i, msg := range messages {
		results[i] = &types.PublishResult{
			MessageID: msg.ID,
			Metadata: types.MessageMetadata{
				Topic:     msg.Type,
				Timestamp: now,
			},
		}
		totalBytes += len(msg.Payload)
	}
	
	// Update stats
	p.mu.Lock()
	p.stats["messages_sent"] = p.stats["messages_sent"].(int) + len(messages)
	p.stats["bytes_sent"] = p.stats["bytes_sent"].(int) + totalBytes
	p.stats["last_batch_size"] = len(messages)
	p.stats["last_batch_time"] = now
	p.mu.Unlock()
	
	return results, nil
}

// PublishAsync publishes a message asynchronously
func (p *Producer) PublishAsync(ctx context.Context, topic string, message []byte, headers types.MessageHeaders) <-chan *types.PublishResult {
	resultChan := make(chan *types.PublishResult, 1)
	
	go func() {
		defer close(resultChan)
		result, err := p.Publish(ctx, topic, message, headers)
		if err != nil {
			// Return error result
			resultChan <- &types.PublishResult{
				Error: err,
				Metadata: types.MessageMetadata{
					Topic:     topic,
					Timestamp: time.Now(),
				},
			}
			return
		}
		resultChan <- result
	}()
	
	return resultChan
}