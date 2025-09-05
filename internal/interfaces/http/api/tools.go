package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/contracts"
	"github.com/promptshield/promptshield/internal/shared/redact"
	ttypes "github.com/promptshield/promptshield/internal/shared/types"
)

// toolExecHandler handles POST /api/tools/exec
func toolExecHandler(opt Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opt.ToolRunner == nil {
			writeErrorJSON(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "tool runner not configured", nil, r)
			return
		}
		tenantStr := strings.TrimSpace(r.Header.Get("X-PS-Tenant-ID"))
		if tenantStr == "" {
			writeErrorJSON(w, http.StatusBadRequest, "MISSING_TENANT", "X-PS-Tenant-ID header is required", nil, r)
			return
		}
		tenantID, err := uuid.Parse(tenantStr)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_TENANT", "invalid tenant id", nil, r)
			return
		}
		var body struct {
			ToolID         string          `json:"tool_id"`
			Args           json.RawMessage `json:"args"`
			ConversationID string          `json:"conversation_id"`
			RequestID      string          `json:"request_id"`
			Endpoint       string          `json:"endpoint"`
			Method         string          `json:"method"`
			Lane           string          `json:"lane"`
			PlanHash       string          `json:"plan_hash"`
			PlanStep       string          `json:"plan_step"`
			TimeoutMs      int64           `json:"timeout_ms"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid json body", nil, r)
			return
		}
		if strings.TrimSpace(body.ToolID) == "" || len(body.Args) == 0 {
			writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", "tool_id and args are required", nil, r)
			return
		}
		// Load preferences from settings (optional)
		prefs := loadToolPrefs(opt, r)
		// Resolve target endpoint/method (for policy match)
		endpoint := coalesce(body.Endpoint, r.Header.Get("X-PS-Endpoint"))
		method := strings.ToUpper(coalesce(body.Method, r.Header.Get("X-PS-Method")))
		// Try to match an endpoint-scoped tenant policy
		var matched *toolPolicySpec
		if opt.DB != nil {
			matched = loadMatchedToolPolicy(opt, r, endpoint, method)
		}
		// Enforce tool allowlist when configured (endpoint policy takes precedence if present)
		if matched != nil && len(matched.AllowedTools) > 0 && !strIn(body.ToolID, matched.AllowedTools) {
			writeErrorJSON(w, http.StatusForbidden, "TOOL_NOT_ALLOWED", "tool not allowed for this endpoint", map[string]any{"tool": body.ToolID, "endpoint": endpoint}, r)
			return
		}
		if len(prefs.ToolsAllowlist) > 0 && !strIn(body.ToolID, prefs.ToolsAllowlist) {
			writeErrorJSON(w, http.StatusForbidden, "TOOL_NOT_ALLOWED", "tool is not allowlisted", map[string]any{"tool": body.ToolID}, r)
			return
		}
		// Require approval token for high-risk tools
		if strIn(body.ToolID, prefs.RequireApprovalTools) || (matched != nil && strIn(body.ToolID, matched.RequireApproval)) {
			tok := strings.TrimSpace(r.Header.Get("X-PS-Approval-Token"))
			if tok == "" {
				writeErrorJSON(w, http.StatusForbidden, "APPROVAL_REQUIRED", "approval required for tool", map[string]any{"tool": body.ToolID}, r)
				return
			}
			// Dev bypass: allow a static token to simplify demos
			if strings.EqualFold(strings.TrimSpace(getEnv("PS_DEV_BYPASS_AUTH", "")), "true") && tok != "approved" {
				writeErrorJSON(w, http.StatusForbidden, "APPROVAL_INVALID", "invalid approval token (dev expects 'approved')", nil, r)
				return
			}
		}
		// Basic egress allowlist for http_fetch
		if strings.EqualFold(body.ToolID, "http_fetch") {
			var a struct {
				URL string `json:"url"`
			}
			_ = json.Unmarshal(body.Args, &a)
			if a.URL == "" {
				writeErrorJSON(w, http.StatusBadRequest, "INVALID_ARGUMENT", "http_fetch requires args.url", nil, r)
				return
			}
			// Prefer endpoint policy egress allowlist when present, else fallback to prefs
			if matched != nil && (len(matched.EgressAllowlist.Schemes) > 0 || len(matched.EgressAllowlist.Hosts) > 0 || len(matched.EgressAllowlist.Paths) > 0) {
				p := toolPrefs{EgressSchemes: matched.EgressAllowlist.Schemes, EgressHosts: matched.EgressAllowlist.Hosts, EgressPaths: matched.EgressAllowlist.Paths}
				if !egressAllowed(a.URL, p) {
					writeErrorJSON(w, http.StatusForbidden, "EGRESS_BLOCKED", "URL not allowed by endpoint policy", map[string]any{"url": a.URL, "endpoint": endpoint}, r)
					return
				}
			} else {
				if !egressAllowed(a.URL, prefs) {
					writeErrorJSON(w, http.StatusForbidden, "EGRESS_BLOCKED", "URL not allowed by egress allowlist", map[string]any{"url": a.URL}, r)
					return
				}
			}
		}

		// Timeout: request override or configured default
		timeout := time.Duration(body.TimeoutMs) * time.Millisecond
		if timeout <= 0 {
			if matched != nil && matched.TimeoutMs > 0 {
				timeout = time.Duration(matched.TimeoutMs) * time.Millisecond
			} else if prefs.TimeoutDefaultMs > 0 {
				timeout = time.Duration(prefs.TimeoutDefaultMs) * time.Millisecond
			} else {
				timeout = 2 * time.Second
			}
		}

		req := contracts.ToolExecRequest{
			TenantID:       tenantID,
			ToolID:         body.ToolID,
			Args:           body.Args,
			ConversationID: body.ConversationID,
			RequestID:      coalesce(body.RequestID, r.Header.Get("X-Request-ID")),
			Endpoint:       coalesce(body.Endpoint, r.Header.Get("X-PS-Endpoint")),
			Method:         coalesce(body.Method, r.Header.Get("X-PS-Method")),
			Lane:           coalesce(body.Lane, r.Header.Get("X-PS-Lane")),
			PlanHash:       coalesce(body.PlanHash, r.Header.Get("X-PS-Plan-Hash")),
			PlanStep:       coalesce(body.PlanStep, r.Header.Get("X-PS-Plan-Step")),
			Timeout:        timeout,
		}

		start := time.Now()
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		resExec, err := opt.ToolRunner.Execute(ctx, req)
		dur := time.Since(start)
		if err != nil {
			auditToolExec(opt, tenantID, req, nil, dur, 500, r)
			writeErrorJSON(w, http.StatusInternalServerError, "TOOL_EXEC_ERROR", err.Error(), nil, r)
			return
		}
		auditToolExec(opt, tenantID, req, &resExec, dur, 200, r)
		w.Header().Set("Content-Type", strings.TrimSpace(resExec.ContentType))
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/json")
		}
		_ = json.NewEncoder(w).Encode(resExec)
	}
}

func auditToolExec(opt Options, tenantID uuid.UUID, req contracts.ToolExecRequest, res *contracts.ToolExecResult, dur time.Duration, status int, r *http.Request) {
	if opt.AuditLogger == nil {
		return
	}
	before := map[string]any{
		"tool_id": req.ToolID,
		"args":    redact.RedactAndTruncate(string(req.Args), 4096),
	}
	after := map[string]any{}
	if res != nil {
		after["result"] = redact.RedactAndTruncate(string(res.Result), 4096)
		after["latency_ms"] = res.LatencyMs
		if res.Provider != "" {
			after["provider"] = res.Provider
		}
		if res.Model != "" {
			after["model"] = res.Model
		}
	}
	meta := map[string]any{
		"endpoint":      req.Endpoint,
		"method":        req.Method,
		"lane":          req.Lane,
		"plan_hash":     req.PlanHash,
		"plan_step":     req.PlanStep,
		"status":        status,
		"processing_ms": dur.Milliseconds(),
	}
	ev := ttypes.AuditEvent{
		TenantID:   &tenantID,
		ActorType:  ttypes.ActorTypeUser,
		Action:     "tool.exec",
		ObjectType: "tool",
		ObjectID:   uuid.New(),
		Before:     before,
		After:      after,
		Metadata:   meta,
		Timestamp:  time.Now().UTC(),
		RequestID:  req.RequestID,
	}
	_ = opt.AuditLogger.LogWithContext(r.Context(), ev)
}

func coalesce(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return strings.TrimSpace(b)
}

// Preferences model and helpers (lightweight, local to tools)
type toolPrefs struct {
	ToolsAllowlist       []string
	RequireApprovalTools []string
	TimeoutDefaultMs     int64
	EgressSchemes        []string
	EgressHosts          []string
	EgressPaths          []string
}

func loadToolPrefs(opt Options, r *http.Request) toolPrefs {
	var p toolPrefs
	// Defaults
	p.EgressSchemes = []string{"http", "https"}
	// Settings repository is optional; if absent, keep defaults (allow all hosts/paths)
	if opt.SettingsRepository == nil {
		return p
	}
	s, err := opt.SettingsRepository.Get(r.Context())
	if err != nil || s == nil || len(s.Settings) == 0 {
		return p
	}
	var raw map[string]any
	if err := json.Unmarshal(s.Settings, &raw); err != nil {
		return p
	}
	prefs, _ := raw["preferences"].(map[string]any)
	if prefs == nil {
		return p
	}
	if v, ok := prefs["tools_allowlist"].([]any); ok {
		for _, e := range v {
			if s, ok := e.(string); ok && strings.TrimSpace(s) != "" {
				p.ToolsAllowlist = append(p.ToolsAllowlist, strings.TrimSpace(s))
			}
		}
	}
	if v, ok := prefs["require_approval_tools"].([]any); ok {
		for _, e := range v {
			if s, ok := e.(string); ok && strings.TrimSpace(s) != "" {
				p.RequireApprovalTools = append(p.RequireApprovalTools, strings.TrimSpace(s))
			}
		}
	}
	if tm, ok := prefs["timeouts_ms"].(map[string]any); ok {
		if d := intFromAny(tm["default"]); d > 0 {
			p.TimeoutDefaultMs = int64(d)
		}
	}
	if eg, ok := prefs["egress_allowlist"].(map[string]any); ok {
		if v, ok := eg["schemes"].([]any); ok {
			p.EgressSchemes = toStrs(v)
		}
		if v, ok := eg["hosts"].([]any); ok {
			p.EgressHosts = toStrs(v)
		}
		if v, ok := eg["paths"].([]any); ok {
			p.EgressPaths = toStrs(v)
		}
	}
	return p
}

func toStrs(v []any) []string {
	var out []string
	for _, e := range v {
		if s, ok := e.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}
func strIn(s string, arr []string) bool {
	for _, e := range arr {
		if strings.EqualFold(s, e) {
			return true
		}
	}
	return false
}

func egressAllowed(raw string, p toolPrefs) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(strings.TrimSuffix(u.Scheme, ":"))
	if len(p.EgressSchemes) > 0 {
		ok := false
		for _, s := range p.EgressSchemes {
			if strings.EqualFold(s, scheme) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if len(p.EgressHosts) > 0 {
		hostOK := false
		for _, h := range p.EgressHosts {
			h = strings.TrimSpace(h)
			if h == "" {
				continue
			}
			if strings.HasPrefix(h, "*.") {
				if strings.HasSuffix(u.Hostname(), strings.TrimPrefix(h, "*")) {
					hostOK = true
					break
				}
			} else if strings.EqualFold(u.Hostname(), h) {
				hostOK = true
				break
			}
		}
		if !hostOK {
			return false
		}
	}
	if len(p.EgressPaths) > 0 {
		path := u.EscapedPath()
		if !strings.HasSuffix(path, "/") {
			path += "/"
		}
		pathOK := false
		for _, glob := range p.EgressPaths {
			glob = strings.TrimSpace(glob)
			if glob == "" {
				continue
			}
			if strings.HasSuffix(glob, "*") {
				prefix := strings.TrimSuffix(glob, "*")
				if strings.HasPrefix(path, prefix) {
					pathOK = true
					break
				}
			} else if path == glob || strings.TrimSuffix(path, "/") == strings.TrimSuffix(glob, "/") {
				pathOK = true
				break
			}
		}
		if !pathOK {
			return false
		}
	}
	return true
}

// getEnv small helper to read env var with default
func getEnv(k, def string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	return v
}

// Endpoint-scoped tool policy spec for matching
type toolPolicySpec struct {
	Scope           string   `json:"scope"`
	Methods         []string `json:"methods"`
	AllowedTools    []string `json:"allowed_tools"`
	RequireApproval []string `json:"require_approval"`
	TimeoutMs       int64    `json:"timeout_ms"`
	EgressAllowlist struct {
		Schemes []string `json:"schemes"`
		Hosts   []string `json:"hosts"`
		Paths   []string `json:"paths"`
	} `json:"egress_allowlist"`
	RequireRoles   []string            `json:"require_roles"`
	RequireHeaders map[string][]string `json:"require_headers"`
}

// loadMatchedToolPolicy loads tenant tool policies and returns the best match for endpoint/method
func loadMatchedToolPolicy(opt Options, r *http.Request, endpoint, method string) *toolPolicySpec {
	// Fetch JSON from tenant_settings (RLS applies); cache per-tenant with short TTL
	tid, ok := GetTenantID(r.Context())
	if !ok {
		return nil
	}
	raw := loadPoliciesJSONFromCacheOrDB(opt, r, tid)
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var root map[string]any
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		return nil
	}
	arr, _ := root["policies"].([]any)
	if len(arr) == 0 {
		return nil
	}
	ep := strings.TrimSpace(endpoint)
	m := strings.ToUpper(strings.TrimSpace(method))
	var best *toolPolicySpec
	bestLen := -1
	for _, it := range arr {
		mm, ok := it.(map[string]any)
		if !ok {
			continue
		}
		scope, _ := mm["scope"].(string)
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		if !scopeMatch(scope, ep) {
			continue
		}
		if vs, ok := mm["methods"].([]any); ok && len(vs) > 0 {
			okm := false
			for _, e := range vs {
				if s, ok := e.(string); ok && strings.EqualFold(strings.TrimSpace(s), m) {
					okm = true
					break
				}
			}
			if !okm {
				continue
			}
		}
		spec := toPolicySpec(mm)
		if l := len(scope); l > bestLen {
			best, bestLen = spec, l
		}
	}
	if best != nil && len(best.RequireRoles) > 0 {
		roles := strings.Split(strings.TrimSpace(r.Header.Get("X-PS-User-Roles")), ",")
		if !anyIntersectFold(roles, best.RequireRoles) {
			return nil
		}
	}
	if best != nil && len(best.RequireHeaders) > 0 {
		if !headersMatch(r, best.RequireHeaders) {
			return nil
		}
	}
	return best
}

func scopeMatch(scope, path string) bool {
	if scope == "" {
		return false
	}
	if strings.HasSuffix(scope, "*") {
		return strings.HasPrefix(path, strings.TrimSuffix(scope, "*"))
	}
	return strings.TrimSuffix(scope, "/") == strings.TrimSuffix(path, "/")
}

func toPolicySpec(m map[string]any) *toolPolicySpec {
	s := &toolPolicySpec{}
	if v, ok := m["scope"].(string); ok {
		s.Scope = strings.TrimSpace(v)
	}
	if v, ok := m["methods"].([]any); ok {
		s.Methods = toStrs(v)
	}
	if v, ok := m["allowed_tools"].([]any); ok {
		s.AllowedTools = toStrs(v)
	}
	if v, ok := m["require_approval"].([]any); ok {
		s.RequireApproval = toStrs(v)
	}
	if tm, ok := m["timeout_ms"].(float64); ok {
		s.TimeoutMs = int64(tm)
	}
	if eg, ok := m["egress_allowlist"].(map[string]any); ok {
		if v, ok := eg["schemes"].([]any); ok {
			s.EgressAllowlist.Schemes = toStrs(v)
		}
		if v, ok := eg["hosts"].([]any); ok {
			s.EgressAllowlist.Hosts = toStrs(v)
		}
		if v, ok := eg["paths"].([]any); ok {
			s.EgressAllowlist.Paths = toStrs(v)
		}
	}
	if rr, ok := m["require_roles"].([]any); ok {
		s.RequireRoles = toStrs(rr)
	}
	if rh, ok := m["require_headers"].(map[string]any); ok {
		s.RequireHeaders = make(map[string][]string, len(rh))
		for k, v := range rh {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			switch tv := v.(type) {
			case string:
				sv := strings.TrimSpace(tv)
				if sv == "" {
					s.RequireHeaders[key] = []string{}
				} else {
					s.RequireHeaders[key] = []string{sv}
				}
			case []any:
				arr := toStrs(tv)
				s.RequireHeaders[key] = arr
			default:
				s.RequireHeaders[key] = []string{}
			}
		}
	}
	return s
}

func anyIntersectFold(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	for _, x := range a {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		for _, y := range b {
			if strings.EqualFold(x, strings.TrimSpace(y)) {
				return true
			}
		}
	}
	return false
}

// Simple per-tenant cache for tool policies JSON
var toolPolicyCache = struct {
	mu sync.RWMutex
	m  map[string]struct {
		raw string
		at  time.Time
	}
}{m: map[string]struct {
	raw string
	at  time.Time
}{}}

func toolPolicyCacheTTL() time.Duration {
	d := strings.TrimSpace(os.Getenv("PS_TOOL_POLICY_CACHE_TTL"))
	if d == "" {
		return 60 * time.Second
	}
	if dur, err := time.ParseDuration(d); err == nil && dur > 0 {
		return dur
	}
	return 60 * time.Second
}

func loadPoliciesJSONFromCacheOrDB(opt Options, r *http.Request, tid uuid.UUID) string {
	// Observe global epoch to decide if we should purge cache
	if epoch := currentPolicyEpoch(opt, r); epoch > getLastSeenEpoch() {
		flushToolPolicyCache()
		setLastSeenEpoch(epoch)
	}
	key := tid.String()
	toolPolicyCache.mu.RLock()
	ce, ok := toolPolicyCache.m[key]
	toolPolicyCache.mu.RUnlock()
	if ok && time.Since(ce.at) < toolPolicyCacheTTL() {
		return ce.raw
	}
	// Query explicit tenant row (RLS still applies)
	const q = `SELECT value FROM tenant_settings WHERE key='tool_policies' AND tenant_id=$1 LIMIT 1`
	var val sql.NullString
	if err := opt.DB.QueryRowContext(r.Context(), q, tid).Scan(&val); err != nil && err != sql.ErrNoRows {
		return ""
	}
	raw := ""
	if val.Valid {
		raw = val.String
	}
	toolPolicyCache.mu.Lock()
	toolPolicyCache.m[key] = struct {
		raw string
		at  time.Time
	}{raw: raw, at: time.Now()}
	toolPolicyCache.mu.Unlock()
	return raw
}

// headersMatch checks presence and/or allowed values (case-sensitive equality) for required headers
func headersMatch(r *http.Request, req map[string][]string) bool {
	if len(req) == 0 {
		return true
	}
	for name, vals := range req {
		if name == "" {
			continue
		}
		got := r.Header.Get(name)
		if strings.TrimSpace(got) == "" {
			return false
		}
		// If no values provided, presence-only
		if len(vals) == 0 {
			continue
		}
		ok := false
		for _, v := range vals {
			if strings.TrimSpace(got) == strings.TrimSpace(v) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

// flushToolPolicyCache clears the in-memory per-tenant policies cache
func flushToolPolicyCache() {
	toolPolicyCache.mu.Lock()
	toolPolicyCache.m = map[string]struct {
		raw string
		at  time.Time
	}{}
	toolPolicyCache.mu.Unlock()
}

// flush a single tenant's cached policies (if present)
func flushToolPolicyCacheTenant(tid string) {
	toolPolicyCache.mu.Lock()
	delete(toolPolicyCache.m, tid)
	toolPolicyCache.mu.Unlock()
}

// Epoch tracking for cluster-wide cache invalidation via DB settings
var policyEpoch struct {
	mu  sync.RWMutex
	val int64
	at  time.Time
}

var lastSeenEpoch struct {
	mu  sync.RWMutex
	val int64
}

func getLastSeenEpoch() int64 {
	lastSeenEpoch.mu.RLock()
	v := lastSeenEpoch.val
	lastSeenEpoch.mu.RUnlock()
	return v
}
func setLastSeenEpoch(v int64) {
	lastSeenEpoch.mu.Lock()
	lastSeenEpoch.val = v
	lastSeenEpoch.mu.Unlock()
}

func currentPolicyEpoch(opt Options, r *http.Request) int64 {
	// If settings repo not configured, no epoch available
	if opt.SettingsRepository == nil {
		return 0
	}
	// Cache epoch briefly to avoid extra reads
	policyEpoch.mu.RLock()
	if time.Since(policyEpoch.at) < 2*time.Second {
		v := policyEpoch.val
		policyEpoch.mu.RUnlock()
		return v
	}
	policyEpoch.mu.RUnlock()
	// Read platform settings and extract tool_policy_epoch
	s, err := opt.SettingsRepository.Get(r.Context())
	if err != nil || s == nil || len(s.Settings) == 0 {
		return 0
	}
	var raw map[string]any
	if err := json.Unmarshal(s.Settings, &raw); err != nil {
		return 0
	}
	var epoch int64
	if v, ok := raw["tool_policy_epoch"].(float64); ok {
		epoch = int64(v)
	}
	policyEpoch.mu.Lock()
	policyEpoch.val = epoch
	policyEpoch.at = time.Now()
	policyEpoch.mu.Unlock()
	return epoch
}
