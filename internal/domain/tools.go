package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Tool represents a user-registered tool/action with capability-based metadata.
type Tool struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	ToolID         string          `json:"tool_id"` // Stable identifier supplied by user (e.g., "search_web")
	Name           string          `json:"name"`    // Human label
	Description    string          `json:"description"`
	CapabilityTags []string        `json:"capability_tags"`      // e.g., read, write, network, file-io, code-exec, shell, db-write, email-send, payment, pii-access, admin-scope
	DataDomains    []string        `json:"data_domains"`         // e.g., PII, PCI, PHI, source-code, internal-docs, public-web
	SideEffect     string          `json:"side_effect"`          // none | reversible | irreversible
	AuthScope      string          `json:"auth_scope"`           // user-delegated | service-account
	ArgSchema      json.RawMessage `json:"arg_schema"`           // JSON summary: params, types; mark ids, enums, regex, free-text
	RiskScore      *int            `json:"risk_score,omitempty"` // optional 0..5
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}
