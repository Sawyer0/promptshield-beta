package nats

import (
    "context"
    "encoding/json"
    "log/slog"
    "time"

    redis "github.com/redis/go-redis/v9"
    pmetrics "github.com/promptshield/promptshield/internal/observability/metrics"
)

const ToolPolicyFlushChannel = "ps.tool_policies.flush"

type ToolPolicyFlush struct {
    TenantID string    `json:"tenant_id,omitempty"`
    Epoch    int64     `json:"epoch,omitempty"`
    Reason   string    `json:"reason,omitempty"`
    At       time.Time `json:"at"`
}

// PublishToolPolicyFlush publishes a flush event via Redis Pub/Sub.
func PublishToolPolicyFlush(addr string, ev ToolPolicyFlush) error {
    if addr == "" { return nil }
    rdb := redis.NewClient(&redis.Options{ Addr: addr })
    defer rdb.Close()
    if ev.At.IsZero() { ev.At = time.Now().UTC() }
    b, _ := json.Marshal(ev)
    err := rdb.Publish(context.Background(), ToolPolicyFlushChannel, b).Err()
    if err == nil {
        scope := "global"; if ev.TenantID != "" { scope = "tenant" }
        pmetrics.PolicyFlushEventsTotal.WithLabelValues("publisher", scope).Inc()
    }
    return err
}

// StartToolPolicyFlushSubscriber subscribes to flush events and invokes cb.
// Returns a function to stop the subscription.
func StartToolPolicyFlushSubscriber(addr string, cb func(ev ToolPolicyFlush)) func() {
    if addr == "" { return func(){} }
    rdb := redis.NewClient(&redis.Options{ Addr: addr })
    pubsub := rdb.Subscribe(context.Background(), ToolPolicyFlushChannel)
    ch := pubsub.Channel()
    stop := make(chan struct{})
    go func() {
        logger := slog.With("component", "policy-flush-subscriber")
        defer func(){ _ = pubsub.Close(); _ = rdb.Close() }()
        for {
            select {
            case <-stop:
                return
            case msg, ok := <-ch:
                if !ok { return }
                var ev ToolPolicyFlush
                if err := json.Unmarshal([]byte(msg.Payload), &ev); err != nil {
                    logger.Warn("invalid flush payload", "err", err)
                    continue
                }
                // best-effort callback
                func(){
                    defer func(){ recover() }()
                    cb(ev)
                }()
            }
        }
    }()
    return func(){ close(stop) }
}
