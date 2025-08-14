//go:build go1.18
// +build go1.18

package rules

import (
	"regexp"
	"testing"
)

func FuzzRegexValidation(f *testing.F) {
	// Add seed corpus
	f.Add(".*")
	f.Add("[a-z]+")
	f.Add("\\d{3}-\\d{2}-\\d{4}")
	f.Add("(?i)password")
	f.Add("^[a-zA-Z0-9]+$")
	f.Add("(group1)(group2)")
	f.Add("[^abc]")
	f.Add("a{2,5}")
	f.Add("(?:non-capturing)")
	f.Add("look(?=ahead)")

	f.Fuzz(func(t *testing.T, pattern string) {
		// Try to compile the regex
		_, err := regexp.Compile(pattern)

		// Create a rule with this pattern
		pack := RulePack{
			APIVersion: "promptshield.io/v1",
			Kind:       "RulePack",
			Metadata:   Metadata{Name: "fuzz-test"},
			Rules: []Rule{
				{
					ID:       "fuzz-regex",
					Level:    2,
					Severity: "ERROR",
					Patterns: []Pattern{{Regex: pattern}},
				},
			},
		}

		// Validate the pack
		errs := ValidatePack(pack)

		// If regexp.Compile failed, ValidatePack should also report an error
		if err != nil {
			if len(errs) == 0 {
				t.Errorf("ValidatePack did not catch invalid regex: %q, regexp error: %v", pattern, err)
			}
		} else {
			// If regex is valid, check that validation passes (except for other potential issues)
			for _, e := range errs {
				if contains(e.Error(), "regex") || contains(e.Error(), "pattern") {
					t.Errorf("ValidatePack reported regex error for valid pattern %q: %v", pattern, e)
				}
			}
		}
	})
}

func FuzzRegexFlags(f *testing.F) {
	// Add seed corpus for flags
	f.Add("i")
	f.Add("m")
	f.Add("s")
	f.Add("ignorecase")
	f.Add("multiline")
	f.Add("invalid")
	f.Add("im")
	f.Add("ims")
	f.Add("")

	f.Fuzz(func(t *testing.T, flagStr string) {
		// Create single-character flags from the string
		flags := []string{}
		for _, r := range flagStr {
			flags = append(flags, string(r))
		}

		if len(flags) == 0 && flagStr != "" {
			flags = []string{flagStr}
		}

		pack := RulePack{
			APIVersion: "promptshield.io/v1",
			Kind:       "RulePack",
			Metadata:   Metadata{Name: "fuzz-flags"},
			Rules: []Rule{
				{
					ID:       "fuzz-regex",
					Level:    2,
					Severity: "ERROR",
					Patterns: []Pattern{{Regex: "test", Flags: flags}},
				},
			},
		}

		// Validate should handle any flag input gracefully
		errs := ValidatePack(pack)

		// Check that invalid flags are caught
		validFlags := map[string]bool{
			"i":          true,
			"m":          true,
			"s":          true,
			"ignorecase": true,
			"multiline":  true,
			"dotall":     true,
		}

		hasInvalidFlag := false
		for _, flag := range flags {
			if !validFlags[flag] && flag != "" {
				hasInvalidFlag = true
				break
			}
		}

		if hasInvalidFlag {
			hasError := false
			for _, e := range errs {
				if contains(e.Error(), "flag") {
					hasError = true
					break
				}
			}
			if !hasError && len(flags) > 0 {
				t.Errorf("ValidatePack did not catch invalid flags: %v", flags)
			}
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && contains(s[1:], substr)
}
