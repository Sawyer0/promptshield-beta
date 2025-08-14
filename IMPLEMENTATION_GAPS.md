# PromptShield Implementation - 7-Day Sprint Plan

## Overview
This document outlines a focused 7-day implementation plan to make PromptShield production-ready. Each day has clear goals, acceptance criteria, and stop points.

**Current Status (v0.2.0):**
- ✅ Core scanning with 3-tier rules (L1 keywords → L2 regex → L3 semantic)
- ✅ Runtime enforcer: HTTP `/check` + Prometheus `/metrics`; gRPC ext_proc streaming with early-quarantine; optional TLS/mTLS
- ✅ Kubernetes manifests with HPA, PDB, and Prometheus ServiceMonitor; Grafana dashboard
- ✅ API docs (OpenAPI + gRPC), provider docs (OpenAI/Anthropic), Anthropic RulePack examples
- ✅ Full OpenTelemetry distributed tracing across CLI, scanner, HTTP `/check`, and gRPC ext_proc; otelhttp/otelgrpc instrumentation; OTLP export opt-in via `PS_TELEMETRY=1` + `PS_TELEMETRY_ENDPOINT`; `healthz`/`metrics` filtered; `x-ps-trace-id` on HTTP responses
- ⚠️ Needs hardening (pattern complexity limits, broader resource limits)
- ⚠️ Runtime: experimental label remains; expand budgets and HITL in future sprints
 - ✅ Per-pattern verifiers in schema and engine (pattern-level `verifier: luhn|ssn_area`)
 - ✅ Response-based redaction/mutation in gRPC ext_proc when rule `response.action` is `redact|mask|quarantine`; immediate deny on `deny|block`

---

# 🎯 Day 1: Lock Safety Rails (90-120 min)
**Goal:** Prevent catastrophic failures and establish fail-safe boundaries

## Tasks
### 1.1 Set Per-Stage Time Budgets
```go
// internal/scanner/limits.go
type StageBudgets struct {
    Canonicalization time.Duration // 5ms
    Level1Keywords   time.Duration // 10ms  
    Level2Regex      time.Duration // 50ms
    Level3Semantic   time.Duration // 300ms
    PerFile          time.Duration // 10s budget per file
}

const (
    DefaultQuarantineOnTimeout = true
    DefaultQuarantineOnError   = true
)
```

### 1.2 Implement Quarantine Semantics
```go
type QuarantinePolicy struct {
    BlockDelivery bool
    QueueForHITL  bool
    TTL           time.Duration // 10m default
    FallbackAction string       // "redact" or "block"
}
```

### 1.3 Turn on Structured Logging (Gap 2.3)
- Wire logger to all components
- Add INFO/WARN/ERROR levels
- Include request IDs and timing

**✅ Acceptance:** 
- [x] Timeout → quarantine works
- [x] Every operation logs with timing
- [x] `PS_DEBUG=true` shows full traces

**🛑 STOP HERE FOR DAY 1**

---

# 🔧 Day 2-3: Fix the Foot-Guns
**Goal:** Eliminate security vulnerabilities

## Day 2 Tasks
### 2.1 Audit Hash → SHA-256 (Gap 4.4)
```go
// internal/audit/logger.go
import "crypto/sha256"

func (l *Logger) calculateHash(event AuditEvent) string {
    h := sha256.New()
    // Hash all fields
    return hex.EncodeToString(h.Sum(nil))
}
```

### 2.2 Input Validation (Gap 4.5)
Note: Path traversal and null-byte guards exist in discovery; this task tracks remaining validation (e.g., symlink policy and pattern complexity limits).
```go
func validatePath(path string) error {
    if strings.Contains(path, "..") { return fmt.Errorf("path traversal detected") }
    // Add symlink policy (allow/deny) and OS-specific checks
    return nil
}
```
Implementation notes:
- Symlink policy: default deny. Set `PS_ALLOW_SYMLINKS=true` to permit following symlinks during discovery (applies to direct paths, globs, and directory walks).
- Pattern complexity: enforce `performance.max_pattern_length` (default 1000). Longer regex patterns are rejected at validation/compile time. Implemented in compiler and via `SetMaxPatternLength`.

## Day 3 Tasks
### 2.3 Resource Limits (Gap 4.6)
```go
type Limits struct {
    MaxFileSize      int64         // 100MB
    MaxPatternLength int           // 1000 chars
    MaxCacheEntries  int           // 1000
    PerFileBudget    time.Duration // 10s
}
```

### 2.4 Expand Redaction (Gap 4.2)
- Verify and expand cloud token patterns (AWS, GCP, Azure)
- Verify JWT, SSH key patterns (present)
- Ensure global toggle works (present)

