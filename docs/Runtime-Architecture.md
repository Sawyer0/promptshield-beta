### PromptShield Runtime Architecture (ps-enforcer)

This document describes the production runtime architecture for the PromptShield enforcer service (`promptshield-enforcer`, alias `ps-enforcer`). It complements the CLI scanner and establishes a data-plane + control-plane pattern suitable for enterprise rollouts.

## High-level overview

```mermaid
flowchart LR
  subgraph Client Apps
    A1[Runtime Service\n(API/Web/Worker)]
    A2[CI/CD Jobs]
    A3[Batch Scans]
  end

  subgraph Sidecar Layer
    B1[Envoy Proxy\n(ext_authz + ext_proc)]
  end

  subgraph Data Plane (ps-enforcer)
    direction TB
    C1[Prefilter\nkeywords/regex AC automaton]
    C2[Vector Lookup\nANN risk fingerprints]
    C3{Decision\nALLOW | QUARANTINE | DENY}
    C4[LLM Guardrails\nBYOM adjudicator pool]
    C5[Budget/SLO Controller\ntimeouts, rate limits]
  end

  subgraph Control Plane
    D1[Policy Registry\nSigned RulePacks, channels]
    D2[Identity & Tenancy\nOIDC/SSO, RBAC]
    D3[Secrets Mgmt\nKMS/Vault]
    D4[Feature Flags & Canary]
  end

  subgraph Management (Console)
    E1[Policy lifecycle\nlint, sign, promote, rollback]
    E2[Runtime controls\nbudgets, provider on/off]
    E3[HITL Review Queue]
    E4[Dashboards\nSLOs, FP/FN, costs]
  end

  subgraph Storage & Observability
    F1[(Operational DB)]
    F2[(Vector Store\nClickHouse/Qdrant)]
    F3[(Audit Log\nhash-chained, append-only)]
    F4[(Evidence Store\nencrypted, TTL)]
    F5[(Metrics/TSDB\nOTel → Grafana)]
  end

  A1 --> B1
  B1 -->|check/stream| C1
  A2 -->|artifacts| C1
  A3 -->|batches| C1

  C1 --> C2 --> C3
  C3 -->|ALLOW| B1
  C3 -->|QUARANTINE| F3
  C3 -->|DENY| F3
  C3 -->|ESCALATE| C4 --> C5 --> C3

  D1 -->|hot-pull| C1
  D4 --> C3
  D2 --> E1
  D3 --> C4

  C1 -->|events| F3
  C3 -->|decisions| F3
  C4 -->|results| F3
  C1 -->|metrics| F5
  C4 -->|metrics| F5

  C4 -->|borderline| E3 -->|override/labels| F2
  E1 -->|promote/canary| D1
```

Key design: fast-first prefilter; escalate to semantic adjudicator only when thresholds/policies demand. Budgets cap latency and cost.

## Integration with Envoy

- Use `ext_authz` for header/context-only, fast ALLOW/DENY or to inject decision headers.
- Use `ext_proc` (External Processing filter) to stream request/response bodies to `ps-enforcer` for content inspection with budgets. Response bodies may be redacted in-place via `CommonResponse.body_mutation` when rules specify `response.action: redact|mask|quarantine`.

Minimal `ext_authz` snippet (HTTP service): see `docs` examples in repo. For response body scanning, add `ext_proc` pointing to the enforcer’s gRPC server (`envoy.service.ext_proc.v3.ExternalProcessor/Process`).

## Enforcer service surface

Two integration modes:

- Envoy `ext_proc` gRPC server (preferred for body streaming)
  - Service: `envoy.service.ext_proc.v3.ExternalProcessor`
  - Method: `Process(stream HttpBodyChunk) returns (stream ProcessingResponse)`
  - Policy: enforce per-request timeout and total `max_stream_bytes`; decision on first threshold hit via scanner-backed evaluation
  - Streaming: sliding-window scan with overlap (`PS_ENFORCER_STREAM_WINDOW`, `PS_ENFORCER_STREAM_OVERLAP`) to bound per-stream memory and avoid boundary misses
  - Backpressure: global stream slots (`PS_ENFORCER_MAX_STREAMS`) and token-bucket rate limiting (`PS_ENFORCER_RPS`/`_BURST`)

- HTTP endpoints (for CI/batch/sidecarless)
  - `GET /healthz` (liveness), `GET /readyz` (readiness gated on rulepack/policy load)
  - `POST /check` → quick allow/deny using headers/context and optional small payload
  - Headers set on success: `x-ps-decision: allow|quarantine|deny`, `x-ps-reason: <rule_id|rationale>`, `x-ps-request-id: <uuid>`
  - JSON response: `{ "decision": "allow|quarantine|deny", "reason": "...", "violations": <int>, "request_id": "..." }`
  - Budgets: request timeout and `PS_ENFORCER_MAX_BODY_BYTES` (default 1MiB)
  - `POST /scan` → full content scan (streaming upload), returns JSON report and decision

gRPC health:

- `grpc.health.v1.Health` is exposed; service statuses reflect overall readiness, ext_proc serving state, and redaction mutation feature readiness.

Decision headers (on ALLOW):

```
x-ps-decision: allow|quarantine|deny
x-ps-reason:  <rule_id or rationale>
```

On DENY/QUARANTINE (HTTP mode), return 403 with a structured JSON body (maps to CLI output schema).

## Policy model (RulePacks)

