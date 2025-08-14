### PromptShield Gateway v1 API – Design Plan

#### Scope & Goals
- Replace CLI workflows with a stable HTTP API while keeping Envoy gRPC ext_proc.
- Keep core logic in `internal/` and `pkg/`; delivery via `enforcer/` (runtime) and `gateway/` (tests).
- Provide a versioned REST surface under `/v1` for rulepacks, runtime config, and admin.

#### Principles
- Base path: `/v1` (JSON only). Prometheus `/metrics` stays unversioned.
- Auth:
  - Requests: `Authorization: Bearer <token>` or `X-PS-Token` (already supported in `/check`).
  - Admin: `Authorization: Bearer <admin-token>` or `X-PS-Admin-Token` (new).
  - Enforce mTLS for admin endpoints when configured (reuse existing TLS/mTLS in HTTP server).
- Responses: Deterministic JSON; structured errors `{code, message, details}`; include `X-PS-API-Version: 1`.
- Compatibility: Backward-compatible additions; legacy root `/check` remains supported; new endpoints live under `/v1`.

---

### Endpoint Catalog

#### Health & Info
- GET `/v1/healthz`: 200 text `ok`
- GET `/v1/readyz`: 200 `ready`, or 503 when rulepack not loaded
- GET `/metrics`: Prometheus text format (unversioned)
- GET `/v1/version`: 200 `{ "version": "...", "commit": "...", "build_date": "..." }`

#### Enforcement
- POST `/v1/check`
  - Purpose: small synchronous decision
  - Query: `mode=observe|redact|quarantine|enforce` (optional override), `fail_on=INFO|WARNING|HIGH|ERROR|CRITICAL`
  - Body: `application/json` or `text/plain`; default cap 1MB (env override via `PS_ENFORCER_MAX_BODY_BYTES`)
  - 200 allow; 403 quarantine/deny
  - Headers: `x-ps-decision`, `x-ps-reason`, `x-ps-request-id`
  - JSON:
    ```json
    { "decision": "allow|quarantine|deny", "reason": "string", "violations": 0, "request_id": "uuid" }
    ```

- POST `/v1/scan`
  - Purpose: larger synchronous scan for non-Envoy clients
  - Content: `application/json` (aggregate) or `application/x-ndjson` (stream)
  - Query: `aggregate=true|false` (default true)
  - 200: aggregate JSON or NDJSON stream of per-record decisions

- gRPC (existing): Envoy `ext_proc` on :9091 implements `envoy.service.ext_proc.v3.ExternalProcessor`.

#### Async Jobs (optional v1)
- POST `/v1/scan:async`: 202 `{ "job_id": "uuid" }`
- GET `/v1/jobs/{job_id}`: 200 status/progress/results URI; 404/410 as appropriate
- DELETE `/v1/jobs/{job_id}`: 204 cancel/cleanup

#### RulePacks
- GET `/v1/rulepacks`: 200 `[{ id,name,version,source,active }]`
- GET `/v1/rulepacks/active`: 200 `{ id,name,version,source }`
- POST `/v1/rulepacks:validate` (admin)
  - Body: YAML (raw) or JSON rulepack
  - 200 `{ valid: true, warnings: [], errors: [] }`
- POST `/v1/rulepacks` (admin)
  - Body: YAML (raw) or multipart
  - Query: `activate=true|false`
  - 201 `{ id,name,version,active }`
- POST `/v1/rulepacks:reload` (admin)
  - Reload from `PS_ENFORCER_RULEPACK` path (or `?path=...`)
  - 200: active pack metadata
- PUT `/v1/rulepacks/active` (admin)
  - Body: `{ id: "..." }`
  - 200: active pack metadata
- DELETE `/v1/rulepacks/{id}` (admin)
  - 204 (no-op when file-only; applies to uploaded packs)

#### Runtime Config (admin)
- GET `/v1/config`
  - Snapshot of effective runtime:
    - `enforcement_mode`, `fail_on`, `redaction.enabled`
    - `max_stream_bytes`, `stream_window`, `stream_overlap`
    - `rps`, `rps_burst`, `inflight_limit_bytes`, `inflight_backoff_ms`
    - `timeouts` (per-rule/request/response)
