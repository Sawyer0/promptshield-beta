### CLI reference

The `promptshield` CLI uses Cobra with global flags and subcommands.

Global flags:

- `--config <path>`: Config file path (default: `promptshield.yaml` or `~/.promptshield/promptshield.yaml`)
- `--output-format <stylish|json|github|ndjson>`: Output format (default: `stylish`)
- `--json`: Shorthand for `--output-format=json`
- Workers: configure via `PS_WORKERS` env or `workers:` in `promptshield.yaml` (0=auto; file-level concurrency)
- Debug: configure via `PS_DEBUG=true` or `debug: true` in config (structured logs to stderr)
- `--quiet`: Reduce log verbosity (errors only) and suppress human-readable output
- Redaction: configure via `PS_REDACTION_ENABLED` or `redaction.enabled` in config (default true). If disabled, a security warning is printed to stderr.

Environment variables (highest precedence after flags):

- `PS_OUTPUT_FORMAT`: Overrides output format
- `PS_WORKERS`: Overrides worker count
 - `PS_SEMANTIC_ENABLED`: `true|false` enable Level‑3 semantic analysis (opt‑in)
 - `PS_SEMANTIC_PROVIDER`: required when L3 is enabled; one of `openai` or `anthropic`
 - For OpenAI: `OPENAI_API_KEY` or `PS_OPENAI_API_KEY`
 - For Anthropic: `ANTHROPIC_API_KEY` or `PS_ANTHROPIC_API_KEY`
 - `PS_SEMANTIC_MAX_CONCURRENCY`: Max in‑flight semantic requests (default 2)
 - `PS_SEMANTIC_CACHE_SIZE`: Cache entries for semantic results (default 1000)
 - `PS_SEMANTIC_CACHE_TTL`: Cache TTL (e.g., `15m`)
 - `PS_ALLOW_PATHS` / `PS_DENY_PATHS`: Comma‑separated path prefixes for discovery allow/deny filtering
 - Telemetry (opt‑in):
   - `PS_TELEMETRY=1`: Enable telemetry collectors
   - `PS_TELEMETRY_ENDPOINT`: OTLP gRPC endpoint (e.g., `otel-collector:4317`)
   - `PS_TELEMETRY_FILE`: Local NDJSON sink file for coarse events
   - `PS_TELEMETRY_SAMPLE`: Sampling rate 0..1 (default 1)
  - `PS_HYPERSCAN`: `true|false` enable Hyperscan fast‑path when the binary was built with Hyperscan support (Docker build arg `ENABLE_HYPERSCAN=1`)
 - `PS_REDACTION_ENABLED`: `true|false` global toggle for log/audit redaction (defaults to true)

Notes:
- If L3 is enabled and any loaded RulePack contains Level‑3 rules, both `PS_SEMANTIC_PROVIDER` and the corresponding API key must be set or the scan will error.
 - In CI environments, output defaults to JSON unless you explicitly set `--output-format`.
 - In CI environments, `--fail-on` defaults to `INFO` unless explicitly set.

Configuration file keys (YAML):

- `output_format: stylish|json|github|ndjson` (stable set)
- `workers: <int>`
 - `rulepack: <path>` (default RulePack path)
  - `composition.strategy: all_matches|first_match|priority_order`
 - `performance.max_length: <int>`
 - `performance.buffer_bytes: <int>` (scanner buffer for long lines; default 16MiB)
 - `performance.chunk_overlap: <int>` (overlap bytes for very long lines; default 8KiB)
 - `performance.timeout: <duration>`
 - `performance.per_rule_timeout: <duration>`
 - `performance.total_scan_timeout: <duration>`
 - `performance.case_sensitive: <bool>` (default keyword behavior)
 - `performance.whole_word: <bool>` (default keyword behavior)
  - `audit_file: <path>` (write audit events; rotates daily when used with the rotating logger)
  - `telemetry.enabled: <bool>` (can also be enabled via `PS_TELEMETRY=1`)
  - `telemetry.endpoint: <host:port>` (OTLP gRPC exporter; enables traces/metrics)
  - `telemetry.file: <path>` (local NDJSON sink for coarse events)
  - `telemetry.sample: <float 0..1>` (sampling rate)

#### Commands

1) `promptshield scan [path|glob]...` (alias: `s`)

Scans one or more inputs. Each positional argument may be a file, a directory (walked recursively with vendor dir skips), or a glob pattern. Results are printed per file. Scanning is parallelized across files and output is merged deterministically in path order.

Flags:

