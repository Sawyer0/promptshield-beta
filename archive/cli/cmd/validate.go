package cmd

import (
	"fmt"
	"os"

	"github.com/promptshield/promptshield/internal/bootstrap"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/spf13/cobra"
)

// validateCmd validates a RulePack file or directory of packs.
var validateCmd = &cobra.Command{
	Use:     "validate <rulepack|dir>",
	Aliases: []string{"v"},
	Short:   "Validate RulePack YAML files",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = bootstrap.From(cmd)
		packs, err := rules.LoadPacks(args[0])
		if err != nil {
			return err
		}
		for _, p := range packs {
			if errs := rules.ValidatePack(p); len(errs) > 0 {
				// print first; future: print all
				return errs[0]
			}
		}
		_, _ = fmt.Fprintln(os.Stdout, "RulePack OK")
		return nil
	},
}

func NewValidateCommand(deps *bootstrap.Deps) *cobra.Command { _ = deps; return validateCmd }
