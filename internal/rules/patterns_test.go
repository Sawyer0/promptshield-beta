package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadPacks_RejectsUncompilableRegex(t *testing.T) {
	dir := t.TempDir()
	yml := `apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: bad-regex
rules:
  - id: ssn-lookahead
    level: 2
    severity: CRITICAL
    patterns:
      - regex: "\\b(?!000)\\d{3}-\\d{2}-\\d{4}\\b"
`
	p := filepath.Join(dir, "pack.yaml")
	require.NoError(t, os.WriteFile(p, []byte(yml), 0o600))

	_, err := LoadPacks(p)
	require.Error(t, err)
	require.Contains(t, err.Error(), `rulepack "bad-regex"`)
	require.Contains(t, err.Error(), `rule "ssn-lookahead"`)
	require.Contains(t, err.Error(), "invalid regex")
}

func TestCheckPatterns_ReportsEveryBadPattern(t *testing.T) {
	pack := RulePack{
		Metadata: Metadata{Name: "p"},
		Rules: []Rule{
			{ID: "ok", Patterns: []Pattern{{Regex: `\d+`}, {Regex: `abc`, Flags: []string{"i"}}}},
			{ID: "backref", Patterns: []Pattern{{Regex: `(a)\1`}}},
			{ID: "fb", Patterns: []Pattern{{Regex: `x`}}, Fallback: &Fallback{Patterns: []Pattern{{Regex: `(?<=a)b`}}}},
		},
	}
	err := CheckPatterns(pack)
	require.Error(t, err)
	require.Contains(t, err.Error(), `rule "backref": pattern 0`)
	require.Contains(t, err.Error(), `rule "fb": fallback pattern 0`)
	require.NotContains(t, err.Error(), `rule "ok"`)

	require.NoError(t, CheckPatterns(RulePack{Rules: []Rule{{ID: "ok", Patterns: []Pattern{{Regex: `\b\d{3}-\d{2}-\d{4}\b`}}}}}))
}

// Every RulePack shipped in the repository must load: a regex RE2 cannot
// compile would otherwise disable its rule.
func TestShippedRulePacksLoad(t *testing.T) {
	var files []string
	for _, pattern := range []string{
		"../../rules/*.yaml",
		"../../demo/rules.yaml",
		"../../docs/examples/rulepacks/*.yaml",
	} {
		m, err := filepath.Glob(pattern)
		require.NoError(t, err)
		files = append(files, m...)
	}
	require.NotEmpty(t, files)
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			_, err := LoadPacks(f)
			require.NoError(t, err)
		})
	}
}