- `--rulepack <file|dir>`: Path to a RulePack YAML file or a directory of RulePacks
- `--context key=value` (repeatable): Merges into each pack’s `context` for evaluating `when` / `unless`
- `--fail-on <severity>`: Exit non-zero if any violation meets/exceeds severity (`INFO|WARNING|HIGH|ERROR|CRITICAL`)
- Metrics file: configure via `PS_METRICS_FILE` or `metrics_file` in config (one-line NDJSON summary) [experimental]
- Trace file: configure via `PS_TRACE_FILE` or `trace_file` in config (one object per line) [experimental]
- Audit file: configure via `PS_AUDIT_FILE` or `audit_file` in config [experimental]
  (redaction is controlled via env/config; no CLI flag)
 - `--demo`: Run a quick demo scan using bundled samples
 - `--no-hints`: Suppress post-scan next steps in stylish mode
 - `-r, --rulepack <file|dir>` shorthand `-r`

Progress:
- Progress is shown by default for human-readable output; suppressed for `--json` and `--quiet`.

Examples:

```bash
promptshield scan --rulepack rules/ ./data/*.txt
promptshield --json scan --rulepack rules input.txt
promptshield scan --rulepack rules --context env=prod --context team=appsec input.txt
PS_WORKERS=8 promptshield scan --rulepack rules ./logs ./prompts
PS_METRICS_FILE=run.ndjson PS_TRACE_FILE=spans.ndjson promptshield scan --rulepack rules --fail-on=ERROR input.txt
# Alternate outputs
promptshield scan --output-format=github --rulepack rules input.txt

Quickstart (under 2 minutes):

```bash
# 1) Initialize config (auto-selects single RulePack under rules/ if present; else creates demo rules)
promptshield init

# 2) Run your first scan (use your own file or the demo data if generated)
promptshield scan:file demo/clean-prompts.json
```
# Composition examples
promptshield scan --rulepack rules input.txt            # default merge (all_matches)
# In pack YAML:
# composition:
#   strategy: first_match
# Or via config:
# composition:
#   strategy: priority_order
# Enable semantic with OpenAI
PS_SEMANTIC_ENABLED=true PS_SEMANTIC_PROVIDER=openai OPENAI_API_KEY=... \
  promptshield scan --rulepack rules input.txt
# Enable semantic with Anthropic
PS_SEMANTIC_ENABLED=true PS_SEMANTIC_PROVIDER=anthropic ANTHROPIC_API_KEY=... \
  promptshield scan --rulepack rules input.txt
```

First‑run helper:

- If no `promptshield.yaml` exists and `rules/` contains exactly one `.yaml`/`.yml` file, the CLI will scaffold `promptshield.yaml` pointing to that pack so you can immediately run:
  ```bash
  promptshield s path/**/*.json
  ```

2) `promptshield scan:file [path|glob]...` (alias: `sf`)

Same behavior as `scan` for files/globs.

3) `promptshield scan:directory [dir]...` (alias: `sd`)

Recursively scans directories.

4) `promptshield rules:create` (alias: `rc`)

Create a skeleton RulePack. Flags: `--dest`, `--force`.

5) `promptshield rules:list` (alias: `rl`)

List rules in a pack or directory. Flag: `--path` (default `rules`).

6) `promptshield rules:validate`

Validate packs with helpful messages. Flags: `--path`, `--json`.

7) `promptshield validate <rulepack|dir>` (alias: `v`)

Validates RulePack YAML(s). Enforces:
 - Required headers (`apiVersion`, `kind`, `metadata.name`)
 - Unique rule IDs
 - Level‑specific requirements (L1 keywords, L2 patterns, L3 semantic model + analysis_prompt)
 - Regex flag correctness

8) `promptshield config print`

Print the effective configuration as JSON.

9) `promptshield demo`

Prepare demo files and run a guided demo.

10) `promptshield benchmark [path|glob]...` (aliases: `bench`, `b`)

Simple scan benchmark. Respects `--rulepack` and `--context`.
Quick performance checks:
  - Make target `bench-quick` runs a focused p95 L1/L2 latency benchmark and asserts p95 ≤ 25ms:
    ```bash
    make bench-quick
    ```
  - Full suite: `make bench`
  - 1GiB streaming test: `make bench-large`

11) `promptshield version`

Prints version, commit, and build date. Values are injected at build time via `-ldflags`.

12) `promptshield auth`

Manage provider credentials using the OS keychain.

