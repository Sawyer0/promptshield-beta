package grpcenforcer

import (
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/contracts"
)

// Options represents runtime tunables for the gRPC enforcer server.
// This duplicates the definition that previously lived in the sub-package
// `server` so that callers can depend on the top-level package without
// introducing import cycles.
// TODO: Consolidate the two copies during follow-up refactor.

type Options struct {
	Timeout         time.Duration
	MaxStreamBytes  int64
	FailOn          string
	RulepackPath    string
	Telemetry       TelemetryCollector
	EnforcementMode string
	// RulepackRepo enables loading rules from persistence layer
	RulepackRepo contracts.RulepackRepository
	// TenantID for loading tenant-specific rulepacks
	TenantID uuid.UUID
	// RedisAddr for live rule updates (if empty, live updates disabled)
	RedisAddr string
}
