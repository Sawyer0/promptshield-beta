### Multi‑Region Inline Enforcement (Enterprise Profile)

Objectives
- Enforce policies inline via Envoy `ext_proc` with mTLS
- Regional isolation: secrets, metrics, usage, and operations per region
- SLOs for availability and latency; alerts and runbooks

Regions & Labeling
- Set `PS_REGION` env in each deployment; use as usage prefix and metrics label
- Prometheus: attach `region` label at scrape; aggregate via Thanos/Cortex

Security/mTLS
- Enforcer HTTP: `PS_ENFORCER_TLS_MODE=require`, mount `/tls/server.crt|key`, `PS_ENFORCER_TLS_CLIENT_CA` when client certs required
- Enforcer gRPC: `PS_ENFORCER_GRPC_TLS_MODE=require`, `PS_ENFORCER_GRPC_TLS_CLIENT_CA` to require client certs from Envoy
- Envoy upstream mTLS configured with client cert and trusted CA

Secrets
- Store certs and tokens in regional secret manager; sync to K8s Secrets; mount into pods
- Rotate on schedule; use overlapping validity for certs

Usage & Quotas
- Enable Redis usage per region:
  - `PS_USAGE_REDIS_ADDR=redis.<region>.svc:6379`
  - `PS_USAGE_PREFIX=$PS_REGION`
  - Optional: `PS_USAGE_TTL_DAYS=35`
- Forward `x-tenant-id` from edge; use per‑tenant quotas via `QuotaStore`

SLOs (example)
- Availability: 99.9% monthly for `/check` and gRPC ext_proc
- Latency: p95 < 300 ms for 64KB bodies at regional concurrency baseline

Alerting (Prometheus examples)
- High violation rate:
  - expr: `sum by (region) (rate(ps_enforcer_decisions_total{decision!="allow"}[5m])) > 50`
- Latency SLO burn:
  - expr: `histogram_quantile(0.95, sum by (le,region) (rate(ps_enforcer_request_duration_seconds_bucket[5m]))) > 0.3`
- Availability drop:
  - expr: `probe_success{job="enforcer-http"} == 0`

Runbooks
- See `docs/runbooks/Runbooks.md` for incident/DR and change management procedures

