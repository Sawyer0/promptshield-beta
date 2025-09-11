package api

import (
    "context"
    "strings"

    "github.com/google/uuid"
    "github.com/promptshield/promptshield/internal/rules"
    ckeys "github.com/promptshield/promptshield/internal/shared/contextkeys"
    "gopkg.in/yaml.v3"
)

// matchesEndpointPattern returns true when the given request path matches the
// assignment target_scope pattern.
// Supported patterns:
//   - "/" or "*" => matches all
//   - exact path, e.g., "/api/payments/refund"
//   - prefix wildcard, e.g., "/v1/*" or "/api/orders/*"
func matchesEndpointPattern(requestPath, pattern string) bool {
    rp := strings.TrimSpace(requestPath)
    if rp == "" {
        rp = "/"
    }
    if !strings.HasPrefix(rp, "/") {
        rp = "/" + rp
    }
    p := strings.TrimSpace(pattern)
    if p == "" || p == "/" || p == "*" {
        return true
    }
    if !strings.HasPrefix(p, "/") {
        p = "/" + p
    }
    // Prefix wildcard "/prefix/*"
    if strings.HasSuffix(p, "/*") {
        base := strings.TrimSuffix(p, "/*")
        if base == "" || base == "/" {
            return true
        }
        if rp == base {
            return true
        }
        return strings.HasPrefix(rp, base+"/")
    }
    // Exact match
    return rp == p
}

// resolveApplicableRulepacks fetches tenant-scoped assignments and returns the
// set of rulepacks whose target_scope matches the given endpoint. Assignments
// are evaluated in repository order (priority desc, created_at asc) and
// deduplicated by rulepack id.
func resolveApplicableRulepacks(ctx context.Context, opt Options, endpoint string, method string) ([]rules.RulePack, error) {
    var out []rules.RulePack
    if opt.AssignmentRepository == nil || opt.RulepackService == nil {
        return out, nil
    }
    tVal := ctx.Value(ckeys.TenantID)
    tStr, _ := tVal.(string)
    tenantID, err := uuid.Parse(strings.TrimSpace(tStr))
    if err != nil {
        return out, nil // no tenant in context → skip assignment resolution
    }
    // List assignments for tenant (already ordered by priority desc in repo)
    list, err := opt.AssignmentRepository.ListByTenant(ctx, tenantID)
    if err != nil {
        return out, err
    }
    if len(list) == 0 {
        return out, nil
    }

    // Normalize method and filter: match when assignment method is "*" or equals request method (case-insensitive)
    reqMethod := strings.ToUpper(strings.TrimSpace(method))
    if reqMethod == "" { reqMethod = "*" }
    // Filter enabled + method + pattern match
    seen := make(map[uuid.UUID]struct{})
    for _, a := range list {
        if a == nil || !a.Enabled {
            continue
        }
        am := strings.ToUpper(strings.TrimSpace(a.Method))
        if am != "*" && am != reqMethod {
            continue
        }
        if !matchesEndpointPattern(endpoint, a.TargetScope) {
            continue
        }
        if _, ok := seen[a.RulepackID]; ok {
            continue
        }
        // Fetch active DSL for rulepack
        dsl, _, err := opt.RulepackService.GetActive(ctx, a.RulepackID)
        if err != nil || len(dsl) == 0 {
            continue
        }
        var rp rules.RulePack
        if err := yaml.Unmarshal(dsl, &rp); err != nil {
            continue
        }
        // Attach source for observability (optional)
        rp.SourcePath = "db:rulepack:" + a.RulepackID.String()
        out = append(out, rp)
        seen[a.RulepackID] = struct{}{}
    }

    return out, nil
}

