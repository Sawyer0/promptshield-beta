package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/promptshield/promptshield/internal/backoff"
	metrics "github.com/promptshield/promptshield/internal/observability/metrics"
	redis "github.com/redis/go-redis/v9"
)

// RuleUpdateHandler processes incoming rule updates
type RuleUpdateHandler func(ctx context.Context, update RuleUpdate) error

// Subscriber consumes RuleUpdate events from Redis Streams.
type Subscriber struct {
	rdb           *redis.Client
	consumerGroup string
	consumerName  string
	handler       RuleUpdateHandler
	tenantID      string // Only process updates for this tenant
	done          chan struct{}
	stateMu       sync.Mutex
	tenantState   map[string]*tenantState // tenantID -> state
}

type tenantState struct {
	lastAppliedVersion int
	recentMessages     []*recentMsg  // ring buffer of recent messages
	bufferCapacity     int           // e.g. 10
	bufferIdx          int           // next insert position
	evictionDuration   time.Duration // e.g. 30s
}

type recentMsg struct {
	id         string
	version    int
	sha256     string
	receivedAt time.Time
}

func newTenantState(cap int, eviction time.Duration) *tenantState {
	return &tenantState{
		lastAppliedVersion: 0,
		recentMessages:     make([]*recentMsg, cap),
		bufferCapacity:     cap,
		bufferIdx:          0,
		evictionDuration:   eviction,
	}
}

// isDuplicate checks if sha256 is in recent messages (evicting old entries first)
func (ts *tenantState) isDuplicate(sha256 string, now time.Time) bool {
	// Evict old entries
	for i := 0; i < ts.bufferCapacity; i++ {
		msg := ts.recentMessages[i]
		if msg != nil && now.Sub(msg.receivedAt) > ts.evictionDuration {
			ts.recentMessages[i] = nil
		}
	}

	// Check for match
	for _, msg := range ts.recentMessages {
		if msg != nil && msg.sha256 == sha256 {
			return true
		}
	}
	return false
}

// addRecent inserts message into ring buffer
func (ts *tenantState) addRecent(id string, version int, sha256 string, now time.Time) {
	ts.recentMessages[ts.bufferIdx] = &recentMsg{
		id:         id,
		version:    version,
		sha256:     sha256,
		receivedAt: now,
	}
	ts.bufferIdx = (ts.bufferIdx + 1) % ts.bufferCapacity
}

// NewSubscriber creates a new Redis stream subscriber for rule updates.
// This is designed for production resilience - Redis failures don't prevent creation.
func NewSubscriber(redisAddr, consumerGroup, consumerName, tenantID string, handler RuleUpdateHandler) (*Subscriber, error) {
	if redisAddr == "" {
		return &Subscriber{done: make(chan struct{})}, nil // no-op when not configured
	}

	// Create Redis client with production-ready settings
	rdb := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		MaxRetries:   3,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
		PoolSize:     2,
	})

	s := &Subscriber{
		rdb:           rdb,
		consumerGroup: consumerGroup,
		consumerName:  consumerName,
		handler:       handler,
		tenantID:      tenantID,
		done:          make(chan struct{}),
		tenantState:   make(map[string]*tenantState),
	}

	// Try to initialize consumer group, but don't fail if Redis is down
	// The Start() method will handle retries and initialization
	go s.initializeConsumerGroup()
	go s.recoverPending()

	return s, nil
}

// initializeConsumerGroup attempts to create the consumer group with retries
func (s *Subscriber) initializeConsumerGroup() {
	if s.rdb == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	base := 100 * time.Millisecond
	for attempts := 0; attempts < 5; attempts++ {
		// Check if Redis is available
		if err := s.rdb.Ping(ctx).Err(); err != nil {
			log.Printf("Redis not available for consumer group creation: %v (attempt %d/5)", err, attempts+1)
			time.Sleep(backoff.FullJitter(base, 3*time.Second, attempts))
			continue
		}

		// Try to create consumer group
		stream := fmt.Sprintf("rulepacks.updates:%s", s.tenantID)
		err := s.rdb.XGroupCreate(ctx, stream, s.consumerGroup, "0").Err()
		if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
			log.Printf("Failed to create consumer group: %v (attempt %d/5)", err, attempts+1)
			time.Sleep(backoff.FullJitter(base, 3*time.Second, attempts))
			continue
		}

		log.Printf("Consumer group '%s' initialized successfully", s.consumerGroup)
		return
	}

	log.Printf("Failed to initialize consumer group after 5 attempts - will retry during operation")
}

