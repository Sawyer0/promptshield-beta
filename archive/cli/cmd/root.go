package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/promptshield/promptshield/internal/audit"
	"github.com/promptshield/promptshield/internal/bootstrap"
	"github.com/promptshield/promptshield/internal/config"
	tel "github.com/promptshield/promptshield/internal/observability/telemetry"
	"github.com/promptshield/promptshield/internal/shared/deprecation"
	"github.com/promptshield/promptshield/internal/shared/output"
	"github.com/promptshield/promptshield/internal/shared/redact"
	"github.com/promptshield/promptshield/internal/shared/termui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile      string
	outputFormat string
	jsonOutput   bool
	quiet        bool
)

// rootCmd is the base command for PromptShield CLI.
var rootCmd = &cobra.Command{
	Use:               "promptshield",
	Short:             "PromptShield – CLI scanner for LLM safety",
	Long:              "PromptShield scans prompts/responses for security risks with a progressive rule system (keywords → regex → semantic).",
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	RunE: func(cmd *cobra.Command, _ []string) error {
		// Friendly onboarding when run without subcommand
		deps := bootstrap.Build(bootstrap.Options{Debug: viper.GetBool("debug"), Quiet: quiet, LogJSON: false, MaxTokenBytes: 0})
		_ = deps
		color := true
		fmt.Fprintln(os.Stdout, termui.Heading(color, "PromptShield - AI Security Scanner"))
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, termui.Heading(color, "Getting started:"))
		fmt.Fprintln(os.Stdout, termui.Bullet(color, "promptshield demo", "Try an interactive demo"))
		fmt.Fprintln(os.Stdout, termui.Bullet(color, "promptshield scan:file file.json", "Scan files or globs"))
		fmt.Fprintln(os.Stdout, termui.Bullet(color, "promptshield scan:directory ./data", "Scan a directory recursively"))
		fmt.Fprintln(os.Stdout, termui.Bullet(color, "promptshield rules:create", "Create a starter RulePack"))
		fmt.Fprintln(os.Stdout)
		return nil
	},
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		deps, ctxWithDeps, err := loadConfigAndDeps(cmd.Context())
		if err != nil {
			return err
		}
		cmd.SetContext(ctxWithDeps)
		// Flags/env/config precedence: flags > env > config > defaults
		if jsonOutput {
			outputFormat = "json"
		}
		_ = viper.BindPFlag("output_format", cmd.Flags().Lookup("output-format"))
		// Auto-switch to JSON in CI unless user explicitly asked otherwise
		// Respect flags: --output-format / --json
		userSetFormat := cmd.Flags().Changed("output-format") || cmd.Flags().Changed("json")
		if isCIEnvironment() && !userSetFormat {
			viper.Set("output_format", "json")
		} else if outputFormat != "" {
			viper.Set("output_format", output.Normalize(outputFormat))
		}
		// Deprecation notice for unstable formats (shared util)
		if warn, msg := deprecation.LegacyOutputFormatWarning(viper.GetString("output_format")); warn {
			fmt.Fprintln(os.Stderr, msg)
		}
		if err := output.Validate(viper.GetString("output_format")); err != nil {
			return err
		}
		// Apply global redaction toggle (default true)
		redaction := viper.GetBool("redaction.enabled")
		redact.SetEnabled(redaction)
		if !redaction {
			// Warn once per invocation on stderr
			fmt.Fprintln(os.Stderr, "[SECURITY WARNING] Redaction is disabled. Sensitive values (API keys, tokens) may appear in logs and audit files.")
		}

		// Audit effective config and overrides once per invocation if enabled
		if deps != nil && deps.Config.AuditFile != "" {
			if rl, e := audit.NewDailyRotatingLogger(deps.Config.AuditFile); e == nil {
				defer func() { _ = rl.Close() }()
				overrides := map[string]any{}
				if cmd.Flags().Changed("output-format") || cmd.Flags().Changed("json") {
					overrides["output_format"] = "flag"
				}
				if cmd.Flags().Changed("quiet") {
					overrides["quiet"] = "flag"
				}
				// Selected env overrides (best-effort)
				if os.Getenv("PS_OUTPUT_FORMAT") != "" {
					overrides["PS_OUTPUT_FORMAT"] = "env"
				}
				if os.Getenv("PS_WORKERS") != "" {
					overrides["PS_WORKERS"] = "env"
				}
				if os.Getenv("PS_DEBUG") != "" {
					overrides["PS_DEBUG"] = "env"
				}
				if os.Getenv("PS_REDACTION_ENABLED") != "" {
					overrides["PS_REDACTION_ENABLED"] = "env"
				}
				// Emit effective snapshot with minimal keys (no secrets)
				_ = rl.Log(audit.Event{Type: "config_effective", Data: audit.SanitizeMap(map[string]any{
					"config_file": viper.ConfigFileUsed(),
					"output":      viper.GetString("output_format"),
					"workers":     viper.GetInt("workers"),
					"debug":       viper.GetBool("debug"),
					"quiet":       quiet,
					"overrides":   overrides,
				})})
			}
		}

		// Telemetry: opt-in via PS_TELEMETRY=1 or telemetry.enabled: true
		if deps != nil {
			enabled := viper.GetBool("telemetry.enabled") || os.Getenv("PS_TELEMETRY") == "1"
			endpoint := viper.GetString("telemetry.endpoint")
			file := viper.GetString("telemetry.file")
			sample := 1.0
			if v := viper.Get("telemetry.sample"); v != nil {
				switch x := v.(type) {
				case float64:
					sample = x
				case int:
					sample = float64(x)
				}
			}
			if enabled && (endpoint != "" || file != "") {
				// Generate or reuse a local machine id file under ~/.promptshield
				mid := readOrCreateMachineID()
				deps.Telemetry = tel.New(tel.Options{Enabled: true, Endpoint: endpoint, File: file, Sample: sample, MachineID: mid, Service: "promptshield", Version: version})
				// Emit startup event
				deps.Telemetry.Collect("startup", map[string]any{
					"version":  version,
					"commit":   commit,
					"os":       runtime.GOOS,
					"arch":     runtime.GOARCH,
					"ci":       isCIEnvironment(),
					"json_out": viper.GetString("output_format") == "json",
				})
			}
		}

		_ = deps
		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command with the provided context.
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

