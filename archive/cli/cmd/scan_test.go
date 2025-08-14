package cmd

import (
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestCLI_Scan_WithRulepack(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/scripts",
		Setup: func(env *testscript.Env) error {
			// Put module root on PATH so `go run ./cmd/promptshield` works inside scripts if needed.
			env.Setenv("PROMPTSHIELD_ROOT", env.WorkDir)
			return nil
		},
	})
}
