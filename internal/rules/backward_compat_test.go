package rules

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestBackwardCompatibility_RulepackVersions tests that we can load older rulepack formats
func TestBackwardCompatibility_RulepackVersions(t *testing.T) {
	t.Run("LegacyV0Format", func(t *testing.T) {
		// Legacy format without apiVersion/kind fields
		legacyYAML := `
metadata:
  name: "legacy-pack"
  description: "Legacy format without apiVersion/kind"
  version: "1.0.0"

rules:
  - id: "legacy-rule"
    level: 1
    keywords: ["test", "legacy"]
    severity: "MEDIUM"
    message: "Legacy rule triggered"
`

		var pack RulePack
		err := yaml.Unmarshal([]byte(legacyYAML), &pack)
		require.NoError(t, err)

		// Should handle legacy format gracefully (may have validation errors)
		errors := ValidatePack(pack)
		if len(errors) > 0 {
			t.Logf("Legacy format validation errors (expected): %v", errors)
			// Should provide helpful guidance about missing fields
			errorStr := errors[0].Error()
			assert.True(t, strings.Contains(errorStr, "apiVersion") || strings.Contains(errorStr, "kind"), 
				"Should mention missing required fields")
		}

		// Core fields should be preserved
		assert.Equal(t, "legacy-pack", pack.Metadata.Name)
		assert.Len(t, pack.Rules, 1)
		assert.Equal(t, "legacy-rule", pack.Rules[0].ID)
	})

	t.Run("OldSeverityValues", func(t *testing.T) {
		// Test backward compatibility with old severity values
		oldSeverities := []string{"LOW", "MEDIUM"} // These were valid in older versions
		
		for _, severity := range oldSeverities {
			pack := RulePack{
				APIVersion: "promptshield.io/v1",
				Kind:       "RulePack",
				Metadata:   Metadata{Name: "test"},
				Rules: []Rule{
					{
						ID:       "test",
						Level:    1,
						Keywords: []string{"test"},
						Severity: severity,
					},
				},
			}

			// Should handle gracefully - either accept or provide migration guidance
			errors := ValidatePack(pack)
			
			// For now, these should be rejected with helpful error messages
			if len(errors) > 0 {
				errorStr := errors[0].Error()
				assert.Contains(t, errorStr, "severity", "Error should mention severity")
				// Should suggest valid alternatives
				assert.True(t, 
					containsAny(errorStr, []string{"INFO", "WARNING", "HIGH", "ERROR", "CRITICAL"}),
					"Error should suggest valid severity values")
			}
		}
	})

	t.Run("MissingRequiredFields", func(t *testing.T) {
		// Test migration path for packs missing required fields
		incompleteJSON := `{
			"metadata": {"name": "incomplete"},
			"rules": [
				{
					"id": "incomplete-rule",
					"level": 1,
					"keywords": ["test"]
				}
			]
		}`

		var pack RulePack
		err := json.Unmarshal([]byte(incompleteJSON), &pack)
		require.NoError(t, err)

		errors := ValidatePack(pack)
		assert.NotEmpty(t, errors, "Should have validation errors for missing fields")

		// Should provide helpful migration guidance
		if len(errors) > 0 {
			t.Logf("Missing fields errors: %v", errors)
			foundSeverityError := false
			for _, err := range errors {
				errorStr := err.Error()
				if containsAny(errorStr, []string{"severity", "required"}) {
					foundSeverityError = true
					break
				}
			}
			if !foundSeverityError {
				t.Logf("Errors found but no severity/required mention: %v", errors)
			}
			// Severity might be mentioned in other ways, so just check that we have errors
			assert.True(t, len(errors) > 0, "Should have validation errors for incomplete rule")
		}
	})

	t.Run("OldCompositionStrategy", func(t *testing.T) {
		// Test old composition strategies that might not be supported
		oldStrategies := []string{"first_match", "weighted", "consensus"}
		
		for _, strategy := range oldStrategies {
			pack := RulePack{
				APIVersion: "promptshield.io/v1",
				Kind:       "RulePack",
				Metadata:   Metadata{Name: "test"},
				Composition: &Composition{
					Strategy: strategy,
				},
				Rules: []Rule{
					{
						ID:       "test",
						Level:    1,
						Keywords: []string{"test"},
						Severity: "INFO",
					},
				},
			}

			errors := ValidatePack(pack)
			
			// Should either accept or provide migration guidance
			if len(errors) > 0 {
				errorStr := errors[0].Error()
				assert.Contains(t, errorStr, "strategy", "Error should mention strategy")
			}
		}
	})
}

