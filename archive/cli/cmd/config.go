package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/promptshield/promptshield/internal/bootstrap"
	"github.com/promptshield/promptshield/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configPrintWithDocs bool
)

var configPrintCmd = &cobra.Command{
	Use:   "print",
	Short: "Print effective configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		deps := bootstrap.From(cmd)
		used := viper.ConfigFileUsed()
		eff := config.ReadEffective(cmd.Context(), used, func(key string) any { return viper.Get(key) })
		if errs := config.Validate(eff); len(errs) > 0 {
			// Print all validation errors and return a combined error
			for _, e := range errs {
				fmt.Fprintln(os.Stderr, e.Error())
			}
			return fmt.Errorf("invalid configuration")
		}
		// Keep DI config in sync for downstream commands
		if deps != nil {
			deps.Config = eff
		}
		// Always emit JSON; redact known secret-like values (defensive; none expected in typed config)
		if configPrintWithDocs {
			payload := map[string]any{
				"config": eff,
				"schema": buildConfigSchema(),
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(payload); err != nil {
				return fmt.Errorf("encoding config with schema: %w", err)
			}
		} else {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(eff); err != nil {
				return fmt.Errorf("encoding config: %w", err)
			}
		}
		return nil
	},
}

func NewConfigCommand(deps *bootstrap.Deps) *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Manage configuration"}
	// print
	configPrintCmd.Flags().BoolVar(&configPrintWithDocs, "with-schema", false, "include documented schema alongside effective config")
	cmd.AddCommand(configPrintCmd)
	// validate
	var configValidateJSON bool
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file and values",
		RunE: func(cmd *cobra.Command, args []string) error {
			used := viper.ConfigFileUsed()
			// Collect errors
			type errItem struct {
				Type    string `json:"type"`
				Error   string `json:"error"`
				Path    string `json:"path,omitempty"`
				Suggest string `json:"suggest,omitempty"`
			}
			var list []errItem
			// Unknown-key check only applies when a file is used
			if used != "" {
				if err := config.ValidateConfigFile(used); err != nil {
					// Attempt to unwrap UnknownKeyError for path/suggest
					var uke config.UnknownKeyError
					if errors.As(err, &uke) {
						list = append(list, errItem{Type: "unknown_key", Error: uke.Error(), Path: uke.Path, Suggest: uke.Suggest})
					} else {
						list = append(list, errItem{Type: "file_error", Error: err.Error()})
					}
				}
			}
			// Validate effective config values (flags/env/defaults already applied)
			eff := config.ReadEffective(cmd.Context(), used, func(key string) any { return viper.Get(key) })
			if vErrs := config.Validate(eff); len(vErrs) > 0 {
				for _, e := range vErrs {
					list = append(list, errItem{Type: "validation", Error: e.Error()})
				}
			}
			if configValidateJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				if err := enc.Encode(list); err != nil {
					return fmt.Errorf("encoding validation errors: %w", err)
				}
				if len(list) > 0 {
					return fmt.Errorf("invalid configuration")
				}
				return nil
			}
			if len(list) > 0 {
				for _, it := range list {
					fmt.Fprintln(os.Stderr, it.Error)
				}
				return fmt.Errorf("invalid configuration")
			}
			if used == "" {
				fmt.Fprintln(os.Stdout, "Config OK (no config file found; using flags/env/defaults)")
			} else {
				fmt.Fprintln(os.Stdout, "Config OK")
			}
			return nil
		},
	}
	validateCmd.Flags().BoolVar(&configValidateJSON, "json", false, "emit machine-readable JSON error list")
	cmd.AddCommand(validateCmd)
	// schema
	cmd.AddCommand(&cobra.Command{
		Use:   "schema",
		Short: "Print JSON Schema for promptshield.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(buildConfigSchema())
		},
	})
	// doctor
	var doctorJSON bool
	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose common configuration issues and provide suggestions",
		RunE: func(cmd *cobra.Command, args []string) error {
			used := viper.ConfigFileUsed()
			type issue struct {
				Severity   string `json:"severity"` // warning|error
				Code       string `json:"code"`
				Message    string `json:"message"`
				Suggestion string `json:"suggestion,omitempty"`
			}
			var issues []issue
			// Unknown keys (file-only)
			if used != "" {
				if err := config.ValidateConfigFile(used); err != nil {
					var uke config.UnknownKeyError
					if errors.As(err, &uke) {
						issues = append(issues, issue{Severity: "error", Code: "unknown_key", Message: uke.Error()})
					} else {
						issues = append(issues, issue{Severity: "error", Code: "config_file", Message: err.Error()})
					}
				}
			}
			// Effective config checks
			eff := config.ReadEffective(cmd.Context(), used, func(key string) any { return viper.Get(key) })
			if vErrs := config.Validate(eff); len(vErrs) > 0 {
				for _, e := range vErrs {
					issues = append(issues, issue{Severity: "error", Code: "validation", Message: e.Error()})
				}
			}
			// Warnings
			of := strings.ToLower(eff.OutputFormat)
			switch of {
			case "ndjson":
				// ndjson is supported
			case "markdown", "csv", "html", "table":
				issues = append(issues, issue{Severity: "error", Code: "removed_output", Message: "Selected output format was removed", Suggestion: "Use 'stylish', 'json', 'github', or 'ndjson'"})
			}
			if eff.Redaction.Enabled == false {
				issues = append(issues, issue{Severity: "warning", Code: "redaction_disabled", Message: "Global redaction is disabled", Suggestion: "Set redaction.enabled: true or export PS_REDACTION_ENABLED=1"})
			}
			if eff.RulepackPath != "" {
				if _, err := os.Stat(eff.RulepackPath); err != nil {
					issues = append(issues, issue{Severity: "warning", Code: "rulepack_missing", Message: "Configured rulepack path not found", Suggestion: "Check 'rulepack:' path or run 'promptshield rules:list --path rules'"})
				}
			}
			if eff.Performance.MaxPatternLength > 0 && eff.Performance.MaxPatternLength < 16 {
				issues = append(issues, issue{Severity: "warning", Code: "max_pattern_length_low", Message: "performance.max_pattern_length is very low", Suggestion: "Use >= 100 to avoid rejecting valid rules"})
			}
			// Emit
			if doctorJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				// adapt to cfgIssue for hasErrors signature
				out := make([]cfgIssue, 0, len(issues))
				for _, it := range issues {
					out = append(out, cfgIssue{Severity: it.Severity, Code: it.Code, Message: it.Message, Suggestion: it.Suggestion})
				}
				if err := enc.Encode(out); err != nil {
					return fmt.Errorf("encoding doctor output: %w", err)
				}
				if hasErrors(out) {
					return fmt.Errorf("configuration issues detected")
				}
				return nil
			}
			// Human-readable
			if len(issues) == 0 {
				fmt.Fprintln(os.Stdout, "No configuration issues detected")
				return nil
			}
			for _, it := range issues {
				line := fmt.Sprintf("[%s] %s", strings.ToUpper(it.Severity), it.Message)
				if it.Suggestion != "" {
					line += fmt.Sprintf("\n  → %s", it.Suggestion)
				}
				fmt.Fprintln(os.Stderr, line)
			}
			// adapt for hasErrors
			out := make([]cfgIssue, 0, len(issues))
			for _, it := range issues {
				out = append(out, cfgIssue{Severity: it.Severity, Code: it.Code, Message: it.Message, Suggestion: it.Suggestion})
			}
			if hasErrors(out) {
				return fmt.Errorf("configuration issues detected")
			}
			return nil
		},
	}
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "emit machine-readable JSON issue list")
	cmd.AddCommand(doctorCmd)
	_ = deps
	return cmd
}