- PUT `/v1/config`
  - Partial update; 200 updated snapshot
- POST `/v1/config:reset`
  - Restore defaults for mutable fields; 200 snapshot

#### Observability & Admin
- GET `/v1/events` (SSE)
  - Streams decision events; filter via `?types=decision,quarantine,deny`
- GET `/v1/stats`
  - Summary JSON from Prometheus counters/histograms
- GET `/v1/usage`
  - Rolling-window usage for billable units (requests, bytes), with window metadata `{ window_start, window_end, counts, bytes }`
- POST `/v1/admin/drain` (admin)
  - 202: begin graceful drain; `readyz` flips to 503
- POST `/v1/admin/shutdown` (admin)
  - 202: graceful exit; optional `?delay=seconds`

#### Licensing (admin)
- GET `/v1/license`
  - 200: `{ org, tier, expires_at, entitlements, licensed }`
- POST `/v1/license`
  - Accepts form or JSON `{ key: "..." }`; verifies and activates in-memory (optional persist)

---

### Schemas

- Error
```json
{ "code": "INVALID_ARGUMENT|UNAUTHORIZED|NOT_FOUND|INTERNAL", "message": "string", "details": {} }
```

- Decision
```json
{ "decision": "allow|quarantine|deny", "reason": "string", "violations": 0, "request_id": "uuid" }
```

---

### Router Design

Mount `/v1` under the existing enforcer mux. Current router setup (reference: `internal/interfaces/http/enforcer/server.go`):
```go
r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write([]byte("ok"))
})
// Readiness gate...
r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) { /* ... */ })
// Prometheus metrics
r.Handle("/metrics", promhttp.Handler())
```

Add a versioned API mux and mount it:
```go
// internal/interfaces/http/api/router.go
package api

import (
    "net/http"
    "time"
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
)

type Options struct {
    AdminToken         string
    AllowInsecureAdmin bool
    ConfigStore        *RuntimeConfigStore
    RulepackManager    *RulepackManager
}

func NewMux(opt Options) http.Handler {
    r := chi.NewRouter()
    r.Use(middleware.Timeout(10 * time.Second))
    r.Use(versionHeader("1"))

    r.Route("/rulepacks", func(r chi.Router) {
        r.Get("/", listRulepacks(opt))
        r.Get("/active", getActiveRulepack(opt))
        r.Group(func(a chi.Router) {
            a.Use(adminAuth(opt))
            a.Post("/validate", validateRulepack(opt))
            a.Post("/", uploadRulepack(opt))
            a.Post("/reload", reloadRulepack(opt))
            a.Put("/active", setActiveRulepack(opt))
            a.Delete("/{id}", deleteRulepack(opt))
        })
    })

    r.Route("/config", func(r chi.Router) {
        r.Get("/", getConfig(opt))
        r.Group(func(a chi.Router) {
            a.Use(adminAuth(opt))
            a.Put("/", putConfig(opt))
            a.Post("/reset", resetConfig(opt))
        })
    })

    r.Get("/version", getVersion())
    r.Get("/stats", getStats())
    r.Group(func(a chi.Router) {
        a.Use(adminAuth(opt))
        a.Post("/admin/drain", drain(opt))
        a.Post("/admin/shutdown", shutdown(opt))
    })

    r.Post("/check", checkHandlerVersioned())   // versioned duplicate
    r.Post("/scan", scanHandler(opt))
    return r
}
```

Mount in enforcer mux:
```go
// internal/interfaces/http/enforcer/server.go (inside NewMux)
adminToken := os.Getenv("PS_ENFORCER_ADMIN_TOKEN")
apiMux := api.NewMux(api.Options{
    AdminToken: adminToken,
    ConfigStore: configStoreRef,
    RulepackManager: rulepackManagerRef,
})
r.Mount("/v1", apiMux)
```

---

### Middleware & Helpers

