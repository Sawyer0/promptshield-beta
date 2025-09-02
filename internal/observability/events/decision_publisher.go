package events

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/infrastructure/messaging/kafka"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// DecisionEvent is the schema for decision stream events (versioned)
// Schema: ps.decision.v1
// Note: payloads are metadata-only; text is not included to preserve data minimization.
// If raw text is needed, emit a quarantine event with a handle to short-TTL storage.
type DecisionEvent struct {
	Schema        string            `json:"schema"`  // e.g., ps.decision.v1
	Version       int               `json:"version"` // 1
	EventID       string            `json:"event_id"`
	OccurredAt    time.Time         `json:"occurred_at"`
	TenantID      string            `json:"tenant_id"`
	Endpoint      string            `json:"endpoint"`
	Method        string            `json:"method"`
	ToolID        string            `json:"tool_id,omitempty"`
	Lane          string            `json:"lane,omitempty"`
	PlanHash      string            `json:"plan_hash,omitempty"`
	PlanStep      int               `json:"plan_step,omitempty"`
	Conversation  string            `json:"conversation_id,omitempty"`
	RequestID     string            `json:"request_id,omitempty"`
	Decision      string            `json:"decision"` // allow|quarantine|deny|replace
	Reason        string            `json:"reason"`
	LatencyMs     int64             `json:"latency_ms"`
	BytesRequest  int64             `json:"bytes_request"`
	BytesResponse int64             `json:"bytes_response"`
	Attributes    map[string]string `json:"attributes,omitempty"`
}

// DecisionPublisher publishes decision events to a streaming sink (Kafka/Redpanda)
type DecisionPublisher struct {
	producer       contracts.MessageProducer
	topicDecisions string
}

// NewDecisionPublisherFromEnv builds a decision publisher if brokers/topics are present.
func NewDecisionPublisherFromEnv() (*DecisionPublisher, error) {
	brokers := strings.TrimSpace(os.Getenv("PS_KAFKA_BROKERS"))
	if brokers == "" {
		brokers = strings.TrimSpace(os.Getenv("PS_AUDIT_KAFKA_BROKERS")) // reuse if set
	}
	if brokers == "" {
		return nil, nil // not configured
	}
	topic := strings.TrimSpace(os.Getenv("PS_TOPIC_DECISIONS"))
	if topic == "" {
		topic = "ps.decisions"
	}
	prod, err := kafka.NewProducer(&kafka.ProducerConfig{Brokers: strings.Split(brokers, ","), Topic: topic, Compression: "zstd", BatchSize: 128, BatchTimeout: 50 * time.Millisecond, RequiredAcks: 1})
	if err != nil {
		return nil, err
	}
	return &DecisionPublisher{producer: prod, topicDecisions: topic}, nil
}

// PublishDecision sends a decision event (metadata only)
func (p *DecisionPublisher) PublishDecision(ctx context.Context, evt *DecisionEvent) error {
	if p == nil || p.producer == nil || evt == nil {
		return nil
	}
	if evt.EventID == "" {
		evt.EventID = uuid.New().String()
	}
	evt.Schema = "ps.decision.v1"
	evt.Version = 1
	if evt.OccurredAt.IsZero() {
		evt.OccurredAt = time.Now().UTC()
	}
	b, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	headers := types.MessageHeaders{
		"event_type": "ps.decision",
		"tenant_id":  evt.TenantID,
		"decision":   evt.Decision,
	}
	_, err = p.producer.Publish(ctx, p.topicDecisions, b, headers)
	return err
}

// Close shuts down the underlying producer
func (p *DecisionPublisher) Close() error {
	if p == nil || p.producer == nil {
		return nil
	}
	return p.producer.Close()
}
