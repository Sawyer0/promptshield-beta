package grpcenforcer

import (
    "context"
    "strings"

    "github.com/google/uuid"
    "github.com/promptshield/promptshield/internal/rules"
    "gopkg.in/yaml.v3"
)

// matchesEndpointPattern returns true if requestPath matches pattern.
// Supported:
// - "/" or "*" => all
// - exact path: "/api/payments/refund"
// - prefix wildcard: "/v1/*", "/api/orders/*"
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
    return rp == p
}

// resolveAssignedRulepacks loads rulepacks assigned to (tenantID, endpoint, method) using
// the assignment repository and rulepack repository.
func (s *Server) resolveAssignedRulepacks(ctx context.Context, tenantID uuid.UUID, endpoint string, method string) ([]rules.RulePack, error) {
    var out []rules.RulePack
    if s.assignmentRepo == nil || s.rulepackRepo == nil || tenantID == uuid.Nil {
        return out, nil
    }
    list, err := s.assignmentRepo.ListByTenant(ctx, tenantID)
    if err != nil || len(list) == 0 {
        return out, err
    }
    // Normalize request method once
    reqMethod := strings.ToUpper(strings.TrimSpace(method))
    if reqMethod == "" { reqMethod = "*" }

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
        dsl, _, err := s.rulepackRepo.GetActive(ctx, a.RulepackID)
        if err != nil || len(dsl) == 0 {
            continue
        }
        var rp rules.RulePack
        if err := yaml.Unmarshal(dsl, &rp); err != nil {
            continue
        }
        rp.SourcePath = "db:rulepack:" + a.RulepackID.String()
        out = append(out, rp)
        seen[a.RulepackID] = struct{}{}
    }
    return out, nil
}

