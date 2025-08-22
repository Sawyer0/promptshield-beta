## PromptShield Architecture Overview (CLI Scanner for LLM Safety)

This document describes the architecture, components, data flows, and security model of PromptShield. It consolidates guidance from internal plans under `plan/` and the implemented TypeScript CLI, while anticipating the Go/Cobra+Viper migration path.

### Design Goals
- Enterprise‑grade from day one; progressive complexity as users mature
- ESLint‑grade UX: clear flags, deterministic output, formatter ecosystem
- Streaming‑first, memory‑bounded processing for large files
- Modular domains with clean ports/adapters; DI wiring
- Security by default: key management, audit, privacy by design
- Distribution ready: single binary path (Go), package managers, CI‑friendly exit codes

### High‑Level System

```mermaid
flowchart TD
  CLI[CLI (Cobra/Commander)] --> CONFIG[Config (flags/env/config file)]
  CLI --> ORCH[Scan Orchestrator]
  CONFIG --> ORCH
  ORCH --> DISC[File Discovery (glob + ignore)]
  ORCH --> PROC[Processors (JSON/NDJSON/TEXT)]
  PROC --> RULES[Rule Engine (regex/keywords)]
  RULES --> VIOLS[Violations]
  ORCH --> METRICS[Metrics]
  VIOLS --> REPORT[Report]
  METRICS --> REPORT
  REPORT --> REND[Renderers (stylish/json/…)]
  REND --> OUTPUT[stdout / file]
```

## Components

### CLI and UX
- Commands: `scan`, `list`, `init`, `validate` (future: `config --print`, `benchmark`, `update`)
- Flags: parity with ESLint‑grade plan (rules discovery, ignore, stdin filename, formatters)
- Exit codes: 0 success, 1 operational/config errors, 2 fail‑on threshold met
- Deterministic output, `--plain` ASCII for CI, color optional

### Configuration
- Precedence: flags > env vars > config file > defaults (Viper in Go; current TS uses flags)
- Suggested files: `promptshield.config.{yaml,json}` or `.promptshieldrc.{yaml,json}` (future)
- `--print-config` (future) for effective config without secrets

### RulePacks (YAML)
- Single responsibility: rule metadata + match strategies (regex, keywords; future: NLP)
- Versioning and headers: version/last_updated (current), evolve towards `apiVersion`, `kind`, `metadata` (see `plan/PromptShield YAML Schema Specification v1.txt`)
- Discovery: `--rulepack` or `--rules-dir` with deterministic selection
- Validation: strict schema, duplicate rule IDs prevented, enabled toggles

### Rule Engine
- Strategy‑based: RegexMatcher, KeywordMatcher; sequence cheap→expensive
- Outputs typed violations with position (start/end/line/column), context window, metadata
- Filtering: severity/category; options for case sensitivity

### Scanning Orchestrator
- Input types: file, directory, glob, stdin, or direct content
- Directory/glob expansion with `.promptshieldignore` and `--ignore-path`; `--ext` for extra types
- Streaming for NDJSON and large JSON; bounded memory; metrics for throughput and usage
- Parallelism strategy (future in TS; part of Go plan) with deterministic merge order

### Processors
- JSON (array/object) and NDJSON streaming; TEXT (future)
- Field selection (`--fields`), `--scan-entire-object`, `--max-objects`, depth limits

### Renderers
- Default `stylish` for console; `json` for machine consumption; others (markdown/csv/html/table/ndjson)
- `--format-options` (future) for renderer‑specific tuning
- ReportService coordinates renderer selection and file writing

### Validation
- `validate <target>` for RulePacks and inputs (format checks, schema, basic sanity)
- Exit with nonzero on invalid; strict mode `/` skip‑warnings support

## Security Architecture (Progressive Security)

This section synthesizes: `plan/Auth  Security Model.txt`, `plan/API Key Management and Rotation.txt`, `plan/Protecting Data in Transit.txt`, `plan/Securing Data at Rest.txt`, `plan/The Principle of Data Minimization.txt`, `plan/Privacy by Design Considerations.txt`.

### Credential & API Key Management
- Hierarchy: env vars → OS keychain (Keychain/Credential Manager/Secret Service) → interactive prompt fallback
- Never store keys in repo or config; validate at startup; `validate` can check credential health
- Optional secret managers (Vault, AWS Secrets Manager) for rotation and policy control

### Authentication & Permissions (future platform integration)
- OAuth 2.0 Device Code + PKCE for CLI login (device flow)
- Contextual permissions: local vs team vs org; least privilege service accounts in CI

