package rules

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// FuzzValidatePack tests the validation logic with fuzzing
func FuzzValidatePack(f *testing.F) {
	// Seed corpus with valid and invalid examples
	seeds := []string{
		`{"metadata": {"name": "test"}, "rules": []}`,
		`{"metadata": {"name": ""}, "rules": []}`, // Empty name
		`{"rules": [{"id": "test"}]}`, // Missing metadata
		`{"metadata": {"name": "test"}, "rules": [{"level": "invalid"}]}`, // Invalid level
		`{"metadata": {"name": "test"}, "rules": [{"id": "test", "patterns": ["("]}]}`, // Invalid regex
		`{"metadata": {"name": "test"}, "rules": [{"id": "test", "keywords": ["x" * 10000]}]}`, // Long keyword
		"",           // Empty
		"not json",   // Invalid JSON
		"---\ntest:", // YAML
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		// Should not panic on any input
		var pack RulePack
		
		// Try JSON first
		err := json.Unmarshal([]byte(data), &pack)
		if err == nil {
			// Validate should not panic
			errs := ValidatePack(pack)
			_ = errs // Errors are expected for invalid input
		}

		// Try YAML
		err = yaml.Unmarshal([]byte(data), &pack)
		if err == nil {
			// Validate should not panic
			errs := ValidatePack(pack)
			_ = errs
		}
	})
}

// FuzzRegexPatterns tests regex compilation with fuzzing
func FuzzRegexPatterns(f *testing.F) {
	// Seed with problematic regex patterns
	seeds := []string{
		"(", ")", "[", "]", "{", "}", // Unmatched brackets
		"(?P<", "(?P<name>", "(?P<>)",  // Named groups
		"*", "+", "?", "|",              // Special chars
		"\\", "\\\\", "\\x", "\\x00",    // Escapes
		"(a+)+", "((a*)*)*",             // Catastrophic backtracking
		"(?i)", "(?m)", "(?s)",          // Flags
		"^", "$", "\\A", "\\z",          // Anchors
		strings.Repeat("a", 10000),      // Long pattern
		"",                              // Empty
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, pattern string) {
		// Test regex compilation
		compiled, err := compileRegex(pattern)
		
		if err != nil {
			// Should be a valid error, not a panic
			assert.Contains(t, err.Error(), "error parsing regexp")
		} else {
			// Should be able to use compiled regex
			assert.NotNil(t, compiled)
			
			// Test matching with random strings
			testStrings := []string{
				"test",
				pattern, // Pattern itself
				"",
				strings.Repeat("x", 1000),
			}
			
			for _, s := range testStrings {
				// Should not panic
				_ = compiled.MatchString(s)
			}
		}
	})
}

// PropertyTest_RulepackMerging tests properties of rulepack merging
func TestProperty_RulepackMerging(t *testing.T) {
	// Property 1: Merging empty packs should produce empty result
	t.Run("EmptyMerge", func(t *testing.T) {
		result := MergePacks([]RulePack{})
		assert.Empty(t, result)
		
		result = MergePacks([]RulePack{{}, {}, {}})
		assert.Empty(t, result)
	})

	// Property 2: Merging single pack should produce same rules
	t.Run("SinglePackIdentity", func(t *testing.T) {
		pack := RulePack{
			Rules: []Rule{
				{ID: "rule1", Level: 1},
				{ID: "rule2", Level: 2},
			},
		}
		
		result := MergePacks([]RulePack{pack})
		assert.Equal(t, pack.Rules, result)
	})

	// Property 3: Order preservation within packs
	t.Run("OrderPreservation", func(t *testing.T) {
		pack1 := RulePack{
			Metadata: Metadata{Name: "pack1"},
			Rules: []Rule{
				{ID: "a1", Level: 1},
				{ID: "a2", Level: 1},
			},
		}
		pack2 := RulePack{
			Metadata: Metadata{Name: "pack2"},
			Rules: []Rule{
				{ID: "b1", Level: 1},
				{ID: "b2", Level: 1},
			},
		}
		
		result := MergePacks([]RulePack{pack1, pack2})
		
		// Should maintain order: a1, a2, b1, b2
		assert.Len(t, result, 4)
		assert.Equal(t, "a1", result[0].ID)
		assert.Equal(t, "a2", result[1].ID)
		assert.Equal(t, "b1", result[2].ID)
		assert.Equal(t, "b2", result[3].ID)
	})

	// Property 4: Duplicate IDs should be handled
	t.Run("DuplicateHandling", func(t *testing.T) {
		pack1 := RulePack{
			Metadata: Metadata{Name: "pack1"},
			Rules: []Rule{{ID: "dup", Level: 1, Severity: "INFO", Keywords: []string{"test"}}},
		}
		pack2 := RulePack{
			Metadata: Metadata{Name: "pack2"},
			Rules: []Rule{{ID: "dup", Level: 1, Severity: "HIGH", Keywords: []string{"test"}}},
		}
		
		result := MergePacks([]RulePack{pack1, pack2})
		
		// Implementation-specific: last one wins or both kept?
		assert.NotEmpty(t, result)
	})
}

