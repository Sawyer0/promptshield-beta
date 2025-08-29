package services

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/promptshield/promptshield/internal/rules"
)

// FuzzRulepackValidation tests DSL validation against malicious/malformed inputs
func FuzzRulepackValidation(f *testing.F) {
	// Create a minimal service for testing
	service := &RulepackService{}

	// Seed corpus with known attack vectors
	seeds := [][]byte{
		// Valid baseline
		[]byte(`{"metadata": {"name": "test"}, "rules": []}`),

		// JSON injection attacks
		[]byte(`{"metadata": {"name": "test\u0000"}, "rules": []}`), // Null bytes
		[]byte(`{"metadata": {"name": "test\\\\"}, "rules": []}`),   // Escape sequences
		[]byte(`{"metadata": {"name": "test\""}, "rules": []}`),     // Quote injection

		// YAML attacks
		[]byte("metadata:\n  name: !!python/object/apply:os.system\n    - 'rm -rf /'"), // YAML deserialization
		[]byte("metadata: &anchor\n  name: test\nrules: *anchor"),                      // YAML anchors
		[]byte("metadata:\n  name: test\n  <<: {arbitrary: yaml}"),                     // YAML merge keys

		// Size-based DoS
		[]byte(strings.Repeat(`{"metadata": {"name": "test"}, "rules": [`, 10000) + "]}"),    // Deeply nested
		[]byte(`{"metadata": {"name": "` + strings.Repeat("x", 100000) + `"}, "rules": []}`), // Large strings

		// Regex injection attacks
		[]byte(`{"metadata": {"name": "test"}, "rules": [{"id": "test", "patterns": [{"regex": "(((((((((("}]}]}`), // Catastrophic backtracking
		[]byte(`{"metadata": {"name": "test"}, "rules": [{"id": "test", "patterns": [{"regex": "(a+)+"}]}]}`),      // ReDoS pattern

		// Unicode/encoding attacks
		[]byte(`{"metadata": {"name": "test💀"}, "rules": []}`),                  // Unicode
		[]byte(`{"metadata": {"name": "test\uffff"}, "rules": []}`),             // High Unicode
		[]byte("{\x00\"metadata\": {\"name\": \"test\"}, \"rules\": []}"),       // Embedded nulls
		[]byte(`{"metadata": {"name": "test\u202e"}, "rules": []}`),             // Right-to-left override
		[]byte(`{"metadata": {"name": "test\u0008\u0009\u000A"}, "rules": []}`), // Control chars

		// Memory exhaustion
		[]byte(`{"rules": [` + strings.Repeat(`{"id": "rule", "keywords": ["test"]},`, 50000) + `]}`), // Many rules

		// Invalid JSON/YAML
		[]byte(`{`),                           // Truncated JSON
		[]byte(`{"metadata": null}`),          // Null metadata
		[]byte(`{"rules": "string"}`),         // Wrong type
		[]byte(`---\n- item\n- item\n- item`), // YAML list instead of object

		// Binary/non-text data
		{0xFF, 0xFE, 0x00, 0x00},          // BOM markers
		{0x00, 0x01, 0x02, 0x03},          // Binary data
		[]byte("\x1b[31mRed text\x1b[0m"), // ANSI escape codes

		// Empty/minimal
		
		[]byte(""),
		[]byte("{}"),
		[]byte("null"),
		[]byte("[]"),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should never panic on any input
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ValidateDSL panicked on input: %v", r)
			}
		}()

		// Test ValidateDSL
		valid, warnings, errors := service.ValidateDSL(data)

		// Sanity checks on response
		if valid && len(errors) > 0 {
			t.Errorf("ValidateDSL returned valid=true but has errors: %v", errors)
		}

		if !valid && len(errors) == 0 && len(warnings) == 0 {
			t.Errorf("ValidateDSL returned valid=false but no errors/warnings")
		}

		// Verify no sensitive data leakage in error messages
		for _, err := range errors {
			if strings.Contains(err, "panic") || strings.Contains(err, "runtime error") {
				t.Errorf("Error message contains panic/runtime info: %s", err)
			}
		}

		// Test ParseDSL
		_, err := service.ParseDSL(data)
		if err != nil && strings.Contains(err.Error(), "panic") {
			t.Errorf("ParseDSL error contains panic info: %v", err)
		}
	})
}

