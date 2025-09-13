**Adapter Guide (detector_event/v1)**

Purpose
- Standardize detector outputs to a single JSON schema and provide easy SDK hooks.

Event schema (required)
- schema_version: "detector_event/v1"
- detector_id, version: strings (stable slug + semver/commit)
- endpoint: route or logical service (e.g., "/v1/chat", "chatbot-prod")
- direction: request | response | tool | model
- decision: allow | alert | redact | quarantine | block
- reason: machine code (e.g., "pii.email", "pi_direct_ignore")
- event_ts: RFC3339 UTC
- correlation_id: request/trace id

Recommended
- confidence (0..1), severity, rule_id, labels, latency_ms, cost
- source (webhook|sdk|sidecar|inline|provider), tenant_id
- controls (framework → [control_ids])
- trace_id/span_id, event_id (idempotency), extensions, signature

HTTP ingestion
- POST /api/detector-events
- Headers: Authorization (Bearer), X-PS-Tenant-ID (optional), X-Idempotency-Key (optional)
- Content-Type: application/vnd.promptshield.detector-event+json

cURL example
- curl -X POST "$PS_API_URL/api/detector-events" \
  -H "Authorization: Bearer $PS_API_TOKEN" \
  -H "Content-Type: application/vnd.promptshield.detector-event+json" \
  -d '{
    "schema_version":"detector_event/v1",
    "detector_id":"promptguard_v2",
    "version":"2.3.1",
    "endpoint":"chatbot-prod",
    "direction":"response",
    "decision":"block",
    "reason":"pii.email",
    "confidence":0.92,
    "event_ts":"2025-09-13T14:02:55Z",
    "correlation_id":"req_abc123"
  }'

Python SDK (stub interface)
- from promptshield_sdk import Client
- client = Client(api_url=os.getenv("PS_API_URL"), token=os.getenv("PS_API_TOKEN"))
- client.emit_event(detector_id="promptguard_v2", version="2.3.1",
  endpoint="chatbot-prod", direction="response",
  decision="block", reason="pii.email",
  confidence=0.92, correlation_id=req_id)

Node SDK (stub interface)
- import { PromptShield } from '@promptshield/sdk'
- const ps = new PromptShield({ apiUrl: process.env.PS_API_URL, token: process.env.PS_API_TOKEN })
- await ps.emitEvent({ detector_id: 'promptguard_v2', version: '2.3.1',
  endpoint: 'chatbot-prod', direction: 'response', decision: 'block', reason: 'pii.email', confidence: 0.92, correlation_id: reqId })

Idempotency & retries
- Provide X-Idempotency-Key per event (hash of detector_id + correlation_id + event_ts)
- Retry on 5xx with jittered backoff; do not retry 4xx schema errors

Schema references
- docs/schemas/seed_bundle.schema.json
- docs/schemas/control.schema.json
- docs/schemas/requirement.schema.json
- docs/schemas/binding.schema.json

