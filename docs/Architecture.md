### Architecture (v0.2.0)

High-level flow:

1) CLI parses flags/env/config (Cobra/Viper) and dispatches to subcommands
2) `scan` loads RulePacks (YAML), merges CLI `--context` into pack contexts
3) `scanner.Scanner` compiles rules and performs streaming line-by-line scanning
4) A file-level worker pool scans files concurrently; results are merged deterministically by path
5) Matches are reported via `internal/report` in `stylish`, `json`, `github`, or `ndjson`

Key packages:

- `cmd/`: CLI entry, commands, flags, completion
- `internal/discovery`: Expands file args into a sorted set of absolute paths
- `internal/rules`: YAML schema types and loader for RulePacks (imports, extends/overrides, composition)
- `internal/scanner`: Streaming scanner and rule compilation/evaluation (configurable buffer/overlap; global regex cache for L2). Regex engine defaults to Go's RE2-compatible `regexp` (safest); optional Hyperscan path is gated behind a build tag and remains off by default.
- `internal/interfaces/http/enforcer`: HTTP runtime `/check` + `/metrics` with budgets and decision headers
- `internal/interfaces/grpc/enforcer`: Envoy `ext_proc` gRPC server with streaming decisions, budgets, and metrics
- `internal/report`: Renderers for stylish/json/github/ndjson (deprecated formats removed)
- `internal/audit`: File and rotating audit loggers with SHA-256 hash chaining
- `internal/shared/redact`: Centralized redaction for sensitive tokens
- `internal/observability/telemetry`: Opt-in OpenTelemetry exporter + optional NDJSON sink; coarse counters only
- `pkg/types`: Public result and violation structs

Streaming and parallel design:

- Uses `bufio.Scanner` with a configurable token buffer (`performance.buffer_bytes`, default ~16 MiB) to handle large JSONL/NDJSON lines
- Tracks bytes and lines in `Metrics` for observability
- Parallel per-file scanning via a worker pool sized by `workers` from config/env (0=auto `runtime.NumCPU()`)
- Deterministic output ensured via indexed results merge

Rule engine (implemented):

- Level 1: keyword evaluation with rule-level and global options (`case_sensitive`, `whole_word`)
- Level 2: precompiled regex patterns with simple flags (`i`, `m`) and multi-match emission; global cache keyed by pattern+flags
- Level 3: semantic analysis adapters (OpenAI/Anthropic) with caching, concurrency limits, and fallbacks
- Conditions: `when`/`unless` checked against merged runtime context; `logic: all` supported for L1/L2

Composition (`all_matches`/`first_match`/`priority_order`), overrides, performance hints, and pack imports/extends are implemented with deterministic merge and recursive import resolution (local/glob/network gated).

Configuration precedence:

1) Flags
2) Environment (`PS_*` with `-` mapped to `_`)
3) Config file: `promptshield.yaml`, `~/.config/promptshield/promptshield.yaml`, `~/.promptshield/promptshield.yaml`, or `XDG_CONFIG_HOME/promptshield/promptshield.yaml`
4) Built-in defaults

Observability & UX:

- Deterministic output ordering across files
- Progress: shown by default for human-readable output; suppressed for JSON/quiet
- Input validation: path traversal guard; configurable max file size per scan
- Metrics: per-file bytes/lines read; optional NDJSON summary via `metrics_file` (env/config). Enforcer metrics: HTTP `/metrics` and process-level gRPC metrics `ps_extproc_streams_total{decision}`, `ps_extproc_bytes_total`, `ps_extproc_stream_duration_seconds{decision}`.
- Tracing: use OpenTelemetry when `telemetry.endpoint` is set. `trace_file` is deprecated; prefer OTLP.
- Structured logs: `debug` (env/config) enables INFO/DEBUG logs to stderr; a `request_id` is attached to logs, traces, and audit entries for correlation; provider adapters log request/response summaries and cache hits (redacted)
- Audit: config-driven `audit_file` writes start/end and per-file events; rotating logger supports daily rotation; hash chaining uses SHA-256 over a canonical payload
- Config validation: unknown keys in `promptshield.yaml` are rejected with actionable errors
Discovery & safety:
- Path traversal guard rejects `..` components and null bytes
- Allow/Deny controls via `PS_ALLOW_PATHS` and `PS_DENY_PATHS` (comma-separated prefixes)
- Heuristic guard for overly broad globs to avoid explosive patterns

