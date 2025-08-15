# Security Notes

## Transport Security

- HTTP TLS: set `PS_ENFORCER_TLS_CERT` and `PS_ENFORCER_TLS_KEY`
- HTTP mTLS: additionally set `PS_ENFORCER_TLS_CLIENT_CA` to require and verify client certs
- gRPC TLS: set `PS_ENFORCER_GRPC_TLS_CERT` and `PS_ENFORCER_GRPC_TLS_KEY`
- gRPC mTLS: additionally set `PS_ENFORCER_GRPC_TLS_CLIENT_CA`

Certificates should be provisioned via Kubernetes Secrets (see `deployments/kubernetes/enforcer.yaml`) or mounted securely in other environments.

## Request Authentication (HTTP)

### User Authentication (API Endpoints)
- Bearer token (Authorization header):
  - `Authorization: Bearer <token>`
  - Enable by setting `PS_ENFORCER_AUTH_TOKEN=<token>`
- Custom header:
  - `X-PS-Token: <token>`
  - Also validated against `PS_ENFORCER_AUTH_TOKEN`

- Optional OIDC (JWT validation):
  - Configure `Options.OIDC.Issuer` (and optional `Audience`) to enable JWT verification for user endpoints.
  - Claims from the verified ID token are attached to request context and used for tenancy resolution.

**Protected User Endpoints:**
- `POST /v1/check` - Security decision API
- `POST /v1/scan` - Batch scanning API
- `POST /v1/scan/async` - Async job submission

### Admin Authentication (Management Endpoints)
- Bearer token with admin privileges:
  - `Authorization: Bearer <admin-token>`
  - Enable by setting `PS_ENFORCER_ADMIN_TOKEN=<admin-token>`
- Custom header:
  - `X-PS-Admin-Token: <admin-token>`

**Protected Admin Endpoints:**
- `GET /v1/license` - License and billing information
- `POST /v1/license` - License key updates
- `GET /v1/usage` - Usage and billing data
- `GET /v1/stats` - Performance statistics
- `GET /v1/events` - Real-time event stream
- `PUT /v1/config` - Runtime configuration changes
- `POST /v1/rulepacks` - Rule pack management
- `POST /v1/admin/shutdown` - Service control

**Security Model:**
- If `PS_ENFORCER_AUTH_TOKEN` is unset, user endpoints allow unauthenticated access (suitable for internal networks)
- Admin endpoints always require authentication when `PS_ENFORCER_ADMIN_TOKEN` is set
- Different tokens provide role-based access control

## Tenancy & Quotas

- Tenancy resolution order:
  1. OIDC JWT claims (`tid`, `tenant`, `org`, `azp`)
  2. Header `x-tenant-id`
- Per-tenant quotas may be enforced when `Options.QuotaStore` is configured. Exceeding limits returns `429 RESOURCE_EXHAUSTED`.

## Limits & Budgets

- Body size limit (HTTP `/check`): default 1MiB; override with `PS_ENFORCER_MAX_BODY_BYTES`
- Processing timeout: `PS_ENFORCER_TIMEOUT` (default 300ms)
- Max stream bytes (gRPC): `PS_ENFORCER_MAX_STREAM_BYTES` (default 5,000,000)
 - Streaming window and overlap (gRPC): `PS_ENFORCER_STREAM_WINDOW`, `PS_ENFORCER_STREAM_OVERLAP`
 - Global concurrency cap: `PS_ENFORCER_MAX_STREAMS`
 - Global rate limiting: `PS_ENFORCER_RPS`, `PS_ENFORCER_RPS_BURST`

## Enforcement Modes & Mutations

- Enforcement mode is controlled via `PS_ENFORCER_ENFORCEMENT_MODE` with values:
  - `observe`: never block; headers/metrics only
  - `redact`: allow but apply redaction mutations when applicable
  - `quarantine`/`enforce`: block on policy violations
- To enable body redaction mutations in gRPC `ext_proc`, set `PS_ENFORCER_REDACTION_MUTATION=true`.

## Headers

Responses include decision headers for observability:
- `x-ps-decision: allow|quarantine|deny`
- `x-ps-reason: rule_id|timeout|body_limit|no_signals`
- `x-ps-request-id: uuid`

## Privacy & Telemetry

- Zero sensitive payloads are emitted in logs/traces/metrics. Inputs to providers are redacted/truncated.
- OpenTelemetry (opt-in) via `telemetry.endpoint` exports traces/metrics; correlate using request IDs.

## Decision Events (privacy)

PromptShield emits minimal decision events for telemetry and metrics with zero content/body. Event schema:

```json
{ "type": "decision", "payload": { "ts": 1712345678, "decision": "allow|quarantine|deny", "rule_id": "optional" } }
```