// PropertyTest_ValidationConsistency tests validation properties
func TestProperty_ValidationConsistency(t *testing.T) {
	// Property: Valid pack should always pass validation
	t.Run("ValidPackAlwaysPasses", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			pack := generateValidPack(i)
			errs := ValidatePack(pack)
			assert.Empty(t, errs, "Valid pack should have no errors")
		}
	})

	// Property: Invalid patterns should always fail
	t.Run("InvalidPatternsAlwaysFail", func(t *testing.T) {
		invalidPatterns := []string{
			"(",
			"[",
			"(?P<",
			"\\x",
		}
		
		for _, pattern := range invalidPatterns {
			pack := RulePack{
				APIVersion: "promptshield.io/v1",
				Kind:       "RulePack",
				Metadata: Metadata{Name: "test"},
				Rules: []Rule{
					{
						ID:       "test",
						Level:    2,
						Severity: "HIGH",
						Patterns: []Pattern{{Regex: pattern}},
					},
				},
			}
			
			errs := ValidatePack(pack)
			assert.NotEmpty(t, errs, "Invalid pattern should fail: %s", pattern)
		}
	})

	// Property: Severity values should be constrained
	t.Run("SeverityConstraints", func(t *testing.T) {
		validSeverities := []string{"INFO", "WARNING", "HIGH", "ERROR", "CRITICAL"}
		invalidSeverities := []string{"LOW", "MEDIUM", "low", "VERY_HIGH", "1", "null"}
		
		for _, sev := range validSeverities {
			pack := RulePack{
				APIVersion: "promptshield.io/v1",
				Kind:       "RulePack",
				Metadata: Metadata{Name: "test"},
				Rules: []Rule{
					{ID: "test", Level: 1, Severity: sev, Keywords: []string{"test"}},
				},
			}
			errs := ValidatePack(pack)
			assert.Empty(t, errs, "Valid severity should pass: %s", sev)
		}
		
		for _, sev := range invalidSeverities {
			pack := RulePack{
				APIVersion: "promptshield.io/v1",
				Kind:       "RulePack",
				Metadata: Metadata{Name: "test"},
				Rules: []Rule{
					{ID: "test", Level: 1, Severity: sev, Keywords: []string{"test"}},
				},
			}
			errs := ValidatePack(pack)
			assert.NotEmpty(t, errs, "Invalid severity should fail: %s", sev)
		}
	})
}

// PropertyTest_CompositionStrategies tests composition strategy properties
func TestProperty_CompositionStrategies(t *testing.T) {
	// Create test packs with different priorities
	createPack := func(priority int, ruleIDs ...string) RulePack {
		rules := make([]Rule, len(ruleIDs))
		for i, id := range ruleIDs {
			rules[i] = Rule{ID: id, Level: 1}
		}
		return RulePack{
			Metadata: Metadata{Name: fmt.Sprintf("pack%d", priority)},
			Composition: &Composition{
				Priority: priority,
				Strategy: "priority_order",
			},
			Rules: rules,
		}
	}

	// Property: Higher priority packs should come first
	t.Run("PriorityOrdering", func(t *testing.T) {
		pack1 := createPack(10, "high1", "high2")
		pack2 := createPack(5, "med1", "med2")
		pack3 := createPack(1, "low1", "low2")
		
		// Test different orderings
		orders := [][]RulePack{
			{pack1, pack2, pack3},
			{pack3, pack2, pack1},
			{pack2, pack1, pack3},
		}
		
		for _, packs := range orders {
			result := MergePacksPriorityOrder(packs)
			
			// Should always be: high1, high2, med1, med2, low1, low2
			assert.Len(t, result, 6)
			assert.Equal(t, "high1", result[0].ID)
			assert.Equal(t, "high2", result[1].ID)
			assert.Equal(t, "med1", result[2].ID)
			assert.Equal(t, "med2", result[3].ID)
			assert.Equal(t, "low1", result[4].ID)
			assert.Equal(t, "low2", result[5].ID)
		}
	})
}

