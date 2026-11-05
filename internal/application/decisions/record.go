// Package decisions keeps a bounded, in-memory history of enforcement
// decisions so operators can inspect recent traffic and summary statistics
// without a database. Prompt content is never stored, only the outcome.
package decisions

import (
	"time"

	pkg "github.com/promptshield/promptshield/pkg/types"
)

// Decision outcomes produced by the enforcer.
const (
	Allow      = "allow"
	Deny       = "deny"
	Quarantine = "quarantine"
)

// Match is one rule that fired for a request.
type Match struct {
	RuleID   string `json:"rule_id"`
	Severity string `json:"severity,omitempty"`
	Category string `json:"category,omitempty"`
	Action   string `json:"action,omitempty"`
	Message  string `json:"message,omitempty"`
}

// Record is the outcome of a single enforcement decision.
type Record struct {
	// Seq is assigned by Log.Add and increases by one per decision; it is
	// the paging cursor for GET /v1/decisions.
	Seq        int64     `json:"seq"`
	RequestID  string    `json:"request_id"`
	Time       time.Time `json:"time"`
	Path       string    `json:"path"`
	Decision   string    `json:"decision"`
	Reason     string    `json:"reason"`
	Status     int       `json:"status"`
	LatencyMs  float64   `json:"latency_ms"`
	BytesIn    int64     `json:"bytes_in"`
	Violations int       `json:"violations"`
	Matches    []Match   `json:"matches"`
}

// MatchesFrom converts scanner violations into decision matches.
func MatchesFrom(violations []pkg.Violation) []Match {
	out := make([]Match, 0, len(violations))
	for _, v := range violations {
		out = append(out, Match{
			RuleID:   v.RuleID,
			Severity: v.Severity,
			Category: v.Category,
			Action:   v.ResponseAction,
			Message:  firstNonEmpty(v.ResponseMessage, v.Message),
		})
	}
	return out
}

// EventData renders the record as an event payload. The keys decision,
// reason, violations and request_id match the original decision event so
// existing SSE consumers keep working.
func (r Record) EventData() map[string]any {
	return map[string]any{
		"seq":        r.Seq,
		"request_id": r.RequestID,
		"time":       r.Time.UTC().Format(time.RFC3339Nano),
		"path":       r.Path,
		"decision":   r.Decision,
		"reason":     r.Reason,
		"status":     r.Status,
		"latency_ms": r.LatencyMs,
		"bytes_in":   r.BytesIn,
		"violations": r.Violations,
		"matches":    r.Matches,
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
