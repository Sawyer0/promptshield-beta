package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/promptshield/promptshield/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderStylish_NoIssues(t *testing.T) {
	res := types.ScanResult{Input: "sample.txt"}
	var buf bytes.Buffer
	require.NoError(t, RenderStylish(&buf, res))
	out := buf.String()
	require.Contains(t, out, "Input: sample.txt")
	require.Contains(t, out, "No issues found")
}

func TestRenderStylish_WithViolations(t *testing.T) {
	res := types.ScanResult{
		Input: "sample.txt",
		Violations: []types.Violation{
			{RuleID: "r1", Message: "m1", Severity: "WARNING", Line: 1, Column: 1},
			{RuleID: "r2", Message: "m2", Severity: "HIGH", Line: 2, Column: 3},
		},
	}
	var buf bytes.Buffer
	require.NoError(t, RenderStylish(&buf, res))
	out := buf.String()
	require.Contains(t, out, "[WARNING] sample.txt:1:1 m1 (r1)")
	require.Contains(t, out, "[HIGH] sample.txt:2:3 m2 (r2)")
}

func TestRenderJSON_RoundTrip(t *testing.T) {
	res := types.ScanResult{
		Input:      "f.txt",
		Violations: []types.Violation{{RuleID: "x", Message: "y", Severity: "INFO", Line: 1, Column: 2}},
		Metrics:    types.Metrics{BytesRead: 2, LinesRead: 1},
	}
	var buf bytes.Buffer
	require.NoError(t, RenderJSON(&buf, res))

	var got types.ScanResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Equal(t, res, got)
}

func TestRenderStylish(t *testing.T) {
	tests := []struct {
		name        string
		result      types.ScanResult
		contains    []string
		notContains []string
	}{
		{
			name: "no violations",
			result: types.ScanResult{
				Input: "clean.txt",
				Metrics: types.Metrics{
					BytesRead: 1024,
					LinesRead: 50,
				},
			},
			contains: []string{
				"Input: clean.txt",
				"No issues found",
			},
		},
		{
			name: "single violation",
			result: types.ScanResult{
				Input: "problem.txt",
				Violations: []types.Violation{
					{
						RuleID:   "secret-001",
						Message:  "Detected hardcoded secret",
						Severity: "CRITICAL",
						Line:     10,
						Column:   15,
					},
				},
			},
			contains: []string{
				"[CRITICAL] problem.txt:10:15 Detected hardcoded secret (secret-001)",
			},
		},
		{
			name: "multiple violations different severities",
			result: types.ScanResult{
				Input: "multi.txt",
				Violations: []types.Violation{
					{RuleID: "info-001", Message: "Info message", Severity: "INFO", Line: 1, Column: 1},
					{RuleID: "warn-001", Message: "Warning message", Severity: "WARNING", Line: 5, Column: 10},
					{RuleID: "high-001", Message: "High severity", Severity: "HIGH", Line: 20, Column: 3},
					{RuleID: "crit-001", Message: "Critical issue", Severity: "CRITICAL", Line: 100, Column: 50},
				},
			},
			contains: []string{
				"[INFO] multi.txt:1:1 Info message (info-001)",
				"[WARNING] multi.txt:5:10 Warning message (warn-001)",
				"[HIGH] multi.txt:20:3 High severity (high-001)",
				"[CRITICAL] multi.txt:100:50 Critical issue (crit-001)",
			},
		},
		{
			name: "violations with long messages",
			result: types.ScanResult{
				Input: "long.txt",
				Violations: []types.Violation{
					{
						RuleID:   "long-001",
						Message:  strings.Repeat("Very long message ", 20),
						Severity: "HIGH",
						Line:     1,
						Column:   1,
					},
				},
			},
			contains: []string{
				"[HIGH] long.txt:1:1",
				"(long-001)",
			},
		},
		{
			name: "metrics display",
			result: types.ScanResult{
				Input: "metrics.txt",
				Metrics: types.Metrics{
					BytesRead: 1048576, // 1MB
					LinesRead: 10000,
				},
			},
			contains: []string{
				"Input: metrics.txt",
				"No issues found",
			},
		},
		{
			name: "empty input filename",
			result: types.ScanResult{
				Input: "",
				Violations: []types.Violation{
					{RuleID: "test", Message: "test", Severity: "HIGH", Line: 1, Column: 1},
				},
			},
			contains: []string{
				"[HIGH] :1:1 test (test)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := RenderStylish(&buf, tt.result)
			require.NoError(t, err)

			output := buf.String()

			for _, expected := range tt.contains {
				assert.Contains(t, output, expected)
			}

			for _, notExpected := range tt.notContains {
				assert.NotContains(t, output, notExpected)
			}
		})
	}
}