Admin auth middleware:
```go
func adminAuth(opt Options) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if opt.AdminToken == "" && !opt.AllowInsecureAdmin {
                writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "admin token required", nil)
                return
            }
            tok := r.Header.Get("Authorization")
            if strings.HasPrefix(strings.ToLower(tok), "bearer ") {
                tok = tok[7:]
            }
            if tok == "" {
                tok = r.Header.Get("X-PS-Admin-Token")
            }
            if tok != opt.AdminToken {
                writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid admin token", nil)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

Error helper:
```go
type apiError struct {
    Code    string         `json:"code"`
    Message string         `json:"message"`
    Details map[string]any `json:"details,omitempty"`
}

func writeError(w http.ResponseWriter, status int, code, msg string, details map[string]any) {
    w.Header().Set("content-type", "application/json")
    w.Header().Set("X-PS-API-Version", "1")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(apiError{Code: code, Message: msg, Details: details})
}
```

Version header middleware:
```go
func versionHeader(v string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("X-PS-API-Version", v)
            next.ServeHTTP(w, r)
        })
    }
}
```

---

### Runtime Config Store (in-memory)

Map to existing tunables used in `internal/interfaces/grpc/enforcer/server.go` (e.g., `Options{Timeout, MaxStreamBytes, FailOn, RulepackPath, Telemetry, EnforcementMode}` and env vars like `PS_ENFORCER_STREAM_WINDOW`, `PS_ENFORCER_RPS`, etc.).

Minimal interfaces:
```go
type RuntimeConfig struct {
    EnforcementMode      string `json:"enforcement_mode"`
    FailOn               string `json:"fail_on"`
    RedactionEnabled     bool   `json:"redaction_enabled"`
    MaxStreamBytes       int64  `json:"max_stream_bytes"`
    StreamWindow         int    `json:"stream_window"`
    StreamOverlap        int    `json:"stream_overlap"`
    RPS                  float64 `json:"rps"`
    RPSBurst             int    `json:"rps_burst"`
    InflightLimitBytes   int64  `json:"inflight_limit_bytes"`
    InflightBackoffMs    int    `json:"inflight_backoff_ms"`
    PerRuleTimeoutMs     int    `json:"per_rule_timeout_ms"`
    RequestTimeoutMs     int    `json:"request_timeout_ms"`
    ResponseTimeoutMs    int    `json:"response_timeout_ms"`
}

type RuntimeConfigStore struct {
    mu  sync.RWMutex
    cfg RuntimeConfig
}

