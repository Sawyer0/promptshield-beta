package scanner

import (
	"fmt"
	"strings"

	"github.com/promptshield/promptshield/pkg/types"
)

// evaluateLine applies all compiled rules to a single line and accumulates violations.
func (s *Scanner) evaluateLine(res *types.ScanResult, line string, lineNum int64, compiledSeen map[string]struct{}) bool {
	lower := strings.ToLower(line)
	matchedAny := false
	// Precompute Aho matches for case-insensitive L1
	var ahoHits []Match
	if s.aho != nil {
		ahoHits = s.aho.FindAll([]byte(lower))
	}
	for _, cr := range s.compiled {
		if !contextMatches(cr.when, cr.unless, s.runtimeContext) {
			continue
		}
		matched := false
		matchCol := 1
		if len(cr.keywords) > 0 {
			any := s.evalKeywords(res, lower, line, lineNum, cr, compiledSeen, ahoHits)
			if any {
				matchedAny = true
				if s.firstMatch {
					return true
				}
				// proceed to next rule
				continue
			}
		}
		if !matched && len(cr.regexes) > 0 {
			// Skip expensive regex on extremely long lines if configured
			if s.maxLineForRegex == 0 || len(line) <= s.maxLineForRegex {
				// Per-rule token gate for L2
				if a := s.ruleTokenAho[cr.id]; a != nil {
					if len(a.FindAll([]byte(lower))) == 0 {
						goto afterRegex
					}
				}
				// metrics: attempt regex
				res.Metrics.RegexAttempts++
				matched, matchCol = evalRegexesFunc(s, res, line, lineNum, cr, compiledSeen)
				if matched {
					matchedAny = true
					if s.firstMatch {
						return true
					}
					continue
				}
			}
		}
	afterRegex:
		if len(cr.regexes) > 0 && s.ruleTokenAho[cr.id] != nil {
			// count skip
			if len(s.ruleTokenAho[cr.id].FindAll([]byte(lower))) == 0 {
				res.Metrics.RegexSkipped++
			}
		}
		if !matched && cr.level == 3 {
			// Per-rule token gate for L3 (reuse L2 tokens)
			if a := s.ruleTokenAho[cr.id]; a != nil {
				if len(a.FindAll([]byte(lower))) == 0 {
					res.Metrics.SemanticSkipped++
					goto afterSemantic
				}
			}
			res.Metrics.SemanticAttempts++
			if s.evaluateSemantic(line, cr, &matchCol) {
				matched = true
			}
		}
	afterSemantic:
		if matched {
			compiledKey := fmt.Sprintf("%s|%d", cr.id, lineNum)
			if _, ok := compiledSeen[compiledKey]; ok {
				continue
			}
			compiledSeen[compiledKey] = struct{}{}
			res.Violations = append(res.Violations, types.Violation{
				RuleID: cr.id, Message: cr.message, Severity: cr.severity, Line: int(lineNum), Column: matchCol, RuleTimeoutMs: cr.timeoutMs,
			})
			matchedAny = true
			if s.firstMatch {
				return true
			}
		}
	}
	return matchedAny
}

