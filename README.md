## Quick Demo

### Option A: Docker Compose (local Envoy + Enforcer + Backend)

1) Start the stack:

```bash
docker compose up -d --build
```

2) Replacement demo (response action: replace):

```bash
curl -sS -i -X POST http://localhost:8080/anything \
  -H 'content-type: application/json' \
  --data-binary '{"data":"replace_me"}'
```

Expected:
- 200 OK
- x-ps-decision: replace
- Body: `REPLACED_BODY`

3) Redaction demo (response action: redact):

```bash
curl -sS -i -X POST http://localhost:8080/anything \
  -H 'content-type: application/json' \
  --data-binary '{"data":"sk_test_abcdefghijklmnopqrstuvwx"}'
```

Expected:
- 200 OK
- Response body with sensitive token masked as `[REDACTED]`

4) Quarantine demo (fast-path leak guard):

```bash
curl -sS -i -X POST http://localhost:8080/anything \
  -H 'content-type: application/json' \
  --data-binary '{"data":"api_key=SECRET123456"}'
```

Expected:
- 403 Forbidden
- x-ps-decision: quarantine

### Option B: Kubernetes (beta)

Prereqs: A running Kubernetes cluster and `kubectl` configured.

```bash
kubectl apply -f deployments/kubernetes/enforcer.yaml --validate=false
kubectl -n promptshield rollout status deploy/promptshield-enforcer
kubectl -n promptshield port-forward svc/promptshield-enforcer 9090:9090 9091:9091 &
```

Then repeat the curl demos above against your Envoy gateway or call the Gateway API directly at `http://localhost:9090/v1/check`.

## PromptShield – LLM API Gateway for LLM Safety (Go + Envoy)

PromptShield is an Envoy-integrated API Gateway that enforces LLM safety policies on requests and responses in real time. It evaluates content using a progressive rule system:

- Level 1: Keyword matching
- Level 2: Regex patterns
- Level 3: Semantic/LLM analysis (opt-in)

Built for production with streaming inspection, deterministic ordering, budgets, and observability.

Hyperscan (advanced): For higher‑throughput regex evaluation, Docker builds can enable an optional Hyperscan fast‑path by passing `--build-arg ENABLE_HYPERSCAN=1`.

### Quickstart

1) Run with Docker Compose (Envoy + Enforcer)

```bash
docker compose up --build -d
```

2) Send a decision request (HTTP Gateway)

```bash
curl -s -X POST http://localhost:9090/v1/check \
  -H 'content-type: text/plain' \
  --data 'hello world' -i | sed -n '1,20p'
```

Headers include `x-ps-decision: allow|quarantine|deny` and `x-ps-reason`.

3) Wire Envoy ext_proc (streaming body inspection)

- Point Envoy to the enforcer gRPC server implementing `envoy.service.ext_proc.v3.ExternalProcessor`.
- See `docs/ENVOY_INTEGRATION.md` and `docs/Envoy.md` for full examples.

### Production Readiness

| Feature | Status | Safe for Production? | Notes |
|---------|--------|---------------------|-------|
| Gateway enforcement (HTTP `/v1/check`) | ✅ Stable | **Yes** | Deterministic headers + JSON payload |
| Envoy gRPC `ext_proc` streaming | ⚠️ Beta | **Limited** | Streaming with budgets; body redaction optional |
| 3‑Tier Rules | ✅ Stable | **Yes** | Keywords, regex, semantic analysis |
| Configuration | ✅ Stable | **Yes** | Env vars + policy bundles (RulePacks) |
| Audit & Metrics | ✅ Stable | **Yes** | Prometheus metrics; hash‑chained audit events |
| RulePack composition/extends | ✅ Supported | **Yes** | Deterministic merge and strategies |

### Features

- Real‑time enforcement over HTTP (`/v1/check`) and Envoy `ext_proc` (streaming)
- RulePacks with L1/L2/L3, composition (`first_match`, `priority_order`), and context gating (`when`/`unless`)
- Budgets and SLOs: per‑request timeout, max stream bytes, LLM call limits
- Deterministic ordering; request correlation via `x-ps-request-id`
- Observability: Prometheus metrics and OpenTelemetry traces
- Optional redaction mutations for response bodies via `ext_proc`

### Configuration

Environment variables (illustrative):

```bash
export PS_ENFORCER_ADDR=:9090
export PS_ENFORCER_GRPC_ADDR=:9091
export PS_ENFORCER_RULEPACK=rules/prompt-injection.yaml
export PS_ENFORCER_TIMEOUT=300ms
export PS_ENFORCER_MAX_BODY_BYTES=1048576
export PS_ENFORCER_FAIL_ON=HIGH
export PS_ENFORCER_ENFORCEMENT_MODE=observe   # observe|redact|quarantine|enforce
export PS_ENFORCER_REDACTION_MUTATION=true    # enable body mutation in ext_proc
# Optional telemetry
export PS_TELEMETRY=1
export PS_TELEMETRY_ENDPOINT=otel-collector:4317
```

Policy bundles (RulePacks) control signals, thresholds, budgets, and actions. See `docs/RulePacks.md`.

### Documentation

- Envoy integration: `docs/ENVOY_INTEGRATION.md`, `docs/Envoy.md`
- Gateway API: `docs/api/Gateway-API-v1.md`, `docs/api/openapi.yaml`
- RulePacks: `docs/RulePacks.md`
- Runtime architecture: `docs/Runtime-Architecture.md`
- Metrics: `docs/api/metrics.md`
- Performance & SLAs: `docs/Performance.md`

### Development

```bash
make fmt
make tidy
make test
```

### License

Copyright © 2025 PromptShield authors.

