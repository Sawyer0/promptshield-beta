/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"

	"log/slog"

	"github.com/google/uuid"
	scancommand "github.com/promptshield/promptshield/internal/application/commands/scan"
	appscan "github.com/promptshield/promptshield/internal/application/scan"
	"github.com/promptshield/promptshield/internal/bootstrap"
	"github.com/promptshield/promptshield/internal/discovery"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
	sharederrors "github.com/promptshield/promptshield/internal/shared/errors"
	sharedseverity "github.com/promptshield/promptshield/internal/shared/severity"
	"github.com/promptshield/promptshield/internal/shared/termui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewScanCommand(deps *bootstrap.Deps) *cobra.Command {
	var (
		rulepackPath string
		contextKVs   []string
		failOn       string
		demo         bool
		noHints      bool
		perfSummary  bool
	)
	cmd := &cobra.Command{
		Use:     "scan [path|glob]...",
		Aliases: []string{"s"},
		Short:   "Scan inputs for LLM security risks (streaming)",
		Args:    cobra.MinimumNArgs(1),
		Example: termui.Dim(true, "  promptshield scan:file prompts.json\n  promptshield scan:directory ./data\n  promptshield scan:file --rulepack rules/*.yaml 'data/**/*.json'\n  promptshield scan:file --json --fail-on ERROR prompts.json"),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Demo mode: reuse demo implementation for onboarding
			if demo {
				return demoCmd.RunE(cmd, nil)
			}
			// Structured logger to stderr (or DI-provided)
			level := slog.LevelInfo
			if deps != nil && deps.Config.Debug {
				level = slog.LevelDebug
			} else if viper.GetBool("debug") {
				level = slog.LevelDebug
			}
			deps := bootstrap.From(cmd)
			var logger *slog.Logger
			if deps != nil && deps.Logger != nil {
				logger = deps.Logger
			} else {
				logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
			}
			// Correlate logs by request_id
			reqID := generateRequestID()
			logger = logger.With("request_id", reqID)

			sc := scanner.New(0)
			if deps != nil && deps.Scanner != nil {
				sc = deps.Scanner
			}
			// Apply max file size from effective config if set
			if deps != nil && deps.Config.Performance.MaxFileSizeBytes > 0 {
				sc.SetFileSizeLimit(deps.Config.Performance.MaxFileSizeBytes)
			}
			// Default 100MB cap when unset to prevent OOM/DoS as per Gap 4.6
			if deps != nil && deps.Config.Performance.MaxFileSizeBytes == 0 {
				sc.SetFileSizeLimit(100 * 1024 * 1024)
			}
			sort.Strings(args)
			// Resolve workers default: Local=2, CI=NumCPU, allow override via flag/config
			numWorkers := deps.Config.Workers
			if numWorkers == 0 {
				numWorkers = viper.GetInt("workers")
			}
			if numWorkers <= 0 {
				if isCIEnvironment() {
					numWorkers = runtime.NumCPU()
				} else {
					numWorkers = 2
				}
			}
			// Build application service and delegate to handler
			// Thread effective config into application service so rule defaults
			// (case_sensitive, whole_word, per_rule_timeout, etc.) are applied.
			svc := appscan.NewService(sc).WithConfig(deps.Config)
			handler := scancommand.NewHandler(svc, logger)

			ctx := cmd.Context()
			// Default fail-on in CI: treat any violation as failure unless overridden
			if failOn == "" && isCIEnvironment() {
				failOn = "INFO"
			}
			// Build override hints for audit: note which values came from flags/env
			overrides := map[string]string{}
			if cmd.Flags().Changed("output-format") || cmd.Flags().Changed("json") {
				overrides["output_format"] = "flag"
			}
			if cmd.Flags().Changed("fail-on") {
				overrides["fail_on"] = "flag"
			}
			if os.Getenv("PS_WORKERS") != "" {
				overrides["workers"] = "env"
			}

			// Telemetry: rulepack usage snapshot (coarse)
			if deps != nil && deps.Telemetry != nil {
				rp := resolveRulepackPath(rulepackPath)
				if rp != "" {
					if packs, e := rules.LoadPacks(rp); e == nil {
						levels := map[int]bool{}
						extends := false
						overrides := false
						logicAll := false
						comp := "all_matches"
						for _, p := range packs {
							if len(p.Extends) > 0 {
								extends = true
							}
							if len(p.Overrides) > 0 {
								overrides = true
							}
							if p.Composition != nil && p.Composition.Strategy != "" {
								comp = p.Composition.Strategy
							}
							for _, r := range p.Rules {
								levels[r.Level] = true
								if r.Logic == "all" {
									logicAll = true
								}
							}
						}
						deps.Telemetry.Collect("rulepack_usage", map[string]any{
							"l1":          levels[1],
							"l2":          levels[2],
							"l3":          levels[3],
							"extends":     extends,
							"overrides":   overrides,
							"composition": comp,
							"logic_all":   logicAll,
						})
					}
				}
			}

			err := handler.Execute(ctx, args, scancommand.Options{
				RulepackPath:  resolveRulepackPath(rulepackPath),
				ContextKVs:    contextKVs,
				Workers:       numWorkers,
				PendingWindow: 256,
				OutputFormat:  pickOutputFormat(deps),
				MetricsFile:   deps.Config.MetricsFile,
				TraceFile:     deps.Config.TraceFile,
				FailOn:        failOn,
				AuditFile:     deps.Config.AuditFile,
				Quiet:         quiet,
				// Progress is default-on for non-JSON output; disabled for JSON or when quiet
				ShowProgress:    viper.GetString("output_format") != "json" && !quiet,
				NoHints:         noHints,
				PerfSummary:     perfSummary,
				EffectiveConfig: deps.Config,
				ConfigFile:      viper.ConfigFileUsed(),
				OverrideHints:   overrides,
				RequestID:       reqID,
			}, os.Stdout)
			if err != nil {
				// Provide onboarding suggestion if no inputs found
				if errors.Is(err, discovery.ErrNoInputFiles) {
					fmt.Fprintln(os.Stderr, "❌ No input files found.")
					fmt.Fprintln(os.Stderr, "\nHelpful tips:")
					fmt.Fprintln(os.Stderr, "  promptshield demo                  # Try an interactive demo")
					fmt.Fprintln(os.Stderr, "  promptshield scan *.json          # Scan common JSON files")
					fmt.Fprintln(os.Stderr, "  promptshield rules:create          # Create a starter RulePack")
					return err
				}
				if errors.Is(err, sharederrors.ErrFailOnThreshold) {
					// Provide a helpful message mapping current threshold and examples
					fmt.Fprintf(os.Stderr, "Scan failed due to --fail-on=%s threshold.\n", strings.ToUpper(failOn))
					fmt.Fprintln(os.Stderr, "Tip: lower the threshold or fix the reported findings.")
					_ = sharedseverity.MeetsThreshold // ensure package referenced when suggestions evolve
				}
				// Actionable messages for semantic provider misconfiguration
				switch {
				case errors.Is(err, sharederrors.ErrSemanticProviderNotSet):
					fmt.Fprintln(os.Stderr, "❌ Semantic analysis enabled but PS_SEMANTIC_PROVIDER is not set (openai|anthropic).")
					fmt.Fprintln(os.Stderr, "   Set PS_SEMANTIC_PROVIDER and the corresponding API key, or disable semantic analysis.")
				case errors.Is(err, sharederrors.ErrUnsupportedProvider):
					fmt.Fprintln(os.Stderr, "❌ Unsupported PS_SEMANTIC_PROVIDER. Use 'openai' or 'anthropic'.")
				case errors.Is(err, sharederrors.ErrOpenAIAPIKeyMissing):
					fmt.Fprintln(os.Stderr, "❌ Missing OpenAI API key. Set via 'promptshield auth set --provider openai' or env OPENAI_API_KEY/PS_OPENAI_API_KEY.")
				case errors.Is(err, sharederrors.ErrAnthropicAPIKeyMissing):
					fmt.Fprintln(os.Stderr, "❌ Missing Anthropic API key. Set via 'promptshield auth set --provider anthropic' or env ANTHROPIC_API_KEY/PS_ANTHROPIC_API_KEY.")
				}
				if deps != nil && deps.Telemetry != nil {
					deps.Telemetry.Collect("error_summary", map[string]any{"kind": "scan_error"})
				}
				return err
			}
			if deps != nil && deps.Config.Debug || viper.GetBool("debug") {
				logger.Info("scan complete")
			}
			if deps != nil && deps.Telemetry != nil {
				deps.Telemetry.Collect("scan_summary", map[string]any{
					"files":         len(args),
					"output_format": strings.ToLower(pickOutputFormat(deps)),
					"workers":       numWorkers,
					"fail_on":       strings.ToUpper(failOn),
				})
			}
			return nil
		},
	}
	// Flags
	cmd.Flags().StringVarP(&rulepackPath, "rulepack", "r", "", "path to a RulePack file or directory")
	cmd.Flags().StringArrayVar(&contextKVs, "context", nil, "context key=value (repeatable; overrides pack defaults)")
	// metrics/trace are config/env only
	cmd.Flags().StringVar(&failOn, "fail-on", "", "fail if any violation meets/exceeds severity (INFO|WARNING|HIGH|ERROR|CRITICAL)")
	// audit/redaction are config/env only
	// Demo flag for quick start
	cmd.Flags().BoolVar(&demo, "demo", false, "run a quick demo scan using bundled samples")
	// Hints/perf are configuration-only now; keep flags as deprecated aliases that print guidance
	cmd.Flags().BoolVar(&noHints, "no-hints", false, "(deprecated) suppress hints; use config 'output.hints: false'")
	cmd.Flags().MarkDeprecated("no-hints", "use 'output.hints: false' in promptshield.yaml or PS_OUTPUT_HINTS=false")
	cmd.Flags().BoolVar(&perfSummary, "perf", false, "(deprecated) perf summary; use 'metrics_file' or 'PS_METRICS_FILE' for summaries")
	cmd.Flags().MarkDeprecated("perf", "use 'metrics_file' in config or PS_METRICS_FILE for summaries")

	// Positional file completion: common text/JSON formats
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) >= 1 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return []string{".txt", ".json", ".jsonl", ".ndjson"}, cobra.ShellCompDirectiveFilterFileExt
	}

	// --rulepack completion: YAML files
	_ = cmd.RegisterFlagCompletionFunc("rulepack", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{".yaml", ".yml"}, cobra.ShellCompDirectiveFilterFileExt
	})

	// --output-format completion: valid values
	_ = cmd.RegisterFlagCompletionFunc("output-format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"stylish", "json", "github", "ndjson"}, cobra.ShellCompDirectiveNoFileComp
	})

	// --fail-on completion: severities
	_ = cmd.RegisterFlagCompletionFunc("fail-on", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"INFO", "WARNING", "HIGH", "ERROR", "CRITICAL"}, cobra.ShellCompDirectiveNoFileComp
	})

	// Bind additional config keys for effective config output
	_ = viper.BindPFlag("rulepack", cmd.Flags().Lookup("rulepack"))
	// metrics/trace are config/env only
	_ = viper.BindPFlag("fail_on", cmd.Flags().Lookup("fail-on"))

	// Defaults for nested config keys
	viper.SetDefault("composition.strategy", "")
	viper.SetDefault("performance.max_length", 0)
	viper.SetDefault("performance.buffer_bytes", 16*1024*1024)
	viper.SetDefault("performance.chunk_overlap", 8*1024)
	viper.SetDefault("performance.timeout", "")
	viper.SetDefault("performance.per_rule_timeout", "")
	viper.SetDefault("performance.total_scan_timeout", "")
	viper.SetDefault("performance.case_sensitive", false)
	viper.SetDefault("performance.whole_word", false)
	viper.SetDefault("performance.gate.enabled", true)
	viper.SetDefault("performance.gate.min_token_len", 3)
	return cmd
}

