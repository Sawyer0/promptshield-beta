// +build integration

package scan_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"github.com/promptshield/promptshield/internal/application/scan"
	"github.com/promptshield/promptshield/internal/config"
	"github.com/promptshield/promptshield/internal/scanner"
	"github.com/promptshield/promptshield/internal/testutil/fixtures"
)

// TestIntegration_EndToEndScanning tests the complete flow from rulepack to scan results
func TestIntegration_EndToEndScanning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	ctx := context.Background()
	tempDir := t.TempDir()
	
	// Create a rulepack file
	rulepackPath := filepath.Join(tempDir, "test-rulepack.json")
	err := ioutil.WriteFile(rulepackPath, []byte(fixtures.ValidRulepackJSON), 0644)
	require.NoError(t, err)
	
	// Create test content with violations
	contentPath := filepath.Join(tempDir, "test-content.txt")
	err = ioutil.WriteFile(contentPath, []byte(fixtures.TextWithViolations), 0644)
	require.NoError(t, err)
	
	// Initialize real scanner
	scannerInstance, err := scanner.New()
	require.NoError(t, err)
	
	// Create scan service with real scanner
	cfg := &config.Config{
		Performance: config.Performance{
			Workers:          2,
			RuleTimeout:      "100ms",
			TotalScanTimeout: "5s",
		},
	}
	
	scanSvc := &scan.Service{
		scanner: scannerInstance,
		config:  cfg,
	}
	
	// Execute scan
	results, err := scanSvc.Scan(ctx, []string{contentPath}, []string{rulepackPath}, scan.Options{
		Workers: 2,
	})
	require.NoError(t, err)
	
	// Verify results
	assert.Len(t, results, 1)
	result := results[0]
	
	// Should detect violations from our test content
	assert.Greater(t, len(result.Violations), 0, "Should detect violations")
	
	// Verify metrics
	assert.Greater(t, result.Metrics.BytesRead, int64(0))
	assert.Greater(t, result.Metrics.LinesRead, int64(0))
}

// TestIntegration_StreamingLargeFile tests streaming scan of large files
func TestIntegration_StreamingLargeFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	ctx := context.Background()
	tempDir := t.TempDir()
	
	// Create a moderately sized file (1MB for testing)
	largePath := filepath.Join(tempDir, "large.txt")
	largeContent := fixtures.GenerateLargeContent(1) // 1MB
	err := ioutil.WriteFile(largePath, []byte(largeContent), 0644)
	require.NoError(t, err)
	
	// Create simple rulepack
	rulepackPath := filepath.Join(tempDir, "simple.yaml")
	simpleRulepack := `
apiVersion: promptshield.io/v1
kind: RulePack
spec:
  rules:
    - id: lorem-detector
      name: Lorem Detector
      level: 1
      severity: LOW
      keywords: ["Lorem", "ipsum"]
`
	err = ioutil.WriteFile(rulepackPath, []byte(simpleRulepack), 0644)
	require.NoError(t, err)
	
	// Initialize scanner
	scannerInstance, err := scanner.New()
	require.NoError(t, err)
	
	cfg := &config.Config{
		Performance: config.Performance{
			Workers:          4,
			TotalScanTimeout: "30s",
		},
	}
	
	scanSvc := &scan.Service{
		scanner: scannerInstance,
		config:  cfg,
	}
	
	// Test streaming mode
	startTime := time.Now()
	results, err := scanSvc.Scan(ctx, []string{largePath}, []string{rulepackPath}, scan.Options{
		Workers: 4,
		Stream:  true,
	})
	duration := time.Since(startTime)
	
	require.NoError(t, err)
	assert.Len(t, results, 1)
	
	// Should process in reasonable time
	assert.Less(t, duration, 5*time.Second, "Should process 1MB file quickly")
	
	// Should detect Lorem patterns
	assert.Greater(t, len(results[0].Violations), 0, "Should find Lorem patterns")
	assert.Equal(t, int64(1024*1024), results[0].Metrics.BytesRead)
}

