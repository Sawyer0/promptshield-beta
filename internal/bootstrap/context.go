package bootstrap

import (
	"context"
	"github.com/spf13/cobra"
)

type depsKey struct{}

// WithDeps returns a child context carrying Deps.
func WithDeps(ctx context.Context, d *Deps) context.Context {
	return context.WithValue(ctx, depsKey{}, d)
}

// From extracts Deps from a cobra.Command's context. Returns nil if missing.
func From(cmd *cobra.Command) *Deps {
	if v := cmd.Context().Value(depsKey{}); v != nil {
		if d, ok := v.(*Deps); ok {
			return d
		}
	}
	return nil
}
