//go:build test
// +build test

package grpcenforcer

import (
	"context"

	nats "github.com/promptshield/promptshield/internal/infrastructure/messaging/nats"
)

// createRuleUpdateHandler creates a handler function for processing rule update messages.
// It is linked into the build only when running `go test` (due to the `test` build tag).
func (s *Server) createRuleUpdateHandler() func(ctx context.Context, update nats.RuleUpdate) error {
	return func(ctx context.Context, update nats.RuleUpdate) error {
		// Reload rules when we receive an update for our tenant
		return s.ReloadRules(ctx)
	}
}
