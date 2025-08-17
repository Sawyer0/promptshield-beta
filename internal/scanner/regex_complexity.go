package scanner

// regexComplexityScore computes a coarse risk score for a regex pattern.
// Heuristics target constructs that tend to cause excessive backtracking in engines
// other than RE2, and generally expensive evaluation even in RE2:
// - Unbounded quantifiers on large classes: .*, .+, \w+, \d+, .{m,}
// - Nested quantified groups: ( ... + )+ or ( ... * )+ etc.
// - Deep alternation chains: many '|'
// - Lazy/greedy modifiers near dot-star sequences
// Score is intentionally simple and cheap to compute. Higher is worse.
func regexComplexityScore(expr string) int {
	if expr == "" {
		return 0
	}
	score := 0
	depth := 0
	// track if the most recent group contained a quantifier
	var groupHasQuant []bool
	groupHasQuant = append(groupHasQuant, false)
	for i := 0; i < len(expr); i++ {
		c := expr[i]
		if c == '\\' { // escape
			if i+1 < len(expr) {
				// classify escaped classes
				switch expr[i+1] {
				case 'w', 'd', 's':
					// don't score baseline class; wait for quantifier
				}
				i++
			}
			continue
		}
		switch c {
		case '(':
			depth++
			groupHasQuant = append(groupHasQuant, false)
		case ')':
			if depth > 0 {
				depth--
				// if a quantifier follows, and inner had quantifiers, bump risk
				if i+1 < len(expr) {
					q := expr[i+1]
					if q == '+' || q == '*' || q == '?' {
						if depth+1 < len(groupHasQuant) && groupHasQuant[depth+1] {
							score += 8 // nested quantifier on quantified group
						} else {
							score += 3 // quantifier on group
						}
					}
				}
				// pop current group flag
				if len(groupHasQuant) > 0 {
					groupHasQuant = groupHasQuant[:len(groupHasQuant)-1]
				}
			}
		case '|':
			score += 1
		case '.':
			// dot with following quantifier is expensive on long lines
			if i+1 < len(expr) {
				switch expr[i+1] {
				case '+':
					score += 5
					if len(groupHasQuant) > 0 {
						groupHasQuant[len(groupHasQuant)-1] = true
					}
				case '*':
					score += 6
					if len(groupHasQuant) > 0 {
						groupHasQuant[len(groupHasQuant)-1] = true
					}
				case '{':
					// .{m,} unbounded upper
					// scan ahead for a closing '}' and comma
					j := i + 2
					upperUnbounded := false
					for ; j < len(expr) && expr[j] != '}'; j++ {
						if expr[j] == ',' {
							// check next non-space char before '}'
							k := j + 1
							for k < len(expr) && expr[k] == ' ' {
								k++
							}
							if k < len(expr) && expr[k] == '}' {
								upperUnbounded = true
							}
							break
						}
					}
					if upperUnbounded {
						score += 6
					}
				}
			}
		case '+', '*':
			// quantified token; if inside a group that already had quantifiers, bump
			if len(groupHasQuant) > 0 {
				if groupHasQuant[len(groupHasQuant)-1] {
					score += 4
				}
				groupHasQuant[len(groupHasQuant)-1] = true
			}
		case '?':
			// lazy/optional near dot/star increases ambiguity slightly
			score += 1
		case '{':
			// {m,} unbounded upper bound (not counting exact {m})
			j := i + 1
			hasComma := false
			for ; j < len(expr) && expr[j] != '}'; j++ {
				if expr[j] == ',' {
					hasComma = true
					break
				}
			}
			if hasComma {
				// if next non-space before '}' is '}', upper unbounded
				k := j + 1
				for k < len(expr) && expr[k] == ' ' {
					k++
				}
				if k < len(expr) && expr[k] == '}' {
					score += 3
				}
			}
		}
	}
	if score < 0 {
		score = 0
	}
	return score
}
