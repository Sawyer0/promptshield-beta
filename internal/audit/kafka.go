package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
	
	"github.com/promptshield/promptshield/internal/infrastructure/messaging/kafka"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/errors"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// KafkaLogger implements audit logging using Kafka message broker
type KafkaLogger struct {
	mu       sync.Mutex
	producer contracts.MessageProducer
	topic    string
	config   *KafkaLoggerConfig
}

// KafkaLoggerConfig contains configuration for Kafka audit logger
type KafkaLoggerConfig struct {
	Brokers      []string
	Topic        string
	BatchSize    int
	BatchTimeout time.Duration
	Async        bool
}

// NewKafkaLogger constructs a Kafka-backed audit logger using the shared infrastructure
func NewKafkaLogger(brokers []string, topic string) (*KafkaLogger, error) {
	if len(brokers) == 0 || strings.TrimSpace(topic) == "" {
		return nil, errors.ConfigurationError("kafka", "brokers and topic are required")
	}
	
	config := &KafkaLoggerConfig{
		Brokers:      brokers,
		Topic:        topic,
		BatchSize:    100,
		BatchTimeout: 100 * time.Millisecond,
		Async:        false, // Synchronous for audit compliance
	}
	
	// Create Kafka producer using shared infrastructure
	producerConfig := &kafka.ProducerConfig{
		Brokers:      brokers,
		Topic:        topic,
		BatchSize:    config.BatchSize,
		BatchTimeout: config.BatchTimeout,
		RequiredAcks: -1, // Wait for all replicas for audit compliance
		MaxAttempts:  5,   // More retries for audit events
		Async:        config.Async,
		Compression:  "snappy",
	}
	
	producer, err := kafka.NewProducer(producerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}
	
	// Connect producer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := producer.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to Kafka: %w", err)
	}
	
	return &KafkaLogger{
		producer: producer,
		topic:    topic,
		config:   config,
	}, nil
}

// Log writes an audit event to Kafka
func (l *KafkaLogger) Log(e Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Sanitize sensitive data
	if e.Data != nil {
		e.Data = SanitizeMap(e.Data)
	}
	
	// Set timestamp and compute hash
	e.Timestamp = time.Now().UTC()
	e.Hash = hashEvent(e)
	
	// Marshal event to JSON
	payload, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}
	
	// Create message headers with metadata
	headers := make(types.MessageHeaders)
	headers["event_type"] = e.Type
	headers["event_hash"] = e.Hash
	headers["timestamp"] = e.Timestamp.Format(time.RFC3339)
	// Extract tenant ID from data if available
	if tenantID, ok := e.Data["tenant_id"]; ok {
		if tid, ok := tenantID.(string); ok && tid != "" {
			headers["tenant_id"] = tid
		}
	}
	
	// Publish to Kafka
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	result, err := l.producer.Publish(ctx, l.topic, payload, headers)
	if err != nil {
		return fmt.Errorf("failed to publish audit event to Kafka: %w", err)
	}
	
	// Log successful publish (optional)
	if result != nil && result.MessageID != "" {
		e.Data["kafka_message_id"] = result.MessageID
	}
	
	return nil
}

// LogBatch writes multiple audit events to Kafka in a batch
func (l *KafkaLogger) LogBatch(events []Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Convert events to queue messages
	messages := make([]types.QueueMessage, 0, len(events))
	for _, e := range events {
		// Sanitize and prepare each event
		if e.Data != nil {
			e.Data = SanitizeMap(e.Data)
		}
		e.Timestamp = time.Now().UTC()
		e.Hash = hashEvent(e)
		
		payload, err := json.Marshal(e)
		if err != nil {
			// Skip invalid events but log error
			continue
		}
		
		headers := make(map[string]string)
		headers["event_type"] = e.Type
		headers["event_hash"] = e.Hash
		// Extract tenant ID from data if available
		if tenantID, ok := e.Data["tenant_id"]; ok {
			if tid, ok := tenantID.(string); ok && tid != "" {
				headers["tenant_id"] = tid
			}
		}
		
		messages = append(messages, types.QueueMessage{
			ID:        e.Hash,
			Type:      l.topic,
			Payload:   payload,
			Headers:   headers,
			CreatedAt: e.Timestamp,
		})
	}
	
	if len(messages) == 0 {
		return nil // No valid messages to send
	}
	
	// Publish batch
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	results, err := l.producer.PublishBatch(ctx, messages)
	if err != nil {
		return fmt.Errorf("failed to publish audit batch to Kafka: %w", err)
	}
	
	// Check for partial failures
	for i, result := range results {
		if result != nil && result.Error != nil {
			// Log individual failures but don't fail entire batch
			fmt.Printf("Failed to publish audit event %d: %v\n", i, result.Error)
		}
	}
	
	return nil
}

// Close gracefully shuts down the Kafka producer
func (l *KafkaLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if l.producer != nil {
		return l.producer.Close()
	}
	return nil
}

// GetStats returns Kafka producer statistics
func (l *KafkaLogger) GetStats() map[string]interface{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if l.producer != nil {
		return l.producer.GetStats()
	}
	return nil
}