// TestIntegration_ConcurrentScanning tests concurrent scan operations
func TestIntegration_ConcurrentScanning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	ctx := context.Background()
	tempDir := t.TempDir()
	
	// Create multiple test files
	numFiles := 10
	var filePaths []string
	for i := 0; i < numFiles; i++ {
		path := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
		content := fmt.Sprintf("File %d content with test keyword", i)
		err := ioutil.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)
		filePaths = append(filePaths, path)
	}
	
	// Create rulepack
	rulepackPath := filepath.Join(tempDir, "concurrent.yaml")
	rulepack := `
apiVersion: promptshield.io/v1
kind: RulePack
spec:
  rules:
    - id: keyword-detector
      level: 1
      severity: LOW
      keywords: ["test", "keyword"]
`
	err := ioutil.WriteFile(rulepackPath, []byte(rulepack), 0644)
	require.NoError(t, err)
	
	// Initialize scanner
	scannerInstance, err := scanner.New()
	require.NoError(t, err)
	
	cfg := &config.Config{
		Performance: config.Performance{
			Workers: 5,
		},
	}
	
	scanSvc := &scan.Service{
		scanner: scannerInstance,
		config:  cfg,
	}
	
	// Run concurrent scans
	done := make(chan bool, numFiles)
	errors := make(chan error, numFiles)
	
	for _, path := range filePaths {
		go func(p string) {
			_, err := scanSvc.Scan(ctx, []string{p}, []string{rulepackPath}, scan.Options{
				Workers: 1,
			})
			if err != nil {
				errors <- err
			}
			done <- true
		}(path)
	}
	
	// Wait for all scans
	for i := 0; i < numFiles; i++ {
		<-done
	}
	
	// Check for errors
	select {
	case err := <-errors:
		t.Fatalf("Concurrent scan failed: %v", err)
	default:
		// No errors
	}
}

// TestIntegration_MultipleRulepacks tests scanning with multiple rulepack files
func TestIntegration_MultipleRulepacks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	ctx := context.Background()
	tempDir := t.TempDir()
	
	// Create multiple rulepacks
	rulepack1Path := filepath.Join(tempDir, "rules1.yaml")
	rulepack1 := `
apiVersion: promptshield.io/v1
kind: RulePack
spec:
  rules:
    - id: rule-from-pack1
      level: 1
      severity: HIGH
      keywords: ["dangerous"]
`
	err := ioutil.WriteFile(rulepack1Path, []byte(rulepack1), 0644)
	require.NoError(t, err)
	
	rulepack2Path := filepath.Join(tempDir, "rules2.yaml")
	rulepack2 := `
apiVersion: promptshield.io/v1
kind: RulePack
spec:
  rules:
    - id: rule-from-pack2
      level: 1
      severity: MEDIUM
      keywords: ["warning"]
`
	err = ioutil.WriteFile(rulepack2Path, []byte(rulepack2), 0644)
	require.NoError(t, err)
	
	// Create content that triggers both rulepacks
	contentPath := filepath.Join(tempDir, "multi-violation.txt")
	content := "This contains dangerous content and warning signs"
	err = ioutil.WriteFile(contentPath, []byte(content), 0644)
	require.NoError(t, err)
	
	// Initialize scanner
	scannerInstance, err := scanner.New()
	require.NoError(t, err)
	
	cfg := &config.Config{}
	scanSvc := &scan.Service{
		scanner: scannerInstance,
		config:  cfg,
	}
	
	// Scan with both rulepacks
	results, err := scanSvc.Scan(ctx, 
		[]string{contentPath}, 
		[]string{rulepack1Path, rulepack2Path}, 
		scan.Options{},
	)
	require.NoError(t, err)
	
	// Should detect violations from both rulepacks
	assert.Len(t, results, 1)
	violations := results[0].Violations
	
	// Check that we have violations from both packs
	var hasRule1, hasRule2 bool
	for _, v := range violations {
		if v.RuleID == "rule-from-pack1" {
			hasRule1 = true
			assert.Equal(t, "HIGH", v.Severity)
		}
		if v.RuleID == "rule-from-pack2" {
			hasRule2 = true
			assert.Equal(t, "MEDIUM", v.Severity)
		}
	}
	
	assert.True(t, hasRule1, "Should have violation from pack 1")
	assert.True(t, hasRule2, "Should have violation from pack 2")
}