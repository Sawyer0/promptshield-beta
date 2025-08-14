## v0.2.0-beta (2024-XX-XX) - Beta Release

**PromptShield is now ready for production CLI use cases!**

### ✅ Production-Ready Features

**Core Scanning:**
- 3-tier rule system: Keywords (Level 1) → Regex (Level 2) → Semantic Analysis (Level 3)
- OpenAI and Anthropic semantic analysis providers with caching and rate limiting
- Streaming architecture with bounded memory (handles large files efficiently)
- File discovery with glob patterns and vendor directory skipping

**Output & Integration:**
- Output formats: stylish (default), json, github, ndjson (removed: markdown, csv, html, table)
- CI auto-detection (defaults to JSON output in CI environments)
- Deterministic output ordering for consistent results
- Shell completion for bash, zsh, fish

**Configuration:**
- Flexible configuration hierarchy: CLI flags > environment > config file > defaults  
- Context gating with `when`/`unless` conditions and CLI overrides
- First-run helper that scaffolds minimal config automatically

**Security & Audit:**
- Basic audit logging with daily rotation
- API key redaction in logs for OpenAI, Anthropic, GitHub tokens
- Configurable redaction system with global toggle

### ⚠️ Security Notes

**Safe for production with limitations:**
- Audit trails now use SHA-256 hash chaining with canonical JSON serialization
- Input validation exists but needs expansion (pattern complexity limits)
- ps-enforcer no longer requires PS_EXPERIMENTAL=true; still considered early-stage, use with care

### 🛡️ Safety Features Added

**RulePack Support:**
- `extends`/`overrides` supported with deterministic merge
- Composition strategies: `all_matches` (default), `first_match`, `priority_order`
- `logic: all` for L1/L2 rules

### 📋 Use Cases

**✅ Recommended for:**
- Pre-commit hooks and CI/CD pipeline scanning
- Batch analysis of training data and prompt libraries  
- Security audits and compliance checking
- Development and testing of LLM applications

**❌ Not recommended for:**
- Real-time enforcement (ps-enforcer experimental)
- Security compliance audit trails (schema may evolve)

### 🚀 Coming Next

**v0.3.0 (4-6 weeks):** Production security hardening
- SHA-256 audit hashing for tamper-evident trails
- Input validation and path traversal protection  
- Resource limits and DoS protection
- Enhanced redaction for cloud provider tokens

See [ROADMAP.md](ROADMAP.md) for complete future plans.

## v0.3.0 (unreleased)

Features:
- NDJSON event streaming (per‑violation + summary)
- Audit logging with daily rotation, hash chain, and global redaction toggle
- Large‑file guardrails: 1 GiB streaming benchmark + memory budget test
- Runtime enforcer (HTTP + gRPC ext_proc) experimental and documented

Improvements:
- Keyword matching options (case_sensitive, whole_word) with config defaults
- Semantic fallback on API errors with optional regex fallbacks
- Configuration discovery (project/XDG/home) + unknown key validation

Docs:
- Output formats, audit schema, runtime architecture, Envoy integration

Breaking changes:
- CLI flag simplification: infrastructure settings moved to env/config only.
  - Workers: use `PS_WORKERS` env or `workers:` in `promptshield.yaml`
  - Debug logging: use `PS_DEBUG` or `debug: true`
  - Redaction: use `PS_REDACTION_ENABLED` or `redaction.enabled`
  - Audit/Metrics/Trace files: use `PS_AUDIT_FILE`, `PS_METRICS_FILE`, `PS_TRACE_FILE` or config (`audit_file`, `metrics_file`, `trace_file`)
  - `fail_on`: use config key or `PS_FAIL_ON`
  - Frequently changed flags remain: `--rulepack/-r`, `--output-format/--json`, `--context`, `--fail-on`, `--quiet`, `--force`


