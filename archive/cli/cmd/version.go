package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// These variables are overridden at build time via -ldflags
var (
	version   = "0.2.0"
	commit    = "dev"
	buildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("promptshield %s (commit %s, built %s)\n", version, commit, buildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
