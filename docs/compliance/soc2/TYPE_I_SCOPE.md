### SOC 2 Type I – Scope Statement (PromptShield Enterprise Inline Enforcement)

System Components (in scope)
- PromptShield Enforcer services (HTTP API on :9090, gRPC `ext_proc` on :9091)
- Gateway API v1 endpoints (`/v1/*`) including config, rulepacks, admin, events, usage, stats, metrics
- Envoy integration (External Processing) as the enforcement plane interface
- Observability: Prometheus `/metrics`, tracing hooks, audit logger (NDJSON)
- Build/release pipeline used to produce production artifacts

Environments & Regions
- Production regions: us-east-1 (primary), eu-west-1 (secondary) — identical controls in each region
- Staging environment mirrors production controls (excluding customer data)

Data Types Processed
- Request/response bodies for policy evaluation (streamed; not stored at rest by default)
- Operational telemetry: metrics, traces (sampled), decision events; optional NDJSON audit logs (redacted)
- Usage counters (per‑tenant, per‑route) in Redis when configured

Trust Services Criteria (TSC)
- Security (required): logical access, network security, change mgmt, logging/monitoring
- Availability: SLOs and monitoring for enforcer services (SLOs below)
- Confidentiality: secrets management, encryption in transit, optional audit redaction, data handling policies

SLOs (initial targets)
- Availability: 99.9% monthly for `/check` and gRPC ext_proc per region
- Latency: p95 ≤ 300 ms for 64KB bodies at baseline concurrency per region
- Error budget policy: 43m/mo downtime; burn alerts at 2%/hour and 5%/hour

Out of Scope
- Customer‑managed Envoy clusters and application backends
- Customer infrastructure, identity providers, and external networks beyond the enforcer ingress/egress

Assumptions/Dependencies
- mTLS is enforced between Envoy and Enforcer in production
- Admin/API endpoints gated by tokens and (optionally) mTLS/OIDC
- Secrets provided via K8s Secrets mounted as files or env; rotated by customer’s secret manager