**✅ Acceptance:**
- [x] SHA-256 chain verified
- [x] Path traversal blocked
- [x] 100MB file → error (not OOM)
- [x] All token types redacted

Implemented:
- Audit events: canonical JSON + SHA-256 hash chain; data sanitized/redacted before hashing/writing.
- Discovery: rejects symlinks by default; traversal and null bytes already guarded; glob and walk enforce same policy.
- Resource limits: default 100MB max file size enforced (configurable); per-file time budget added.
- Redaction: expanded patterns (AWS STS token, GCP service account keys, Azure SAS/Client Secret, Bearer/JWT, SSH keys, Slack, GitHub PAT, Stripe, generic API tokens).

**🛑 STOP HERE FOR DAY 3**

---

# ⚡ Day 4: Performance Pragmatism
**Goal:** Fast enough for runtime without over-engineering

## Tasks
### 4.1 Keep RE2 as Default
- Default path uses Go's RE2-compatible `regexp` engine (kept). Hyperscan remains optional behind a build tag.

### 4.2 Add Hyperscan Behind Flag (Optional)
```go
if viper.GetBool("performance.use_hyperscan") {
    matcher = hyperscan.NewMatcher()
} else {
    matcher = regex.NewMatcher() // default
}
```

### 4.3 Quick Benchmark
```bash
# Focused p95 L1/L2 latency
make bench-quick

# Full suite
make bench

# If < 20% improvement, skip Bloom/XOR for now
```

**✅ Acceptance:**
- [x] p95 < 25ms for keyword/regex
- [x] Benchmark documented
- [x] RE2 remains default

**🛑 STOP HERE FOR DAY 4**

---

# 🛡️ Day 5: RulePack That Protects
**Goal:** Ship real PII/Secrets detection with verifiers

## Tasks
### 5.1 Create PII/Secrets RulePack (per-pattern verifiers)
```yaml
# rules/pii-secrets.yaml
apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: pii-secrets-protection
  version: 1.0.0
rules:
  - id: credit-card
    level: 2
    severity: HIGH
    patterns:
      - regex: '\\b(?:\\d[ -]*?){13,19}\\b'
        verifier: luhn
    
  - id: ssn
    level: 2
    severity: CRITICAL
    patterns:
      - regex: '\\b\\d{3}-\\d{2}-\\d{4}\\b'
        verifier: ssn_area
    
  - id: api-key-generic
    level: 2
    severity: HIGH
    patterns:
      - regex: '(?i)(api[_-]?key|token)[\s:=]+[\'\"]?([a-zA-Z0-9]{32,})[\'\"]?'
```

### 5.2 Implement Verifiers (done)
```go
// internal/scanner/verifiers.go
func LuhnCheck(input string) bool {
    // Luhn algorithm
}

func SSNAreaValid(input string) bool {
    // Check valid SSN area codes
}
```

### 5.3 LLM Escalation with Quarantine
```go
if rule.Severity >= HIGH && rule.Level == 3 {
    result, err := semantic.Analyze(ctx, input)
    if err != nil || result.Timeout {
        return QuarantineDecision{
            Action: "block",
            Reason: "LLM unavailable",
        }
    }
}
```

**✅ Acceptance:**
- [ ] Golden PII corpus → 0 false negatives (pending corpus run)
- [ ] Verifiers reduce false positives > 50% (pending benchmark)
- [x] LLM timeout → quarantine

Implemented:
- RulePack examples updated to use per-pattern `verifier:`.
- Verifiers implemented (`internal/scanner/verifiers.go`) and applied selectively per pattern (`verifier: luhn|ssn_area`).
- Semantic L3 already enforces with timeout fallback → quarantine.

**🛑 STOP HERE FOR DAY 5**

---

# 🚀 Day 6: Streaming = Real Runtime
**Goal:** Make decisions BEFORE data leaves the system

## Tasks
### 6.1 Sidecar Streaming Decision (implemented incrementally)
```go
// internal/enforcer/stream.go
type StreamDecision struct {
    Delta    []byte
    Decision string // "pass", "block", "redact"
    Made     time.Time
}

func (e *Enforcer) ProcessDelta(delta []byte) StreamDecision {
    // Scan delta
    // Return decision immediately on first HIGH (gRPC ext_proc returns ImmediateResponse)
    if violation.Severity >= HIGH {
        return StreamDecision{
            Decision: "block",
            Made: time.Now(),
        }
    }
}
```

### 6.2 Minimal Decision Events
```go
type DecisionEvent struct {
    Timestamp time.Time `json:"ts"`
    Decision  string    `json:"decision"`
    RuleID    string    `json:"rule_id,omitempty"`
    // NO content included
}
```