func (s *RuntimeConfigStore) Get() RuntimeConfig { s.mu.RLock(); defer s.mu.RUnlock(); return s.cfg }
func (s *RuntimeConfigStore) Update(p RuntimeConfig) RuntimeConfig { s.mu.Lock(); defer s.mu.Unlock(); /* merge+validate */; s.cfg = merge(s.cfg, p); return s.cfg }
```

Integrate with gRPC server by reading from store when constructing `grpcenforcer.NewWithOptions` or making selected fields atomically visible (e.g., update `Server` fields guarded by mutex/atomics).

---

### Rulepack Manager (ephemeral)

Use existing loader `rules.LoadPacks(...)`. Provide a process-local registry to upload, validate, and switch active packs (file-backed can remain via `PS_ENFORCER_RULEPACK`). Minimal interface:
```go
type RulepackMeta struct { ID, Name, Version, Source string; Active bool }
type RulepackManager struct {
    mu       sync.RWMutex
    activeID string
    packs    map[string]LoadedPack // LoadedPack wraps parsed packs for scanner
}
func (m *RulepackManager) List() []RulepackMeta {}
func (m *RulepackManager) Active() RulepackMeta {}
func (m *RulepackManager) Validate(data []byte) (bool, []string, []string) {}
func (m *RulepackManager) Upload(data []byte, activate bool) (RulepackMeta, error) {}
func (m *RulepackManager) SetActive(id string) error {}
func (m *RulepackManager) ReloadFrom(path string) (RulepackMeta, error) {}
```

On activation, reload scanners in both HTTP `/check` and gRPC ext_proc using the new pack set.

---

### OpenAPI & Documentation
- Update `docs/api/openapi.yaml` with new `/v1/*` endpoints.
- Keep `docs/api/grpc.md` for Envoy ext_proc; add config/examples for `/v1/check` integrations.
- Document auth/mTLS expectations in `docs/api/security.md`.

---

### Example Requests

Health and version:
```bash
curl -sf http://localhost:9090/v1/healthz
curl -sf http://localhost:9090/v1/version
```

Check (text):
```bash
curl -s -X POST \
  -H "Authorization: Bearer $PS_TOKEN" \
  --data-binary @payload.txt \
  http://localhost:9090/v1/check
```

Validate rulepack (admin):
```bash
curl -s -X POST \
  -H "Authorization: Bearer $PS_ADMIN_TOKEN" \
  --data-binary @rules.yaml \
  http://localhost:9090/v1/rulepacks:validate
```

Update config (admin):
```bash
curl -s -X PUT \
  -H "Authorization: Bearer $PS_ADMIN_TOKEN" \
  -H "content-type: application/json" \
  -d '{"enforcement_mode":"enforce","fail_on":"HIGH"}' \
  http://localhost:9090/v1/config
```

---

### Implementation Plan (Phased)

1) Router & scaffolding
- Create `internal/interfaces/http/api/` with router, helpers, stubs.
- Mount at `/v1` from enforcer mux.

2) Info & read-only
- GET `/v1/version`
- GET `/v1/rulepacks`, `/v1/rulepacks/active`
- GET `/v1/config`

3) Mutations (admin)
- POST `/v1/rulepacks:validate`, `/v1/rulepacks:reload`, PUT `/v1/rulepacks/active`
- PUT `/v1/config`, POST `/v1/config:reset`

4) Scans
- POST `/v1/scan` (aggregate JSON and NDJSON streaming)
- (Optional) Async jobs endpoints

5) Observability & admin
- GET `/v1/stats` (summarize Prometheus)
- GET `/v1/events` (SSE)
- POST `/v1/admin/drain`, `/v1/admin/shutdown`

6) OpenAPI & tests
- Update `docs/api/openapi.yaml` and add unit/integration tests under `gateway/`.

---

### Testing Strategy
- Unit tests for API handlers (table-driven in `internal/interfaces/http/api/`).
- Golden tests for JSON responses.
- Integration tests (httptest) for auth and error codes.
- Envoy e2e smoke tests with `docker-compose.yaml`.

---

### Backward Compatibility
- Keep legacy root `/check` for a deprecation window; offer `/v1/check` as the stable path.
- No changes to gRPC ext_proc contract.

---

### References
- HTTP enforcer mux and handlers: `internal/interfaces/http/enforcer/server.go`
- gRPC ext_proc server: `internal/interfaces/grpc/enforcer/server.go`
- Scanner/rules core: `internal/scanner/*`, `internal/rules/*`, `pkg/types/*`
- OpenAPI (current): `docs/api/openapi.yaml`
- Gateway tests: `gateway/http_test.go`, `gateway/grpc_test.go`



### Monetization & Licensing

#### Business Model
- Tiered licensing via signed tokens:
  - **Evaluation**: rate-limited, headers watermark, no premium features
  - **Pro**: full L1/L2, configurable policies, moderate RPS, basic support
  - **Enterprise**: high RPS, L3 semantics, async jobs, SSO/SAML, priority support

#### Current Capabilities in Code
- License check and evaluation limiter live in `internal/license/license.go` (ed25519 signature verification; 10 req/min eval bucket; headers/logging).
- HTTP `/check` enforces evaluation limit and sets `X-PromptShield-License` header in `internal/interfaces/http/enforcer/server.go`.

#### Licensing Token (ed25519)
- `PROMPTSHIELD_LICENSE_KEY` contains a base64url payload and signature joined by `.`.
- `PROMPTSHIELD_LICENSE_PUBLIC_KEY` provides base64url public key for verification.
- Suggested payload fields: `org`, `tier`, `expires_at`, `entitlements`.

Example (JSON payload before signing):
```json
{
  "org": "Acme Corp",
  "tier": "enterprise",
  "expires_at": "2030-01-01T00:00:00Z",
  "entitlements": {
    "max_rps": 2500,
    "features": {"l3_semantic": true, "async_jobs": true, "sso": true}
  }
}
```

#### Entitlements (Runtime Types)
```go
// internal/license/license.go
type Entitlements struct {
    MaxRPS   float64         `json:"max_rps"`
    Features map[string]bool `json:"features"`
}

type License struct {
    Organization string       `json:"org"`
    ExpiresAt    time.Time    `json:"expires_at"`
    Tier         string       `json:"tier"`
    Entitlements Entitlements `json:"entitlements"`
}
```

#### Gating Points
- Global RPS limit: prefer `entitlements.max_rps` to size the limiter in gRPC and HTTP paths (falls back to env `PS_ENFORCER_RPS`).
- Feature gates:
  - L3 semantics: `features.l3_semantic` required in `internal/scanner/semantic.go`
  - Async scans: `features.async_jobs` required for `/v1/scan:async`
  - SSO/Admin UX: `features.sso` required for future auth providers
- Headers: add `x-ps-license-tier` and `x-ps-license-expiry` on responses when licensed.

Gating example:
```go
// internal/scanner/semantic.go
if !license.IsLicensed() || !license.HasFeature("l3_semantic") {
    return Result{}, errors.New("feature not available: requires Pro or Enterprise license")
}
```

Apply MaxRPS in gRPC server:
```go
// internal/interfaces/grpc/enforcer/server.go (NewWithOptions)
if ent, ok := license.Entitlement(); ok && ent.MaxRPS > 0 {
    burst := 1
    if b := os.Getenv("PS_ENFORCER_RPS_BURST"); b != "" { /* parse */ }
    s.limiter = rate.NewLimiter(rate.Limit(ent.MaxRPS), burst)
}
```

#### License Endpoints (Admin)
- GET `/v1/license`
  - 200: `{ org, tier, expires_at, entitlements, licensed }`
- POST `/v1/license`
  - Admin-only; accepts form or JSON `{ key: "..." }`; verifies and activates in-memory (optional persist)

Example handlers:
```go
func getLicense() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        l := license.Info()
        _ = json.NewEncoder(w).Encode(map[string]any{
            "org": l.Organization,
            "tier": l.Tier,
            "expires_at": l.ExpiresAt,
            "entitlements": l.Entitlements,
            "licensed": license.IsLicensed(),
        })
    }
}

