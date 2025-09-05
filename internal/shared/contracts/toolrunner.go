package contracts

import (
    "context"
    "encoding/json"
    "time"

    "github.com/google/uuid"
)

type ToolExecRequest struct {
    TenantID      uuid.UUID       `json:"tenant_id"`
    ToolID        string          `json:"tool_id"`
    Args          json.RawMessage `json:"args"`
    ConversationID string         `json:"conversation_id,omitempty"`
    RequestID     string          `json:"request_id,omitempty"`
    Endpoint      string          `json:"endpoint,omitempty"`
    Method        string          `json:"method,omitempty"`
    Lane          string          `json:"lane,omitempty"`
    PlanHash      string          `json:"plan_hash,omitempty"`
    PlanStep      string          `json:"plan_step,omitempty"`
    Timeout       time.Duration   `json:"-"`
}

type ToolExecResult struct {
    Result      json.RawMessage   `json:"result"`
    ContentType string            `json:"content_type,omitempty"`
    StartedAt   time.Time         `json:"started_at,omitempty"`
    CompletedAt time.Time         `json:"completed_at,omitempty"`
    LatencyMs   int64             `json:"latency_ms,omitempty"`
    Provider    string            `json:"provider,omitempty"`
    Model       string            `json:"model,omitempty"`
    TokensIn    int               `json:"tokens_in,omitempty"`
    TokensOut   int               `json:"tokens_out,omitempty"`
    CostUSD     float64           `json:"cost_usd,omitempty"`
    Metadata    map[string]any    `json:"metadata,omitempty"`
}

type ToolRunner interface { Execute(ctx context.Context, req ToolExecRequest) (ToolExecResult, error) }
