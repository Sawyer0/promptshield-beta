## PromptShield – CLI scanner for LLM safety (v0.2.0)

PromptShield is a streaming-first CLI that scans prompts and responses for security and safety risks using a progressive rule system:

- Level 1: Keyword matching
- Level 2: Regex patterns
- Level 3: Semantic/LLM analysis (opt-in)

It is designed with developer experience, performance, and enterprise readiness in mind.

### Quickstart

1) Build from source

```bash
make build
./bin/promptshield --version
```

2) Run a scan

```bash
./bin/promptshield scan --rulepack rules path/to/file.txt
```

3) JSON output

```bash
./bin/promptshield --json scan --rulepack rules path/to/file.txt
```

4) Validate rule packs

```bash
./bin/promptshield validate rules/
```

## ⚠️ Production Readiness

| Feature | Status | Safe for Production? | Notes |
|---------|--------|---------------------|-------|
| CLI Scanning | ✅ Stable | **Yes** | Core functionality works reliably |
| 3-Tier Rules | ✅ Stable | **Yes** | Keywords, regex, semantic analysis |
| Semantic Analysis | ✅ Stable | **Yes** | OpenAI & Anthropic providers with caching |
| Output Formats | ✅ Stable | **Yes** | JSON, stylish, github, ndjson |
| Configuration | ✅ Stable | **Yes** | Flags, env vars, config files |
| Audit Logging | ⚠️ Beta | **Limited** | Uses SHA-256 hash chain; schema may evolve |
| ps-enforcer | ⚠️ Experimental | **Limited** | HTTP `/check` and gRPC `ext_proc`; budgets and decision headers; not production-ready |
| RulePack extends | ✅ Supported | **Yes** | Inheritance and overrides via deterministic merge |
| RulePack composition | ✅ Supported | **Yes** | `all_matches` (default), `first_match`, `priority_order` |
| Rule response actions | ⚠️ Accepted | **Limited** | Parsed and passed through; enforcement actions are not applied by CLI scan |

### Security Notes

**Current limitations for production use:**
// ... unchanged lines ...
- **ps-enforcer**: Experimental gRPC ext_proc streaming enforcer with budgets and scanner-backed decisions; not for production access control yet

**Recommended for production:**
- ✅ Use CLI scanning for batch analysis and CI/CD pipelines
- ✅ Use semantic analysis for advanced threat detection
- ✅ Validate RulePacks before deployment (automatic validation prevents broken configs)
- ❌ Don't rely on audit trails for security compliance yet
- ❌ Don't use ps-enforcer for real enforcement yet

### Features implemented

- Streaming scanner (bounded memory) with file-level parallelism and deterministic output ordering; configurable buffer (`performance.buffer_bytes`) and chunk overlap (`performance.chunk_overlap`)
- File discovery from paths, directories (recursive with common vendor dir skips), and glob patterns
- RulePacks with Level 1 (keywords), Level 2 (regex), and Level 3 (semantic) matching (L3 opt-in); composition strategies (`first_match`, `priority_order`)
- Context gating via `when`/`unless`, with CLI overrides using `--context key=value`
- Output formats: `stylish` (default), `json`, `github`, `ndjson` (JSON includes optional `rule_timeout_ms` when applicable)
- Progress shown by default for human-readable output (suppressed for `--json` and `--quiet`); deterministic ordering preserved; request correlation via `request_id`
- CI auto-detect: defaults to `--output-format=json` in CI unless explicitly set; `--fail-on` defaults to `INFO` in CI
- First‑run helper: if no `promptshield.yaml` exists and `rules/` contains a single pack, a minimal config is scaffolded automatically
- Optional built-in baseline keyword rules for common risks (disabled by default; use RulePacks for production)
- Configuration via flags, environment (`PS_*`), or a config file with precedence
- Telemetry (opt-in): `telemetry.enabled`/`PS_TELEMETRY=1` with `telemetry.endpoint` for OTLP and/or `telemetry.file` for NDJSON sink
- Concurrency control via `PS_WORKERS` or `workers` in config (0 or unset = auto; CI defaults to `NumCPU()`; local default=2)
 - Redaction with verifiers: credit cards (Luhn), API keys/tokens; allowlist/denylist for discovery via `PS_ALLOW_PATHS`/`PS_DENY_PATHS`

### Current limitations

- None for Level 3: provider adapters (OpenAI and Anthropic) are available and used only when opted‑in via env and rule packs specify `semantic.model` and `semantic.analysis_prompt`.
- `--debug` flag exists but logging hooks are not active yet (no structured logs)
- `response` actions are accepted in packs but not executed by the CLI scanner; use reporting to act on results
- Very long single lines (> ~1MiB) can exceed the current per-line buffer; large JSONL is supported within that bound

### Configuration

PromptShield uses a standard hierarchy (highest to lowest):

1) CLI flags
2) Environment variables (`PS_*`, with `-` mapped to `_`)
3) Config file (`promptshield.yaml` in the current directory or `~/.promptshield/promptshield.yaml`)
4) Defaults

Example config:

```yaml
output_format: json
workers: 4
rulepack: rules/example.yaml
composition:
  strategy: first_match
performance:
  per_rule_timeout: 50ms
  case_sensitive: false
  whole_word: false
```

Environment override example:

```bash
export PS_OUTPUT_FORMAT=json
export PS_WORKERS=8
export PS_SEMANTIC_ENABLED=true
export PS_SEMANTIC_PROVIDER=openai   # or 'anthropic'
export OPENAI_API_KEY=...            # or ANTHROPIC_API_KEY for anthropic
 # Optional telemetry
 export PS_TELEMETRY=1
 export PS_TELEMETRY_ENDPOINT=otel-collector:4317
 export PS_TELEMETRY_FILE=spans.ndjson
 export PS_TELEMETRY_SAMPLE=1
```

### RulePacks

Minimal example (see `rules/example.yaml`):

```yaml
apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: example-pack
  version: 0.1.0
rules:
  - id: demo-ignore-previous
    name: Detect 'ignore previous instructions'
    level: 1
    severity: HIGH
    keywords: ["ignore previous instructions"]
  - id: demo-api-key-regex
    name: API key style token
    level: 2
    severity: WARNING
    patterns:
      - regex: "(?i)sk-[a-z0-9]{10,}"
        flags: [ignorecase]
  - id: sem-manipulation
    level: 3
    severity: ERROR
    semantic:
      model: gpt-4o-mini
      analysis_prompt: |
        Respond VIOLATION or SAFE for: {input}
      confidence_threshold: 0.85
      fallback_on_error: true
    fallback:
      patterns:
        - regex: "\\b(jailbreak|DAN)\\b"
          flags: [ignorecase]
```

CLI context overrides can be provided with `--context key=value` (repeatable). These merge into each pack’s `context` and are used to evaluate `when`/`unless` conditions.

### Documentation

- See `docs/CLI.md` for command and flag reference
- See `docs/RulePacks.md` for the RulePack schema and examples (including Level‑3 semantics)
- See `docs/Output.md` for output formats and sample payloads
- See `docs/Architecture.md` for internals and data flow

### Shell completion

PromptShield provides shell completion through Cobra. Use the built-in `completion` command:

```bash
./bin/promptshield completion bash   > /etc/bash_completion.d/promptshield
./bin/promptshield completion zsh    > ~/.zsh/completions/_promptshield
./bin/promptshield completion fish   > ~/.config/fish/completions/promptshield.fish
```

### Development

Common tasks:

```bash
make fmt
make tidy
make test
```

### License

Copyright © 2025 PromptShield authors.


