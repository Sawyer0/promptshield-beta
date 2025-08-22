package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanResult_ShouldBlock(t *testing.T) {
	t.Run("ShouldBlockTrue", func(t *testing.T) {
		result := &ScanResult{
			ScanInfo: ScanInfo{
				ShouldBlock: true,
			},
		}
		assert.True(t, result.ShouldBlock())
	})

	t.Run("ShouldBlockFalse", func(t *testing.T) {
		result := &ScanResult{
			ScanInfo: ScanInfo{
				ShouldBlock: false,
			},
		}
		assert.False(t, result.ShouldBlock())
	})
}

func TestScanInfo_Serialization(t *testing.T) {
	scanInfo := ScanInfo{
		TotalViolations:   3,
		ScanStatus:        "success",
		ScanDurationMs:    150,
		RulesProcessed:    25,
		RulesSkipped:      2,
		RulesTimedOut:     1,
		Level1DurationMs:  50,
		Level2DurationMs:  75,
		Level3DurationMs:  25,
		CacheHits:         12,
		CacheMisses:       8,
		ShouldBlock:       true,
		BlockReason:       "high-severity-rule",
		HighestSeverity:   "CRITICAL",
		TriggerRuleCount:  3,
		PeakMemoryBytes:   1024 * 1024,
		CPUTimeMs:         145,
	}

	// Test JSON marshaling
	data, err := json.Marshal(scanInfo)
	require.NoError(t, err)

	// Test JSON unmarshaling
	var unmarshaled ScanInfo
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// Verify all fields are preserved
	assert.Equal(t, scanInfo.TotalViolations, unmarshaled.TotalViolations)
	assert.Equal(t, scanInfo.ScanStatus, unmarshaled.ScanStatus)
	assert.Equal(t, scanInfo.ScanDurationMs, unmarshaled.ScanDurationMs)
	assert.Equal(t, scanInfo.RulesProcessed, unmarshaled.RulesProcessed)
	assert.Equal(t, scanInfo.ShouldBlock, unmarshaled.ShouldBlock)
	assert.Equal(t, scanInfo.BlockReason, unmarshaled.BlockReason)
	assert.Equal(t, scanInfo.HighestSeverity, unmarshaled.HighestSeverity)
}

func TestScanResult_CompleteSerialization(t *testing.T) {
	result := ScanResult{
		Input: "test-input",
		Violations: []Violation{
			{
				RuleID:   "test-rule",
				Message:  "Test violation",
				Severity: "HIGH",
				Line:     1,
				Column:   10,
			},
		},
		Metrics: Metrics{
			BytesRead:        256,
			LinesRead:        5,
			RegexAttempts:    10,
			SemanticAttempts: 2,
		},
		ScanInfo: ScanInfo{
			TotalViolations:  1,
			ScanStatus:       "success",
			ScanDurationMs:   100,
			RulesProcessed:   15,
			ShouldBlock:      true,
			BlockReason:      "test-rule",
			HighestSeverity:  "HIGH",
			TriggerRuleCount: 1,
		},
		DurationMs: 100,
	}

	// Test JSON marshaling of complete ScanResult
	data, err := json.Marshal(result)
	require.NoError(t, err)

	// Test JSON unmarshaling
	var unmarshaled ScanResult
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// Verify structure integrity
	assert.Equal(t, result.Input, unmarshaled.Input)
	assert.Len(t, unmarshaled.Violations, 1)
	assert.Equal(t, result.Violations[0].RuleID, unmarshaled.Violations[0].RuleID)
	assert.Equal(t, result.ScanInfo.TotalViolations, unmarshaled.ScanInfo.TotalViolations)
	assert.Equal(t, result.ScanInfo.ShouldBlock, unmarshaled.ScanInfo.ShouldBlock)
	assert.True(t, unmarshaled.ShouldBlock(), "ShouldBlock() method should work after unmarshaling")
}

func TestScanInfo_DefaultValues(t *testing.T) {
	// Test zero values work correctly
	scanInfo := ScanInfo{}
	
	assert.Equal(t, 0, scanInfo.TotalViolations)
	assert.Equal(t, "", scanInfo.ScanStatus)
	assert.Equal(t, false, scanInfo.ShouldBlock)
	assert.Equal(t, "", scanInfo.BlockReason)
	assert.Equal(t, 0, scanInfo.TriggerRuleCount)

	// Test in ScanResult
	result := ScanResult{
		Input:    "test",
		ScanInfo: scanInfo,
	}
	assert.False(t, result.ShouldBlock())
}