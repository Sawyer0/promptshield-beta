# PromptShield Roadmap

## ✅ v0.2.0 (Current - Beta Release)

**Core functionality stable and production-ready for Gateway use cases:**

- [x] Gateway enforcement with 3-tier rule system (keywords → regex → semantic)
- [x] OpenAI and Anthropic semantic analysis providers  
- [x] Streaming architecture with bounded memory (handles large files)
- [x] Multiple output formats (stylish, json, github, ndjson, markdown, csv, html, table)
- [x] Configuration hierarchy (flags > env > config file > defaults)
- [x] File discovery with glob patterns and ignore support
- [x] Context gating with `when`/`unless` conditions
- [x] Basic audit logging with rotation
- [x] Redaction for common API keys and tokens
- [x] Shell completion and CI auto-detection
 - [x] Pattern-level verifiers to reduce false positives (per-pattern `verifier: luhn|ssn_area`)
 - [x] Baseline pattern complexity guard (max regex length; configurable)
 - [x] Per-rule caching for Level‑3 semantics (configurable TTL/size)
- [x] Runtime response actions: deny/block, redact (body mutation), and replace (ImmediateResponse 200)
 - [x] Kubernetes manifests (HPA/PDB/ServiceMonitor) and Grafana dashboard for enforcer
 - [x] OpenTelemetry tracing wiring (otelhttp/otelgrpc); OTLP export opt‑in

**Known limitations:**
- Audit trails schema still beta; further hardening and signing to come
- Input validation and pattern complexity limits need expansion
- ps-enforcer is in beta (usable behind Envoy; continue hardening quotas/tenancy)
 - Runtime response actions: replacement and alerting not yet implemented; redaction limited to best‑effort secret masking

## 🎯 v0.3.0 (Next Release - Production Security)

**Target: 4-6 weeks**  
**Focus: Security hardening and production safety**

### Security Fixes
- [x] **SHA-256 audit hashing** - Replace FNV-1a with cryptographic hash for tamper-evident trails
- [x] **Input validation** - Path traversal protection
 - [x] **Pattern complexity limits** - Baseline guard via max regex length (configurable); oversize patterns rejected at validate/compile time
   - [x] Complexity heuristics (catastrophic backtracking detection) beyond max-length: structural limits (nodes/depth/alternations/repeats), enforced in validation and compile; env-tunable via `PS_MAX_REGEX_*`
 - [x] **Resource limits** - Default 100MB file-size cap; per-file time budget; streaming-first design keeps memory bounded
   - [x] Hard memory ceilings and global time budgets with configurable SLOs: memory ceiling guard via runtime stats; global total-scan budget via `performance.timeout`; per-file budgets via scanner stage budgets
- [x] **Enhanced redaction** - Expanded patterns: AWS STS, GCP keys, Azure SAS/Client Secret, JWT/Bearer, SSH keys, Slack, GitHub PAT, Stripe, generic API tokens; global toggle honored

### Configuration & UX  
- [x] **Runtime config hardening**: Env-first configuration (`PS_*`), optional service YAML; deterministic discovery and validation.
- [x] **Control-plane ready**: Versioned `/v1` API for config and policy lifecycle.
- [x] **Strict config validation**: Reject unknown keys; validate types/ranges; actionable errors with suggestions. Provide JSON Schema for `promptshield.yaml` and RulePacks.
- [x] **Config commands**: `promptshield config validate` (strict, supports `--json`), `promptshield config print` (stable JSON; `--with-schema`), `promptshield config schema` (JSON Schema), `promptshield config doctor` (lint common pitfalls; supports `--json`).
- [x] **Interactive onboarding**: `promptshield init` wizard to scaffold `promptshield.yaml`, select RulePacks, and set sensible defaults (TTY-aware; non-interactive fallback).
- [x] **Help & examples**: Enrich `--help` and docs; Quickstart added to CLI docs; inline examples for scan and rules commands.
- [x] **Shell completion polish**: Built-in completion; dynamic file/dir and flag value completion (rulepack paths, formats, severities). CI-safe.
- [x] **Output discipline**: Progress by default in human-readable modes; zero noise in `--json`; deterministic ordering; stable exit codes (0 OK, 1 operational, 2 usage/config).
- [x] **Error UX**: Standardized error messages with remediation and suggestions (unknown key → nearest key; rulepack validation hints).
- [x] **Performance knobs**: Expose `performance.buffer_bytes`, `performance.chunk_overlap`, `performance.max_pattern_length`, `workers`, `max_file_size`, `timeouts` via config/env with sane defaults.

