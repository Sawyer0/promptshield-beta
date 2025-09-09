package rules

import (
    "fmt"
    "os"
    "regexp"
    "regexp/syntax"
    "strconv"
    "strings"
)

// ComplexityScore provides a coarse measure of regex structural cost.
type ComplexityScore struct {
    Nodes        int
    MaxDepth     int
    Alternations int
    Repeats      int
}

type complexityLimits struct {
    maxNodes        int
    maxDepth        int
    maxAlternations int
    maxRepeats      int
}

func limitsFromEnv() complexityLimits {
    // Conservative defaults suitable for RE2 while preventing pathological patterns.
    lim := complexityLimits{maxNodes: 750, maxDepth: 30, maxAlternations: 300, maxRepeats: 1000}
    if v := os.Getenv("PS_MAX_REGEX_NODES"); v != "" {
        if n, err := strconv.Atoi(v); err == nil && n > 0 { lim.maxNodes = n }
    }
    if v := os.Getenv("PS_MAX_REGEX_DEPTH"); v != "" {
        if n, err := strconv.Atoi(v); err == nil && n > 0 { lim.maxDepth = n }
    }
    if v := os.Getenv("PS_MAX_REGEX_ALTERNATIONS"); v != "" {
        if n, err := strconv.Atoi(v); err == nil && n > 0 { lim.maxAlternations = n }
    }
    if v := os.Getenv("PS_MAX_REGEX_REPEATS"); v != "" {
        if n, err := strconv.Atoi(v); err == nil && n > 0 { lim.maxRepeats = n }
    }
    return lim
}

// prefixedRegex adds inline flags consistent with compileRegexStrict.
func prefixedRegex(expr string, flags []string) string {
    if expr == "" { return expr }
    allowed := map[string]bool{"ignorecase": true, "i": true, "multiline": true, "m": true}
    prefix := ""
    for _, f := range flags {
        f = strings.ToLower(f)
        if !allowed[f] { continue }
        switch f {
        case "ignorecase", "i":
            prefix = "(?i)" + prefix
        case "multiline", "m":
            prefix = "(?m)" + prefix
        }
    }
    return prefix + expr
}

// ComputeRegexComplexity parses the regex and returns a structural complexity score.
func ComputeRegexComplexity(expr string, flags []string) (ComplexityScore, error) {
    var score ComplexityScore
    if expr == "" {
        return score, fmt.Errorf("empty regex")
    }
    // Parse using RE2 syntax for safety.
    parsed, err := syntax.Parse(prefixedRegex(expr, flags), syntax.Perl)
    if err != nil {
        return score, err
    }
    var walk func(re *syntax.Regexp, depth int)
    walk = func(re *syntax.Regexp, depth int) {
        if re == nil { return }
        score.Nodes++
        if depth > score.MaxDepth { score.MaxDepth = depth }
        switch re.Op {
        case syntax.OpAlternate:
            score.Alternations += len(re.Sub) - 1
            for _, sub := range re.Sub { walk(sub, depth+1) }
            return
        case syntax.OpRepeat:
            score.Repeats++
            // Treat nested repeats as additional depth cost
            walk(re.Sub[0], depth+2)
            return
        }
        for _, sub := range re.Sub { walk(sub, depth+1) }
    }
    walk(parsed.Simplify(), 1)
    return score, nil
}

// CheckRegexComplexity returns an error if the regex exceeds limits.
func CheckRegexComplexity(expr string, flags []string) error {
    score, err := ComputeRegexComplexity(expr, flags)
    if err != nil {
        return err
    }
    lim := limitsFromEnv()
    if score.Nodes > lim.maxNodes || score.MaxDepth > lim.maxDepth || score.Alternations > lim.maxAlternations || score.Repeats > lim.maxRepeats {
        return fmt.Errorf("regex too complex (nodes=%d, depth=%d, alts=%d, repeats=%d; limits nodes<=%d depth<=%d alts<=%d repeats<=%d)",
            score.Nodes, score.MaxDepth, score.Alternations, score.Repeats, lim.maxNodes, lim.maxDepth, lim.maxAlternations, lim.maxRepeats)
    }
    return nil
}

// CompileWithComplexity enforces complexity limits then compiles with RE2.
func CompileWithComplexity(expr string, flags []string) (*regexp.Regexp, error) {
    if err := CheckRegexComplexity(expr, flags); err != nil {
        return nil, err
    }
    return compileRegexStrict(expr, flags)
}