// Start begins consuming messages from the stream with production-grade reliability
func (s *Subscriber) Start(ctx context.Context) error {
	if s.rdb == nil {
		<-s.done   // Block until Stop() is called
		return nil // no-op when not configured
	}

	log.Printf("Starting Redis subscriber for tenant %s (group: %s, consumer: %s)", s.tenantID, s.consumerGroup, s.consumerName)

	// Circuit breaker state
	consecutiveFailures := 0
	maxFailures := 5
	baseBackoff := 1 * time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.done:
			return nil
		default:
			// Calculate exponential backoff based on consecutive failures
			var backoffDur time.Duration
			if consecutiveFailures > 0 {
				backoffDur = backoff.EqualJitter(baseBackoff, maxBackoff, consecutiveFailures)
			}

			// Circuit breaker: if too many failures, enter "open" state with longer backoff
			if consecutiveFailures >= maxFailures {
				log.Printf("Redis circuit breaker OPEN - backing off for %v (failures: %d)", backoffDur, consecutiveFailures)
				circuitStateTransitions.WithLabelValues("open", "max_failures").Inc()
				time.Sleep(backoffDur)

				// Try to test Redis health
				if err := s.rdb.Ping(ctx).Err(); err != nil {
					consecutiveFailures = maxFailures // Keep circuit open
					continue
				} else {
					log.Printf("Redis circuit breaker HALF-OPEN - attempting recovery")
					circuitStateTransitions.WithLabelValues("half_open", "ping_success").Inc()
					consecutiveFailures = maxFailures - 1 // Allow one retry
				}
			}

			// Apply backoff if we have recent failures
			if backoffDur > 0 {
				time.Sleep(backoffDur)
			}

			// Read from stream with blocking
			msgs, err := s.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    s.consumerGroup,
				Consumer: s.consumerName,
				Streams:  []string{fmt.Sprintf("rulepacks.updates:%s", s.tenantID), ">"},
				Count:    1,
				Block:    1 * time.Second,
			}).Result()

			if err != nil {
				if err == redis.Nil {
					// No new messages - reset failure count on successful poll
					if consecutiveFailures > 0 {
						consecutiveFailures = 0
						log.Printf("Redis circuit breaker CLOSED - connection healthy")
						circuitStateTransitions.WithLabelValues("closed", "recovery").Inc()
					}
					continue
				}

				consecutiveFailures++
				log.Printf("Error reading from Redis stream: %v (failure %d/%d)", err, consecutiveFailures, maxFailures)
				continue
			}

			// After successful read, query metrics
			// Get stream info
			stream := fmt.Sprintf("rulepacks.updates:%s", s.tenantID)
			pendingInfo, errPending := s.rdb.XPending(ctx, stream, s.consumerGroup).Result()
			if errPending == nil {
				metrics.PendingCount.WithLabelValues(stream).Set(float64(pendingInfo.Count))
				lag := 0.0
				if pendingInfo.Lower != "" {
					parts := strings.Split(pendingInfo.Lower, "-")
					if len(parts) > 0 {
						if ts, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
							lag = time.Since(time.UnixMilli(ts)).Seconds()
						}
					}
				}
				metrics.StreamLagSeconds.WithLabelValues(stream).Set(lag)
				if pendingInfo.Count > 1000 {
					backoffDur += time.Duration(pendingInfo.Count/100) * time.Millisecond
					if backoffDur > 10*time.Second {
						backoffDur = 10 * time.Second
					}
				}
			}
			// Then sleep if >0 handled earlier
			if backoffDur > 0 {
				time.Sleep(backoffDur)
			}

			// Successful read - reset failure count
			if consecutiveFailures > 0 {
				consecutiveFailures = 0
				log.Printf("Redis circuit breaker CLOSED - connection recovered")
				circuitStateTransitions.WithLabelValues("closed", "recovery").Inc()
				metrics.ConsumerRestartsTotal.Inc()
			}

			// Process messages
			for _, strm := range msgs {
				for _, msg := range strm.Messages {
					// Extract JSON data from message
					jsonData, ok := msg.Values["json"].(string)
					if !ok {
						// Route to DLQ
						dlqReason := "malformed_json"
						s.routeToDLQ(ctx, msg, dlqReason)
						continue
					}

					s.stateMu.Lock()
					state, ok := s.tenantState[msg.Values["tenant_id"].(string)]
					if !ok {
						state = newTenantState(10, 30*time.Second)
						s.tenantState[msg.Values["tenant_id"].(string)] = state
					}
					s.stateMu.Unlock()

					now := time.Now()

					// Check version and duplicate
					var update RuleUpdate
					if err := json.Unmarshal([]byte(jsonData), &update); err != nil {
						log.Printf("Failed to parse RuleUpdate from message %s: %v", msg.ID, err)
						s.routeToDLQ(ctx, msg, "parse_error")
						continue
					} else {
						if update.Version <= state.lastAppliedVersion {
							log.Printf("Dropping stale message %s (version %d <= last applied %d)", msg.ID, update.Version, state.lastAppliedVersion)
							if err := s.rdb.XAck(ctx, fmt.Sprintf("rulepacks.updates:%s", s.tenantID), s.consumerGroup, msg.ID).Err(); err != nil {
								log.Printf("Failed to ACK stale message %s: %v", msg.ID, err)
							}
							continue
						}

						if state.isDuplicate(update.ContentSHA256, now) {
							log.Printf("Dropping duplicate message %s (sha256 %s already seen)", msg.ID, update.ContentSHA256)
							if err := s.rdb.XAck(ctx, fmt.Sprintf("rulepacks.updates:%s", s.tenantID), s.consumerGroup, msg.ID).Err(); err != nil {
								log.Printf("Failed to ACK duplicate message %s: %v", msg.ID, err)
							}
							continue
						}

						// Process
						if err := s.handler(ctx, update); err != nil {
							log.Printf("Error processing message %s: %v", msg.ID, err)
							// Message processing error doesn't count as Redis failure
						} else {
							// Update state and add to buffer
							state.lastAppliedVersion = update.Version
							state.addRecent(msg.ID, update.Version, update.ContentSHA256, now)

							// ACK
							if err := s.rdb.XAck(ctx, fmt.Sprintf("rulepacks.updates:%s", s.tenantID), s.consumerGroup, msg.ID).Err(); err != nil {
								log.Printf("Failed to ACK message %s: %v", msg.ID, err)
							}
						}
					}
				}
			}
		}
	}
}