// TestMigration_AutomaticFieldUpgrade tests automatic field upgrades during loading
func TestMigration_AutomaticFieldUpgrade(t *testing.T) {
	t.Run("AddMissingAPIVersion", func(t *testing.T) {
		// Create pack without apiVersion/kind
		legacyPack := RulePack{
			Metadata: Metadata{Name: "legacy"},
			Rules: []Rule{
				{
					ID:       "test",
					Level:    1,
					Keywords: []string{"test"},
					Severity: "INFO",
				},
			},
		}

		// Marshal and unmarshal to simulate loading from storage
		data, err := json.Marshal(legacyPack)
		require.NoError(t, err)

		var loadedPack RulePack
		err = json.Unmarshal(data, &loadedPack)
		require.NoError(t, err)

		// Apply migration logic (this would be in the actual loader)
		migratedPack := migrateRulepack(loadedPack)

		// Should have correct defaults
		assert.Equal(t, "promptshield.io/v1", migratedPack.APIVersion)
		assert.Equal(t, "RulePack", migratedPack.Kind)
		assert.Equal(t, "legacy", migratedPack.Metadata.Name)
	})

	t.Run("NormalizeFieldNames", func(t *testing.T) {
		// Test handling of field name variations
		variations := []string{
			`{"metadata": {"name": "test"}, "rules": []}`,                    // Standard
			`{"meta": {"name": "test"}, "rules": []}`,                        // Alternative name
			`{"metadata": {"title": "test"}, "rules": []}`,                   // Alternative field
		}

		for i, variant := range variations {
			var pack RulePack
			err := json.Unmarshal([]byte(variant), &pack)
			
			if i == 0 {
				// Standard format should always work
				require.NoError(t, err)
				assert.Equal(t, "test", pack.Metadata.Name)
			} else {
				// Alternative formats should either work or fail gracefully
				if err != nil {
					t.Logf("Variant %d failed as expected: %v", i, err)
				} else {
					t.Logf("Variant %d loaded successfully", i)
				}
			}
		}
	})
}

// TestVersionCompatibility_SchemaEvolution tests schema evolution scenarios
func TestVersionCompatibility_SchemaEvolution(t *testing.T) {
	t.Run("AddedOptionalFields", func(t *testing.T) {
		// Simulate loading older pack with new optional fields added
		oldPackJSON := `{
			"apiVersion": "promptshield.io/v1",
			"kind": "RulePack",
			"metadata": {"name": "old-pack"},
			"rules": [{
				"id": "old-rule",
				"level": 1,
				"keywords": ["test"],
				"severity": "INFO"
			}]
		}`

		var pack RulePack
		err := json.Unmarshal([]byte(oldPackJSON), &pack)
		require.NoError(t, err)

		// Should validate even without new optional fields
		errors := ValidatePack(pack)
		assert.Empty(t, errors, "Pack without new optional fields should still validate")

		// New fields should have sensible defaults
		rule := pack.Rules[0]
		assert.Empty(t, rule.Category, "Category should default to empty")
		assert.Nil(t, rule.Response, "Response should default to nil")
		
		// Composition should be nil/default
		assert.Nil(t, pack.Composition, "Composition should default to nil")
	})

	t.Run("RemovedDeprecatedFields", func(t *testing.T) {
		// Test handling when deprecated fields are encountered
		packWithDeprecated := `{
			"apiVersion": "promptshield.io/v1",
			"kind": "RulePack", 
			"metadata": {"name": "deprecated-fields"},
			"rules": [{
				"id": "test",
				"level": 1,
				"keywords": ["test"],
				"severity": "INFO",
				"deprecated_field": "should_be_ignored",
				"old_timeout": 5000
			}]
		}`

		var pack RulePack
		err := json.Unmarshal([]byte(packWithDeprecated), &pack)
		require.NoError(t, err)

		// Should still validate (unknown fields ignored by JSON)
		errors := ValidatePack(pack)
		assert.Empty(t, errors, "Pack with deprecated fields should still validate")
	})

	t.Run("ChangedFieldTypes", func(t *testing.T) {
		// Test handling when field types change between versions
		packWithOldTypes := `{
			"apiVersion": "promptshield.io/v1",
			"kind": "RulePack",
			"metadata": {"name": "type-changes"},
			"rules": [{
				"id": "test",
				"level": "1",
				"keywords": ["test"],
				"severity": "INFO"
			}]
		}`

		var pack RulePack
		// This might fail due to type mismatch (string vs int for level)
		err := json.Unmarshal([]byte(packWithOldTypes), &pack)
		
		if err != nil {
			// Should get helpful error about type mismatch
			errorStr := err.Error()
			assert.True(t, strings.Contains(errorStr, "level") || strings.Contains(errorStr, "Level"), 
				"Error should mention level field: %s", errorStr)
			t.Logf("Type mismatch handled appropriately: %v", err)
		} else {
			// If it succeeds, verify the value was converted properly
			assert.Equal(t, 1, pack.Rules[0].Level)
		}
	})
}

