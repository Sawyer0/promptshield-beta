package rules

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// joinErrs converts a slice of errors to a single space-joined string of their messages.
func joinErrs(errs []error) string {
	if len(errs) == 0 {
		return ""
	}
	msgs := make([]string, 0, len(errs))
	for _, e := range errs {
		msgs = append(msgs, e.Error())
	}
	return strings.Join(msgs, " ")
}

func TestValidatePack(t *testing.T) {
	tests := []struct {
		name        string
		pack        RulePack
		wantErr     bool
		errContains []string
	}{
		{
			name: "valid pack with all levels",
			pack: RulePack{
				APIVersion: "promptshield.io/v1",
				Kind:       "RulePack",
				Metadata:   Metadata{Name: "valid-pack", Version: "1.0.0"},
				Rules: []Rule{
					{ID: "kw1", Level: 1, Severity: "INFO", Keywords: []string{"password"}},
					{ID: "rx1", Level: 2, Severity: "ERROR", Patterns: []Pattern{{Regex: `\d{3}-\d{2}-\d{4}`}}},
					{ID: "sem1", Level: 3, Severity: "CRITICAL", Semantic: &Semantic{Model: "gpt-4", AnalysisPrompt: "Classify: {input}"}},
				},
			},
			wantErr: false,
		},

		{
			name: "missing required fields",
			pack: RulePack{
				APIVersion: "",
				Kind:       "",
				Metadata:   Metadata{Name: ""},
			},
			wantErr:     true,
			errContains: []string{"apiVersion", "kind", "metadata.name"},
		},
		{
			name: "duplicate rule IDs",
			pack: RulePack{
				APIVersion: "promptshield.io/v1",
				Kind:       "RulePack",
				Metadata:   Metadata{Name: "dup-test"},
				Rules: []Rule{
					{ID: "rule1", Level: 1, Keywords: []string{"test"}},
					{ID: "rule1", Level: 2, Patterns: []Pattern{{Regex: "test"}}},
				},
			},
			wantErr:     true,
			errContains: []string{"duplicate rule id"},
		},
		{
			name: "invalid regex pattern",
			pack: RulePack{
				APIVersion: "promptshield.io/v1",
				Kind:       "RulePack",
				Metadata:   Metadata{Name: "regex-test"},
				Rules: []Rule{
					{ID: "bad-regex", Level: 2, Patterns: []Pattern{{Regex: "("}}},
				},
			},
			wantErr:     true,
			errContains: []string{"regex error"},
		},
		{
			name: "level 1 missing keywords",
			pack: RulePack{
				APIVersion: "promptshield.io/v1",
				Kind:       "RulePack",
				Metadata:   Metadata{Name: "level1-test"},
				Rules: []Rule{
					{ID: "no-keywords", Level: 1, Severity: "ERROR"},
				},
			},
			wantErr:     true,
			errContains: []string{"level 1", "keywords"},
		},
		{
			name: "level 2 missing patterns",
			pack: RulePack{
				APIVersion: "promptshield.io/v1",
				Kind:       "RulePack",
				Metadata:   Metadata{Name: "level2-test"},
				Rules: []Rule{
					{ID: "no-patterns", Level: 2, Severity: "ERROR"},
				},
			},
			wantErr:     true,
			errContains: []string{"level 2", "patterns"},
		},
		{
			name: "level 3 missing semantic config",
			pack: RulePack{
				APIVersion: "promptshield.io/v1",
				Kind:       "RulePack",
				Metadata:   Metadata{Name: "level3-test"},
				Rules: []Rule{
					{ID: "no-semantic", Level: 3, Severity: "ERROR"},
				},
			},
			wantErr:     true,
			errContains: []string{"level 3", "semantic"},
		},
		{
			name: "invalid severity value",
			pack: RulePack{
				APIVersion: "promptshield.io/v1",
				Kind:       "RulePack",
				Metadata:   Metadata{Name: "severity-test"},
				Rules: []Rule{
					{ID: "bad-severity", Level: 1, Severity: "EXTREME", Keywords: []string{"test"}},
				},
			},
			wantErr:     true,
			errContains: []string{"severity"},
		},
		{
			name: "invalid regex flags",
			pack: RulePack{
				APIVersion: "promptshield.io/v1",
				Kind:       "RulePack",
				Metadata:   Metadata{Name: "flags-test"},
				Rules: []Rule{
					{ID: "bad-flags", Level: 2, Patterns: []Pattern{{Regex: "test", Flags: []string{"invalid"}}}},
				},
			},
			wantErr:     true,
			errContains: []string{"flag"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidatePack(tt.pack)

			if tt.wantErr {
				require.NotEmpty(t, errs)
				errStr := joinErrs(errs)
				for _, expected := range tt.errContains {
					assert.Contains(t, errStr, expected)
				}
			} else {
				require.Empty(t, errs)
			}
		})
	}
}

