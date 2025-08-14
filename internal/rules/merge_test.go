package rules

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func boolPtr(b bool) *bool { return &b }

func TestMergePacks_ExtendsOverrides(t *testing.T) {
	base := RulePack{
		Metadata: Metadata{Name: "base"},
		Rules:    []Rule{{ID: "r1", Severity: "WARNING", Level: 1, Keywords: []string{"a"}}, {ID: "r2", Severity: "HIGH", Level: 1, Keywords: []string{"a"}}},
	}
	ext := RulePack{
		Metadata:  Metadata{Name: "ext"},
		Extends:   []string{"base"},
		Rules:     []Rule{{ID: "r2", Severity: "CRITICAL", Level: 1, Keywords: []string{"a"}}, {ID: "r3", Severity: "INFO", Level: 1, Keywords: []string{"a"}}},
		Overrides: []Override{{RuleID: "r1", Severity: "ERROR"}, {RuleID: "r3", Enabled: boolPtr(false)}},
	}

	got := MergePacks([]RulePack{ext, base})

	// IDs should be r1, r2 (r3 disabled by override)
	require.Len(t, got, 2)
	require.Equal(t, "r1", got[0].ID)
	require.Equal(t, "ERROR", got[0].Severity)
	require.Equal(t, "r2", got[1].ID)
	require.Equal(t, "CRITICAL", got[1].Severity)
}

func TestMergePacks_PriorityOrder_FirstWins(t *testing.T) {
	base := RulePack{
		Metadata: Metadata{Name: "base"},
		Rules:    []Rule{{ID: "dup", Severity: "WARNING", Level: 1, Keywords: []string{"a"}}},
	}
	app := RulePack{
		Metadata:    Metadata{Name: "app"},
		Extends:     []string{"base"},
		Composition: &Composition{Strategy: "priority_order"},
		Rules:       []Rule{{ID: "dup", Severity: "ERROR", Level: 1, Keywords: []string{"a"}}},
	}
	got := MergePacksPriorityOrder([]RulePack{app, base})
	require.Len(t, got, 1)
	require.Equal(t, "dup", got[0].ID)
	// base should win due to first-wins semantics after extends order, since base is visited before app
	require.Equal(t, "WARNING", got[0].Severity)
}