func (s *Scanner) evalKeywords(res *types.ScanResult, lower string, line string, lineNum int64, cr compiledRule, compiledSeen map[string]struct{}, ahoHits []Match) bool {
	any := false
	if !cr.caseSensitive && s.aho != nil && len(ahoHits) > 0 {
		found := map[string]int{}
		firstIdx := -1
		for _, m := range ahoHits {
			kw := s.aho.patterns[m.PatternIndex]
			idx := m.Index
			if cr.wholeWord {
				leftOK := idx == 0 || !isWordChar(rune(lower[idx-1]))
				rightPos := idx + len(kw)
				rightOK := rightPos >= len(lower) || (rightPos < len(lower) && !isWordChar(rune(lower[rightPos])))
				if !(leftOK && rightOK) {
					continue
				}
			}
			if _, ok := found[kw]; !ok {
				found[kw] = idx
				if firstIdx < 0 {
					firstIdx = idx
				}
			}
			if !cr.requireAll {
				key := fmt.Sprintf("%s|%d|kw|%s|%d", cr.id, lineNum, kw, idx)
				if _, ok := compiledSeen[key]; ok {
					continue
				}
				compiledSeen[key] = struct{}{}
				v := types.Violation{RuleID: cr.id, Message: cr.message, Severity: cr.severity, Line: int(lineNum), Column: idx + 1, RuleTimeoutMs: cr.timeoutMs}
				v.Category = cr.category
				if cr.response != nil {
					v.ResponseAction = cr.response.Action
					v.ResponseMessage = cr.response.Message
					v.ResponseReplacement = cr.response.Replacement
				}
				res.Violations = append(res.Violations, v)
				any = true
				if s.firstMatch {
					return true
				}
			}
		}
		if cr.requireAll && len(cr.keywords) > 0 {
			allPresent := true
			for _, kw := range cr.keywords {
				if _, ok := found[kw]; !ok {
					allPresent = false
					break
				}
			}
			if allPresent {
				col := 1
				if firstIdx >= 0 {
					col = firstIdx + 1
				}
				key := fmt.Sprintf("%s|%d|kw|all", cr.id, lineNum)
				if _, ok := compiledSeen[key]; !ok {
					compiledSeen[key] = struct{}{}
					v := types.Violation{RuleID: cr.id, Message: cr.message, Severity: cr.severity, Line: int(lineNum), Column: col, RuleTimeoutMs: cr.timeoutMs}
					v.Category = cr.category
					if cr.response != nil {
						v.ResponseAction = cr.response.Action
						v.ResponseMessage = cr.response.Message
						v.ResponseReplacement = cr.response.Replacement
					}
					res.Violations = append(res.Violations, v)
					any = true
				}
			}
		}
		return any
	}
	if cr.requireAll && len(cr.keywords) > 0 {
		allPresent := true
		firstIdx := -1
		for _, kw := range cr.keywords {
			haystack := lower
			target := kw
			if cr.caseSensitive {
				haystack = line
				target = kw
			}
			idx := strings.Index(haystack, target)
			if idx < 0 {
				allPresent = false
				break
			}
			if cr.wholeWord {
				leftOK := idx == 0 || !isWordChar(rune(haystack[idx-1]))
				rightPos := idx + len(target)
				rightOK := rightPos >= len(haystack) || (rightPos < len(haystack) && !isWordChar(rune(haystack[rightPos])))
				if !(leftOK && rightOK) {
					allPresent = false
					break
				}
			}
			if firstIdx < 0 {
				firstIdx = idx
			}
		}
		if allPresent {
			col := 1
			if firstIdx >= 0 {
				col = firstIdx + 1
			}
			key := fmt.Sprintf("%s|%d|kw|all", cr.id, lineNum)
			if _, ok := compiledSeen[key]; !ok {
				compiledSeen[key] = struct{}{}
				res.Violations = append(res.Violations, types.Violation{
					RuleID: cr.id, Message: cr.message, Severity: cr.severity, Line: int(lineNum), Column: col, RuleTimeoutMs: cr.timeoutMs,
				})
				any = true
			}
		}
		return any
	}
	for _, kw := range cr.keywords {
		haystack := lower
		target := kw
		if cr.caseSensitive {
			haystack = line
			target = kw
		}
		if idx := strings.Index(haystack, target); idx >= 0 {
			if cr.wholeWord {
				leftOK := idx == 0 || !isWordChar(rune(haystack[idx-1]))
				rightPos := idx + len(target)
				rightOK := rightPos >= len(haystack) || (rightPos < len(haystack) && !isWordChar(rune(haystack[rightPos])))
				if !(leftOK && rightOK) {
					continue
				}
			}
			key := fmt.Sprintf("%s|%d|kw|%s|%d", cr.id, lineNum, kw, idx)
			if _, ok := compiledSeen[key]; ok {
				continue
			}
			compiledSeen[key] = struct{}{}
			v := types.Violation{RuleID: cr.id, Message: cr.message, Severity: cr.severity, Line: int(lineNum), Column: idx + 1, RuleTimeoutMs: cr.timeoutMs}
			v.Category = cr.category
			if cr.response != nil {
				v.ResponseAction = cr.response.Action
				v.ResponseMessage = cr.response.Message
				v.ResponseReplacement = cr.response.Replacement
			}
			res.Violations = append(res.Violations, v)
			any = true
			if s.logger != nil {
				s.logger.Debug("keyword match", "rule", cr.id, "kw", kw, "line", lineNum, "col", idx+1, "case_sensitive", cr.caseSensitive, "whole_word", cr.wholeWord)
			}
		}
	}
	return any
}

