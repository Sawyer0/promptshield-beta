### Gateway Responses and Telemetry

HTTP Gateway (`/v1/check`) returns:

- Decision headers: `x-ps-decision: allow|quarantine|deny`, `x-ps-reason`, `x-ps-request-id` (and `x-ps-trace-id` when tracing is enabled)
- JSON body: `{ "decision": "allow|quarantine|deny", "reason": "string", "violations": 0, "request_id": "uuid" }`

gRPC (Envoy ext_proc) injects decision headers and may mutate response bodies when enabled.

### Audit events

When `audit_file` (or `PS_AUDIT_FILE`) is set, the enforcer writes NDJSON audit events with hash chaining:

- Files rotate daily: `<base>.YYYY-MM-DD.ndjson`
- Sensitive values are redacted (API keys, tokens) before writing. You can disable this with `redaction.enabled: false` (or `PS_REDACTION_ENABLED=false`).

Event shapes (illustrative):

```
{ "timestamp": "2025-01-01T00:00:00Z", "type": "config_effective", "data": { "enforcement_mode": "observe", "fail_on": "HIGH" }, "hash": "...", "prev_hash": "..." }
{ "timestamp": "2025-01-01T00:00:01Z", "type": "decision", "data": { "decision": "allow", "rule_id": null }, "hash": "...", "prev_hash": "..." }
{ "timestamp": "2025-01-01T00:00:02Z", "type": "decision", "data": { "decision": "quarantine", "rule_id": "pii-quarantine" }, "hash": "...", "prev_hash": "..." }
```

Notes:

- Hash chaining uses SHA‑256 over a canonical JSON payload
- Audit event payloads never include raw secrets; redaction is applied recursively