RulePacks control:
- Signals: keywords, regex, semantic tasks (analysis prompts)
- Thresholds: risk scores, similarity, confidence
- Budgets: `timeout_ms`, `max_llm_calls`, `max_p95_ms`, `max_cost_cents`
- Actions: `allow`, `quarantine`, `deny`, `escalate`
  - Response mapping: `response.action` supports `redact`, `replace`, `deny`/`block`, `quarantine`, `alert` (alert/replace staged; redact/block delivered baseline)
- Context gating: `when`/`unless` over tenant/route/env keys
- Composition: `first_match` vs `priority_order`

RulePacks are signed and versioned in the policy registry; `ps-enforcer` hot-pulls bundles (ETag/If-None-Match) and validates signatures before activation.

## Decision engine

1) Prefilter (streaming): Aho–Corasick keyword matcher and precompiled regex with per-line and per-file limits. Deterministic order retained.
2) Vector lookup (optional): approximate nearest neighbor against “risk fingerprints” for similarity-based flags.
3) Decision: evaluate thresholds; if borderline or policy demands, escalate.
4) LLM guardrails (escalation): call adjudicator with strict JSON schema; enforce timeouts, concurrency, and cost budgets.
5) Action: ALLOW, QUARANTINE, or DENY. Emit decision headers and audit/metrics.

## Budgets and SLOs

- Per-request: `timeout_ms`, `max_llm_calls`, `max_stream_bytes`, enforcement mode (`PS_ENFORCER_ENFORCEMENT_MODE`)
- Per-tenant budgets: p95 latency ceiling, daily cost caps
- Circuit breaker: fail-safe modes (e.g., `on_error: pass_with_quarantine`)
- Targets (initial): p95 ≤ 300ms without escalation; p95 ≤ 700ms with one escalation

### Enforcement modes

- `observe`: never block; emit headers/metrics only
- `redact`: allow but mutate body with redaction when applicable (`PS_ENFORCER_REDACTION_MUTATION=true`)
- `quarantine`/`enforce`: block and annotate with decision headers; provenance headers optional

## Tenancy and identity

- Read `x-tenant-id` and map to policy channels and budgets
- Support OIDC service accounts for control-plane APIs; RBAC for management plane

## Observability and audit

- Metrics: OTel counters/histograms (files scanned, p50/p95/p99, escalation rate, cost). Prometheus: HTTP `/metrics` for HTTP enforcer; gRPC ext_proc exports process-level metrics `ps_extproc_streams_total{decision}`, `ps_extproc_bytes_total`, `ps_extproc_stream_duration_seconds{decision}`.
- Tracing: spans for prefilter, vector lookup, LLM calls
- Audit: hash-chained NDJSON with daily rotation; events for `config_effective`, `scan_start`, `scan_file`, `scan_summary`, `scan_end`
- Redaction: enabled by default; set `redaction.enabled: false` (or `PS_REDACTION_ENABLED=false`) to disable, which prints a security warning

## Security posture

- mTLS (Envoy ↔ enforcer); service tokens for CI/batch
- No secrets in logs/audit; centralized redaction
- Signed RulePacks; publisher allowlist/revocation
- Input limits: max body size, line length, streaming backpressure
- Rate limits and per-tenant circuit breakers

## Deployment topologies

1) Sidecar in pod: Envoy + `ps-enforcer` alongside app for low latency
2) Node or cluster service: central `ps-enforcer` with Envoy over mTLS
3) CI/Batch: call HTTP `/scan` directly (no Envoy)

Scale horizontally; enforcer instances are stateless (policies cached; state in external stores).

## Configuration (service)

YAML/env keys (illustrative):

```
enforcer:
  listen: 0.0.0.0:9090
  workers: 0              # 0=NumCPU
  redaction_enabled: true
  budgets:
    timeout_ms: 300
    max_llm_calls: 1
    max_stream_bytes: 5_000_000
  semantic:
    enabled: false
    provider: openai|anthropic
    max_concurrency: 2
    cache_size: 1000
    cache_ttl: 15m
policy:
  source: file|https
  file: rules/prod.yaml
  https_endpoint: https://registry/packs/prod
  verify_signatures: true
tenancy:
  header: x-tenant-id
tls:
  mtls_enabled: true
  ca_file: /etc/ps/ca.pem
  cert_file: /etc/ps/tls.crt
  key_file: /etc/ps/tls.key
```

## Failure modes and defaults

- Policy load failure: serve last-known-good; emit audit + metric
- LLM provider errors: honor `fallback_on_error`; use regex fallbacks; budget controller records cost/latency
- Timeouts: abort escalation; derive decision from prefilter results and policy

## Testing strategy

- Unit: prefilter, rule compilation, context gating, budgets, redaction
- Integration: Envoy `ext_authz` ALLOW/DENY; `ext_proc` streaming with chunk limits; signed-bundle activation
- Performance: 1 GiB streaming scan; concurrency and p95 under budgets
- Chaos: provider timeouts, registry outages, policy rollback

## Roadmap

- Full ext_proc gRPC server implementation with backpressure and chunk budgets
- Signed bundle registry and cosign verification
- Vector store integration for risk fingerprints (ClickHouse/Qdrant)
- HITL queue (API + console) with label feedback loop
- Multi-region tenancy and data residency controls

## References

- Envoy ext_authz: `https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/ext_authz_filter`
- Envoy ext_proc: `https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/ext_proc_filter`
- OPA bundles: `https://openpolicyagent.org/docs/management-bundles`
- OWASP LLM Top 10: `https://owasp.org/www-project-top-10-for-large-language-model-applications/`