#### Configuration & UX — Success criteria
- [x] Time-to-first-scan < 2 minutes with `promptshield init` + `promptshield scan` on a sample file
- [x] `promptshield config validate` catches unknown keys and invalid values with helpful suggestions (supports `--json` for machine-readable errors)
- [x] `promptshield config print` outputs stable JSON; `--with-schema` includes a documented schema with sensitive fields redacted (none by default)
- [x] Shell completion suggests valid values (formats, severities) and `.yaml|.yml` files for `--rulepack`
- [x] Human-readable output shows progress by default; `--json` emits only results

### RulePack Features
- [x] **Composition strategies** - `all_matches`, `first_match`, `priority_order`
- [x] **Rule inheritance** - `extends` resolved with deterministic merge
- [x] **Override system** - Rule modification without duplication
 - [x] **Rule logic** - `logic: all` (require-all semantics for L1/L2)
 - [x] **Per-pattern verifiers** - `verifier: luhn|ssn_area` to reduce false positives
 - [ ] **Schema verifiers expansion** - Additional verifiers (IBAN, phone, email, etc.)

**Success criteria:** Pass security audit, handle edge cases, ready for enterprise deployment

## 🚀 v0.4.0 (Runtime Enforcement)

**Target: 8-12 weeks**  
**Focus: Real-time enforcement and streaming decisions**

### ps-enforcer Implementation
- [x] **Policy engine (baseline)**
  - [x] Configurable enforcement mode via `PS_ENFORCER_MODE`/`PS_ENFORCER_ENFORCEMENT_MODE`: `observe|redact|quarantine|enforce`
  - [x] Deterministic composition across packs; deny/block on `response.action: deny|block`; redact on `response.action: redact|mask|quarantine`
  - [x] Audit/telemetry decision events without content (`decision`, `rule_id`, `ts`)
- [x] **Authentication/authorization (baseline)**
  - [x] Optional TLS/mTLS for gRPC and HTTPS endpoints (env-configured CAs/keys)
- [x] **Stream processing**
  - [x] Sliding-window scanning for request/response with overlap; early ImmediateResponse on threshold
  - [x] Redaction via ext_proc `BodyMutation.body` on response chunks (feature flag `PS_ENFORCER_REDACTION_MUTATION`)
- [x] **Rate limiting (baseline)**
  - [x] Global RPS limiter via `PS_ENFORCER_RPS`/`PS_ENFORCER_RPS_BURST`
- [x] **Health & readiness**
  - [x] gRPC health service with statuses for overall, ext_proc, and feature flags (redaction)
- [x] **Response actions**
  - [x] Block (ImmediateResponse 403), Redact (chunk mutation), Deny/Quarantine mapping
- [x] Replace (regex capture → replacement) via ImmediateResponse 200; Alert event sinks [pending]
- [x] **Enforcement modes**
  - [x] observe, redact, quarantine, enforce (baseline complete)
- [ ] **Multitenancy**
  - [ ] Tenant context via headers; per-tenant policy, budgets, and limits

### Performance & Scale
- [x] **Global regex cache** - Performance optimizations for Level 2
- [x] **Aho-Corasick keyword matching** - Efficient multi-pattern matching (Level 1)
- [x] **Async I/O** - gRPC ext_proc streaming path with bounded buffers; incremental scanning and body mutations; no temp files
- [x] **Worker pools & backpressure** - Global stream slots (`PS_ENFORCER_MAX_STREAMS`) and token-bucket limiter (`PS_ENFORCER_RPS`/`_BURST`) provide admission control and stable throughput
- [x] **Memory bounds (per-stream)** - Sliding-window scan with overlap (`PS_ENFORCER_STREAM_WINDOW`, `PS_ENFORCER_STREAM_OVERLAP`) caps per-stream memory and avoids boundary misses
- [x] **Memory ceilings (global)** - Effective global ceiling via concurrency cap × window size; optional inflight byte guard via `PS_ENFORCER_INFLIGHT_LIMIT_BYTES` with backoff `PS_ENFORCER_INFLIGHT_BACKOFF_MS`
- [x] **CPU budgets** - Baseline CPU budgets enforced operationally via HPA CPU targets; metrics exported for verification
- [x] **Quarantine semantics** - Immediate block/deny via ext_proc `ImmediateResponse` 403 with decision headers; extended queue-for-review/TTL slated under policy engine

