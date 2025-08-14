package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/promptshield/promptshield/internal/encoding/jsonx"
	"github.com/promptshield/promptshield/pkg/types"
)

// RenderStylish prints a human-friendly report.
func RenderStylish(w io.Writer, res types.ScanResult) error {
	// Backward-compatible default: no color, minimal spacing
	return RenderStylishWithOptions(w, res, StylishOptions{Color: false, Spacing: false})
}

// StylishOptions controls human-readable rendering.
type StylishOptions struct {
	Color   bool
	Spacing bool
}

// RenderStylishWithOptions prints a human-friendly report with optional color and spacing.
func RenderStylishWithOptions(w io.Writer, res types.ScanResult, opt StylishOptions) error {
	// Header
	header := fmt.Sprintf("Input: %s\n", res.Input)
	if _, err := io.WriteString(w, maybeDim(opt.Color, header)); err != nil {
		return err
	}
	if opt.Spacing {
		if _, err := io.WriteString(w, "\n"); err != nil {
			return err
		}
	}
	if len(res.Violations) == 0 {
		line := "No issues found\n"
		if _, err := io.WriteString(w, maybeGreen(opt.Color, line)); err != nil {
			return err
		}
		return nil
	}
	for i, v := range res.Violations {
		sev := strings.ToUpper(v.Severity)
		sevStr := fmt.Sprintf("[%s]", sev)
		sevStr = colorSeverity(opt.Color, sev, sevStr)
		msg := fmt.Sprintf(" %s:%d:%d %s (%s)\n", res.Input, v.Line, v.Column, v.Message, v.RuleID)
		if _, err := io.WriteString(w, sevStr+msg); err != nil {
			return err
		}
		if opt.Spacing && i < len(res.Violations)-1 {
			if _, err := io.WriteString(w, "\n"); err != nil {
				return err
			}
		}
	}
	return nil
}

const (
	ansiReset   = "\x1b[0m"
	ansiBold    = "\x1b[1m"
	ansiFaint   = "\x1b[2m"
	ansiRed     = "\x1b[31m"
	ansiMagenta = "\x1b[35m"
	ansiYellow  = "\x1b[33m"
	ansiCyan    = "\x1b[36m"
	ansiGreen   = "\x1b[32m"
)

func colorSeverity(enabled bool, sev string, s string) string {
	if !enabled {
		return s
	}
	switch sev {
	case "CRITICAL":
		return ansiBold + ansiRed + s + ansiReset
	case "ERROR":
		return ansiRed + s + ansiReset
	case "HIGH":
		return ansiMagenta + s + ansiReset
	case "WARNING":
		return ansiYellow + s + ansiReset
	case "INFO":
		return ansiCyan + s + ansiReset
	default:
		return s
	}
}

func maybeDim(enabled bool, s string) string {
	if !enabled {
		return s
	}
	return ansiFaint + s + ansiReset
}

func maybeGreen(enabled bool, s string) string {
	if !enabled {
		return s
	}
	return ansiGreen + s + ansiReset
}

// RenderJSON writes the raw result as JSON.
func RenderJSON(w io.Writer, res types.ScanResult) error {
	enc := jsonx.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(res)
}

// RenderGitHub writes GitHub Actions annotation lines for each violation.
// Format: ::error file=path,line=1,col=1::message (rule)
func RenderGitHub(w io.Writer, res types.ScanResult) error {
	if len(res.Violations) == 0 {
		return nil
	}
	for _, v := range res.Violations {
		level := "error"
		switch v.Severity {
		case "INFO":
			level = "notice"
		case "WARNING":
			level = "warning"
		case "ERROR", "CRITICAL":
			level = "error"
		}
		if _, err := fmt.Fprintf(w, "::%s file=%s,line=%d,col=%d::%s (%s)\n", level, res.Input, v.Line, v.Column, v.Message, v.RuleID); err != nil {
			return err
		}
	}
	return nil
}

// RenderNDJSON writes one compact JSON object per line (newline-delimited JSON).
// (removed) legacy non-streaming NDJSON renderer; prefer NDJSONEventWriter in ndjson.go

// StreamNDJSON writes each ScanResult as a line as they are produced.
// Callers should provide an iterator/callback; this adapter simplifies usage.
// (removed) NDJSONStreamer; use NDJSONEventWriter for streaming pipelines
