package scancommand

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/promptshield/promptshield/internal/application/scan"
	"github.com/promptshield/promptshield/internal/audit"
	cfg "github.com/promptshield/promptshield/internal/config"
	"github.com/promptshield/promptshield/internal/discovery"
	obmetrics "github.com/promptshield/promptshield/internal/observability/metrics"
	"github.com/promptshield/promptshield/internal/report"
	"github.com/promptshield/promptshield/internal/shared/deprecation"
	sharederrors "github.com/promptshield/promptshield/internal/shared/errors"
	"github.com/promptshield/promptshield/internal/shared/severity"
	"github.com/promptshield/promptshield/pkg/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	// codes not used here; error recording is sufficient
	"go.opentelemetry.io/otel/trace"
)

// Options represent CLI-facing scan settings mapped to application concerns.
type Options struct {
	RulepackPath  string
	ContextKVs    []string
	Workers       int
	PendingWindow int

	OutputFormat string // stylish|json|ndjson
	MetricsFile  string // optional
	TraceFile    string // optional
	FailOn       string // optional severity
	AuditFile    string // optional audit events file (config-driven)
	// Quiet mode: when true, suppress human-readable output unless errors; used for --quiet
	Quiet bool
	// ShowProgress prints progress to stderr during scanning; ignored for JSON output
	ShowProgress bool
	// NoHints suppresses post-scan next steps summary in stylish mode
	NoHints bool
	// PerfSummary prints aggregated perf counters to stdout (JSON) at the end
	PerfSummary bool

	// RequestID for correlating logs, traces, and audit entries
	RequestID string

	// Effective configuration snapshot and hints for audit logging
	EffectiveConfig cfg.Config
	ConfigFile      string
	OverrideHints   map[string]string
}

// Handler coordinates a scan using the application service and renders results.
type Handler struct {
	svc    *scan.Service
	logger *slog.Logger
}

