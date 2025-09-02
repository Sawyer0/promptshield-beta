package planstate

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	redis "github.com/redis/go-redis/v9"
)

// RedisPlanState stores plan and lane state in Redis with TTL
type RedisPlanState struct {
	rdb    *redis.Client
	prefix string
}

func NewRedisPlanState(rdb *redis.Client, prefix string) *RedisPlanState {
	if prefix == "" {
		prefix = "ps"
	}
	return &RedisPlanState{rdb: rdb, prefix: prefix}
}

func keyPlan(prefix string, tenant uuid.UUID, conv string) string {
	return fmt.Sprintf("%s:plan:%s:%s", prefix, tenant.String(), conv)
}

func keyLane(prefix string, tenant uuid.UUID, conv string) string {
	return fmt.Sprintf("%s:lane:%s:%s", prefix, tenant.String(), conv)
}

type planEnvelope struct {
	Plan     json.RawMessage `json:"plan"`
	PlanHash string          `json:"plan_hash"`
}

func (s *RedisPlanState) PutPlan(ctx context.Context, tenantID uuid.UUID, conversationID string, plan json.RawMessage, planHash string, ttl time.Duration) error {
	if s == nil || s.rdb == nil {
		return nil
	}
	env := planEnvelope{Plan: plan, PlanHash: planHash}
	b, _ := json.Marshal(env)
	return s.rdb.Set(ctx, keyPlan(s.prefix, tenantID, conversationID), b, ttl).Err()
}

func (s *RedisPlanState) GetPlan(ctx context.Context, tenantID uuid.UUID, conversationID string) (json.RawMessage, string, error) {
	if s == nil || s.rdb == nil {
		return nil, "", nil
	}
	res, err := s.rdb.Get(ctx, keyPlan(s.prefix, tenantID, conversationID)).Bytes()
	if err != nil {
		return nil, "", err
	}
	var env planEnvelope
	if json.Unmarshal(res, &env) != nil {
		return nil, "", nil
	}
	return env.Plan, env.PlanHash, nil
}

func (s *RedisPlanState) PutLane(ctx context.Context, tenantID uuid.UUID, conversationID string, lane string, ttl time.Duration) error {
	if s == nil || s.rdb == nil {
		return nil
	}
	return s.rdb.Set(ctx, keyLane(s.prefix, tenantID, conversationID), lane, ttl).Err()
}

func (s *RedisPlanState) GetLane(ctx context.Context, tenantID uuid.UUID, conversationID string) (string, error) {
	if s == nil || s.rdb == nil {
		return "", nil
	}
	return s.rdb.Get(ctx, keyLane(s.prefix, tenantID, conversationID)).Val(), nil
}

var _ contracts.PlanState = (*RedisPlanState)(nil)
