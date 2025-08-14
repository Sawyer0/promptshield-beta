package rules

import (
	"sort"
)

// MergePacks produces a deterministic merged list of rules from the provided
// packs, applying extends and overrides where possible. Imports are currently
// ignored and assumed to be pre-resolved by the caller.
//
// Semantics:
//   - Packs are processed in deterministic order by metadata.name.
//   - For each pack, any referenced pack names in Extends are processed first
//     (if present in the provided set).
//   - Rules are merged by ID: later definitions (in processing order) replace
//     earlier ones.
//   - Overrides are applied after rules are merged, in processing order.
//   - Rules with Enabled=false are excluded from the resulting set.
func MergePacks(packs []RulePack) []Rule {
	// Index packs by name for extends resolution.
	byName := make(map[string]RulePack, len(packs))
	names := make([]string, 0, len(packs))
	for _, p := range packs {
		n := p.Metadata.Name
		byName[n] = p
		names = append(names, n)
	}
	sort.Strings(names)

	visited := make(map[string]bool)
	order := make([]string, 0, len(names))
	var visit func(string)
	visit = func(n string) {
		if visited[n] {
			return
		}
		visited[n] = true
		p, ok := byName[n]
		if !ok {
			return
		}
		for _, dep := range p.Extends {
			if _, ok := byName[dep]; ok {
				visit(dep)
			}
		}
		order = append(order, n)
	}
	for _, n := range names {
		visit(n)
	}

	// Merge rules in processing order.
	byID := make(map[string]Rule)
	for _, n := range order {
		p := byName[n]
		for _, r := range p.Rules {
			byID[r.ID] = r
		}
	}
	// Apply overrides in processing order.
	for _, n := range order {
		p := byName[n]
		for _, ov := range p.Overrides {
			r, ok := byID[ov.RuleID]
			if !ok {
				continue
			}
			if ov.Severity != "" {
				r.Severity = ov.Severity
			}
			if ov.Enabled != nil {
				r.Enabled = ov.Enabled
			}
			byID[ov.RuleID] = r
		}
	}

	// Materialize list, excluding disabled rules. Stable by rule ID.
	ids := make([]string, 0, len(byID))
	for id := range byID {
		// Skip disabled
		r := byID[id]
		if r.Enabled != nil && *r.Enabled == false {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]Rule, 0, len(ids))
	for _, id := range ids {
		out = append(out, byID[id])
	}
	return out
}

// MergePacksPriorityOrder merges rules preserving pack and rule order with
// first-wins semantics. Extends are respected by processing dependencies
// first. Later duplicates are ignored.
func MergePacksPriorityOrder(packs []RulePack) []Rule {
	// Index packs by name
	byName := make(map[string]RulePack, len(packs))
	for _, p := range packs {
		byName[p.Metadata.Name] = p
	}
	// Deterministic pack order: resolve extends DFS then sort remaining by name
	visited := make(map[string]bool)
	var order []string
	var visit func(string)
	visit = func(n string) {
		if visited[n] {
			return
		}
		visited[n] = true
		p, ok := byName[n]
		if !ok {
			return
		}
		for _, dep := range p.Extends {
			if _, ok := byName[dep]; ok {
				visit(dep)
			}
		}
		order = append(order, n)
	}
	// Start with sorted names to make traversal deterministic
	var names []string
	for n := range byName {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		visit(n)
	}

	// First-wins: keep first encountered rule id
	seen := make(map[string]struct{})
	var out []Rule
	for _, n := range order {
		p := byName[n]
		for _, r := range p.Rules {
			if _, ok := seen[r.ID]; ok {
				continue
			}
			seen[r.ID] = struct{}{}
			out = append(out, r)
		}
	}
	// Apply overrides in pack order
	for _, n := range order {
		p := byName[n]
		for i := range out {
			for _, ov := range p.Overrides {
				if out[i].ID != ov.RuleID {
					continue
				}
				if ov.Severity != "" {
					out[i].Severity = ov.Severity
				}
				if ov.Enabled != nil {
					out[i].Enabled = ov.Enabled
				}
			}
		}
	}
	// Filter disabled
	filtered := make([]Rule, 0, len(out))
	for _, r := range out {
		if r.Enabled != nil && *r.Enabled == false {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}
