## PromptShield Tech Stack (v0.2.0)

This document describes the complete technical stack used by PromptShield, derived from the codebase at v0.2.0. Versions are taken from `go.mod` unless noted otherwise.

## Language, Runtime, Build

- **Language**: Go 1.24.6 (`go.mod`)
- **Build system**: Make (`Makefile`), Go toolchain
  - `make build-enforcer` builds `ps-enforcer` with ldflags
  - `make test`, `make bench`, quick/large benches
- **Containerization**: Multi-stage Dockerfiles (`Dockerfile`, `Dockerfile.enforcer`)
- **Orchestration**: Kubernetes manifests (`deployments/kubernetes/enforcer.yaml`)
- **Compose**: `docker-compose.yaml` for local Envoy + Enforcer + Backend

## Configuration

- **Config management**: `spf13/viper v1.20.1`
  - Primary configuration via environment (`PS_*`) and service YAML
  - Validates unknown config keys and provides suggestions where applicable

## Core Architecture (Code Layout)

- `internal/bootstrap`: Dependency wiring (logger, scanner, telemetry, semantic adapters)
- `internal/config`: Typed config, defaults, validation, unknown key checks
- `internal/rules`: RulePack schema, loader, imports/extends, merge, validation
- `internal/scanner`: Streaming scanner, compilation and evaluation (L1/L2/L3)
- `internal/observability`: Telemetry, metrics, tracing
- `internal/audit`: Structured audit events with hash chaining + rotation
- `internal/interfaces/http|grpc/enforcer`: Runtime enforcement services
- `pkg/types`: Public API types (stable)

## Rule Engine

- **Levels**:
  - Level 1: Keyword matching (Aho-Corasick), case/word-boundary options
  - Level 2: Regex patterns (Go RE2); compiled with global cache
  - Level 3: Semantic/LLM analysis (opt-in; providers below)
- **Composition**: `all_matches` (default), `first_match`, `priority_order`
- **Context gating**: `when`/`unless` using merged runtime context
- **Validation**: Strict YAML schema validation with actionable errors
- **Schema**: `internal/rules/schema.json` with JSON Schema via `santhosh-tekuri/jsonschema/v5 v5.3.1`

## Streaming Scanner & Performance

- **Design**: Streaming-first; never loads whole files; deterministic ordering
- **Concurrency**: File-level worker pool sized by `workers` (0 = auto) using `sourcegraph/conc`
- **Deterministic merge**: Results merged in path-sorted order
- **Large lines**: Configurable token buffer (`performance.buffer_bytes`) and chunk overlap
- **Pre-filters**: Optional Bloom filter (`bits-and-blooms/bloom/v3 v3.6.0`)
- **Keyword engine**: `cloudflare/ahocorasick`
- **Regex engine**: Go RE2; optional Hyperscan fast‑path via build tags (`flier/gohs v1.2.3`).
  - Docker: enable with build arg `ENABLE_HYPERSCAN=1` (multi‑arch Linux base installs `libhyperscan5`).
  - Windows/macOS users build and run via Docker; no native Hyperscan install required on the host.

## Semantic Analysis (Level 3)

- **Providers**:
  - OpenAI: `github.com/openai/openai-go v1.12.0`
  - Anthropic: `github.com/anthropics/anthropic-sdk-go v1.9.1`
