package events

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/promptshield/promptshield/internal/infrastructure/messaging/kafka"
	"github.com/promptshield/promptshield/internal/shared/types"
)

// RawEvent captures raw request/response metadata + token counts (no full bodies by design)
type RawEvent struct {
	Schema     string    `json:"schema"`
	Version    int       `json:"version"`
	OccurredAt time.Time `json:"occurred_at"`
	Direction  string    `json:"direction"` // request|response
	TenantID   string    `json:"tenant_id"`
	Endpoint   string    `json:"endpoint"`
	Method     string    `json:"method"`
	RequestID  string    `json:"request_id"`
	Bytes      int64     `json:"bytes"`
}

// SanitizedEvent captures sanitized snippets or hashes if enabled (optional)
type SanitizedEvent struct {
	Schema     string    `json:"schema"`
	Version    int       `json:"version"`
	OccurredAt time.Time `json:"occurred_at"`
	Direction  string    `json:"direction"`
	TenantID   string    `json:"tenant_id"`
	Endpoint   string    `json:"endpoint"`
	Method     string    `json:"method"`
	RequestID  string    `json:"request_id"`
	Bytes      int64     `json:"bytes"`
	Snippet    string    `json:"snippet,omitempty"`
}

type StreamPublishers struct {
	rawProd        *kafka.Producer
	sanitizedProd  *kafka.Producer
	rawTopic       string
	sanitizedTopic string
}

func NewStreamPublishersFromEnv() (*StreamPublishers, error) {
	brokers := strings.TrimSpace(os.Getenv("PS_KAFKA_BROKERS"))
	if brokers == "" {
		return nil, nil
	}
	rawTopic := os.Getenv("PS_TOPIC_RAW")
	if rawTopic == "" {
		rawTopic = "ps.raw"
	}
	sanTopic := os.Getenv("PS_TOPIC_SANITIZED")
	if sanTopic == "" {
		sanTopic = "ps.sanitized"
	}
	bp := strings.Split(brokers, ",")
	rpI, err := kafka.NewProducer(&kafka.ProducerConfig{Brokers: bp, Topic: rawTopic, Compression: "zstd", BatchSize: 256, BatchTimeout: 50 * time.Millisecond, RequiredAcks: 1})
	if err != nil {
		return nil, err
	}
	spI, err := kafka.NewProducer(&kafka.ProducerConfig{Brokers: bp, Topic: sanTopic, Compression: "zstd", BatchSize: 256, BatchTimeout: 50 * time.Millisecond, RequiredAcks: 1})
	if err != nil {
		return nil, err
	}
	rp := rpI.(*kafka.Producer)
	sp := spI.(*kafka.Producer)
	return &StreamPublishers{rawProd: rp, sanitizedProd: sp, rawTopic: rawTopic, sanitizedTopic: sanTopic}, nil
}

func (p *StreamPublishers) PublishRaw(ctx context.Context, ev *RawEvent) error {
	if p == nil || p.rawProd == nil || ev == nil {
		return nil
	}
	ev.Schema = "ps.raw.v1"
	ev.Version = 1
	if ev.OccurredAt.IsZero() {
		ev.OccurredAt = time.Now().UTC()
	}
	b, _ := json.Marshal(ev)
	_, err := p.rawProd.Publish(ctx, p.rawTopic, b, types.MessageHeaders{"event_type": "ps.raw", "tenant_id": ev.TenantID})
	return err
}

func (p *StreamPublishers) PublishSanitized(ctx context.Context, ev *SanitizedEvent) error {
	if p == nil || p.sanitizedProd == nil || ev == nil {
		return nil
	}
	ev.Schema = "ps.sanitized.v1"
	ev.Version = 1
	if ev.OccurredAt.IsZero() {
		ev.OccurredAt = time.Now().UTC()
	}
	b, _ := json.Marshal(ev)
	_, err := p.sanitizedProd.Publish(ctx, p.sanitizedTopic, b, types.MessageHeaders{"event_type": "ps.sanitized", "tenant_id": ev.TenantID})
	return err
}