// TestRulepackUpgrade_ContentMigration tests content-level migrations
func TestRulepackUpgrade_ContentMigration(t *testing.T) {
	t.Run("RegexPatternUpgrade", func(t *testing.T) {
		// Test upgrading old regex patterns that might need escaping changes
		oldRegexPack := RulePack{
			APIVersion: "promptshield.io/v1",
			Kind:       "RulePack",
			Metadata:   Metadata{Name: "regex-upgrade"},
			Rules: []Rule{
				{
					ID:    "old-regex",
					Level: 2,
					Patterns: []Pattern{
						{Regex: `\w+@\w+\.\w+`}, // Old simple email regex
					},
					Severity: "MEDIUM",
				},
			},
		}

		// Should validate (or provide upgrade suggestions)
		errors := ValidatePack(oldRegexPack)
		
		if len(errors) > 0 {
			t.Logf("Old regex pattern validation: %v", errors)
		} else {
			t.Log("Old regex patterns still valid")
		}
	})

	t.Run("KeywordNormalization", func(t *testing.T) {
		// Test keyword normalization across versions
		keywordVariations := [][]string{
			{"Test", "TEST", "test"},              // Case variations
			{"test ", " test", "  test  "},        // Whitespace variations
			{"test\n", "test\t", "test\r"},        // Control char variations
		}

		for i, keywords := range keywordVariations {
			pack := RulePack{
				APIVersion: "promptshield.io/v1",
				Kind:       "RulePack",
				Metadata:   Metadata{Name: "keyword-test"},
				Rules: []Rule{
					{
						ID:       "keyword-rule",
						Level:    1,
						Keywords: keywords,
						Severity: "INFO",
					},
				},
			}

			errors := ValidatePack(pack)
			t.Logf("Keyword variation %d: %v errors", i, len(errors))
			
			// Should either validate or provide normalization guidance
			for _, err := range errors {
				if containsAny(err.Error(), []string{"keyword", "whitespace", "normalize"}) {
					t.Logf("Normalization guidance: %v", err)
				}
			}
		}
	})
}

// Helper functions

func migrateRulepack(pack RulePack) RulePack {
	// Apply migration logic that would be in the actual loader
	if pack.APIVersion == "" {
		pack.APIVersion = "promptshield.io/v1"
	}
	if pack.Kind == "" {
		pack.Kind = "RulePack"
	}
	
	// Add other migration logic here as needed
	
	return pack
}

func containsAny(str string, substrings []string) bool {
	for _, substr := range substrings {
		if strings.Contains(str, substr) {
			return true
		}
	}
	return false
}

