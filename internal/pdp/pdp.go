package pdp

import (
	"context"
	"encoding/json"
	"time"
)

type Decision string

const (
	Permit        Decision = "PERMIT"
	Deny          Decision = "DENY"
	Indeterminate Decision = "INDETERMINATE"
	NotApplicable Decision = "NOT_APPLICABLE"
)

type Subject struct {
	UserID     string                 `json:"userId"`
	TenantID   string                 `json:"tenantId"`
	Roles      []string               `json:"roles,omitempty"`
	Attributes map[string]any         `json:"attributes,omitempty"`
}

type Resource struct {
	Type       string                 `json:"type"`            // e.g., document, tool, message
	ID         string                 `json:"id,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
	Attributes map[string]any         `json:"attributes,omitempty"`
}

type Environment struct {
	CorrelationID string                 `json:"correlationId,omitempty"`
	IP            string                 `json:"ip,omitempty"`
	Time          time.Time              `json:"time"`
	Attributes    map[string]any         `json:"attributes,omitempty"`
}

type Request struct {
	Subject     Subject     `json:"subject"`
	Action      string      `json:"action"` // e.g., "rag.query", "tool.invoke", "message.send"
	Resource    Resource    `json:"resource"`
	Environment Environment `json:"environment"`
}

type Obligation struct {
	Type  string `json:"type"`           // e.g., filterTag, allowScope, mask, redactField
	Key   string `json:"key,omitempty"`
	Value any    `json:"value,omitempty"`
}

type Response struct {
	Decision    Decision        `json:"decision"`
	Obligations []Obligation    `json:"obligations,omitempty"`
	Reason      string          `json:"reason,omitempty"`
	Risk        float64         `json:"risk,omitempty"`
	Cacheable   bool            `json:"cacheable,omitempty"`
	Provider    string          `json:"provider,omitempty"`
	Raw         json.RawMessage `json:"raw,omitempty"`
	// Local-only metadata
	TTL time.Duration `json:"-"`
}

type Client interface {
	Evaluate(ctx context.Context, req Request) (Response, error)
}