- **Wiring**: Enabled only when `PS_SEMANTIC_ENABLED=true` and `PS_SEMANTIC_PROVIDER` is set (`openai` or `anthropic`)
- **API keys**: Resolved via OS keyring (preferred) or env (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`)
- **Concurrency & cache**: Tuned by `PS_SEMANTIC_MAX_CONCURRENCY`, `PS_SEMANTIC_CACHE_SIZE`, `PS_SEMANTIC_CACHE_TTL`
- **Rule requirements**: RulePacks must specify `semantic.model` and `semantic.analysis_prompt`; no built-in defaults

## Safety Guards

- **Path safety**: Null-byte/path traversal rejection
- **Allow/Deny**: `PS_ALLOW_PATHS`, `PS_DENY_PATHS` prefix filters (legacy file scanning support)

## Data Formats & Encoding

- **JSON encoder**: Pluggable `encoding/json` or `goccy/go-json v0.10.5` via `internal/encoding/jsonx`
  - Select with `PS_JSON_ENCODER=std|fast`
- **NDJSON**: Optional streamed event writer for large runs
- **Gateway responses**: JSON decision payloads + headers

## Observability

- **Logging**: `log/slog` with text or JSON handlers; control via env (e.g., `PS_DEBUG`/log level)
- **Redaction**: Central redaction (`internal/shared/redact`) with verifiers (e.g., Luhn for cards)
- **Metrics**:
  - Prometheus client `client_golang v1.23.0`
  - Gateway: coarse counters via OTel metrics; optional NDJSON sink
  - Enforcer HTTP: `/metrics` endpoint
  - gRPC ext_proc: counters/histograms (streams, duration, bytes, rule hits, redactions)
- **Tracing**: OpenTelemetry SDK v1.37.0; `otelhttp` and `otelgrpc` instrumentation
  - Enabled with `telemetry.enabled` and `telemetry.endpoint` (OTLP/GRPC)
  - Sampling via `telemetry.sample` (0–1)
- **Dashboards**: `monitoring/dashboards/promptshield-enforcer.json`

## Audit Logging

- **Format**: JSON events with SHA-256 hash chaining (tamper-evident)
- **Writers**: File logger and rotating logger (`lumberjack v2.2.1`)
- **Sanitization**: All event payloads sanitized through redaction before hashing/writing
- **Config key**: `audit_file`

## Runtime Services (Enforcer)

- **HTTP Enforcer** (`internal/interfaces/http/enforcer`)
  - Framework: `go-chi/chi v5`
  - Endpoints: `/healthz`, `/readyz`, `/check`, `/metrics`
  - Tracing: `otelhttp` handler wrapper; emits `x-ps-trace-id` when available
  - Decisions: allow/quarantine/deny (stub), emits headers `x-ps-decision`, `x-ps-reason`
  - TLS/mTLS: Optional via `PS_ENFORCER_TLS_*` env
  - Licensing: Evaluation vs licensed headers via `internal/license`
- **gRPC Enforcer (Envoy ext_proc)** (`internal/interfaces/grpc/enforcer`)
  - API: Envoy `ext_proc v3` (`envoyproxy/go-control-plane`)
  - Interceptors: recovery + logging + `otelgrpc` stats handler
  - Streaming: Sliding windows with overlap for cross-chunk matching; rate limiter; inflight bytes ceiling; global stream slots
  - Decisions: threshold-based + response-action aware; optional redaction mutations for response chunks
  - TLS/mTLS: Optional via `PS_ENFORCER_GRPC_TLS_*` env

## Security & Privacy

- **Secrets storage**: OS keyring for provider API keys (`99designs/keyring v1.2.2`)
- **Redaction**: Tokens and secrets masked everywhere (logs, audit)
- **Input validation**: Path traversal, null-byte, symlink policy, broad glob guard
- **Network**: TLS/mTLS support in HTTP/gRPC enforcers
- **License-aware limits**: Evaluation-mode rate limiting headers in HTTP enforcer

## Testing & Quality

- **Testing**: Standard `go test`; table-driven tests across `internal/*`
- **Fuzzing**: Present for discovery/rule validation
- **Benchmarks**: Scanner performance benches (including 1GiB scenarios)
- **Assertions**: `stretchr/testify v1.10.0`
- **Linting**: `golangci-lint` target in `Makefile`

## Distribution & Deployment

- **Binaries**: `bin/ps-enforcer`
- **Docker images**: Multi-stage builds, non-root user, rules baked under `/rules`
- **Kubernetes**: Deployment, Service, HPA, PDB, and optional ServiceMonitor
- **Envoy**: Example ext_proc integration (`Envoy.md`, `envoy-config.yaml`)

## Public API Surface

- **Package**: `pkg/types` – stable types for scan results and violations

## Key Environment Variables

- Core:
  - `PS_WORKERS`, `PS_DEBUG`, `PS_REDACTION_ENABLED`
  - `PS_ALLOW_PATHS`, `PS_DENY_PATHS`, `PS_ALLOW_SYMLINKS`
  - `PS_JSON_ENCODER=std|fast`
- Telemetry:
  - `PS_TELEMETRY=1`, `PS_TELEMETRY_ENDPOINT`, `PS_TELEMETRY_FILE`, `PS_TELEMETRY_SAMPLE`
- Semantic (L3):
  - `PS_SEMANTIC_ENABLED=true`, `PS_SEMANTIC_PROVIDER=openai|anthropic`
  - `OPENAI_API_KEY` or `ANTHROPIC_API_KEY` (or saved to OS keyring)
  - Tuning: `PS_SEMANTIC_MAX_CONCURRENCY`, `PS_SEMANTIC_CACHE_SIZE`, `PS_SEMANTIC_CACHE_TTL`
- Enforcer (HTTP):
  - `PS_ENFORCER_ADDR`, `PS_ENFORCER_RULEPACK`, `PS_ENFORCER_AUTH_TOKEN`
  - TLS: `PS_ENFORCER_TLS_CERT`, `PS_ENFORCER_TLS_KEY`, `PS_ENFORCER_TLS_CLIENT_CA`
  - Body limits: `PS_ENFORCER_MAX_BODY_BYTES`
- Enforcer (gRPC ext_proc):
  - `PS_ENFORCER_GRPC_ADDR`
  - TLS: `PS_ENFORCER_GRPC_TLS_CERT`, `PS_ENFORCER_GRPC_TLS_KEY`, `PS_ENFORCER_GRPC_TLS_CLIENT_CA`
  - Streaming tunables: `PS_ENFORCER_STREAM_WINDOW`, `PS_ENFORCER_STREAM_OVERLAP`
  - Rate limit: `PS_ENFORCER_RPS`, `PS_ENFORCER_RPS_BURST`
  - Inflight limits: `PS_ENFORCER_INFLIGHT_LIMIT_BYTES`, `PS_ENFORCER_INFLIGHT_BACKOFF_MS`
  - Mutation: `PS_ENFORCER_REDACTION_MUTATION=true|false`

## Notable Third-Party Libraries (selected)

- Config: `viper v1.20.1`
- Observability: `go.opentelemetry.io/* v1.37.0`, `otelhttp v0.62.0`, `otelgrpc v0.62.0`, `prometheus/client_golang v1.23.0`
- HTTP & gRPC: `go-chi/chi v5.0.12`, `google.golang.org/grpc v1.74.2`, `envoyproxy/go-control-plane v1.32.4`
- JSON: `goccy/go-json v0.10.5`
- Patterns & Scanning: `cloudflare/ahocorasick`, `flier/gohs v1.2.3` (optional), `bits-and-blooms/bloom/v3 v3.6.0`
- Discovery: `bmatcuk/doublestar/v4 v4.9.1`, `go-git` gitignore matcher
- Utilities: `sourcegraph/conc`, `hashicorp/golang-lru/v2`, `hashicorp/go-retryablehttp`
- Semantics: `openai-go v1.12.0`, `anthropic-sdk-go v1.9.1`
- Security & Storage: `99designs/keyring v1.2.2`, `lumberjack v2.2.1`
- Validation: `santhosh-tekuri/jsonschema/v5 v5.3.1`

## References

- RulePacks: `docs/RulePacks.md` and `internal/rules/README.md`
- Output formats: `docs/Output.md`
- Architecture overview: `docs/Architecture.md`
- Envoy integration: `docs/Envoy.md` and `docs/ENVOY_INTEGRATION.md`


