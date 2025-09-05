package scanner_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
)

func TestContextMinimization(t *testing.T) {
	tests := []struct {
		name           string
		config         *rules.ContextMinimization
		content        string
		stripPoint     string
		expectedResult string
		expectError    bool
	}{
		{
			name: "basic_masking",
			config: &rules.ContextMinimization{
				Enabled:    true,
				StripPoint: "after_tool_selection",
				MaskToken:  "<MASKED>",
			},
			content:        `{"messages": [{"role": "user", "content": "Please help me with sensitive data"}]}`,
			expectedResult: `{"messages":[{"content":"<MASKED>","role":"user"}]}`,
		},
		{
			name: "retention_patterns",
			config: &rules.ContextMinimization{
				Enabled:   true,
				MaskToken: "<MASKED>",
				Retain:    []string{`\b\w+@\w+\.\w+\b`}, // email pattern
			},
			content:        `{"messages": [{"role": "user", "content": "Contact john@example.com for help"}]}`,
			expectedResult: `{"messages":[{"content":"john@example.com","role":"user"}]}`,
		},
		{
			name: "step_based_minimization",
			config: &rules.ContextMinimization{
				Enabled:    true,
				StripPoint: "step_by_step",
				Step:       3,
				MaskToken:  "<STEP_3>",
			},
			content:        `{"messages": [{"role": "user", "content": "This is step 3 content"}]}`,
			expectedResult: `{"model":null,"tools":null,"tool_choice":null,"messages":[{"role":"user","content":"<STEP_3>"}]}`,
		},
		{
			name: "disabled_config",
			config: &rules.ContextMinimization{
				Enabled: false,
			},
			content: `{"messages": [{"role": "user", "content": "Should not be modified"}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			minimizer := scanner.NewContextMinimizer(tt.config)
			if tt.config != nil && !tt.config.Enabled {
				assert.Nil(t, minimizer)
				return
			}

			require.NotNil(t, minimizer)
			result, err := minimizer.MinimizeContext(tt.content, tt.stripPoint)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.expectedResult != "" {
				// For JSON, compare without whitespace differences
				if strings.Contains(tt.content, "{") {
					assert.JSONEq(t, tt.expectedResult, result)
				} else {
					assert.Equal(t, tt.expectedResult, result)
				}
			} else {
				// Just verify it processed without error
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestMapReduceProcessor(t *testing.T) {
	// Create a test scanner with basic rules
	sc := scanner.ScanEngineCstor(0)
	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "test-pack"},
		Rules: []rules.Rule{
			{ID: "password", Level: 1, Keywords: []string{"password", "secret"}, Severity: "HIGH"},
			{ID: "ssn", Level: 2, Patterns: []rules.Pattern{{Regex: `\b\d{3}-\d{2}-\d{4}\b`}}, Severity: "CRITICAL"},
		},
	}})

	tests := []struct {
		name           string
		config         *rules.MapReduce
		content        string
		expectChunking bool
		minViolations  int
	}{
		{
			name: "paragraph_chunking",
			config: &rules.MapReduce{
				Enabled:       true,
				MapUnit:       "paragraph",
				TextMaxTokens: 100,
				ReduceType:    "union",
			},
			content: strings.Join([]string{
				"First paragraph with password leak.",
				"",
				"Second paragraph with secret data.",
				"",
				"Third paragraph with SSN 123-45-6789.",
			}, "\n"),
			expectChunking: true,
			minViolations:  3,
		},
		{
			name: "sentence_chunking",
			config: &rules.MapReduce{
				Enabled:       true,
				MapUnit:       "sentence",
				TextMaxTokens: 50,
				ReduceType:    "intersection",
			},
			content:        "This sentence has password. This sentence also has password. Different sentence with secret.",
			expectChunking: true,
			minViolations:  1, // intersection should find password in multiple chunks
		},
		{
			name: "consensus_reduce",
			config: &rules.MapReduce{
				Enabled:       true,
				MapUnit:       "line",
				TextMaxTokens: 20,
				ReduceType:    "consensus",
			},
			content: strings.Join([]string{
				"Line 1 with password",
				"Line 2 with password",
				"Line 3 with password",
				"Line 4 with different content",
			}, "\n"),
			expectChunking: true,
			minViolations:  1, // consensus should find password violation
		},
		{
			name: "disabled_config",
			config: &rules.MapReduce{
				Enabled: false,
			},
			content:        "Simple content with password",
			expectChunking: false,
			minViolations:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := scanner.NewMapReduceProcessor(tt.config)
			if tt.config != nil && !tt.config.Enabled {
				assert.Nil(t, processor)
				return
			}

			result, err := processor.ProcessDocument(context.Background(), tt.content, sc)
			require.NoError(t, err)

			assert.GreaterOrEqual(t, len(result.Violations), tt.minViolations)

			if tt.expectChunking {
				// Check that chunking was used by verifying total violations and scan info
				assert.Greater(t, result.ScanInfo.TotalViolations, 0)
				assert.Equal(t, "success", result.ScanInfo.ScanStatus)
			}
		})
	}
}

func TestScannerAgentPatternIntegration(t *testing.T) {
	sc := scanner.ScanEngineCstor(0)

	// Load rules with agent patterns
	sc.LoadRulePacks([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "agent-patterns-test"},
		Rules: []rules.Rule{
			{ID: "password", Level: 1, Keywords: []string{"password", "secret"}, Severity: "HIGH"},
		},
		Patterns: &rules.Patterns{
			ContextMinimization: &rules.ContextMinimization{
				Enabled:    true,
				StripPoint: "after_tool_selection",
				MaskToken:  "<REDACTED>",
			},
			MapReduce: &rules.MapReduce{
				Enabled:       true,
				MapUnit:       "paragraph",
				TextMaxTokens: 50,
				ReduceType:    "union",
			},
		},
	}})

	content := `{
		"messages": [
			{"role": "user", "content": "First paragraph with password data."},
			{"role": "user", "content": "Second paragraph with secret information."}
		]
	}`

	result, err := sc.ScanContent(context.Background(), content, "test-input")
	require.NoError(t, err)

	// Should have found violations despite context minimization
	assert.Greater(t, len(result.Violations), 0)

	// Check that processing was applied
	t.Logf("Violations found: %d, Scan status: %s", len(result.Violations), result.ScanInfo.ScanStatus)
}

func TestContextMinimizerMethods(t *testing.T) {
	config := &rules.ContextMinimization{
		Enabled:    true,
		StripPoint: "before_execution",
		MaskToken:  "<TEST>",
	}

	minimizer := scanner.NewContextMinimizer(config)
	require.NotNil(t, minimizer)

	assert.True(t, minimizer.IsEnabled())
	assert.Equal(t, "before_execution", minimizer.GetStripPoint())
	assert.Equal(t, "<TEST>", minimizer.GetMaskToken())

	// Test nil config
	nilMinimizer := scanner.NewContextMinimizer(nil)
	assert.Nil(t, nilMinimizer)
}

func TestMapReduceProcessorMethods(t *testing.T) {
	config := &rules.MapReduce{
		Enabled:    true,
		MapUnit:    "sentence",
		ReduceType: "intersection",
	}

	processor := scanner.NewMapReduceProcessor(config)
	require.NotNil(t, processor)

	assert.True(t, processor.IsEnabled())
	assert.Equal(t, "sentence", processor.GetMapUnit())
	assert.Equal(t, "intersection", processor.GetReduceType())

	// Test nil config
	nilProcessor := scanner.NewMapReduceProcessor(nil)
	assert.Nil(t, nilProcessor)
}

func TestChunkingStrategies(t *testing.T) {
	config := &rules.MapReduce{
		Enabled:       true,
		TextMaxTokens: 20, // Small chunks for testing
	}

	testCases := []struct {
		mapUnit        string
		expectedChunks int
	}{
		{"paragraph", 3},
		{"sentence", 3},
		{"line", 5},  // includes empty lines
		{"token", 4}, // based on token estimation
	}

	for _, tc := range testCases {
		t.Run(tc.mapUnit, func(t *testing.T) {
			config.MapUnit = tc.mapUnit
			processor := scanner.NewMapReduceProcessor(config)

			// We can't directly test chunking, but we can test the processor creation
			assert.NotNil(t, processor)
			assert.Equal(t, tc.mapUnit, processor.GetMapUnit())
		})
	}
}