// Setup loads config/deps and attaches all subcommands constructed with DI.
func Setup(ctx context.Context) error {
	deps, ctxWithDeps, err := loadConfigAndDeps(ctx)
	if err != nil {
		return err
	}
	rootCmd.SetContext(ctxWithDeps)
	AttachCommands(deps)
	// Emit telemetry for setup when enabled
	if deps != nil && deps.Telemetry != nil {
		deps.Telemetry.Collect("setup", map[string]any{"version": version, "ci": isCIEnvironment()})
	}
	return nil
}

// AttachCommands wires all subcommands built with the provided deps.
func AttachCommands(deps *bootstrap.Deps) {
	// Avoid duplicate attachments by clearing existing subcommands if needed.
	// Cobra does not provide a clear API; assume first-time call in main.
	rootCmd.AddCommand(NewScanCommand(deps))
	rootCmd.AddCommand(NewScanFileCommand(deps))
	rootCmd.AddCommand(NewScanDirectoryCommand(deps))
	rootCmd.AddCommand(NewRulesListCommand(deps))
	rootCmd.AddCommand(NewRulesCreateCommand(deps))
	rootCmd.AddCommand(NewRulesValidateCommand(deps))
	rootCmd.AddCommand(NewInitCommand(deps))
	rootCmd.AddCommand(NewValidateCommand(deps))
	cfgCmd := NewConfigCommand(deps)
	rootCmd.AddCommand(cfgCmd)
	rootCmd.AddCommand(NewUpdateCommand(deps))
	rootCmd.AddCommand(NewAuthCommand(deps))

	// Root-level alias: doctor → config doctor
	rootCmd.AddCommand(&cobra.Command{
		Use:   "doctor",
		Short: "Diagnose common configuration issues and provide suggestions",
		RunE: func(cmd *cobra.Command, args []string) error {
			dc, _, _ := cfgCmd.Find([]string{"doctor"})
			if dc == nil {
				return fmt.Errorf("doctor command not available")
			}
			dc.SetArgs(args)
			dc.SetIn(cmd.InOrStdin())
			dc.SetOut(cmd.OutOrStdout())
			dc.SetErr(cmd.ErrOrStderr())
			return dc.Execute()
		},
	})
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (default: promptshield.yaml or ~/.promptshield/promptshield.yaml)")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output-format", "stylish", "output format: stylish|json|github|ndjson")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "shorthand for --output-format=json")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "reduce log verbosity (errors only)")

	viper.SetEnvPrefix("PS")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("output_format", "stylish")
	viper.SetDefault("workers", 0)
	viper.SetDefault("redaction.enabled", true)
	viper.SetDefault("debug", false)
	viper.SetDefault("audit_file", "")
	viper.SetDefault("metrics_file", "")
	viper.SetDefault("trace_file", "")
	viper.SetDefault("fail_on", "")

	// Explicit env bindings for nested/critical keys
	_ = viper.BindEnv("workers", "PS_WORKERS")
	_ = viper.BindEnv("debug", "PS_DEBUG")
	_ = viper.BindEnv("redaction.enabled", "PS_REDACTION_ENABLED")
	_ = viper.BindEnv("audit_file", "PS_AUDIT_FILE")
	_ = viper.BindEnv("metrics_file", "PS_METRICS_FILE")
	_ = viper.BindEnv("trace_file", "PS_TRACE_FILE")
	_ = viper.BindEnv("fail_on", "PS_FAIL_ON")
	_ = viper.BindEnv("rulepack", "PS_RULEPACK")
	_ = viper.BindEnv("telemetry.enabled", "PS_TELEMETRY")
	_ = viper.BindEnv("telemetry.endpoint", "PS_TELEMETRY_ENDPOINT")
	_ = viper.BindEnv("telemetry.file", "PS_TELEMETRY_FILE")
	_ = viper.BindEnv("telemetry.sample", "PS_TELEMETRY_SAMPLE")
}

