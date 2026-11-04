package processors

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/promptshield/promptshield/internal/jobs"
	"github.com/promptshield/promptshield/internal/scanner"
	"github.com/promptshield/promptshield/pkg/types"
)

// ScanProcessor handles asynchronous scan jobs
type ScanProcessor struct {
	scanner *scanner.Scanner
}

// NewScanProcessor creates a new scan processor with the provided scanner
func NewScanProcessor(sc *scanner.Scanner) *ScanProcessor {
	return &ScanProcessor{
		scanner: sc,
	}
}

// Type returns the job type this processor handles
func (p *ScanProcessor) Type() string {
	return "scan"
}

// Process executes the scan job
func (p *ScanProcessor) Process(ctx context.Context, job *jobs.Job) error {
	// Parse scan options from metadata
	options := parseScanOptions(job.Metadata)
	
	// Determine input name for the scan
	inputName := "async-job"
	if name, ok := job.Metadata["input_name"].(string); ok {
		inputName = name
	}
	
	// Create a reader from the job input
	reader := strings.NewReader(string(job.Input))
	
	// Update progress to indicate scanning started
	job.Progress = 10
	
	// Perform the scan
	result, err := p.scanner.ScanReader(ctx, reader, inputName)
	if err != nil {
		return err
	}
	
	// Update progress
	job.Progress = 80
	
	// Format the result based on requested format
	formattedResult, err := formatScanResult(result, options.OutputFormat)
	if err != nil {
		return err
	}
	
	// Store the result
	job.Result = json.RawMessage(formattedResult)
	job.Progress = 100
	
	return nil
}

// ScanOptions contains options for scan jobs
type ScanOptions struct {
	OutputFormat string `json:"output_format"`
	FailOn       string `json:"fail_on"`
}

// parseScanOptions extracts scan options from job metadata
func parseScanOptions(metadata map[string]interface{}) ScanOptions {
	options := ScanOptions{
		OutputFormat: "json", // default
		FailOn:       "",     // default
	}
	
	if format, ok := metadata["output_format"].(string); ok {
		options.OutputFormat = format
	}
	
	if failOn, ok := metadata["fail_on"].(string); ok {
		options.FailOn = failOn
	}
	
	return options
}

// formatScanResult formats the scan result according to the requested output format
func formatScanResult(result types.ScanResult, format string) ([]byte, error) {
	switch strings.ToLower(format) {
	case "json":
		return json.Marshal(map[string]interface{}{
			"violations": result.Violations,
			"metrics":    result.Metrics,
			"input":      result.Input,
		})
	case "ndjson":
		// For async jobs, return violations as NDJSON lines
		var lines []string
		for _, v := range result.Violations {
			if b, err := json.Marshal(v); err == nil {
				lines = append(lines, string(b))
			}
		}
		return []byte(strings.Join(lines, "\n")), nil
	case "simple":
		// Simple text format
		if len(result.Violations) == 0 {
			return []byte("No violations found"), nil
		}
		var lines []string
		for _, v := range result.Violations {
			lines = append(lines, v.RuleID+": "+v.Message)
		}
		return []byte(strings.Join(lines, "\n")), nil
	default:
		// Default to JSON
		return json.Marshal(result)
	}
}