func setLicense() http.HandlerFunc { // admin
    return func(w http.ResponseWriter, r *http.Request) {
        key := r.FormValue("key")
        if key == "" {
            var body struct{ Key string `json:"key"` }
            _ = json.NewDecoder(r.Body).Decode(&body)
            key = body.Key
        }
        if key == "" { writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "missing key", nil); return }
        os.Setenv("PROMPTSHIELD_LICENSE_KEY", key)
        license.Check()
        w.WriteHeader(http.StatusNoContent)
    }
}
```

#### Usage Metering (Privacy-First)
- Emit minimal decision events (counts, bytes) to telemetry if `PS_TELEMETRY_ENDPOINT` is set; otherwise optional NDJSON sink.
- Summarize every N events or T seconds; if `PS_BILLING_ENDPOINT` is set, POST usage summaries (org, counts, bytes, tier).

#### Environment Variables
- `PROMPTSHIELD_LICENSE_KEY`: signed license token
- `PROMPTSHIELD_LICENSE_PUBLIC_KEY`: base64url public key
- `PS_ENFORCER_ADMIN_TOKEN`: admin API guard
- `PS_TELEMETRY_ENDPOINT` / `PS_TELEMETRY_FILE` / `PS_TELEMETRY_SAMPLE`: observability sinks
- `PS_BILLING_ENDPOINT` (optional): usage summary endpoint
- `PS_USAGE_SUMMARY_INTERVAL` (default 60s)

#### Example Requests
```bash
# Inspect license (read-only)
curl -s http://localhost:9090/v1/license

# Set/rotate license (admin token required)
curl -s -X POST \
  -H "Authorization: Bearer $PS_ADMIN_TOKEN" \
  -d '{"key":"<SIGNED_TOKEN>"}' \
  http://localhost:9090/v1/license
```

#### Rollout Checklist
- Extend `license.License` to include `Entitlements` and helpers: `HasFeature`, `Entitlement()`.
- Gate premium features and apply MaxRPS across HTTP/gRPC.
- Implement `/v1/license` endpoints (GET/POST) under admin auth.
- Add optional usage summaries and docs (pricing, setup, rotation).
- Tests: verification, gating behavior, rate-limit paths.