// Ensure providers flush after command execution
func init() {
	rootCmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		if deps := bootstrap.From(cmd); deps != nil && deps.Telemetry != nil {
			// best-effort shutdown with short timeout
			ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Second)
			defer cancel()
			_ = deps.Telemetry.Shutdown(ctx)
		}
	}
}

// readOrCreateMachineID stores a random ID at ~/.promptshield/machine_id and returns it.
func readOrCreateMachineID() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(home, ".promptshield")
	_ = os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, "machine_id")
	if b, err := os.ReadFile(path); err == nil {
		s := strings.TrimSpace(string(b))
		if len(s) > 0 {
			return s
		}
	}
	id := generateRequestID()
	// best-effort write
	_ = os.WriteFile(path, []byte(id), fs.FileMode(0o644))
	return id
}

// (removed) legacy initConfig stub was unused

func loadConfigAndDeps(ctx context.Context) (*bootstrap.Deps, context.Context, error) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Enhanced discovery: project root, XDG, home fallback
		viper.SetConfigName("promptshield")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		// XDG config home
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			viper.AddConfigPath(filepath.Join(xdg, "promptshield"))
		}
		// Home fallback
		if home, err := os.UserHomeDir(); err == nil {
			viper.AddConfigPath(filepath.Join(home, ".config", "promptshield"))
			viper.AddConfigPath(filepath.Join(home, ".promptshield"))
		}
	}
	if err := viper.ReadInConfig(); err != nil {
		var nf viper.ConfigFileNotFoundError
		if errors.As(err, &nf) {
			used := viper.ConfigFileUsed()
			// First-run helper: scaffold promptshield.yaml if no config exists and a single rulepack is present
			scaffoldIfFirstRun()
			d := bootstrap.Build(bootstrap.Options{Debug: viper.GetBool("debug"), Quiet: quiet, LogJSON: false, MaxTokenBytes: 0})
			eff := config.ReadEffective(ctx, used, func(key string) any { return viper.Get(key) })
			d.Config = eff
			c := bootstrap.WithDeps(ctx, d)
			return d, c, nil
		}
		return nil, ctx, fmt.Errorf("reading config: %w", err)
	}
	// Validate unknown keys if a config file was used
	used := viper.ConfigFileUsed()
	if err := config.ValidateConfigFile(used); err != nil {
		return nil, ctx, fmt.Errorf("invalid config file %s: %w", used, err)
	}
	// Build and attach DI container to context for all subcommands
	d := bootstrap.Build(bootstrap.Options{Debug: viper.GetBool("debug"), Quiet: quiet, LogJSON: false, MaxTokenBytes: 0})
	// Compute effective typed config and store into deps
	eff := config.ReadEffective(ctx, used, func(key string) any { return viper.Get(key) })
	if errs := config.Validate(eff); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, e.Error())
		}
		return nil, ctx, fmt.Errorf("invalid configuration")
	}
	d.Config = eff
	c := bootstrap.WithDeps(ctx, d)
	return d, c, nil
}

// scaffoldIfFirstRun creates a minimal promptshield.yaml if none exists and
// exactly one rulepack YAML is found under ./rules/ (non-recursive). It is a
// no-op if a config already exists or if multiple/zero packs are present.
func scaffoldIfFirstRun() {
	// Do not overwrite existing file
	if _, err := os.Stat("promptshield.yaml"); err == nil {
		return
	}
	// Look for exactly one YAML under rules/
	entries, err := os.ReadDir("rules")
	if err != nil {
		return
	}
	var candidates []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".yaml") || strings.HasSuffix(strings.ToLower(name), ".yml") {
			candidates = append(candidates, filepath.ToSlash(filepath.Join("rules", name)))
		}
	}
	if len(candidates) != 1 {
		return
	}
	// Write minimal config pointing to the single rulepack and sensible defaults
	content := "rulepack: " + candidates[0] + "\noutput_format: stylish\n"
	// Best-effort write; ignore errors to avoid blocking startup
	_ = os.WriteFile("promptshield.yaml", []byte(content), 0o644)
}

// isCIEnvironment returns true if common CI environment variables are present.
func isCIEnvironment() bool {
	// Common CI env vars across providers
	keys := []string{
		"CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "BUILD_ID", "CIRCLECI", "TRAVIS", "APPVEYOR",
	}
	for _, k := range keys {
		if v := os.Getenv(k); v != "" && strings.ToLower(v) != "false" && v != "0" {
			return true
		}
	}
	return false
}

// note: deprecation helpers moved to internal/shared/deprecation