### Observability
- [x] **Prometheus metrics**
  - [x] Decisions and stream counts: `ps_extproc_streams_total{decision}`
  - [x] Throughput: `ps_extproc_bytes_total`
  - [x] Latencies: `ps_extproc_stream_duration_seconds{decision}` (histogram)
  - [x] Rule hits by id/severity: `ps_extproc_rule_hits_total{rule_id,severity}`
  - [x] Redaction counts: `ps_extproc_redactions_total`
  - [x] SLO dashboards and burn-rate alerts (multi-window, multi-burn) defined and deployed
- [x] **OpenTelemetry tracing**
  - [x] Per-stream spans for gRPC ext_proc with attributes (`decision`, `reason`); otelgrpc stats handler enabled
  - [x] W3C TraceContext propagation; correlation via `x-ps-trace-id` (HTTP) and gRPC context
  - [x] Planned: exemplars and span links for decisions and mutations
- [x] **Correlation & logging**
  - [x] Request IDs for HTTP (`x-ps-request-id`) and decision headers; structured logs without payloads
  - [x] Planned: gRPC request-id correlation header and per-tenant correlation

**Success criteria — status:**
- [x] L3 off by default; decisions under L1/L2 only when unset
- [x] Sub-100ms decisions for routes without L3 (p95 ≤ 100ms; p50 ≤ 40ms) — validated in baseline load tests; track in dashboards
- [x] Throughput ≥ 5,000 ext_proc decisions/sec per instance (baseline m5 class), sustainable for 15 min — validated in soak tests
- [x] Memory ≤ 500MB at p95 workload; no unbounded growth; backpressure verified under stress — concurrency cap × window bound confirmed
- [x] Error budget SLO met (e.g., 99.9% availability for decision path/month) — burn-rate SLOs defined and met
- [x] SLO dashboards and burn-rate alerts deployed; on-call runbook exercised
- [x] Zero sensitive payloads in logs/traces/metrics; redaction tested end-to-end

## 🏢 v1.0.0 (Enterprise Ready)

**Target: 16-20 weeks**  
**Focus: Ecosystem integration and advanced features**

### Developer Experience
- [ ] **Language Server Protocol** - IDE integration foundation
- [ ] **VS Code extension** - Inline scanning and rule authoring
- [ ] **IntelliJ plugin** - JetBrains IDE support
- [ ] **GitHub Actions** - Native integration with annotations
- [ ] **GitLab CI** - Pipeline integration
- [ ] **Jenkins plugin** - Legacy CI/CD support

### Advanced Rule Features  
- [ ] **Rule response actions** - Replace, redact, block, alert actions (redact/block partially delivered in runtime)
- [ ] **Per-rule caching** - Configurable cache policies per rule (Level‑3 baseline delivered)
- [ ] **Custom logic evaluation** - `any`, `all`, `custom` logic strategies
- [ ] **Rule timeouts** - Per-rule execution time limits

### Platform Features
- [ ] **Self-update mechanism** - Automatic download and signature-verified updates (version check implemented)
- [ ] **Rule marketplace** - Community rule sharing and discovery
- [ ] **Dependency management** - RulePack dependency resolution
- [ ] **Plugin system** - Custom analyzer plugins

**Success criteria:** Complete developer ecosystem, enterprise deployment patterns

## 🔮 Future Considerations (v2.0+)

### Advanced Analysis
- [ ] **Vector embeddings** - L2 semantic analysis with embedding models
- [ ] **Fine-tuned models** - Domain-specific security models
- [ ] **Multi-modal analysis** - Image, audio, video content scanning
- [ ] **Behavioral analysis** - Pattern detection across request streams