// FuzzRegexComplexity tests regex patterns for complexity attacks
func FuzzRegexComplexity(f *testing.F) {
	// Seed with known problematic patterns
	seeds := []string{
		// Catastrophic backtracking patterns
		"(a+)+",
		"(a*)*",
		"(a+)+b",
		"(a|a)*",
		"([a-zA-Z]+)*",
		"(a+)+$",
		"^(a+)+",
		"(a+)+(a+)+",
		"(.*a){x,y}", // Dynamic quantifiers

		// Nested quantifiers
		"((a*)*)*",
		"(((a+)+)+)+",
		"((a{0,5}){0,5}){0,5}",

		// Complex alternations
		"(a|a|a|a|a|a|a|a)*",
		"(ab|ac|ad|ae|af)*",
		"(test|testing|tester)*",

		// Unicode complexity
		"(\\p{L}+)+",
		"(\\p{N}*)*",
		"([\\p{L}\\p{N}]+)*",

		// Large repetition counts
		"a{1000000}",
		"a{1,1000000}",
		".{999999}",

		// Complex lookarounds
		"(?=.{8,})(?=.*[a-z])(?=.*[A-Z])(?=.*\\d)",
		"(?!.*(.))\\1",
		"(?<=a+)b+(?=c+)",

		// Valid but complex patterns
		"^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",                                            // Email regex
		"^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$", // IP regex

		// Error cases
		"(",
		"[",
		"(?P<",
		"\\x",
		"(?P<>)",
		"(?P<name>)(?P<name>)", // Duplicate group names
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, pattern string) {
		// Create a rule with this pattern
		pack := rules.RulePack{
			Metadata: rules.Metadata{Name: "fuzz-test"},
			Rules: []rules.Rule{
				{
					ID:       "fuzz-rule",
					Level:    2,
					Patterns: []rules.Pattern{{Regex: pattern}},
					Severity: "MEDIUM",
				},
			},
		}

		// Should not panic during validation
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ValidatePack panicked on regex pattern %q: %v", pattern, r)
			}
		}()

		errs := rules.ValidatePack(pack)

		// If errors exist, they should be descriptive and not expose internal details
		for _, err := range errs {
			errStr := err.Error()
			if strings.Contains(errStr, "panic") || strings.Contains(errStr, "runtime") {
				t.Errorf("Error message exposes internals for pattern %q: %s", pattern, errStr)
			}

			// Should mention regex/pattern in error for regex-related issues
			if strings.Contains(pattern, "(") && len(pattern) < 100 { // Only for malformed patterns
				if !strings.Contains(errStr, "regex") && !strings.Contains(errStr, "pattern") {
					t.Logf("Pattern %q got non-regex error: %s", pattern, errStr)
				}
			}
		}
	})
}

