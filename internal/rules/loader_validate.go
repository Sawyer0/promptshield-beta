package rules

import "fmt"

// validateSupportedFeatures checks for RulePack features that are parsed but not yet implemented
func validateSupportedFeatures(pack RulePack) error {
	// Extends/overrides are supported by merge logic; allow them.
	// Composition strategies supported: all_matches (default), first_match, priority_order
	if pack.Composition != nil {
		switch pack.Composition.Strategy {
		case "", "all_matches", "first_match", "priority_order":
			// ok
		default:
			return fmt.Errorf("unsupported composition strategy %q (valid: all_matches, first_match, priority_order)", pack.Composition.Strategy)
		}
	}

	// Rule-level validation
	for _, rule := range pack.Rules {
		if err := validateRuleFeatures(rule); err != nil {
			return fmt.Errorf("in rule '%s': %w", rule.ID, err)
		}
	}
	return nil
}

// validateRuleFeatures checks for unsupported rule-level features
func validateRuleFeatures(rule Rule) error {
	// logic: support any (default) and all; custom not yet supported
	switch rule.Logic {
	case "", "any", "all":
	default:
		return fmt.Errorf("unsupported logic %q (valid: any, all)", rule.Logic)
	}
	// response: allowed (affects enforcement/rendering, not validation)
	// cache: allowed (global caches may still apply)
	return nil
}
