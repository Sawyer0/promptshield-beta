# PDP Integration Guide

This gateway integrates with a vendor-neutral Policy Decision Point (PDP). You can plug in OPA (Rego), Cedar, or any external PDP. The gateway sends a canonical Subject–Action–Resource–Environment (SARE) request and expects a compact response: decision + obligations + reason.

## Modes
- Sidecar HTTP (recommended): Run OPA locally and point PS_PDP_ENDPOINT to http://127.0.0.1:8181/v1/data/your/entry
- In-process (future option): Evaluate Rego in-process using OPA’s Go SDK (no network). The current code is structured to allow this without changing handlers.

## Env Variables
- PS_PDP_ENDPOINT: PDP evaluate endpoint URL (empty disables PDP)
- PS_PDP_API_KEY: Bearer token (optional)
- PS_PDP_TIMEOUT_MS: Request timeout to PDP (default 2000)
- PS_PDP_FAIL_OPEN_TOOL=false: fail-closed for tool.invoke if errors
- PS_PDP_FAIL_OPEN_CHECK=false: fail-closed for /check if errors
- PS_PDP_CACHE_TTL_MS: decision cache TTL (default ~1500)
- PS_PDP_CACHE_MAX_ENTRIES: max cache entries (default 10000)
- PS_PDP_POLICY_EPOCH: epoch to invalidate cache entries on policy changes

## Request Model (SARE)
{
  "subject": {"userId":"u1","tenantId":"t1","roles":["admin"]},
  "action": "tool.invoke",
  "resource": {"type":"tool","id":"http_fetch","attributes":{"endpoint":"/v1/things"}},
  "environment": {"correlationId":"...","time":"...","attributes":{"path":"/api/tools/exec","method":"POST"}}
}

## Response Model
Top-level (preferred) or OPA nested result are supported.

Top-level:
{
  "decision": "PERMIT|DENY|INDETERMINATE|NOT_APPLICABLE",
  "obligations": [{"type":"mask","key":"pattern","value":"\\bSSN\\b"}],
  "reason": "policy_name",
  "risk": 0.0,
  "ttlMs": 2000,
  "cacheable": true,
  "provider": "opa"
}

OPA nested (optional):
{"result": { "decision": "PERMIT", ... }}

## Example Rego Policy (allow/deny with obligations)
package your.entry

import future.keywords.if

# input carries SARE fields under input.subject, input.action, input.resource, input.environment

permit := {
  "decision": "PERMIT",
  "reason": "default_allow",
  "cacheable": true,
  "ttlMs": 2000,
}

deny(reason) := {
  "decision": "DENY",
  "reason": reason,
  "cacheable": true,
  "ttlMs": 2000,
}

# Example: deny tool.invoke on http_fetch unless endpoint is allowlisted
allowlist := {"/v1/safe", "/v1/data"}

result := deny("tool_not_allowed") if {
  input.action == "tool.invoke"
  input.resource.type == "tool"
  input.resource.id == "http_fetch"
  not input.resource.attributes.endpoint in allowlist
}

result := permit if {
  # otherwise permit
}

## Caching & Epoch
The gateway wraps the PDP client with a TTL cache. It keys decisions by (tenant,user,action,resource,attrsHash,policyEpoch). Bump PS_PDP_POLICY_EPOCH (or wire a runtime update) to invalidate cached decisions after policy changes.

## Security & Air-Gap
- Sidecar OPA keeps PDP calls local. No external egress is required.
- For stricter setups, use in-process evaluation (future option) or block egress at the network layer.

## Metrics
- ps_gateway_cache_operations_total{operation=hit|miss|error,cache_type=pdp}
- ps_time_to_first_decision_seconds

## Obligations
Standardize and evolve obligations (filterTag, allowScope, mask, redactField, maxTokens). The gateway already wires tool gating and output redaction; RAG retrieval filters can be added by mapping allowScope/exclude_tags to DB/vector filters.