func TestValidatePack_Errors(t *testing.T) {
	p := RulePack{
		APIVersion: "promptshield.io/v1",
		Kind:       "RulePack",
		Metadata:   Metadata{Name: "p"},
		Rules: []Rule{
			{ID: "dup", Level: 1, Keywords: []string{"a"}},
			{ID: "dup", Level: 1, Keywords: []string{"a"}},
			{ID: "rx", Level: 2, Patterns: []Pattern{{Regex: "(", Flags: []string{"x"}}}},
		},
	}
	errs := ValidatePack(p)
	require.NotEmpty(t, errs)
}

func TestValidateRegexFlags(t *testing.T) {
	tests := []struct {
		name    string
		flags   []string
		wantErr bool
	}{
		{
			name:    "valid flags",
			flags:   []string{"i", "m"},
			wantErr: false,
		},
		{
			name:    "ignorecase flag",
			flags:   []string{"ignorecase"},
			wantErr: false,
		},
		{
			name:    "multiline flag",
			flags:   []string{"multiline"},
			wantErr: false,
		},
		{
			name:    "empty flags",
			flags:   []string{},
			wantErr: false,
		},
		{
			name:    "invalid flag",
			flags:   []string{"invalid"},
			wantErr: true,
		},
		{
			name:    "mixed valid and invalid",
			flags:   []string{"i", "invalid", "m"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pack := RulePack{
				APIVersion: "promptshield.io/v1",
				Kind:       "RulePack",
				Metadata:   Metadata{Name: "flag-test"},
				Rules: []Rule{
					{
						ID:       "test",
						Level:    2,
						Severity: "ERROR",
						Patterns: []Pattern{{Regex: "test", Flags: tt.flags}},
					},
				},
			}

			errs := ValidatePack(pack)
			if tt.wantErr {
				require.NotEmpty(t, errs)
				assert.Contains(t, joinErrs(errs), "flag")
			} else {
				require.Empty(t, errs)
			}
		})
	}
}

func TestValidateLevelFieldCombinations(t *testing.T) {
	tests := []struct {
		name    string
		rule    Rule
		wantErr bool
		errMsg  string
	}{
		{
			name: "level 1 with keywords only",
			rule: Rule{
				ID:       "test",
				Level:    1,
				Severity: "ERROR",
				Keywords: []string{"password", "secret"},
			},
			wantErr: false,
		},
		{
			name: "level 1 with patterns (allowed by current validator)",
			rule: Rule{
				ID:       "test",
				Level:    1,
				Severity: "ERROR",
				Keywords: []string{"password"},
				Patterns: []Pattern{{Regex: "test"}},
			},
			wantErr: false,
		},
		{
			name: "level 2 with patterns only",
			rule: Rule{
				ID:       "test",
				Level:    2,
				Severity: "ERROR",
				Patterns: []Pattern{{Regex: `\d+`}},
			},
			wantErr: false,
		},
		{
			name: "level 2 with keywords (allowed by current validator)",
			rule: Rule{
				ID:       "test",
				Level:    2,
				Severity: "ERROR",
				Keywords: []string{"test"},
				Patterns: []Pattern{{Regex: "test"}},
			},
			wantErr: false,
		},
		{
			name: "level 3 with semantic only",
			rule: Rule{
				ID:       "test",
				Level:    3,
				Severity: "ERROR",
				Semantic: &Semantic{
					Model:          "gpt-4",
					AnalysisPrompt: "Analyze for security issues",
				},
			},
			wantErr: false,
		},
		{
			name: "level 3 with keywords (allowed by current validator)",
			rule: Rule{
				ID:       "test",
				Level:    3,
				Severity: "ERROR",
				Keywords: []string{"test"},
				Semantic: &Semantic{Model: "gpt-4", AnalysisPrompt: "Analyze: {input}"},
			},
			wantErr: false,
		},
		{
			name: "empty keywords for level 1",
			rule: Rule{
				ID:       "test",
				Level:    1,
				Severity: "ERROR",
				Keywords: []string{},
			},
			wantErr: true,
			errMsg:  "requires non-empty keywords",
		},
		{
			name: "empty patterns for level 2",
			rule: Rule{
				ID:       "test",
				Level:    2,
				Severity: "ERROR",
				Patterns: []Pattern{},
			},
			wantErr: true,
			errMsg:  "requires at least one regex pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pack := RulePack{
				APIVersion: "promptshield.io/v1",
				Kind:       "RulePack",
				Metadata:   Metadata{Name: "test"},
				Rules:      []Rule{tt.rule},
			}

			errs := ValidatePack(pack)
			if tt.wantErr {
				require.NotEmpty(t, errs)
				if tt.errMsg != "" {
					assert.Contains(t, joinErrs(errs), tt.errMsg)
				}
			} else {
				require.Empty(t, errs)
			}
		})
	}
}
