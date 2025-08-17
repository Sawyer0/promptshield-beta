package audit

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"
)

// KafkaLogger is a thin interface wrapper so we can swap underlying client if needed
// without adding a heavy dependency here. Implementations should be created in this
// file to keep imports centralized.

type KafkaLogger struct {
	mu    sync.Mutex
	prod  kafkaProducer
	topic string
}

// kafkaProducer captures the subset we need from a Kafka client implementation.
type kafkaProducer interface {
	Produce(ctx context.Context, topic string, key []byte, value []byte) error
	Close() error
}

// NewKafkaLogger constructs a Kafka-backed audit logger using a minimal internal producer.
func NewKafkaLogger(brokers []string, topic string) (*KafkaLogger, error) {
	if len(brokers) == 0 || strings.TrimSpace(topic) == "" {
		return nil, errors.New("brokers and topic are required")
	}
	p, err := newBuiltinProducer(brokers)
	if err != nil {
		return nil, err
	}
	return &KafkaLogger{prod: p, topic: topic}, nil
}

func (l *KafkaLogger) Log(e Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e.Data != nil {
		e.Data = SanitizeMap(e.Data)
	}
	e.Timestamp = time.Now().UTC()
	// Hash chaining at sink layer is optional; we include event hash
	e.Hash = hashEvent(e)
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return l.prod.Produce(context.Background(), l.topic, nil, b)
}

func (l *KafkaLogger) Close() error { return l.prod.Close() }

// Minimal builtin producer implementation using a lightweight client. To keep
// dependencies constrained, we implement a tiny, buffered producer over the
// standard library + a well-known protocol library would be ideal; however,
// for this repository we provide a stub that can be wired to a real client by
// swapping the build tag or adding a thin adapter later.

type builtinProducer struct{ brokers []string }

func newBuiltinProducer(brokers []string) (*builtinProducer, error) {
	// Placeholder implementation. In production, replace with a battle-tested client
	// (e.g., segmentio/kafka-go or confluent-kafka-go) behind the kafkaProducer interface.
	return &builtinProducer{brokers: append([]string(nil), brokers...)}, nil
}

func (p *builtinProducer) Produce(_ context.Context, _ string, _ []byte, _ []byte) error {
	// No-op stub. Replace with real implementation.
	return nil
}

func (p *builtinProducer) Close() error { return nil }
