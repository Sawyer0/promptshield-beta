package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPriorityBasedMerging(t *testing.T) {
	t.Run("HigherPriorityWins", func(t *testing.T) {
		// Create packs with explicit priorities
		lowPriorityPack := RulePack{
			Metadata: Metadata{Name: "low"},
			Composition: &Composition{
				Strategy: "priority_order",
				Priority: 1, // Lower priority
			},
			Rules: []Rule{
				{ID: "shared", Level: 1, Severity: "LOW", Keywords: []string{"test"}},
				{ID: "low-only", Level: 1, Severity: "INFO", Keywords: []string{"low"}},
			},
		}
		
		highPriorityPack := RulePack{
			Metadata: Metadata{Name: "high"},
			Composition: &Composition{
				Strategy: "priority_order", 
				Priority: 10, // Higher priority
			},
			Rules: []Rule{
				{ID: "shared", Level: 1, Severity: "CRITICAL", Keywords: []string{"test"}}, // Should win
				{ID: "high-only", Level: 1, Severity: "HIGH", Keywords: []string{"high"}},
			},
		}
		
		// Test both orders to ensure priority wins regardless of slice order
		result1 := MergePacksPriorityOrder([]RulePack{lowPriorityPack, highPriorityPack})
		result2 := MergePacksPriorityOrder([]RulePack{highPriorityPack, lowPriorityPack})
		
		// Both should produce identical results
		require.Len(t, result1, 3)
		require.Len(t, result2, 3)
		
		// Find the shared rule
		var sharedRule1, sharedRule2 *Rule
		for i := range result1 {
			if result1[i].ID == "shared" {
				sharedRule1 = &result1[i]
			}
		}
		for i := range result2 {
			if result2[i].ID == "shared" {
				sharedRule2 = &result2[i]
			}
		}
		
		require.NotNil(t, sharedRule1)
		require.NotNil(t, sharedRule2)
		
		// High priority pack should win (CRITICAL severity)
		assert.Equal(t, "CRITICAL", sharedRule1.Severity)
		assert.Equal(t, "CRITICAL", sharedRule2.Severity)
		
		// Results should be identical regardless of input order
		assert.Equal(t, result1, result2)
	})
	
	t.Run("DefaultPriorityZero", func(t *testing.T) {
		packWithPriority := RulePack{
			Metadata: Metadata{Name: "priority"},
			Composition: &Composition{
				Strategy: "priority_order",
				Priority: 5,
			},
			Rules: []Rule{
				{ID: "shared", Level: 1, Severity: "HIGH", Keywords: []string{"test"}},
			},
		}
		
		packWithoutPriority := RulePack{
			Metadata: Metadata{Name: "default"},
			// No composition = priority 0
			Rules: []Rule{
				{ID: "shared", Level: 1, Severity: "LOW", Keywords: []string{"test"}},
			},
		}
		
		result := MergePacksPriorityOrder([]RulePack{packWithoutPriority, packWithPriority})
		
		require.Len(t, result, 1)
		// Pack with explicit priority 5 should win over default priority 0
		assert.Equal(t, "HIGH", result[0].Severity)
	})
	
	t.Run("SamePriorityFallsBackToName", func(t *testing.T) {
		pack1 := RulePack{
			Metadata: Metadata{Name: "zebra"}, // Comes last alphabetically
			Composition: &Composition{
				Strategy: "priority_order",
				Priority: 5,
			},
			Rules: []Rule{
				{ID: "shared", Level: 1, Severity: "LOW", Keywords: []string{"test"}},
			},
		}
		
		pack2 := RulePack{
			Metadata: Metadata{Name: "alpha"}, // Comes first alphabetically
			Composition: &Composition{
				Strategy: "priority_order",
				Priority: 5, // Same priority
			},
			Rules: []Rule{
				{ID: "shared", Level: 1, Severity: "HIGH", Keywords: []string{"test"}},
			},
		}
		
		result := MergePacksPriorityOrder([]RulePack{pack1, pack2})
		
		require.Len(t, result, 1)
		// "alpha" should win due to alphabetical sorting when priorities are equal
		assert.Equal(t, "HIGH", result[0].Severity)
	})
	
	t.Run("PriorityWithExtends", func(t *testing.T) {
		basePack := RulePack{
			Metadata: Metadata{Name: "base"},
			Composition: &Composition{
				Priority: 1,
			},
			Rules: []Rule{
				{ID: "base-rule", Level: 1, Severity: "INFO", Keywords: []string{"base"}},
				{ID: "shared", Level: 1, Severity: "LOW", Keywords: []string{"test"}},
			},
		}
		
		extendingPack := RulePack{
			Metadata: Metadata{Name: "extending"},
			Composition: &Composition{
				Priority: 10, // Higher priority, but extends base
			},
			Extends: []string{"base"}, // Extends base pack
			Rules: []Rule{
				{ID: "extending-rule", Level: 1, Severity: "HIGH", Keywords: []string{"extending"}},
				{ID: "shared", Level: 1, Severity: "CRITICAL", Keywords: []string{"test"}},
			},
		}
		
		result := MergePacksPriorityOrder([]RulePack{basePack, extendingPack})
		
		require.Len(t, result, 3)
		
		// Find shared rule
		var sharedRule *Rule
		for i := range result {
			if result[i].ID == "shared" {
				sharedRule = &result[i]
				break
			}
		}
		
		require.NotNil(t, sharedRule)
		// Base pack wins due to extends semantics (first-wins after dependency resolution)
		// This is correct behavior - extends dependencies are processed first regardless of priority
		assert.Equal(t, "LOW", sharedRule.Severity)
	})
}

func TestGetPriorityHelper(t *testing.T) {
	t.Run("NilComposition", func(t *testing.T) {
		pack := RulePack{Metadata: Metadata{Name: "test"}}
		assert.Equal(t, 0, getPriority(pack))
	})
	
	t.Run("WithPriority", func(t *testing.T) {
		pack := RulePack{
			Metadata: Metadata{Name: "test"},
			Composition: &Composition{Priority: 42},
		}
		assert.Equal(t, 42, getPriority(pack))
	})
}