### Transport Security
- TLS for all outbound calls; certificate pinning optional
- Privacy‑first scanning mode: local prefilter before any remote semantic analysis

### Data at Rest
- Encrypted storage for any persisted data; 0600 perms on sensitive files
- Optional SQLCipher (SQLite) or encrypted PG; per‑user access control

### Data Minimization & Privacy by Design
- Retention modes: full‑audit, metrics‑only, ephemeral (no persistent content)
- Audit/metrics never include raw prompts by default; redaction options
- Deletion tooling for data subject requests and incident response

### Audit Trails & Monitoring
- Append‑only, hash‑chained logs; integrity checks; OpenTelemetry‑compatible JSON events
- Alerts on spikes, repeated API failures, unusual scan contexts

## Observability & Error Handling
- Structured logging (levels: DEBUG/INFO/WARN/ERROR)
- Metrics: objectsScanned, processingTime, memory, streamingUsed; optional NDJSON metrics file (future)
- Error taxonomy: ValidationError, ConfigurationError, FileSystemError, FailOnThresholdError (mapped to exit 2)

## Performance & Scalability
Sourced from `plan/Performance  Scalability.txt`.
- Streaming‑first parsers; fixed buffers; early feedback
- Parallelization: file‑level worker pools (adaptive), deterministic output
- Memory tiers: working/cache/spill-to-disk
- Optimization roadmap: regex caches, bloom filters, semantic rules as last resort

## Distribution & Updates
Sourced from `plan/Core Distribution Stack.txt` and `plan/CrossPlatform Binary Distribution.txt`.
- GoReleaser + GitHub Releases: cross‑platform binaries, checksums, optional signatures
- Package managers: Homebrew (tap), Snap, Chocolatey; Docker image for container use
- Optional update checker; opt‑in self‑update with verification

## Extensibility
Sourced from `plan/Gaps Resolution - Proposed Solutions.txt` and `plan/YAML Schema Design Best Practices.txt`.
- Ports/adapters across domains; registries for matchers/renderers
- Rule types via strategy pattern (keyword/regex/semantic)
- Schema evolution with `apiVersion`; JSON Schema for IDE hints; marketplace‑ready metadata
- ADRs in `docs/adr/` for major decisions (DI, logging, update policy, schema versioning)

## Primary Flows

### Scan Flow

```mermaid
sequenceDiagram
  participant U as User
  participant CLI as CLI
  participant C as Config
  participant D as Discovery
  participant O as Orchestrator
  participant F as FileReader
  participant P as Processor
  participant E as RuleEngine
  participant R as Renderers

  U->>CLI: promptshield scan <input> [flags]
  CLI->>C: Resolve config (flags/env/defaults)
  CLI->>D: Resolve rulepack via --rulepack/--rules-dir
  CLI->>O: Execute scan(config)
  O->>F: Expand directory/globs; apply ignore
  loop For each file/object
    O->>P: Process content (stream if large)
    P->>E: Apply rules (regex/keywords)
    E-->>O: Violations
  end
  O-->>CLI: Results + metrics
  CLI->>R: Render (stylish/json/...)
  R-->>U: Output (stdout/file); exit with 0/1/2
```

### Init Flow
- Input: filename + options (`--name`, `--description`, `--force`)
- Output: YAML RulePack stub with version/last_updated and example rule

### List Flow
- Input: `--rulepack` or `--rules-dir`
- Loads RulePack, filters by `--category`, `--severity`, `--enabled-only`
- Outputs concise rule list

### Validate Flow
- Validates rulepack or input format; strict mode and output formatting supported

## Migration to Go (Cobra+Viper)
See `docs/GO_MIGRATION_PLAN.md` for command parity, module layout, and two‑week plan. The TS CLI remains the oracle for flags, outputs, and exit codes until the Go binary reaches parity.

## Security & Compliance Checklist (Rollup)
- Never commit secrets; use env/OS keychain; validate credentials early
- Enforce TLS; consider pinning; sanitize before remote calls
- Default to metrics‑only retention; redaction for logs; deletion tooling
- Append‑only audit with integrity; alerts for anomalies
- RulePack schema versioned, validated, and documented

## ADR Index (suggested)
- ADR‑0001: DI composition strategy (explicit constructors; optional Wire later)
- ADR‑0002: Logging and audit (slog, integrity chaining)
- ADR‑0003: Update policy (check‑only by default; signed artifacts)
- ADR‑0004: RulePack schema versioning and marketplace metadata



