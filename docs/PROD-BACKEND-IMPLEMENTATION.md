## PromptShield Production Backend: Rulepack Persistence, Distribution, and Gateway Live Enforcement

### Goals
- Durable, multi-tenant storage of user-authored DSL rulepacks (no built-ins).
- Versioned updates with audit and approval flows.
- Low-latency distribution to gateways; live enforcement with tiered escalation (keywords → regex → LLM).
- Backward-compatible with current mock gateway and demo scripts.

### High-Level Architecture
- Source of truth: PostgreSQL (JSONB rulepacks + relational metadata).
- Distribution bus: Redis Streams or NATS for low-latency fan-out; optional Kafka for CDC.
- Control Plane API: validates, versions, stores, publishes updates.
- Gateways: subscribe to updates, fetch assigned pack(s), enforce locally.

### Repo Structure (additions)
- `internal/infrastructure/persistence/postgres/` — PG adapters (rulepacks, versions, assignments, audits).
- `internal/application/services/` — RulepackService (validate, version, assign).
- `internal/interfaces/http/api/` — REST/JSON (and SSE) endpoints for rule lifecycle and enforcement.
- `internal/infrastructure/messaging/` — NATS/Redis publisher/subscriber.
- `migrations/` — SQL migrations.
- `gateway/main.go` — unified gateway binary (control plane + enforcement).
- `configs/` — config files (env, DSN, NATS/Redis URLs).

### Data Model (PostgreSQL)
```sql
-- migrations/0001_init.sql
CREATE TABLE tenants (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rulepacks (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT,
  current_version_id UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rulepack_versions (
  id UUID PRIMARY KEY,
  rulepack_id UUID NOT NULL REFERENCES rulepacks(id) ON DELETE CASCADE,
  version INT NOT NULL,
  dsl  JSONB NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('draft','approved','active','archived')),
  created_by UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(rulepack_id, version)
);

ALTER TABLE rulepacks
  ADD CONSTRAINT fk_current_version
  FOREIGN KEY (current_version_id) REFERENCES rulepack_versions(id);

CREATE TABLE assignments (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  rulepack_id UUID NOT NULL REFERENCES rulepacks(id) ON DELETE CASCADE,
  target_scope TEXT NOT NULL, -- e.g., 'global', 'env:prod', 'app:web', 'gateway:gw-123'
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE audits (
  id UUID PRIMARY KEY,
  tenant_id UUID,
  actor_id UUID,
  action TEXT NOT NULL,
  object_type TEXT NOT NULL,
  object_id UUID NOT NULL,
  diff JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Helpful indexes
CREATE INDEX idx_rulepack_versions_rulepack ON rulepack_versions(rulepack_id);
CREATE INDEX idx_rulepack_versions_status ON rulepack_versions(status);
CREATE INDEX idx_rulepack_versions_dsl_gin ON rulepack_versions USING GIN (dsl);
```

Optional: PostgreSQL Row-Level Security templates for tenant isolation.

### Control Plane API (HTTP)

Endpoints (implemented under `internal/interfaces/http/api/`):
- `POST /v1/rulepacks` — create rulepack (tenant_id, name, description)
- `POST /v1/rulepacks/{id}/versions` — upload DSL (YAML/JSON), validate against schema, create new version (status=draft)
- `POST /v1/rulepacks/{id}/versions/{ver}/approve` — approval flow
- `POST /v1/rulepacks/{id}/versions/{ver}/activate` — set as current_version_id; publish update event
- `POST /v1/assignments` — bind rulepack to target scope; publish update event
- `GET /v1/rulepacks/{id}` — metadata + current_version
- `GET /v1/rulepacks/{id}/versions/{ver}` — fetch DSL JSONB
- `GET /v1/stream` — server-sent events for updates (alt: gRPC stream)

Validation notes
- Use `github.com/santhosh-tekuri/jsonschema/v5` to validate the canonical JSON object.
- Reuse and extend schema from `tools/demos/mockgateway/main.go`.

### Distribution

Preferred: NATS or Redis Streams
- On activation or assignment change, control plane publishes `{tenant_id, target_scope, rulepack_id, version, checksum}` to `rulepacks.updates`.
- Gateways subscribe (by tenant/scope), compare checksum, fetch DSL if changed.

Example publisher (`internal/infrastructure/messaging/nats/publisher.go`):
```go
type RuleUpdate struct {
  TenantID    string `json:"tenantId"`
  TargetScope string `json:"targetScope"`
  RulepackID  string `json:"rulepackId"`
  Version     int    `json:"version"`
  Checksum    string `json:"checksum"`
}

func (p *Publisher) PublishRuleUpdate(ctx context.Context, u RuleUpdate) error {
  b, _ := json.Marshal(u)
  return p.js.PublishAsync("rulepacks.updates", b)
}
```