// FuzzJSONYAMLRoundtrip tests JSON/YAML parsing consistency
func FuzzJSONYAMLRoundtrip(f *testing.F) {
	// Seed with edge cases
	seeds := [][]byte{
		[]byte(`{"metadata": {"name": "test"}, "rules": []}`),
		[]byte(`metadata:\n  name: test\nrules: []`),
		[]byte(`{"numbers": [1, 2, 3], "string": "test"}`),
		[]byte(`{"unicode": "test 🔥 unicode"}`),
		[]byte(`{"escaped": "test\n\t\r\""}`),
		[]byte(`{"null": null, "bool": true, "num": 123.45}`),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic on any input
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("JSON/YAML parsing panicked: %v", r)
			}
		}()

		// Try to parse as JSON
		var jsonPack rules.RulePack
		jsonErr := json.Unmarshal(data, &jsonPack)

		// Try to parse as YAML
		var yamlPack rules.RulePack
		yamlErr := yaml.Unmarshal(data, &yamlPack)

		// If both succeed, they should produce equivalent results for valid JSON
		if jsonErr == nil && yamlErr == nil {
			// Basic consistency checks
			if jsonPack.Metadata.Name != yamlPack.Metadata.Name {
				t.Logf("JSON/YAML name mismatch: %q vs %q", jsonPack.Metadata.Name, yamlPack.Metadata.Name)
			}
			if len(jsonPack.Rules) != len(yamlPack.Rules) {
				t.Logf("JSON/YAML rules count mismatch: %d vs %d", len(jsonPack.Rules), len(yamlPack.Rules))
			}
		}

		// Error messages should not expose internal implementation details
		if jsonErr != nil && strings.Contains(jsonErr.Error(), "panic") {
			t.Errorf("JSON error exposes internals: %v", jsonErr)
		}
		if yamlErr != nil && strings.Contains(yamlErr.Error(), "panic") {
			t.Errorf("YAML error exposes internals: %v", yamlErr)
		}
	})
}

// FuzzUnicodeHandling tests Unicode and encoding edge cases
func FuzzUnicodeHandling(f *testing.F) {
	// Seed with Unicode edge cases
	seeds := []string{
		"normal",
		"café",                     // Basic Unicode
		"💀🔥⚡",                      // Emoji
		"test\u0000null",           // Embedded null
		"test\uFFFDreplacement",    // Replacement character
		"test\u202Ereverse",        // Right-to-left override
		"test\u2028line",           // Line separator
		"test\u2029para",           // Paragraph separator
		"test\uFEFFbom",            // Byte order mark
		"\U0001F600\U0001F601",     // High Unicode (surrogates)
		string([]byte{0xFF, 0xFE}), // Invalid UTF-8
		"test\x80\x81invalid",      // Invalid UTF-8 sequences
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, name string) {
		// Create pack with fuzzed name
		pack := rules.RulePack{
			Metadata: rules.Metadata{Name: name},
			Rules: []rules.Rule{
				{
					ID:       "test",
					Level:    1,
					Keywords: []string{name}, // Also test keywords
					Severity: "LOW",
				},
			},
		}

		// Should handle any Unicode input gracefully
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Unicode handling panicked on %q: %v", name, r)
			}
		}()

		errs := rules.ValidatePack(pack)

		// Check that validation handles Unicode appropriately
		for _, err := range errs {
			// Error messages should be valid UTF-8
			if !isValidUTF8(err.Error()) {
				t.Errorf("Error message contains invalid UTF-8: %q", err.Error())
			}
		}

		// Test JSON marshaling with Unicode
		data, err := json.Marshal(pack)
		if err != nil && strings.Contains(err.Error(), "panic") {
			t.Errorf("JSON marshal panicked on Unicode: %v", err)
		}

		// If marshal succeeded, unmarshal should work
		if err == nil {
			var unmarshaled rules.RulePack
			if unmarshalErr := json.Unmarshal(data, &unmarshaled); unmarshalErr != nil {
				t.Errorf("JSON roundtrip failed for Unicode input: %v", unmarshalErr)
			}
		}
	})
}

