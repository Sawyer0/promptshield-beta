package scanner

import (
	"strings"
	"time"

	"github.com/promptshield/promptshield/internal/rules"
)

// LoadRulePacks atomically replaces all rules in the scanner.
// This method is thread-safe and designed for live rule updates in production.
func (s *Scanner) LoadRulePacks(packs []rules.RulePack) {
	if s.logger != nil {
		s.logger.Debug("loading rule packs", "count", len(packs))
	}
	// Choose composition strategy
	usePriority := false
	for _, p := range packs {
		if p.Composition != nil && strings.EqualFold(p.Composition.Strategy, "priority_order") {
			usePriority = true
			break
		}
	}
	// Override with scanner-level composition strategy if set
	switch strings.ToLower(s.compositionStrategy) {
	case "priority_order":
		usePriority = true
	case "first_match":
		s.firstMatch = true
	}
    var merged []rules.Rule
	if usePriority {
		merged = rules.MergePacksPriorityOrder(packs)
	} else {
		merged = rules.MergePacks(packs)
	}
    // Apply pattern length guard before compiling to avoid wasting CPU on extreme patterns
    if s.maxPatternLength > 0 {
        trimmed := make([]rules.Rule, 0, len(merged))
        for _, r := range merged {
            if len(r.Patterns) > 0 {
                kept := make([]rules.Pattern, 0, len(r.Patterns))
                for _, p := range r.Patterns {
                    if len(p.Regex) <= s.maxPatternLength {
                        kept = append(kept, p)
                    }
                }
                r.Patterns = kept
            }
            if r.Fallback != nil && len(r.Fallback.Patterns) > 0 {
                keptFb := make([]rules.Pattern, 0, len(r.Fallback.Patterns))
                for _, p := range r.Fallback.Patterns {
                    if len(p.Regex) <= s.maxPatternLength {
                        keptFb = append(keptFb, p)
                    }
                }
                r.Fallback.Patterns = keptFb
            }
            trimmed = append(trimmed, r)
        }
        merged = trimmed
    }
    compiled := compileRules(merged, s.defaultRuleTimeoutMs, s.defaultCaseSensitive, s.defaultWholeWord)
	
	// Atomic replacement: compile new rules first, then swap atomically
	// This prevents race conditions during live rule updates
	s.compiled = compiled
	if s.logger != nil {
		s.logger.Debug("replaced compiled rules", "count", len(compiled))
	}
	// Derive composition/performance settings from packs (most restrictive wins)
	for _, p := range packs {
		if p.Composition != nil && strings.EqualFold(p.Composition.Strategy, "first_match") {
			s.firstMatch = true
		}
		if p.Performance != nil {
			if p.Performance.MaxLength > 0 {
				if s.maxLineForRegex == 0 || p.Performance.MaxLength < s.maxLineForRegex {
					s.maxLineForRegex = p.Performance.MaxLength
				}
				// Build Aho-Corasick over level-1 keywords
				var kw []string
				for _, cr := range s.compiled {
					if cr.level == 1 {
						for _, k := range cr.keywords {
							if k != "" {
								kw = append(kw, k)
							}
						}
					}
				}
				if len(kw) > 0 {
					s.aho = NewAho(kw)
				} else {
					s.aho = nil
				}
				// Build per-rule token automata for precise gating; honor min_token_len if set
				if p.Performance.Gate.MinTokenLen > 0 {
					minTokenLen = p.Performance.Gate.MinTokenLen
				}
				s.ruleTokenAho = make(map[string]*Aho)
				for _, cr := range s.compiled {
					if len(cr.literalTokens) > 0 {
						s.ruleTokenAho[cr.id] = NewAho(cr.literalTokens)
					}
				}
			}
			if p.Performance.Timeout != "" {
				if d, err := time.ParseDuration(p.Performance.Timeout); err == nil {
					if s.fileTimeout == 0 || d < s.fileTimeout {
						s.fileTimeout = d
					}
				}
			}
			// Gate enable/disable; if disabled set map to nil
			if !p.Performance.Gate.Enabled {
				s.ruleTokenAho = nil
			}
		}
	}
}