func buildConfigSchema() map[string]any {
	// Minimal documented JSON Schema (draft-07 style), stable keys only
	return map[string]any{
		"$schema":              "http://json-schema.org/draft-07/schema#",
		"title":                "PromptShield Config",
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"output_format": map[string]any{"type": "string", "enum": []string{"stylish", "json", "github", "ndjson"}, "description": "Output format for results"},
			"workers":       map[string]any{"type": "integer", "minimum": 0, "description": "Number of file-level workers (0=auto)"},
			"debug":         map[string]any{"type": "boolean", "description": "Enable debug logging to stderr"},
			"color":         map[string]any{"type": "boolean", "description": "Force color on/off for stylish output (optional)"},
			"audit_file":    map[string]any{"type": "string", "description": "Path to audit log file (rotating)"},
			"redaction": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"enabled": map[string]any{"type": "boolean", "description": "Redact common secrets in logs/audit"},
				},
			},
			"rulepack":     map[string]any{"type": "string", "description": "Path to a RulePack file or directory"},
			"metrics_file": map[string]any{"type": "string", "description": "Write one-line NDJSON summary (experimental)"},
			"trace_file":   map[string]any{"type": "string", "description": "Write per-span NDJSON (deprecated; prefer OpenTelemetry)"},
			"fail_on":      map[string]any{"type": "string", "enum": []string{"INFO", "WARNING", "HIGH", "ERROR", "CRITICAL"}, "description": "Exit non-zero if any violation meets/exceeds this severity"},
			"composition": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"strategy": map[string]any{"type": "string", "enum": []string{"first_match", "priority_order", ""}, "description": "Composition strategy for rule evaluation"},
				},
			},
			"performance": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"max_length":         map[string]any{"type": "integer", "minimum": 0, "description": "Max line length for regex evaluation (0=unlimited)"},
					"max_file_size":      map[string]any{"type": "integer", "minimum": 0, "description": "Max file size in bytes (default 100MB)"},
					"max_pattern_length": map[string]any{"type": "integer", "minimum": 0, "description": "Reject regex longer than this to avoid runaway complexity"},
					"timeout":            map[string]any{"type": "string", "description": "Default scan timeout (e.g., 10s)"},
					"per_rule_timeout":   map[string]any{"type": "string", "description": "Default per-rule timeout budget"},
					"total_scan_timeout": map[string]any{"type": "string", "description": "Total scan timeout budget"},
					"case_sensitive":     map[string]any{"type": "boolean", "description": "Default case sensitivity for keyword rules"},
					"whole_word":         map[string]any{"type": "boolean", "description": "Default whole-word match for keyword rules"},
					"buffer_bytes":       map[string]any{"type": "integer", "minimum": 0, "description": "Scanner token buffer size for long lines"},
					"chunk_overlap":      map[string]any{"type": "integer", "minimum": 0, "description": "Overlap bytes when chunking long lines"},
					"gate": map[string]any{
						"type":                 "object",
						"additionalProperties": false,
						"properties": map[string]any{
							"enabled":       map[string]any{"type": "boolean", "description": "Enable literal-token gating for regex/semantic"},
							"min_token_len": map[string]any{"type": "integer", "minimum": 0, "description": "Minimum literal token length for gates"},
						},
					},
				},
			},
			"telemetry": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"enabled":  map[string]any{"type": "boolean", "description": "Enable OTLP exporters (privacy-first, coarse spans)"},
					"endpoint": map[string]any{"type": "string", "description": "OTLP gRPC endpoint (host:port)"},
					"file":     map[string]any{"type": "string", "description": "Local NDJSON sink file for coarse spans"},
					"sample":   map[string]any{"type": "number", "minimum": 0, "maximum": 1, "description": "Sampling rate 0..1"},
				},
			},
		},
	}
}

// local struct mirror for hasErrors signature
type cfgIssue struct{ Severity, Code, Message, Suggestion string }

func hasErrors(issues []cfgIssue) bool {
	for _, it := range issues {
		if strings.EqualFold(it.Severity, "error") {
			return true
		}
	}
	return false
}
