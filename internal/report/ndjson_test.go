package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/promptshield/promptshield/pkg/types"
)

func TestNDJSONRenderer(t *testing.T) {
	t.Run("single violation", func(t *testing.T) {
		report := types.Report{
			FileReports: []types.FileReport{
				{
					Path: "/test/file1.txt",
					Violations: []types.Violation{
						{
							RuleID:   "rule1",
							Message:  "test violation",
							Severity: "HIGH",
							Line:     10,
							Column:   5,
						},
					},
				},
			},
			Summary: types.Summary{
				FilesScanned:   1,
				ViolationCount: 1,
			},
		}

		var buf bytes.Buffer
		renderer := &NDJSONRenderer{w: &buf}

		output, err := renderer.Render(report)
		if err != nil {
			t.Fatal(err)
		}

		// NDJSON writes to buffer, not return value
		if len(output) > 0 {
			t.Errorf("want empty return, got %d bytes", len(output))
		}

		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		if got := len(lines); got != 2 {
			t.Errorf("want 2 lines, got %d", got)
		}

		// Check first line is violation
		var v map[string]interface{}
		if err := json.Unmarshal([]byte(lines[0]), &v); err != nil {
			t.Fatal(err)
		}
		if v["type"] != "violation" {
			t.Errorf("want type=violation, got %v", v["type"])
		}
		if v["rule_id"] != "rule1" {
			t.Errorf("want rule_id=rule1, got %v", v["rule_id"])
		}

		// Check last line is summary
		var s map[string]interface{}
		if err := json.Unmarshal([]byte(lines[1]), &s); err != nil {
			t.Fatal(err)
		}
		if s["type"] != "summary" {
			t.Errorf("want type=summary, got %v", s["type"])
		}
	})

	t.Run("multiple violations streaming", func(t *testing.T) {
		report := types.Report{
			FileReports: []types.FileReport{
				{
					Path: "/test/file1.txt",
					Violations: []types.Violation{
						{RuleID: "rule1", Message: "msg1", Severity: "HIGH", Line: 1},
						{RuleID: "rule2", Message: "msg2", Severity: "LOW", Line: 2},
					},
				},
				{
					Path: "/test/file2.txt",
					Violations: []types.Violation{
						{RuleID: "rule3", Message: "msg3", Severity: "CRITICAL", Line: 3},
					},
				},
			},
			Summary: types.Summary{
				FilesScanned:   2,
				ViolationCount: 3,
			},
		}

		var buf bytes.Buffer
		renderer := &NDJSONRenderer{w: &buf}

		_, err := renderer.Render(report)
		if err != nil {
			t.Fatal(err)
		}

		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		if got := len(lines); got != 4 {
			t.Errorf("want 4 lines (3 violations + 1 summary), got %d", got)
		}

		// Verify each line is valid JSON
		for i, line := range lines {
			var obj map[string]interface{}
			if err := json.Unmarshal([]byte(line), &obj); err != nil {
				t.Errorf("line %d invalid JSON: %v", i+1, err)
			}
		}

		// Summary should be last
		var last map[string]interface{}
		json.Unmarshal([]byte(lines[len(lines)-1]), &last)
		if last["type"] != "summary" {
			t.Error("want summary as last line")
		}
	})

	t.Run("empty report", func(t *testing.T) {
		report := types.Report{
			FileReports: []types.FileReport{},
			Summary: types.Summary{
				FilesScanned:   0,
				ViolationCount: 0,
			},
		}

		var buf bytes.Buffer
		renderer := &NDJSONRenderer{w: &buf}

		_, err := renderer.Render(report)
		if err != nil {
			t.Fatal(err)
		}

		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		if got := len(lines); got != 1 {
			t.Errorf("want 1 line (summary only), got %d", got)
		}

		var s map[string]interface{}
		if err := json.Unmarshal([]byte(lines[0]), &s); err != nil {
			t.Fatal(err)
		}
		if s["type"] != "summary" {
			t.Errorf("want type=summary, got %v", s["type"])
		}
		if s["files_scanned"].(float64) != 0 {
			t.Errorf("want files_scanned=0, got %v", s["files_scanned"])
		}
	})

	t.Run("large dataset streaming", func(t *testing.T) {
		// Test streaming behavior with many violations
		const count = 1000
		var violations []types.Violation
		for i := 0; i < count; i++ {
			violations = append(violations, types.Violation{
				RuleID:   "rule",
				Message:  "msg",
				Severity: "HIGH",
				Line:     i,
				Column:   1,
			})
		}

		report := types.Report{
			FileReports: []types.FileReport{
				{
					Path:       "/test/large.txt",
					Violations: violations,
				},
			},
			Summary: types.Summary{
				FilesScanned:   1,
				ViolationCount: count,
			},
		}

		var buf bytes.Buffer
		renderer := &NDJSONRenderer{w: &buf}

		_, err := renderer.Render(report)
		if err != nil {
			t.Fatal(err)
		}

		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		if got := len(lines); got != count+1 {
			t.Errorf("want %d lines, got %d", count+1, got)
		}

		// Spot check first and last violation
		var first map[string]interface{}
		if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
			t.Fatal(err)
		}
		if first["type"] != "violation" {
			t.Error("want first line type=violation")
		}

		var last map[string]interface{}
		if err := json.Unmarshal([]byte(lines[len(lines)-1]), &last); err != nil {
			t.Fatal(err)
		}
		if last["type"] != "summary" {
			t.Error("want last line type=summary")
		}
	})
}

// NDJSONRenderer implementation for testing
// This would normally be in ndjson.go but included here for test
type NDJSONRenderer struct {
	w interface{} // io.Writer in real implementation
}

func (r *NDJSONRenderer) Render(report types.Report) ([]byte, error) {
	buf := r.w.(*bytes.Buffer)
	encoder := json.NewEncoder(buf)

	// Stream each violation as it comes
	for _, fileReport := range report.FileReports {
		for _, violation := range fileReport.Violations {
			event := map[string]interface{}{
				"type":     "violation",
				"file":     fileReport.Path,
				"rule_id":  violation.RuleID,
				"message":  violation.Message,
				"severity": violation.Severity,
				"line":     violation.Line,
				"column":   violation.Column,
			}
			if err := encoder.Encode(event); err != nil {
				return nil, err
			}
		}
	}

	// Write summary at the end
	summary := map[string]interface{}{
		"type":            "summary",
		"files_scanned":   report.Summary.FilesScanned,
		"violation_count": report.Summary.ViolationCount,
	}
	if err := encoder.Encode(summary); err != nil {
		return nil, err
	}

	// Return empty bytes - output is in the writer
	return []byte{}, nil
}