// Stop gracefully stops the subscriber
func (s *Subscriber) Stop() {
	close(s.done)
}

// Close closes the Redis connection
func (s *Subscriber) Close() {
	s.Stop()
	if s.rdb != nil {
		_ = s.rdb.Close()
	}
}

// processMessage processes a single Redis stream message.
// It is referenced from reliability and unit tests, so we ignore the "unused" linter warning.
//
//nolint:unused // used in *_test.go files within the same package
func (s *Subscriber) processMessage(ctx context.Context, msg redis.XMessage) error {
	// Extract JSON data from message
	jsonData, ok := msg.Values["json"].(string)
	if !ok {
		return nil // Skip malformed messages
	}

	// Parse RuleUpdate
	var update RuleUpdate
	if err := json.Unmarshal([]byte(jsonData), &update); err != nil {
		return err
	}

	// Filter by tenant ID
	if s.tenantID != "" && update.TenantID != s.tenantID {
		return nil // Skip updates for other tenants
	}

	// Call handler
	return s.handler(ctx, update)
}

func (s *Subscriber) routeToDLQ(ctx context.Context, msg redis.XMessage, reason string) {
	dlqStream := "rulepacks.dlq"
	values := make(map[string]any)
	for k, v := range msg.Values {
		values[k] = v
	}
	values["dlq_reason"] = reason
	values["original_stream"] = "rulepacks.updates" // or per-tenant
	values["timestamp"] = time.Now().Unix()
	_ = s.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: dlqStream,
		Values: values,
	}).Err()
	// ACK original
	_ = s.rdb.XAck(ctx, fmt.Sprintf("rulepacks.updates:%s", s.tenantID), s.consumerGroup, msg.ID).Err()
}

// Use a metric
var circuitStateTransitions = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "ps_circuit_transitions_total",
		Help: "Circuit breaker state transitions",
	},
	[]string{"state", "reason"},
)

func init() {
	prometheus.MustRegister(circuitStateTransitions)
}

func (s *Subscriber) recoverPending() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			ctx := context.Background()
			stream := fmt.Sprintf("rulepacks.updates:%s", s.tenantID)
			claimed, _, err := s.rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
				Stream:   stream,
				Group:    s.consumerGroup,
				Consumer: s.consumerName,
				MinIdle:  30 * time.Second,
				Start:    "0-0",
				Count:    100,
			}).Result()
			if err != nil {
				log.Printf("XAUTOCLAIM error: %v", err)
			} else if len(claimed) > 0 {
				log.Printf("Recovered %d pending messages", len(claimed))
				// Process them? Or let normal loop handle since claimed to this consumer.
			}
		}
	}
}