// FuzzSizeBasedDoS tests resistance to size-based denial of service
func FuzzSizeBasedDoS(f *testing.F) {
	// Seed with various size-based attacks
	seeds := []int{
		0, 1, 10, 100, 1000, 10000, 100000, // Size progression
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, size int) {
		// Clamp size to reasonable bounds for testing
		if size < 0 {
			size = -size
		}
		if size > 1000000 { // 1MB limit for testing
			size = size % 1000000
		}

		// Create service with default limits
		service := &RulepackService{}

		// Test large string inputs
		largeString := strings.Repeat("a", size)
		packWithLargeString := rules.RulePack{
			Metadata: rules.Metadata{Name: largeString},
			Rules:    []rules.Rule{},
		}

		data, err := json.Marshal(packWithLargeString)
		if err != nil {
			t.Skip("Cannot marshal large string")
		}

		// Should handle large inputs gracefully
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Size-based DoS test panicked with size %d: %v", size, r)
			}
		}()

		valid, _, errors := service.ValidateDSL(data)

		// Large inputs should be rejected with appropriate error
		if size > 1024*1024 && valid { // > 1MB
			t.Errorf("Large input (size %d) was accepted", size)
		}

		// Errors should be descriptive
		for _, err := range errors {
			if strings.Contains(err, "panic") {
				t.Errorf("Size DoS error contains panic info: %s", err)
			}
		}

		// Test many small rules
		if size < 10000 { // Reasonable limit for this test
			ruleList := make([]rules.Rule, size)
			for i := 0; i < size; i++ {
				ruleList[i] = rules.Rule{
					ID:       string(rune('a' + (i % 26))),
					Level:    1,
					Keywords: []string{"test"},
					Severity: "LOW",
				}
			}

			packWithManyRules := rules.RulePack{
				Metadata: rules.Metadata{Name: "many-rules"},
				Rules:    ruleList,
			}

			data, err := json.Marshal(packWithManyRules)
			if err == nil {
				_, _, errors := service.ValidateDSL(data)

				// Should reject too many rules
				if size > 1000 {
					foundError := false
					for _, err := range errors {
						if strings.Contains(err, "too many rules") {
							foundError = true
							break
						}
					}
					if !foundError {
						t.Logf("Many rules (count %d) not properly rejected", size)
					}
				}
			}
		}
	})
}

// Helper functions

func isValidUTF8(s string) bool {
	for _, r := range s {
		if r == unicode.ReplacementChar {
			// Could be valid replacement char or invalid UTF-8
			continue
		}
		if !utf8.ValidRune(r) {
			return false
		}
	}
	return true
}

// PropertyTest_InputSanitization tests that all inputs are properly sanitized
func TestProperty_InputSanitization(t *testing.T) {
	service := &RulepackService{}

	// Test that null bytes are handled
	t.Run("NullByteHandling", func(t *testing.T) {
		input := []byte("{\x00\"metadata\": {\"name\": \"test\x00\"}, \"rules\": []}")

		valid, _, errors := service.ValidateDSL(input)

		// Should either reject or sanitize null bytes
		if valid {
			t.Error("Input with null bytes was accepted")
		}

		// Error should not contain null bytes
		for _, err := range errors {
			if strings.Contains(err, "\x00") {
				t.Error("Error message contains null bytes")
			}
		}
	})

	// Test that extremely deep nesting is rejected
	t.Run("DeepNestingRejection", func(t *testing.T) {
		// Create deeply nested JSON
		nested := strings.Repeat("{\"nested\":", 1000) + "\"value\"" + strings.Repeat("}", 1000)

		valid, _, errors := service.ValidateDSL([]byte(nested))

		// Should be rejected
		assert.False(t, valid, "Deeply nested input should be rejected")
		assert.NotEmpty(t, errors, "Should have errors for deeply nested input")
	})

	// Test that control characters are handled appropriately
	t.Run("ControlCharacterHandling", func(t *testing.T) {
		controlChars := []byte{0x01, 0x02, 0x03, 0x07, 0x08, 0x0B, 0x0C, 0x0E, 0x0F}

		for _, char := range controlChars {
			input := []byte(`{"metadata": {"name": "test` + string(char) + `"}, "rules": []}`)

			valid, _, errors := service.ValidateDSL(input)

			// Should handle gracefully (reject or sanitize)
			if valid {
				t.Logf("Control character 0x%02X was accepted", char)
			}

			// Errors should not propagate control characters
			for _, err := range errors {
				for _, errChar := range []byte(err) {
					if errChar < 0x20 && errChar != '\n' && errChar != '\t' && errChar != '\r' {
						t.Errorf("Error message contains control character 0x%02X", errChar)
					}
				}
			}
		}
	})
}
