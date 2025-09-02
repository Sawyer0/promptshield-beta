package contracts

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PlanState defines a minimal interface to persist per-conversation plan/lane state with TTL
// Keys are scoped by tenant and conversation id.
// Implementations must be safe for concurrent use and optimized for low latency.
type PlanState interface {
	PutPlan(ctx context.Context, tenantID uuid.UUID, conversationID string, plan json.RawMessage, planHash string, ttl time.Duration) error
	GetPlan(ctx context.Context, tenantID uuid.UUID, conversationID string) (json.RawMessage, string, error)
	PutLane(ctx context.Context, tenantID uuid.UUID, conversationID string, lane string, ttl time.Duration) error
	GetLane(ctx context.Context, tenantID uuid.UUID, conversationID string) (string, error)
}
