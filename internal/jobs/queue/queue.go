package queue

import (
	"context"
)

// Message represents a unit of work for asynchronous processing.
// It mirrors the core fields needed to route and execute a job.
type Message struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Input    []byte                 `json:"input"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Handler processes a single message. Returning a non-nil error will
// cause the queue implementation to NACK or leave the message pending
// for retry according to its semantics.
type Handler func(ctx context.Context, msg Message) error

// DurableQueue defines a minimal, implementation-agnostic API for
// durable job delivery with at-least-once semantics.
//
// Implementations should:
// - Persist messages so they survive process/node restarts
// - Provide consumer concurrency and fair distribution
// - Ack only on successful processing; retry on failure
// - Support visibility timeouts to avoid message loss on crashes
type DurableQueue interface {
	// Enqueue persists a message and returns its assigned ID.
	Enqueue(ctx context.Context, msg Message) (string, error)
	// RunConsumers starts n consumer loops until ctx is cancelled.
	// The implementation is responsible for ACK/NACK based on handler error.
	RunConsumers(ctx context.Context, n int, handler Handler) error
	// Close releases any resources.
	Close() error
}