func NewHandler(svc *scan.Service, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// Execute runs the scan and writes formatted results to out.
func (h *Handler) Execute(ctx context.Context, args []string, opt Options, out io.Writer) error {
	// Tracing via OpenTelemetry (trace_file deprecated)
	if opt.TraceFile != "" && h.logger != nil {
		h.logger.Warn(deprecation.ExperimentalFeatureMessage("trace_file is deprecated; use OpenTelemetry export via telemetry.endpoint", "", ""))
	}

	// Start run span
	tr := otel.Tracer("promptshield/cli")
	ctx, runSpan := tr.Start(ctx, "scan_run", trace.WithAttributes(attribute.String("request_id", opt.RequestID)))
	// Optional audit logging (config-driven)
	var auditLogger audit.Logger
	var auditClose func()
	if opt.AuditFile != "" {
		if h.logger != nil {
			h.logger.Warn(deprecation.ExperimentalFeatureMessage("audit logging (audit_file)", "", ""))
		}
		if rl, err := audit.NewDailyRotatingLogger(opt.AuditFile); err == nil {
			auditLogger = rl
			auditClose = func() { _ = rl.Close() }
			// Capture a sanitized, minimal snapshot of effective config and any overrides
			startData := map[string]any{
				"args":        args,
				"config_file": opt.ConfigFile,
				"output":      opt.OutputFormat,
				"workers":     opt.Workers,
				"fail_on":     opt.FailOn,
			}
			if opt.RequestID != "" {
				startData["request_id"] = opt.RequestID
			}
			if opt.OverrideHints != nil {
				startData["overrides"] = opt.OverrideHints
			}
			// Do not log rulepack contents; only its path
			if opt.RulepackPath != "" {
				startData["rulepack"] = opt.RulepackPath
			}
			_ = auditLogger.Log(audit.Event{Type: "scan_start", Data: startData})
			defer func() {
				_ = auditLogger.Log(audit.Event{Type: "scan_end", Data: map[string]any{}})
				if auditClose != nil {
					auditClose()
				}
			}()
		} else if f, err2 := os.OpenFile(opt.AuditFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err2 == nil {
			// Fallback to non-rotating logger if rotation init fails
			fl := audit.NewFileLogger(f)
			auditLogger = fl
			auditClose = func() { _ = f.Close() }
			_ = auditLogger.Log(audit.Event{Type: "scan_start", Data: map[string]any{"args": args}})
			defer func() {
				_ = auditLogger.Log(audit.Event{Type: "scan_end", Data: map[string]any{}})
				if auditClose != nil {
					auditClose()
				}
			}()
		}
	}
	of := strings.ToLower(opt.OutputFormat)
	// If NDJSON streaming is requested, switch to streaming path for bounded memory
	if of == "ndjson" {
		if ok, msg := deprecation.LegacyOutputFormatWarning("ndjson"); ok && h.logger != nil {
			h.logger.Warn(msg)
		}
		// Event-per-violation NDJSON streaming
		var vioHits bool
		events := report.NewNDJSONEventWriter(out)
		var total int
		var done int
		var filesScanned int
		var violationCount int
		var agg types.Metrics
		if opt.ShowProgress && !opt.Quiet {
			if paths, err := discovery.DiscoverPaths(args); err == nil {
				total = len(paths)
				if total > 0 {
					fmt.Fprintf(os.Stderr, "Scanning %d %s...\n", total, plural(total, "file", "files"))
				}
			}
		}
		// Stream deterministic ordered results and emit per-violation events
		if err := h.svc.Stream(ctx, args, scan.StreamOptions{
			RulepackPath:  opt.RulepackPath,
			ContextKVs:    opt.ContextKVs,
			Workers:       opt.Workers,
			PendingWindow: opt.PendingWindow,
			Emit: func(r types.ScanResult) error {
				filesScanned++
				agg.BytesRead += r.Metrics.BytesRead
				agg.LinesRead += r.Metrics.LinesRead
				agg.RegexAttempts += r.Metrics.RegexAttempts
				agg.RegexSkipped += r.Metrics.RegexSkipped
				agg.SemanticAttempts += r.Metrics.SemanticAttempts
				agg.SemanticSkipped += r.Metrics.SemanticSkipped
				obmetrics.IncFilesScanned()
				obmetrics.AddViolations(len(r.Violations))
				for _, v := range r.Violations {
					violationCount++
					if err := events.WriteViolation(r.Input, v); err != nil {
						return err
					}
				}
				if opt.ShowProgress && !opt.Quiet && total > 0 {
					done++
					fmt.Fprintf(os.Stderr, "%d/%d %s\n", done, total, r.Input)
				}
				if auditLogger != nil {
					_ = auditLogger.Log(audit.Event{Type: "scan_file", Data: map[string]any{"input": r.Input, "violations": len(r.Violations)}})
				}
				if opt.FailOn != "" {
					for _, v := range r.Violations {
						if severity.MeetsThreshold(v.Severity, opt.FailOn) {
							vioHits = true
						}
					}
				}
				return nil
			},
		}); err != nil {
			runSpan.RecordError(err)
			runSpan.End()
			return err
		}
		// Emit final summary line
		if err := events.WriteSummary(filesScanned, violationCount); err != nil {
			runSpan.RecordError(err)
			runSpan.End()
			return err
		}
		// Optional: write audit metrics summary once
		if auditLogger != nil {
			_ = auditLogger.Log(audit.Event{Type: "scan_summary", Data: map[string]any{"files": filesScanned, "violations": violationCount}})
		}
		// Optional perf summary line
		if opt.PerfSummary {
			fmt.Fprintf(out, "{\"regex_attempts\":%d,\"regex_skipped\":%d,\"semantic_attempts\":%d,\"semantic_skipped\":%d,\"bytes_read\":%d,\"lines_read\":%d}\n",
				agg.RegexAttempts, agg.RegexSkipped, agg.SemanticAttempts, agg.SemanticSkipped, agg.BytesRead, agg.LinesRead)
		}
		runSpan.End()
		if opt.FailOn != "" && vioHits {
			return sharederrors.ErrFailOnThreshold
		}
		// metrics file is a summary; for streaming mode, we would need to buffer aggregates. Keep behavior consistent with non-streaming for now by skipping metrics in streaming mode.
		return nil
	}

	// For non-JSON outputs with progress enabled, stream to show progress and accumulate
	if opt.ShowProgress && !opt.Quiet && of != "json" {
		var results []types.ScanResult
		var total int
		var done int
		if paths, err := discovery.DiscoverPaths(args); err == nil {
			total = len(paths)
			if total > 0 {
				fmt.Fprintf(os.Stderr, "Scanning %d %s...\n", total, plural(total, "file", "files"))
			}
		}
		err := h.svc.Stream(ctx, args, scan.StreamOptions{
			RulepackPath:  opt.RulepackPath,
			ContextKVs:    opt.ContextKVs,
			Workers:       opt.Workers,
			PendingWindow: opt.PendingWindow,
			Emit: func(r types.ScanResult) error {
				results = append(results, r)
				if opt.ShowProgress && total > 0 {
					done++
					fmt.Fprintf(os.Stderr, "%d/%d %s\n", done, total, r.Input)
				}
				return nil
			},
		})
		if err != nil {
			runSpan.RecordError(err)
			runSpan.End()
			return err
		}
		runSpan.End()
		// Render accumulated results
		for _, r := range results {
			switch of {
			case "github":
				if err := report.RenderGitHub(out, r); err != nil {
					return err
				}
			default:
				if opt.Quiet {
					// suppress
				} else {
					if err := report.RenderStylishWithOptions(out, r, report.StylishOptions{Color: true, Spacing: true}); err != nil {
						return err
					}
					obmetrics.IncFilesScanned()
					obmetrics.AddViolations(len(r.Violations))
				}
			}
		}
		// Post-scan hints
		if of == "stylish" && !opt.NoHints && !opt.Quiet {
			printHints(out, results)
		}
		if opt.PerfSummary {
			var agg types.Metrics
			for _, r := range results {
				agg.BytesRead += r.Metrics.BytesRead
				agg.LinesRead += r.Metrics.LinesRead
				agg.RegexAttempts += r.Metrics.RegexAttempts
				agg.RegexSkipped += r.Metrics.RegexSkipped
				agg.SemanticAttempts += r.Metrics.SemanticAttempts
				agg.SemanticSkipped += r.Metrics.SemanticSkipped
			}
			fmt.Fprintf(out, "{\"regex_attempts\":%d,\"regex_skipped\":%d,\"semantic_attempts\":%d,\"semantic_skipped\":%d,\"bytes_read\":%d,\"lines_read\":%d}\n",
				agg.RegexAttempts, agg.RegexSkipped, agg.SemanticAttempts, agg.SemanticSkipped, agg.BytesRead, agg.LinesRead)
		}
		// Fail-on threshold check
		if opt.FailOn != "" {
			for _, r := range results {
				for _, v := range r.Violations {
					if severity.MeetsThreshold(v.Severity, opt.FailOn) {
						return sharederrors.ErrFailOnThreshold
					}
				}
			}
		}
		return nil
	}

	results, err := h.svc.Scan(ctx, args, scan.Options{
		RulepackPath:  opt.RulepackPath,
		ContextKVs:    opt.ContextKVs,
		Workers:       opt.Workers,
		PendingWindow: opt.PendingWindow,
	})
	if err != nil {
		if errors.Is(err, discovery.ErrNoInputFiles) {
			runSpan.RecordError(err)
			runSpan.End()
			return err
		}
		runSpan.RecordError(err)
		runSpan.End()
		return err
	}

	// Render results
	for _, r := range results {
		switch of {
		case "json":
			if err := report.RenderJSON(out, r); err != nil {
				runSpan.RecordError(err)
				runSpan.End()
				return err
			}
		case "github":
			if err := report.RenderGitHub(out, r); err != nil {
				runSpan.RecordError(err)
				runSpan.End()
				return err
			}
		default:
			if opt.Quiet {
				// In quiet mode, suppress standard output for human format
			} else {
				// Default stylish with color and spacing for better readability
				if err := report.RenderStylishWithOptions(out, r, report.StylishOptions{Color: true, Spacing: true}); err != nil {
					runSpan.RecordError(err)
					runSpan.End()
					return err
				}
			}
		}
	}
	runSpan.End()

	// Fail-on threshold policy: if any violation meets/exceeds severity, return threshold error
	if opt.FailOn != "" {
		for _, r := range results {
			for _, v := range r.Violations {
				if severity.MeetsThreshold(v.Severity, opt.FailOn) {
					return sharederrors.ErrFailOnThreshold
				}
			}
		}
	}

	// Optional NDJSON metrics summary
	if opt.MetricsFile != "" {
		if h.logger != nil {
			h.logger.Warn(deprecation.ExperimentalFeatureMessage("metrics output (metrics_file)", "--output-format=json", ""))
		}
		f, err := os.Create(opt.MetricsFile)
		if err != nil {
			return err
		}
		mw := obmetrics.NewNDJSONWriter(f)
		sum := obmetrics.Summary{}
		sum.Files = len(results)
		var vio int
		var bytesRead, linesRead int64
		var min, max, sumDur int64
		for i, r := range results {
			vio += len(r.Violations)
			bytesRead += r.Metrics.BytesRead
			linesRead += r.Metrics.LinesRead
			sum.RegexAttempts += r.Metrics.RegexAttempts
			sum.RegexSkipped += r.Metrics.RegexSkipped
			sum.SemanticAttempts += r.Metrics.SemanticAttempts
			sum.SemanticSkipped += r.Metrics.SemanticSkipped
			if i == 0 || r.DurationMs < min {
				min = r.DurationMs
			}
			if r.DurationMs > max {
				max = r.DurationMs
			}
			sumDur += r.DurationMs
		}
		sum.Violations = vio
		sum.BytesRead = bytesRead
		sum.LinesRead = linesRead
		sum.DurationMsMin = min
		sum.DurationMsMax = max
		sum.DurationMsSum = sumDur
		if err := mw.WriteSummary(sum); err != nil {
			_ = f.Close()
			return fmt.Errorf("write metrics: %w", err)
		}
		_ = f.Close()
	}

	// Optional perf summary to stdout
	if opt.PerfSummary {
		var agg types.Metrics
		for _, r := range results {
			agg.BytesRead += r.Metrics.BytesRead
			agg.LinesRead += r.Metrics.LinesRead
			agg.RegexAttempts += r.Metrics.RegexAttempts
			agg.RegexSkipped += r.Metrics.RegexSkipped
			agg.SemanticAttempts += r.Metrics.SemanticAttempts
			agg.SemanticSkipped += r.Metrics.SemanticSkipped
		}
		fmt.Fprintf(out, "{\"regex_attempts\":%d,\"regex_skipped\":%d,\"semantic_attempts\":%d,\"semantic_skipped\":%d,\"bytes_read\":%d,\"lines_read\":%d}\n",
			agg.RegexAttempts, agg.RegexSkipped, agg.SemanticAttempts, agg.SemanticSkipped, agg.BytesRead, agg.LinesRead)
	}
	// Post-scan hints for stylish
	if of == "stylish" && !opt.NoHints && !opt.Quiet {
		printHints(out, results)
	}
	return nil
}

// printHints renders a compact next-steps summary for stylish output.
func printHints(w io.Writer, results []types.ScanResult) {
	totalFiles := len(results)
	totalIssues := 0
	critical := 0
	for _, r := range results {
		totalIssues += len(r.Violations)
		for _, v := range r.Violations {
			if strings.ToUpper(v.Severity) == "CRITICAL" {
				critical++
			}
		}
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "✓ Scan complete: %d %s found (%d critical) across %d %s\n", totalIssues, plural(totalIssues, "issue", "issues"), critical, totalFiles, plural(totalFiles, "file", "files"))
	fmt.Fprintln(w, "  → Run 'promptshield scan:file --json <paths>' for machine-readable output")
	fmt.Fprintln(w, "  → Run 'promptshield rules:create' to customize rules")
	fmt.Fprintln(w, "  → Run 'promptshield rules:validate --path rules' to check packs")
}

// plural returns the correct singular/plural form based on n.
func plural(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
