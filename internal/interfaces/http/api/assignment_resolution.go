package api

import (
    "context"
    "database/sql"
    "regexp"
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
    // Fast path: use endpoint_rulepack_snapshots if DB is available
    if opt.DB != nil {
        rpIDs, err := snapshotRulepacks(ctx, opt, tenantID, endpoint, method)
        if err == nil && len(rpIDs) > 0 {
            seen := make(map[uuid.UUID]struct{})
            for _, id := range rpIDs {
                if _, ok := seen[id]; ok { continue }
                dsl, _, err := opt.RulepackService.GetActive(ctx, id)
                if err != nil || len(dsl) == 0 { continue }
                var rp rules.RulePack
                if err := yaml.Unmarshal(dsl, &rp); err != nil { continue }
                rp.SourcePath = "db:snapshot:rulepack:" + id.String()
                out = append(out, rp)
                seen[id] = struct{}{}
            }
            if len(out) > 0 {
                return out, nil
            }
            // Log snapshot miss (no IDs found) for observability
            logSnapshotMiss(ctx, opt, tenantID, endpoint, method)
        } else {
            // Error or empty result: still log miss
            logSnapshotMiss(ctx, opt, tenantID, endpoint, method)
        }
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

// snapshotRulepacks returns ordered rulepack IDs from endpoint_rulepack_snapshots.
func snapshotRulepacks(ctx context.Context, opt Options, tenantID uuid.UUID, endpoint, method string) ([]uuid.UUID, error) {
    // Normalize path similar to DB function normalize_path_template
    path := normalizePathForSnapshot(endpoint)
    m := strings.ToUpper(strings.TrimSpace(method))
    if m == "" { m = "ANY" }
    // Prefer method-specific, then ANY
    try := func(meth string) ([]uuid.UUID, error) {
        const q = `SELECT rulepack_id
                   FROM endpoint_rulepack_snapshots,
                        unnest(rulepack_ids) WITH ORDINALITY AS t(rulepack_id, ord)
                   WHERE tenant_id = $1 AND method = $2 AND endpoint_template = $3
                   ORDER BY ord`
        rows, err := opt.DB.QueryContext(ctx, q, tenantID, meth, path)
        if err != nil { return nil, err }
        defer rows.Close()
        var ids []uuid.UUID
        for rows.Next() {
            var idStr string
            if err := rows.Scan(&idStr); err != nil { return nil, err }
            if id, err := uuid.Parse(strings.TrimSpace(idStr)); err == nil {
                ids = append(ids, id)
            }
        }
        if err := rows.Err(); err != nil { return nil, err }
        return ids, nil
    }
    if ids, err := try(m); err == nil && len(ids) > 0 { return ids, nil }
    if ids, err := try("ANY"); err == nil && len(ids) > 0 { return ids, nil }
    return nil, sql.ErrNoRows
}

var (
    reUUID   = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
    reNumber = regexp.MustCompile(`/[0-9]+`)
)

func normalizePathForSnapshot(p string) string {
    s := strings.TrimSpace(p)
    if s == "" { return "/" }
    if !strings.HasPrefix(s, "/") { s = "/" + s }
    s = reUUID.ReplaceAllString(s, "/:uuid")
    s = reNumber.ReplaceAllString(s, "/:id")
    for strings.Contains(s, "//") { s = strings.ReplaceAll(s, "//", "/") }
    return s
}

func logSnapshotMiss(ctx context.Context, opt Options, tenantID uuid.UUID, endpoint, method string) {
    if opt.DB == nil { return }
    tpl := normalizePathForSnapshot(endpoint)
    // best-effort; ignore error
    _, _ = opt.DB.ExecContext(ctx,
        "INSERT INTO endpoint_snapshot_misses (tenant_id, method, endpoint, template) VALUES ($1,$2,$3,$4)",
        tenantID, strings.ToUpper(strings.TrimSpace(method)), endpoint, tpl,
    )
}

