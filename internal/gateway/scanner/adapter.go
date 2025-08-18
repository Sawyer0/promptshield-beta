package scanner

import (
	"time"

	"github.com/google/uuid"
	scannerpkg "github.com/promptshield/promptshield/internal/scanner"
	sharedtypes "github.com/promptshield/promptshield/internal/shared/types"
	pkgtypes "github.com/promptshield/promptshield/pkg/types"
)

// ConvertScanResult converts a scanner result to a shared types result
func ConvertScanResult(scannerResult *pkgtypes.ScanResult, tenantID *uuid.UUID, requestID string) *sharedtypes.ScanResult {
	// Convert violations
	violations := make([]*sharedtypes.PolicyViolation, len(scannerResult.Violations))
	for i, v := range scannerResult.Violations {
		violations[i] = &sharedtypes.PolicyViolation{
			RuleID:              v.RuleID,
			Message:             v.Message,
			Severity:            v.Severity,
			Line:                v.Line,
			Column:              v.Column,
			Action:              "deny", // Default action for gateway
			Category:            v.Category,
			Confidence:          1.0, // High confidence for rule-based detection
			RuleTimeoutMs:       v.RuleTimeoutMs,
			ResponseAction:      v.ResponseAction,
			ResponseMessage:     v.ResponseMessage,
			ResponseReplacement: v.ResponseReplacement,
		}
	}

	// Convert metrics
	metrics := &sharedtypes.ScanMetrics{
		BytesRead:        scannerResult.Metrics.BytesRead,
		LinesRead:        scannerResult.Metrics.LinesRead,
		RegexAttempts:    scannerResult.Metrics.RegexAttempts,
		RegexSkipped:     scannerResult.Metrics.RegexSkipped,
		SemanticAttempts: scannerResult.Metrics.SemanticAttempts,
		SemanticSkipped:  scannerResult.Metrics.SemanticSkipped,
		ProcessingTime:   time.Duration(scannerResult.DurationMs) * time.Millisecond,
	}

	// Convert scan info
	scanInfo := &sharedtypes.ScanInfo{
		TotalViolations:  len(violations),
		ScanStatus:       "success",
		ScanDurationMs:   scannerResult.DurationMs,
		RulesProcessed:   len(scannerResult.Violations),
		ShouldBlock:      len(violations) > 0,
		TriggerRuleCount: len(violations),
	}

	// Determine highest severity and block reason
	if len(violations) > 0 {
		highestSeverity := "low"
		blockReason := violations[0].RuleID

		for _, v := range violations {
			if severityToLevel(v.Severity) > severityToLevel(highestSeverity) {
				highestSeverity = v.Severity
				blockReason = v.RuleID
			}
		}

		scanInfo.HighestSeverity = highestSeverity
		scanInfo.BlockReason = blockReason
	}

	// Create enforcement decision
	decision := &sharedtypes.EnforcementDecision{
		Allow:       len(violations) == 0,
		Reason:      scanInfo.BlockReason,
		Violations:  violations,
		ProcessedAt: time.Now(),
		Latency:     time.Duration(scannerResult.DurationMs) * time.Millisecond,
	}

	return &sharedtypes.ScanResult{
		ID:         uuid.New().String(),
		Input:      scannerResult.Input,
		Violations: violations,
		Metrics:    metrics,
		Decision:   decision,
		ScanInfo:   scanInfo,
		TenantID:   tenantID,
		RequestID:  requestID,
		CreatedAt:  time.Now(),
		Metadata: map[string]interface{}{
			"source":               "llm_gateway_scanner",
			"original_duration_ms": scannerResult.DurationMs,
		},
	}
}

// ConvertViolationSeverity converts string severity to ViolationSeverity
func ConvertViolationSeverity(severity string) sharedtypes.ViolationSeverity {
	switch severity {
	case "critical":
		return sharedtypes.SeverityCritical
	case "high":
		return sharedtypes.SeverityHigh
	case "medium":
		return sharedtypes.SeverityMedium
	case "low":
		return sharedtypes.SeverityLow
	default:
		return sharedtypes.SeverityMedium
	}
}

// severityToLevel converts severity string to numeric level for comparison
func severityToLevel(severity string) int {
	switch severity {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

// CreateGatewayScanner creates a scanner configured for LLM Gateway use
func CreateGatewayScanner() *scannerpkg.Scanner {
	scanner := scannerpkg.ScanEngineCstor(16 * 1024 * 1024) // 16MB buffer for large LLM requests

	// Configure for gateway performance
	scanner.SetBufferBytes(1024 * 1024)                  // 1MB line buffer
	scanner.SetMaxStreamBytes(50 * 1024 * 1024)          // 50MB max request size
	scanner.SetTotalScanBudget(10 * time.Second)         // 10s timeout
	scanner.SetMaxResidentMemoryBytes(500 * 1024 * 1024) // 500MB memory limit

	// Enable quarantine mode for gateway
	scanner.SetQuarantineOnTimeout(true)
	scanner.SetQuarantineOnError(true)

	return scanner
}
