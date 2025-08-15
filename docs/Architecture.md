### Architecture (Gateway + Enforcer)

High-level flow:

1) Envoy forwards requests to the enforcer via `ext_authz` (headers) and/or `ext_proc` (streaming bodies)
2) Enforcer loads RulePacks (YAML) and evaluates content using L1/L2/L3 with budgets
3) Decision engine returns ALLOW/QUARANTINE/DENY and sets decision headers; optional body mutation for redaction
4) Observability via Prometheus metrics and OpenTelemetry traces; audit events are hash‑chained
5) Audit & Events via append-only event store; orchestration via Saga pattern; optional chaos testing

Key packages:

- `internal/rules`: YAML schema types and loader for RulePacks (imports, extends/overrides, composition)
- `internal/scanner`: Streaming scanner and rule compilation/evaluation (configurable buffer/overlap; global regex cache for L2). Regex engine defaults to Go's RE2-compatible `regexp`; optional Hyperscan path is gated behind a build tag.
- `internal/interfaces/http/enforcer`: HTTP runtime `/healthz`, `/readyz`, `/metrics` and mounts versioned API under `/v1` including `/v1/check`, `/v1/scan`, async job endpoints, rulepack/config management, and admin endpoints. Legacy root `/check` remains supported for compatibility.
- `internal/interfaces/grpc/enforcer`: Envoy `ext_proc` gRPC server with streaming decisions, budgets, and metrics
- `internal/observability/telemetry`: Opt-in OpenTelemetry exporter; Prometheus registries
- `internal/audit`: File and rotating audit loggers with SHA‑256 hash chaining
- `internal/audit/eventstore`: Append-only event store (file-backed) for audit/event sourcing
- `internal/shared/redact`: Centralized redaction for sensitive tokens
- `pkg/types`: Public result and violation structs
 - `internal/application/commands/scan`: CLI command handler and Saga coordinator for multi-step scans
 - `internal/observability/chaos`: Fault injection controller (env-gated)

Streaming and parallel design:

- Sliding-window scanning for gRPC streams with overlap; bounded memory per stream
- Deterministic decision ordering; request correlation via request IDs
- Budgets: per-request timeout, max stream bytes, LLM concurrency and cache

Rule engine:

- Level 1: keyword evaluation with rule-level and global options (`case_sensitive`, `whole_word`)
- Level 2: precompiled regex patterns with simple flags (`i`, `m`) and multi-match emission; global cache keyed by pattern+flags
- Level 3: semantic analysis adapters (OpenAI/Anthropic) with caching, concurrency limits, and fallbacks
- Conditions: `when`/`unless` checked against runtime context; `logic: all` supported for L1/L2

Composition (`all_matches`/`first_match`/`priority_order`), overrides, performance hints, and pack imports/extends are implemented with deterministic merge and recursive import resolution.

Observability & UX:

- Decision headers: `x-ps-decision`, `x-ps-reason`, `x-ps-request-id`
- Metrics: HTTP `/metrics` and gRPC ext_proc process metrics `ps_extproc_streams_total{decision}`, `ps_extproc_bytes_total`, `ps_extproc_stream_duration_seconds{decision}`
- Tracing: use OpenTelemetry when `telemetry.endpoint` is set
- Audit: hash‑chained events with daily rotation

