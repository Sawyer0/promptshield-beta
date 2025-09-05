package nats

import (
	"context"
	"encoding/json"
	"fmt"

	redis "github.com/redis/go-redis/v9"
)

type RuleUpdate struct {
	TenantID      string `json:"tenantId"`
	TargetScope   string `json:"targetScope"`
	RulepackID    string `json:"rulepackId"`
	Version       int    `json:"version"`
	ContentSHA256 string `json:"contentSha256"`
}

// Publisher publishes RuleUpdate events to Redis Streams (stream name: rulepacks.updates).
// If redisAddr is empty, PublishRuleUpdate becomes a no-op so the code compiles without Redis.
type Publisher struct{ rdb *redis.Client }

func NewPublisher(redisAddr string) (*Publisher, error) {
	if redisAddr == "" {
		return &Publisher{}, nil // no-op
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &Publisher{rdb: rdb}, nil
}

func (p *Publisher) PublishRuleUpdate(ctx context.Context, upd RuleUpdate) error {
	if p.rdb == nil {
		return nil // no-op when not configured
	}
	data, _ := json.Marshal(upd)
	return p.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: fmt.Sprintf("rulepacks.updates:%s", upd.TenantID),
		Values: map[string]any{"json": data},
		MaxLen: 10000,
	}).Err()
}

func (p *Publisher) Close() {
	if p.rdb != nil {
		_ = p.rdb.Close()
	}
}

// ----- DLQ publisher for blocked/errored decisions -----

type DLQ struct{ rdb *redis.Client }

func NewDLQ(redisAddr string) (*DLQ, error) {
	if redisAddr == "" {
		return &DLQ{}, nil
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &DLQ{rdb: rdb}, nil
}

// Publish writes a small JSON payload to the DLQ stream (default: ps.dlq.decisions).
func (d *DLQ) Publish(ctx context.Context, stream string, payload map[string]any) error {
	if d.rdb == nil {
		return nil
	}
	if stream == "" {
		stream = "ps.dlq.decisions"
	}
	data, _ := json.Marshal(payload)
	return d.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]any{"json": data},
		MaxLen: 100000,
	}).Err()
}

func (d *DLQ) Close() {
	if d.rdb != nil {
		_ = d.rdb.Close()
	}
}