Gateway subscriber (pseudo; integrate into gateway bootstrap):
```go
func (g *Gateway) startRuleUpdates(ctx context.Context) error {
  _, err := g.js.Subscribe("rulepacks.updates", func(m *nats.Msg) {
    var u RuleUpdate; _ = json.Unmarshal(m.Data, &u)
    if u.TenantID != g.tenantID { return }
    // Check current assigned checksum; if different, fetch DSL
    go g.fetchAndApply(ctx, u)
  })
  return err
}

func (g *Gateway) fetchAndApply(ctx context.Context, u RuleUpdate) {
  dsl, checksum := g.gatewayClient.GetRulepack(ctx, u.RulepackID, u.Version)
  if checksum == g.cacheChecksum { return }
  cr, err := CompileRules(dsl) // reuse engine from `tools/demos/mockgateway/main.go`
  if err != nil { g.logger.Error("compile", "err", err); return }
  g.rules.Store(cr)
  g.cacheChecksum = checksum
}
```

### Persistence Layer (Postgres)

Interfaces (`internal/application/services/rulepacks.go`):
```go
type RulepackRepository interface {
  Create(ctx context.Context, tenantID uuid.UUID, name, desc string) (uuid.UUID, error)
  CreateVersion(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, status string, createdBy uuid.UUID) (uuid.UUID, error)
  GetActive(ctx context.Context, packID uuid.UUID) (json.RawMessage, int, error)
  Activate(ctx context.Context, packID, versionID uuid.UUID) error
}
```

Example PG adapter (`internal/infrastructure/persistence/postgres/rulepacks.go`):
```go
func (r *pgRulepackRepo) CreateVersion(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, status string, createdBy uuid.UUID) (uuid.UUID, error) {
  var id uuid.UUID
  err := r.db.QueryRowContext(ctx, `INSERT INTO rulepack_versions (id, rulepack_id, version, dsl, status, created_by)
    VALUES (gen_random_uuid(), $1, $2, $3, $4, $5) RETURNING id`, packID, version, dsl, status, createdBy).Scan(&id)
  if err != nil { return uuid.Nil, fmt.Errorf("create version: %w", err) }
  return id, nil
}
```

### Gateway Integration
- Replace mock file persistence with control plane client + subscriber.
- Keep local in-memory compiled rules; hot reload on events.
- Fallback: pull on startup; then stream for updates.
- Respect tenant context from config/env.

### LLM Escalation
- Control plane passes `llm` config in DSL as-is.
- Optional policy: per-tenant thresholds, classifier names.
- Gateway: if `LLM_EVAL_ENABLED=1`, evaluate post-rules (mock or live as configured).

### Observability & Security
- Control plane and gateway expose Prometheus metrics and OTel tracing.
- mTLS between gateway and control plane.
- Per-tenant RLS in Postgres; service account with limited privileges.
- Signed rulepack versions: store checksum/signature; gateways verify before apply.

### Rollout & Zero-Downtime
1. Introduce control plane service (readiness probes, health).
2. Add gateway client with pull-only; deploy.
3. Enable streaming; switch to push+pull hybrid.
4. Migrate demo to use control plane upload (in addition to `/v1/rules` on gateway during transition).

### Testing Strategy
- Unit: validation, compilation, matching engine (keywords/regex/LLM) — high coverage.
- Integration: PG repo + HTTP API + NATS/Redis distribution.
- End-to-end: spin up control plane + gateway; upload pack; assert enforcement.
- Load: measure P50/P95 with 100s of rules; ensure no CPU spikes.

### File References (current & to-be)
- Current gateway engine (mock): `tools/demos/mockgateway/main.go`
- Demo scripts: `tools/demos/*.sh`
- Example rulepacks: `docs/examples/rulepacks/*.yaml`
- New persistence: `internal/infrastructure/persistence/postgres/` (to implement)
- New services: `internal/application/services/` (to implement)
- Unified HTTP API: `internal/interfaces/http/api/` + `gateway/main.go` (implemented)
- Messaging: `internal/infrastructure/messaging/` (to implement)
- Migrations: `migrations/*.sql` (to add)

### Code Snippets

Validator setup (control plane):
```go
compiler := jsonschema.NewCompiler()
_ = compiler.AddResource("mem://dsl-schema.json", strings.NewReader(dslSchemaJSON))
schema, _ := compiler.Compile("mem://dsl-schema.json")
var js interface{}
_ = json.Unmarshal(canonicalJSON, &js)
if err := schema.Validate(js); err != nil { /* 400 */ }
```

SSE endpoint for updates (simplified):
```go
func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Content-Type", "text/event-stream")
  ch := h.bus.Subscribe(r.Context(), tenantTopic)
  for ev := range ch {
    fmt.Fprintf(w, "data: %s\n\n", ev.JSON)
    if f, ok := w.(http.Flusher); ok { f.Flush() }
  }
}
```

Gateway apply (redaction target example):
```go
// Inbound redaction target example
redact := redactConfig{fields: []string{"choices[].message.content"}, replacement: "[REDACTED]"}
body, _ = applyRedactions(body, redact.fields, redact.replacement)
```

### Phased Delivery
1. Migrations + PG repo + JSON schema validation (2–3 days)
2. Control plane REST (create/version/activate/assign) (3–4 days)
3. Messaging bus integration + gateway subscriber (3–4 days)
4. SSE as an interim (1–2 days)
5. Security (mTLS/JWT) + audits (2–3 days)
6. Load tests + tuning + documentation (2–3 days)

### Risks & Mitigations
- Very large rulepacks: paginate updates; gzip; chunked transfer.
- Regex catastrophic backtracking: sandbox/timeout, preflight checks.
- LLM latency spikes: circuit breaker; sample only where configured.


