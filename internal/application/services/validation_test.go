package services

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"github.com/promptshield/promptshield/internal/testutil/fixtures"
)

func TestValidationService_ValidateDSL(t *testing.T) {
	svc := &ValidationService{}

	tests := []struct {
		name        string
		rawDSL      []byte
		wantErr     bool
		errContains []string // Multiple error messages to check
	}{
		{
			name:    "TP-2.1 accepts valid JSON",
			rawDSL:  []byte(fixtures.ValidRulepackJSON),
			wantErr: false,
		},
		{
			name:    "TP-2.2 accepts valid YAML",
			rawDSL:  []byte(fixtures.ValidRulepackYAML),
			wantErr: false,
		},
		{
			name:        "TP-2.3 rejects invalid format",
			rawDSL:      []byte(fixtures.InvalidRulepackJSON),
			wantErr:     true,
			errContains: []string{"apiVersion", "metadata.name"},
		},
		{
			name:    "TP-2.4 aggregates schema errors",
			rawDSL:  []byte(fixtures.RulepackWithSchemaErrors),
			wantErr: true,
			errContains: []string{
				"metadata.name", // Required field validation
				"rules",         // Minimum rules validation
			},
		},
		{
			name:        "empty input",
			rawDSL:      []byte(""),
			wantErr:     true,
			errContains: []string{"apiVersion", "rules"},
		},
		{
			name:        "malformed JSON",
			rawDSL:      []byte("{not valid json}"),
			wantErr:     true,
			errContains: []string{"apiVersion", "rules"},
		},
		{
			name:        "malformed YAML",
			rawDSL:      []byte(":\n  - this is\n bad yaml"),
			wantErr:     true,
			errContains: []string{"invalid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidateDSL(context.Background(), tt.rawDSL)
			
			if tt.wantErr {
				require.Error(t, err)
				errStr := err.Error()
				for _, contains := range tt.errContains {
					assert.Contains(t, strings.ToLower(errStr), strings.ToLower(contains),
						"Error should contain '%s', got: %s", contains, errStr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidationService_NormalizeDSL(t *testing.T) {
	svc := &ValidationService{}

	tests := []struct {
		name        string
		rawDSL      []byte
		wantErr     bool
		checkResult func(t *testing.T, result []byte)
	}{
		{
			name:    "TP-2.5 YAML to JSON conversion",
			rawDSL:  []byte(fixtures.ValidRulepackYAML),
			wantErr: false,
			checkResult: func(t *testing.T, result []byte) {
				// Result should be valid JSON with capitalized field names
				assert.Contains(t, string(result), `"APIVersion"`)
				assert.Contains(t, string(result), `"promptshield.io/v1"`)
				assert.Contains(t, string(result), `"Kind"`)
				assert.Contains(t, string(result), `"RulePack"`)
				
				// Should preserve all fields
				assert.Contains(t, string(result), `"test-rule-1"`)
				assert.Contains(t, string(result), `"HIGH"`)
			},
		},
		{
			name:    "TP-2.6 JSON idempotent",
			rawDSL:  []byte(fixtures.ValidRulepackJSON),
			wantErr: false,
			checkResult: func(t *testing.T, result []byte) {
				// Normalized JSON should be structurally equivalent
				// Note: formatting may differ but content should be same
				assert.Contains(t, string(result), `"APIVersion"`)
				assert.Contains(t, string(result), `"test-rule-1"`)
			},
		},
		{
			name:    "minimal valid DSL",
			rawDSL:  []byte(fixtures.MinimalValidDSL),
			wantErr: false,
			checkResult: func(t *testing.T, result []byte) {
				// Should be valid JSON with expected structure
				assert.Contains(t, string(result), `"APIVersion"`)
				assert.Contains(t, string(result), `"Kind"`)
				assert.Contains(t, string(result), `"Rules"`)
			},
		},
		{
			name:        "invalid input",
			rawDSL:      []byte("not valid json or yaml at all"),
			wantErr:     true,
			checkResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.NormalizeDSL(context.Background(), tt.rawDSL)
			
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
				
				// Verify result is valid JSON
				var jsonCheck interface{}
				err := json.Unmarshal(result, &jsonCheck)
				assert.NoError(t, err, "Result should be valid JSON")
			}
		})
	}
}

func TestValidationService_NormalizeDSL_Deterministic(t *testing.T) {
	svc := &ValidationService{}
	
	// Test that normalization is deterministic
	inputs := [][]byte{
		[]byte(fixtures.ValidRulepackYAML),
		[]byte(fixtures.ValidRulepackJSON),
	}
	
	for _, input := range inputs {
		t.Run("deterministic output", func(t *testing.T) {
			result1, err1 := svc.NormalizeDSL(context.Background(), input)
			require.NoError(t, err1)
			
			result2, err2 := svc.NormalizeDSL(context.Background(), input)
			require.NoError(t, err2)
			
			// Multiple calls should produce identical results
			assert.Equal(t, result1, result2, "Normalization should be deterministic")
		})
	}
}

func TestValidationService_ConcurrentValidation(t *testing.T) {
	svc := &ValidationService{}
	
	// Test concurrent validation doesn't cause issues
	done := make(chan bool, 20)
	
	for i := 0; i < 10; i++ {
		go func() {
			err := svc.ValidateDSL(context.Background(), []byte(fixtures.ValidRulepackJSON))
			assert.NoError(t, err)
			done <- true
		}()
		
		go func() {
			_, err := svc.NormalizeDSL(context.Background(), []byte(fixtures.ValidRulepackYAML))
			assert.NoError(t, err)
			done <- true
		}()
	}
	
	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}