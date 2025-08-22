package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	appscan "github.com/promptshield/promptshield/internal/application/scan"
	"github.com/promptshield/promptshield/internal/scanner"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var benchmarkCmd = &cobra.Command{
	Use:     "benchmark [path|glob]...",
	Aliases: []string{"bench", "b"},
	Short:   "Run a simple scan benchmark",
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		sc := scanner.New(0)
		svc := appscan.NewService(sc)
		results, err := svc.Scan(cmd.Context(), args, appscan.Options{
			RulepackPath:  viper.GetString("rulepack"),
			ContextKVs:    viper.GetStringSlice("context"),
			Workers:       viper.GetInt("workers"),
			PendingWindow: 256,
		})
		if err != nil {
			return err
		}
		duration := time.Since(start)
		var regexAttempts, regexSkipped, semAttempts, semSkipped int64
		for _, r := range results {
			regexAttempts += r.Metrics.RegexAttempts
			regexSkipped += r.Metrics.RegexSkipped
			semAttempts += r.Metrics.SemanticAttempts
			semSkipped += r.Metrics.SemanticSkipped
		}
		out := map[string]any{
			"files":             len(results),
			"duration_ms":       duration.Milliseconds(),
			"regex_attempts":    regexAttempts,
			"regex_skipped":     regexSkipped,
			"semantic_attempts": semAttempts,
			"semantic_skipped":  semSkipped,
		}
		if outputFormat == "json" || jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}
		fmt.Fprintf(os.Stdout, "files=%d duration_ms=%d\n", out["files"], out["duration_ms"])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(benchmarkCmd)
	// Reuse the same rulepack/context flags as scan
	benchmarkCmd.Flags().StringP("rulepack", "r", "", "path to a RulePack file or directory")
	benchmarkCmd.Flags().StringArray("context", nil, "context key=value (repeatable; overrides pack defaults)")
	_ = viper.BindPFlag("rulepack", benchmarkCmd.Flags().Lookup("rulepack"))
	_ = viper.BindPFlag("context", benchmarkCmd.Flags().Lookup("context"))
}