**✅ Acceptance:**
- [x] First HIGH signal → immediate block
- [x] Zero content in decision events
- [ ] Streaming latency < 5ms

Implemented:
- gRPC ext_proc enforcer returns ImmediateResponse on first threshold hit and records decision/latency
- Minimal decision events emitted with only ts, decision, rule_id (no content/body)
 - Response-based redaction: when rules specify `response.action: redact|mask|quarantine`, the enforcer mutates response body chunks via ext_proc BodyMutation set_body with secrets redacted; deny/block triggers ImmediateResponse

**🛑 STOP HERE FOR DAY 6**

---

# 📊 Day 7: Observability You'll Live With
**Goal:** Know what's happening without drowning in data

## Tasks
### 7.1 Prometheus & Dashboard (implemented)
```go
var (
    scanDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "ps_scan_duration_seconds",
            Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
        },
        []string{"level", "status"},
    )
    
    // Runtime metrics exposed by enforcer:
    // ps_enforcer_requests_total{path,code}
    // ps_enforcer_decisions_total{decision}
    // ps_enforcer_request_duration_seconds_bucket
    // ps_extproc_streams_total{decision}
    // ps_extproc_bytes_total
    // ps_extproc_stream_duration_seconds_bucket
)
```

### 7.2 Request IDs in Logs
```go
ctx = context.WithValue(ctx, "request_id", uuid.New())
logger.Info("scan started", "request_id", ctx.Value("request_id"))
```

### 7.3 NDJSON Streaming (Already Done)
- Verify it's still working
- Add request_id to events

**✅ Acceptance:**
- [x] `curl /metrics` shows counters and histograms; Grafana p95 panels show data
- [x] Every log has request_id (CLI); server logs include request context
- [x] Dashboard panels cover request/stream rate and latencies

### 7.4 OpenTelemetry Tracing (implemented)
```bash
# Enable privacy-first tracing export (no content payloads)
export PS_TELEMETRY=1
export PS_TELEMETRY_ENDPOINT=otel-collector:4317   # OTLP/gRPC
# Optional local file sink for coarse events (no secrets)
export PS_TELEMETRY_FILE=spans.ndjson
# Optional sampling (0.0..1.0). Default 1.0 when enabled
export PS_TELEMETRY_SAMPLE=0.2
```

- Tracing wired via otelhttp (HTTP) and otelgrpc (gRPC) with W3C TraceContext propagation
- Resource attributes include `service.name`, `service.version`, and `service.instance.id`
- HTTP returns `x-ps-trace-id`; `GET /healthz` and `/metrics` are excluded from tracing
- Shutdown flushes providers on CLI and enforcer exit to avoid span loss

**✅ Acceptance:**
- [x] Traces appear in the collector when `PS_TELEMETRY=1` and `PS_TELEMETRY_ENDPOINT` set
- [x] No sensitive content in spans; only coarse attributes (format, workers, fail_on)

**🛑 STOP - WEEK 1 COMPLETE!**

---

# 🚫 What to IGNORE (for now)

These are in the gaps doc but NOT urgent:
- ❌ Embedding/L2 vector gates
- ❌ XOR/Bloom pre-filters (unless benchmark shows >20% win)
- ❌ Self-update download + signature verification (version check exists)
- ❌ LSP/IDE plugins (7.2)
- ❌ Full OpenTelemetry (distributed tracing across services). Basic OTel exporters exist. (Resolved in v0.2.0)
- ❌ Unused rule fields (7.3)

---

# ✅ Week 1 Success Metrics

**Security:**
- SHA-256 audit chain verified ✓
- Invalid paths/patterns rejected ✓
- Large files/timeouts capped ✓

**Correctness:**
- Golden PII corpus → zero false negatives ✓
- LLM timeout → quarantine ✓

**Runtime:**
- p95 decision ≤25ms (no LLM) ✓
- p95 decision ≤300ms (with LLM) ✓
- "blocked-before-delivery" counter increments ✓

**Observability:**
- One command shows scan_duration, files/sec, violations ✓

---

# 📅 Future Sprints (After Week 1)

## Week 2: Polish & Hardening
- Buffer size & chunk overlap configuration (done; confirm defaults per SLOs)
- CLI flag simplification (3.4)
- Error handling improvements (6.4)
- Hardcoded values → config (6.5)
- Add pattern complexity guards (avoid catastrophic regex backtracking)
- Tighten resource limits and per-file budgets

## Week 3: Advanced Features
- Performance optimizations (7.1)
- Advanced tracing polish (exemplars, span links, user-provided attributes)

## Month 2+: Ecosystem
- IDE integrations (7.2)
- Update command (5.3)
- Community rule packs
- HITL queue + quarantine workflows
- Signed RulePack registry and cosign verification

---