// evalRegexesFunc can be overridden via build tags (e.g., Hyperscan). Default uses Go regexp.
var evalRegexesFunc = (*Scanner).evalRegexesStandard

func (s *Scanner) evalRegexesStandard(res *types.ScanResult, line string, lineNum int64, cr compiledRule, compiledSeen map[string]struct{}) (bool, int) {
	matched := false
	matchCol := 1
	if cr.requireAll && len(cr.regexes) > 0 {
		earliest := -1
		for _, rx := range cr.regexes {
			loc := rx.FindStringIndex(line)
			if loc == nil {
				return false, 1
			}
			if earliest < 0 || loc[0] < earliest {
				earliest = loc[0]
			}
		}
		key := fmt.Sprintf("%s|%d|rx|all", cr.id, lineNum)
		if _, ok := compiledSeen[key]; !ok {
			compiledSeen[key] = struct{}{}
			col := 1
			if earliest >= 0 {
				col = earliest + 1
			}
			res.Violations = append(res.Violations, types.Violation{
				RuleID: cr.id, Message: cr.message, Severity: cr.severity, Line: int(lineNum), Column: col, RuleTimeoutMs: cr.timeoutMs,
			})
			matched = true
			matchCol = col
			if s.logger != nil {
				s.logger.Debug("regex all-match", "rule", cr.id, "line", lineNum, "col", matchCol)
			}
		}
		return matched, matchCol
	}
	for i, rx := range cr.regexes {
		matches := rx.FindAllStringIndex(line, -1)
		for _, loc := range matches {
			// Optional per-pattern verifier
			if i < len(cr.verifiers) {
				verifier := cr.verifiers[i]
				if verifier == "luhn" {
					if !LuhnCheck(line[loc[0]:loc[1]]) {
						continue
					}
				} else if verifier == "ssn_area" {
					if !SSNAreaValid(line[loc[0]:loc[1]]) {
						continue
					}
				} else if verifier == "iban" {
					if !IBANValid(line[loc[0]:loc[1]]) {
						continue
					}
				} else if verifier == "email" {
					if !EmailValid(line[loc[0]:loc[1]]) {
						continue
					}
				}
			}
			key := fmt.Sprintf("%s|%d|rx|%d", cr.id, lineNum, loc[0])
			if _, ok := compiledSeen[key]; ok {
				continue
			}
			compiledSeen[key] = struct{}{}
			v := types.Violation{RuleID: cr.id, Message: cr.message, Severity: cr.severity, Line: int(lineNum), Column: loc[0] + 1, RuleTimeoutMs: cr.timeoutMs}
			v.Category = cr.category
			if cr.response != nil {
				v.ResponseAction = cr.response.Action
				v.ResponseMessage = cr.response.Message
				v.ResponseReplacement = cr.response.Replacement
			}
			res.Violations = append(res.Violations, v)
			matched = true
			matchCol = loc[0] + 1
			if s.logger != nil {
				s.logger.Debug("regex match", "rule", cr.id, "line", lineNum, "col", matchCol)
			}
		}
	}
	return matched, matchCol
}

func (s *Scanner) evaluateLongLine(res *types.ScanResult, line []byte, lineNum int64, overlap int) {
	if len(line) <= s.bufferSizeBytes {
		compiledRuleSeen := make(map[string]struct{})
		s.evaluateLine(res, string(line), lineNum, compiledRuleSeen)
		return
	}
	if overlap < 0 {
		overlap = 0
	}
	step := s.bufferSizeBytes
	compiledRuleSeen := make(map[string]struct{})
	for start := 0; start < len(line); start += step {
		end := start + step
		if end > len(line) {
			end = len(line)
		}
		if end < len(line) && end+overlap <= len(line) {
			end += overlap
		}
		s.evaluateLine(res, string(line[start:end]), lineNum, compiledRuleSeen)
	}
}
