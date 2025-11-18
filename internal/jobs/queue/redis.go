package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/promptshield/promptshield/internal/util/tracing"
	redis "github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
)

// RedisQueue is a durable queue backed by Redis using a stream/consumer group.
// It provides at-least-once delivery with visibility timeouts.
type RedisQueue struct {
	client   *redis.Client
	stream   string
	group    string
	consumer string
	visTTL   time.Duration
}

var redisQueueTracer = otel.Tracer("promptshield/redis/queue")

type redisPayload struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Input    []byte                 `json:"input"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewRedisQueue creates a queue using Redis Streams.
// stream: name of the stream (e.g., "ps.jobs"); group: consumer group (e.g., "ps.workers").
func NewRedisQueue(addr, password string, db int, stream, group, consumer string, visTTL time.Duration) (*RedisQueue, error) {
	if stream == "" || group == "" || consumer == "" {
		return nil, errors.New("stream, group, and consumer are required")
	}
	rc := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db})
	rq := &RedisQueue{client: rc, stream: stream, group: group, consumer: consumer, visTTL: visTTL}
	// Ensure group exists
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rq.ensureGroup(ctx); err != nil {
		return nil, err
	}
	return rq, nil
}

func (q *RedisQueue) ensureGroup(ctx context.Context) error {
	// Create stream implicitly if needed; MKSTREAM
	// Try creating group; ignore BUSYGROUP
	ctx, span := tracing.TraceRedisCommand(redisQueueTracer, ctx, "XGROUP CREATE", q.stream)
	defer span.End()
	if err := q.client.XGroupCreateMkStream(ctx, q.stream, q.group, "$").Err(); err != nil {
		if !strings.Contains(err.Error(), "BUSYGROUP") {
			return fmt.Errorf("create group: %w", err)
		}
	}
	return nil
}

func (q *RedisQueue) Enqueue(ctx context.Context, msg Message) (string, error) {
	payload := redisPayload(msg)
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	args := &redis.XAddArgs{Stream: q.stream, Values: map[string]any{"data": raw}}
	ctxAdd, span := tracing.TraceRedisCommand(redisQueueTracer, ctx, "XADD", q.stream)
	id, err := q.client.XAdd(ctxAdd, args).Result()
	span.End()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (q *RedisQueue) RunConsumers(ctx context.Context, n int, handler Handler) error {
	if n <= 0 {
		n = 1
	}
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					errCh <- nil
					return
				default:
				}
				// Read from stream using consumer group; block for visTTL/2
				ctxRead, spanRead := tracing.TraceRedisCommand(redisQueueTracer, ctx, "XREADGROUP", q.stream)
				msgs, err := q.client.XReadGroup(ctxRead, &redis.XReadGroupArgs{
					Group:    q.group,
					Consumer: q.consumer,
					Streams:  []string{q.stream, ">"},
					Count:    10,
					Block:    q.visTTL / 2,
				}).Result()
				spanRead.End()
				if err != nil && !errors.Is(err, redis.Nil) {
					// brief backoff on error
					select {
					case <-time.After(200 * time.Millisecond):
					case <-ctx.Done():
						return
					}
					continue
				}
				for _, s := range msgs {
					for _, x := range s.Messages {
						raw := []byte(fmt.Sprint(x.Values["data"]))
						var p redisPayload
						if err := json.Unmarshal(raw, &p); err != nil {
							_ = q.client.XAck(ctx, q.stream, q.group, x.ID).Err()
							continue
						}
						m := Message(p)
						hctx, cancel := context.WithTimeout(ctx, q.visTTL)
						err := handler(hctx, m)
						cancel()
						if err == nil {
							_ = q.client.XAck(ctx, q.stream, q.group, x.ID).Err()
						} else {
							// Do not ACK: message remains pending and will be retried after visibility timeout
							_ = q.client.XClaimJustID(ctx, &redis.XClaimArgs{Stream: q.stream, Group: q.group, Consumer: q.consumer, MinIdle: q.visTTL, Messages: []string{x.ID}}).Err()
						}
					}
				}
			}
		}()
	}
	// Wait until context cancelled
	<-ctx.Done()
	close(errCh)
	return nil
}

func (q *RedisQueue) Close() error { return q.client.Close() }