func generateRequestID() string { return uuid.NewString() }

func pickOutputFormat(deps *bootstrap.Deps) string {
	if deps != nil && deps.Config.OutputFormat != "" {
		return deps.Config.OutputFormat
	}
	return viper.GetString("output_format")
}

// resolveRulepackPath selects the rulepack path based on flag, config, or a built-in default.
func resolveRulepackPath(flag string) string {
	// Re-implemented here to avoid leaking internal types; use viper to read effective config
	if flag != "" {
		return flag
	}
	if v := viper.GetString("rulepack"); v != "" {
		return v
	}
	// Fallback to built-in basic rules if present
	if _, err := os.Stat("rules/basic-security.yaml"); err == nil {
		return "rules/basic-security.yaml"
	}
	return ""
}

// NewScanFileCommand exposes the scan:file command with a short alias.
func NewScanFileCommand(deps *bootstrap.Deps) *cobra.Command {
	c := NewScanCommand(deps)
	c.Use = "scan:file [path|glob]..."
	c.Aliases = []string{"sf"}
	c.Short = "Scan files or globs for LLM security risks"
	return c
}

// NewScanDirectoryCommand exposes the scan:directory command with a short alias.
func NewScanDirectoryCommand(deps *bootstrap.Deps) *cobra.Command {
	c := NewScanCommand(deps)
	c.Use = "scan:directory [dir]..."
	c.Aliases = []string{"sd"}
	c.Short = "Scan a directory recursively for LLM security risks"
	// Optional: could validate args are directories; keep flexible to match prior behavior
	return c
}
