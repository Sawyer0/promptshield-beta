//go:build hyperscan

package scanner

import (
	"fmt"
	"os"
	"strconv"

	// Hyperscan Go bindings; optional when built with -tags hyperscan
	hs "github.com/flier/gohs/hyperscan"
	"github.com/promptshield/promptshield/pkg/types"
)

// Override evalRegexesFunc to use Hyperscan when built with the hyperscan tag and enabled via env.
func init() {
	if enabled, _ := strconv.ParseBool(os.Getenv("PS_HYPERSCAN")); enabled {
		evalRegexesFunc = (*Scanner).evalRegexesHyperscan
	}
}

func (s *Scanner) evalRegexesHyperscan(res *types.ScanResult, line string, lineNum int64, cr compiledRule, compiledSeen map[string]struct{}) (bool, int) {
	if len(cr.regexes) == 0 {
		return false, 1
	}
	// Compile all patterns into a single Hyperscan database lazily per rule id.
	// For simplicity, compile per call; in production cache by rule id.
	exprs := make([]hs.Expression, 0, len(cr.regexes))
	for _, rx := range cr.regexes {
		// Use the original string; regexp.Regexp doesn't expose it reliably, so we fallback to String().
		// Users should avoid patterns that stringify differently.
		exprs = append(exprs, hs.Expression{Expression: rx.String(), Flags: hs.Caseless | hs.SomLeftMost})
	}
	db, err := hs.CompileMulti(exprs...)
	if err != nil {
		if s.logger != nil {
			s.logger.Debug("hyperscan compile error", "rule", cr.id, "error", err)
		}
		return s.evalRegexesStandard(res, line, lineNum, cr, compiledSeen)
	}
	scratch, err := hs.AllocScratch(db)
	if err != nil {
		return s.evalRegexesStandard(res, line, lineNum, cr, compiledSeen)
	}
	matched := false
	matchCol := 1
	cb := func(id uint, from, to uint64, flags uint, context uint64) error {
		key := fmt.Sprintf("%s|%d|rx|%d", cr.id, lineNum, from)
		if _, ok := compiledSeen[key]; ok {
			return nil
		}
		compiledSeen[key] = struct{}{}
		v := types.Violation{RuleID: cr.id, Message: cr.message, Severity: cr.severity, Line: int(lineNum), Column: int(from) + 1, RuleTimeoutMs: cr.timeoutMs}
		v.Category = cr.category
		if cr.response != nil {
			v.ResponseAction = cr.response.Action
			v.ResponseMessage = cr.response.Message
			v.ResponseReplacement = cr.response.Replacement
		}
		res.Violations = append(res.Violations, v)
		matched = true
		matchCol = int(from) + 1
		return nil
	}
	_ = db.Scan([]byte(line), scratch, cb, 0)
	return matched, matchCol
}
