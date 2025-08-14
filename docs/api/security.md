# Security Notes

## Transport Security

- HTTP TLS: set `PS_ENFORCER_TLS_CERT` and `PS_ENFORCER_TLS_KEY`
- HTTP mTLS: additionally set `PS_ENFORCER_TLS_CLIENT_CA` to require and verify client certs
- gRPC TLS: set `PS_ENFORCER_GRPC_TLS_CERT` and `PS_ENFORCER_GRPC_TLS_KEY`
- gRPC mTLS: additionally set `PS_ENFORCER_GRPC_TLS_CLIENT_CA`

Certificates should be provisioned via Kubernetes Secrets (see `deployments/kubernetes/enforcer.yaml`) or mounted securely in other environments.

## Request Authentication (HTTP)

- Bearer token (Authorization header):
  - `Authorization: Bearer <token>`
  - Enable by setting `PS_ENFORCER_AUTH_TOKEN=<token>`
- Custom header:
  - `X-PS-Token: <token>`
  - Also validated against `PS_ENFORCER_AUTH_TOKEN`

If `PS_ENFORCER_AUTH_TOKEN` is unset, the `/check` endpoint allows unauthenticated access (suitable for sidecar-only or internal networks).

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