- `promptshield auth set --provider openai` → prompts for and stores `OPENAI_API_KEY`
- `promptshield auth set --provider anthropic` → prompts for and stores `ANTHROPIC_API_KEY`

The CLI prefers keys from the OS keychain; it falls back to environment variables if no key is stored.

## ps-enforcer (runtime)

`ps-enforcer` provides HTTP and gRPC (Envoy `ext_proc`) surfaces for runtime enforcement. It streams request/response bodies with bounded memory, applies policy decisions, and can mutate content (redaction/replacement) when enabled.

Environment variables:

```
# Endpoints
PS_ENFORCER_ADDR=:9090                  # HTTP server address
PS_ENFORCER_GRPC_ADDR=:9091             # gRPC ext_proc address

# RulePack & policy
PS_ENFORCER_RULEPACK=rules/prod.yaml    # RulePack to load
PS_ENFORCER_FAIL_ON=HIGH                # severity threshold: INFO|WARNING|HIGH|ERROR|CRITICAL
PS_ENFORCER_ENFORCEMENT_MODE=observe    # observe|redact|quarantine|enforce
PS_ENFORCER_REDACTION_MUTATION=true     # apply BodyMutation for redaction when applicable

# Budgets and streaming
PS_ENFORCER_TIMEOUT=300ms               # per-request processing timeout
PS_ENFORCER_MAX_STREAM_BYTES=5000000    # max total bytes per gRPC stream
PS_ENFORCER_STREAM_WINDOW=65536         # sliding window size (bytes)
PS_ENFORCER_STREAM_OVERLAP=4096         # overlap between windows (bytes)

# Backpressure and rate limits
PS_ENFORCER_MAX_STREAMS=50              # global concurrent stream cap
PS_ENFORCER_RPS=100                     # token-bucket rate (requests/second)
PS_ENFORCER_RPS_BURST=20                # burst size

# TLS/mTLS (HTTP)
PS_ENFORCER_TLS_CERT=/path/server.crt
PS_ENFORCER_TLS_KEY=/path/server.key
PS_ENFORCER_TLS_CLIENT_CA=/path/ca.crt  # enables mTLS validation

# TLS/mTLS (gRPC)
PS_ENFORCER_GRPC_TLS_CERT=/path/server.crt
PS_ENFORCER_GRPC_TLS_KEY=/path/server.key
PS_ENFORCER_GRPC_TLS_CLIENT_CA=/path/ca.crt
```

Endpoints:

- `/healthz` — liveness
- `/readyz` — readiness (true when RulePack/policy loaded)
- `/check` — minimal allow/quarantine/redact for small payloads
- `/metrics` — Prometheus metrics (HTTP enforcer)

gRPC health:

- `grpc.health.v1.Health` exposed; service statuses reflect ext_proc and feature readiness

gRPC ext_proc decision headers:

```
x-ps-decision: allow|quarantine|deny
x-ps-reason:  <rule_id|timeout|body_limit|no_signals>
```

gRPC ext_proc metrics (process-level):

- `ps_extproc_streams_total{decision}`
- `ps_extproc_bytes_total`
- `ps_extproc_stream_duration_seconds{decision}`

#### Completion

Shell completion is provided via Cobra’s `completion` command. The `scan` command also offers positional file-extension filtering and `--rulepack` flag file completion for `.yaml` / `.yml`.


## Kubernetes (Helm) Quick Start

```yaml
# values.yaml (minimal)
image:
  repository: ghcr.io/promptshield/enforcer
  tag: v0.2.0
  pullPolicy: IfNotPresent

env:
  # Rulepack location
  PS_ENFORCER_RULEPACK: /rules/basic-security.yaml
  # Streaming performance and backpressure (minimal safe defaults)
  PS_ENFORCER_STREAM_WINDOW: "65536"
  PS_ENFORCER_STREAM_OVERLAP: "4096"
  PS_ENFORCER_MAX_STREAMS: "50"           # global concurrent streams
  PS_ENFORCER_RPS: "100"                  # token-bucket requests/sec
  PS_ENFORCER_RPS_BURST: "20"
  # Optional global inflight memory ceiling
  PS_ENFORCER_INFLIGHT_LIMIT_BYTES: "67108864"   # 64MB across streams
  PS_ENFORCER_INFLIGHT_BACKOFF_MS: "5"           # admission backoff

service:
  type: ClusterIP
  ports:
    http: 9090
    grpc: 9091

volumeMounts:
  - name: rules
    mountPath: /rules
volumes:
  - name: rules
    configMap:
      name: promptshield-rules
```