func TestRenderJSON(t *testing.T) {
	tests := []struct {
		name   string
		result types.ScanResult
	}{
		{
			name: "empty result",
			result: types.ScanResult{
				Input:      "empty.txt",
				Violations: []types.Violation{},
				Metrics:    types.Metrics{},
			},
		},
		{
			name: "full result with all fields",
			result: types.ScanResult{
				Input: "full.txt",
				Violations: []types.Violation{
					{
						RuleID:   "test-001",
						Message:  "Test violation",
						Severity: "HIGH",
						Line:     42,
						Column:   10,
					},
					{
						RuleID:   "test-002",
						Message:  "Another violation",
						Severity: "WARNING",
						Line:     100,
						Column:   1,
					},
				},
				Metrics: types.Metrics{
					BytesRead: 2048,
					LinesRead: 100,
				},
			},
		},
		{
			name: "special characters in messages",
			result: types.ScanResult{
				Input: "special.txt",
				Violations: []types.Violation{
					{
						RuleID:   "special-001",
						Message:  `Message with "quotes" and \backslashes\`,
						Severity: "INFO",
						Line:     1,
						Column:   1,
					},
				},
			},
		},
		{
			name: "unicode in messages",
			result: types.ScanResult{
				Input: "unicode.txt",
				Violations: []types.Violation{
					{
						RuleID:   "unicode-001",
						Message:  "Message with 日本語 and émojis 🔒",
						Severity: "WARNING",
						Line:     1,
						Column:   1,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := RenderJSON(&buf, tt.result)
			require.NoError(t, err)

			// Verify valid JSON
			var parsed types.ScanResult
			err = json.Unmarshal(buf.Bytes(), &parsed)
			require.NoError(t, err)

			// Verify round-trip
			assert.Equal(t, tt.result.Input, parsed.Input)
			assert.Equal(t, len(tt.result.Violations), len(parsed.Violations))
			assert.Equal(t, tt.result.Metrics, parsed.Metrics)

			// Verify specific fields
			for i, v := range tt.result.Violations {
				assert.Equal(t, v.RuleID, parsed.Violations[i].RuleID)
				assert.Equal(t, v.Message, parsed.Violations[i].Message)
				assert.Equal(t, v.Severity, parsed.Violations[i].Severity)
				assert.Equal(t, v.Line, parsed.Violations[i].Line)
				assert.Equal(t, v.Column, parsed.Violations[i].Column)
			}
		})
	}
}

func TestRenderJSON_Ordering(t *testing.T) {
	// Test that JSON field ordering is consistent
	result := types.ScanResult{
		Input: "order.txt",
		Violations: []types.Violation{
			{RuleID: "z", Message: "last", Severity: "LOW", Line: 3, Column: 3},
			{RuleID: "a", Message: "first", Severity: "HIGH", Line: 1, Column: 1},
			{RuleID: "m", Message: "middle", Severity: "MEDIUM", Line: 2, Column: 2},
		},
		Metrics: types.Metrics{
			BytesRead: 100,
			LinesRead: 10,
		},
	}

	var buf1, buf2 bytes.Buffer
	require.NoError(t, RenderJSON(&buf1, result))
	require.NoError(t, RenderJSON(&buf2, result))

	// Output should be identical
	assert.Equal(t, buf1.String(), buf2.String())
}

func TestRenderStylish_ViolationOrdering(t *testing.T) {
	// Test that violations appear in the order they're provided
	result := types.ScanResult{
		Input: "order.txt",
		Violations: []types.Violation{
			{RuleID: "third", Message: "3", Severity: "LOW", Line: 30, Column: 1},
			{RuleID: "first", Message: "1", Severity: "HIGH", Line: 10, Column: 1},
			{RuleID: "second", Message: "2", Severity: "MEDIUM", Line: 20, Column: 1},
		},
	}

	var buf bytes.Buffer
	require.NoError(t, RenderStylish(&buf, result))
	output := buf.String()

	// Find positions of each violation in output
	pos1 := strings.Index(output, "(first)")
	pos2 := strings.Index(output, "(second)")
	pos3 := strings.Index(output, "(third)")

	// They should appear in the same order as in the input
	assert.True(t, pos3 < pos1, "third should appear before first")
	assert.True(t, pos1 < pos2, "first should appear before second")
}