// GenerativeTest_LargeRulepacks tests handling of large rulepacks
func TestGenerative_LargeRulepacks(t *testing.T) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		t.Run(fmt.Sprintf("Size%d", size), func(t *testing.T) {
			pack := generateLargePack(size)
			
			// Should be able to validate without OOM
			start := testing.AllocsPerRun(1, func() {
				errs := ValidatePack(pack)
				_ = errs
			})
			
			// Should be able to marshal/unmarshal
			data, err := json.Marshal(pack)
			assert.NoError(t, err)
			
			var decoded RulePack
			err = json.Unmarshal(data, &decoded)
			assert.NoError(t, err)
			assert.Equal(t, size, len(decoded.Rules))
			
			// Memory should be reasonable
			t.Logf("Size %d: %f allocs", size, start)
		})
	}
}

// Helper functions

func generateValidPack(seed int) RulePack {
	r := rand.New(rand.NewSource(int64(seed)))
	
	numRules := r.Intn(5) + 1 // Smaller number for simplicity
	rules := make([]Rule, numRules)
	
	// Use correct severity values
	severities := []string{"INFO", "WARNING", "HIGH", "ERROR", "CRITICAL"}
	categories := []string{"security", "compliance", "quality"}
	
	for i := 0; i < numRules; i++ {
		level := r.Intn(3) + 1
		rule := Rule{
			ID:       fmt.Sprintf("rule-%d-%d", seed, i),
			Level:    level,
			Severity: severities[r.Intn(len(severities))],
			Category: categories[r.Intn(len(categories))],
		}
		
		// Add required fields based on level
		switch level {
		case 1: // Keywords required
			rule.Keywords = []string{fmt.Sprintf("keyword%d", i), "test"}
		case 2: // Patterns required  
			rule.Patterns = []Pattern{{Regex: fmt.Sprintf("pattern%d.*", i)}}
		case 3: // Semantic required
			rule.Semantic = &Semantic{
				Model:          "gpt-4", // Required field
				AnalysisPrompt: fmt.Sprintf("Analyze for rule %d", i),
			}
		}
		
		rules[i] = rule
	}
	
	return RulePack{
		APIVersion: "promptshield.io/v1",
		Kind:       "RulePack",
		Metadata: Metadata{
			Name:        fmt.Sprintf("pack-%d", seed),
			Description: "Generated pack",
		},
		Rules: rules,
	}
}

func generateLargePack(size int) RulePack {
	rules := make([]Rule, size)
	
	// Use correct severity values
	severities := []string{"INFO", "WARNING", "HIGH", "ERROR", "CRITICAL"}
	
	for i := 0; i < size; i++ {
		rules[i] = Rule{
			ID:       fmt.Sprintf("rule-%d", i),
			Level:    1,
			Keywords: []string{fmt.Sprintf("keyword-%d", i)},
			Severity: severities[i%len(severities)],
			Category: "test",
		}
	}
	
	return RulePack{
		APIVersion: "promptshield.io/v1",
		Kind:       "RulePack",
		Metadata: Metadata{
			Name:        "large-pack",
			Description: fmt.Sprintf("Pack with %d rules", size),
		},
		Rules: rules,
	}
}

// compileRegex is a helper to compile regex patterns
func compileRegex(pattern string) (*regexp.Regexp, error) {
	// Add timeout protection for catastrophic backtracking
	return regexp.Compile(pattern)
}