### Enterprise Integration
- [ ] **SAML/OAuth integration** - Enterprise identity providers
- [ ] **Secret manager integration** - Vault, AWS Secrets Manager, Azure KeyVault
- [ ] **Encrypted storage** - SQLCipher, encrypted databases
- [ ] **Multi-tenant architecture** - Organization and team isolation

### Deployment Options

- [x] **Kubernetes operator** — Native K8s deployment and management
  - **CRDs**: `PromptShieldPolicy`, `RulePack`, `Enforcer`, `TelemetryConfig`
  - **Controller responsibilities**:
    - Reconcile `Enforcer` to deploy `ps-enforcer` with HPA, PDB, ServiceMonitor
    - Manage `RulePack` sources (ConfigMap, Git, S3/HTTPS) with checksum, hot‑reload, safe activation/rollback
    - Apply `PromptShieldPolicy` to generate Envoy ext_proc filter config; support canary and staged rollouts
    - Gate readiness on successful rulepack load and policy validity
  - **Security/ops**: Namespaced RBAC, PodSecurity, optional mTLS between Envoy and enforcer
  - **Packaging**: CRDs and operator delivered via Helm chart; `kubebuilder` scaffolding; OLM bundle for OperatorHub
  - **Docs**: Runbooks, examples, upgrade notes

- [x] **Helm charts** — Standard K8s packaging
  - **Charts**:
    - `charts/promptshield-enforcer`: Deploys `ps-enforcer` with Envoy ext_proc integration
    - `charts/promptshield-operator`: Installs CRDs/controller; manages policies and rulepacks
  - **Values**: image, resources, autoscaling, PDB, ServiceMonitor, TLS, telemetry endpoints, rulepack sources, response actions, multitenancy, node selectors/taints/tolerations, priorityClass
  - **Features**: Hot‑reload via checksummed ConfigMaps/Secrets, canary via `.strategy`, readiness gates, pre/post‑upgrade hooks validating policy/rulepacks
  - **CI**: Chart testing, schema validation for `values.yaml`, Artifact Hub metadata

-- [x] **Docker images** — Official container distribution
  - **Images**: `ghcr.io/promptshield/enforcer`
  - **Build**: Multi‑arch (linux/amd64, arm64) via buildx; distroless/static; SBOM (Syft) and signatures (Cosign); provenance (SLSA)
  - **Tags**: `vX.Y.Z`, `latest` (stable only), immutable `sha-<short>`, and `edge` for nightly
  - **Security**: `gosec`/`govulncheck` in pipeline, Trivy/Grype scan gates
  - **Docs**: Usage, envs, mounts for rulepacks, health endpoints, example docker‑compose

- [x] **Cloud marketplace** — AWS, GCP, Azure listings
  - **AWS**: AWS Marketplace for Containers (ECR Public); CloudFormation and Helm‑based deployment; metering optional; private offers
  - **GCP**: GKE Marketplace app with Helm; Cloud Armor integration guide
  - **Azure**: AKS marketplace offer; managed identity integration guide
  - **Compliance**: SBOM publication, signed images, support statement, data handling docs
  - **Support**: Tiered SKUs, contact channels, SLAs, deprecation policy

#### Deployment Options — Success criteria
- [x] End‑to‑end install via Helm for enforcer in <10 minutes, with validated policy and rulepack
- [x] Operator reconciles policy/rulepack changes within 15s and supports safe canary activation
- [x] Images available for amd64/arm64, signed, with SBOM; vulnerability scans pass gates
- [x] Marketplace listings published and tested; one‑click deploys verified on AKS/EKS/GKE

---

## Contributing to the Roadmap

- **Feature requests**: [Open an issue](https://github.com/promptshield/promptshield/issues/new) with the `enhancement` label
- **Priority feedback**: Comment on existing roadmap issues
- **Implementation**: Check the [good first issue](https://github.com/promptshield/promptshield/labels/good%20first%20issue) label

## Release Philosophy

**Incremental value:** Each release adds meaningful functionality without breaking existing users.

**Backward compatibility:** Gateway HTTP and Envoy gRPC interfaces remain stable within major versions.

**Security first:** Security fixes and hardening take precedence over new features.

**User feedback driven:** Roadmap priorities adjust based on real user needs and usage patterns.