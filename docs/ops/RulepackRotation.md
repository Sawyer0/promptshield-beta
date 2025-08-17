# 📜 RulePack Rotation & Rollback Playbook

> Applies to PromptShield Enforcer ≥ v0.2.0

| Step | Command / Action | Notes |
|------|------------------|-------|
|1. Validate new RulePack locally|`promptshield rules:validate mypack.yaml`|Fail fast before upload |
|2. Upload as **draft**|`POST /v1/rulepacks?activate=false`|Idempotency-Key recommended |
|3. Approve & Activate|`PUT /v1/rulepacks/active` with `If-Match:"<current-etag>"`|Optimistic-lock ensures no concurrent change |
|4. Monitor canary window| Prometheus alert on `ps_rulepack_activations_total` & error SLOs |Delay can be configured via `PS_RULEPACK_CANARY_DELAY_SECONDS` |
|5. Rollback (fail-closed opt-in)|`PUT /v1/rulepacks/active` with previous pack ID|Alternatively set `PS_ENFORCER_MODE=enforce` to block on violations |
|6. GC old versions|Automatic: background purge keeps last `PS_RULEPACK_RETENTION`|Manual: `POST /v1/rulepacks/gc` (future) |

## Fail-Closed vs Fail-Open

| Mode | Env | Behaviour |
|------|-----|-----------|
|**observe** (default) | `PS_ENFORCER_MODE=observe` | Always 200 / allow, headers indicate decision |
|**quarantine** | `PS_ENFORCER_MODE=quarantine` | 403 on signals; body preserved |
|**enforce** | `PS_ENFORCER_MODE=enforce` | 403 on signals; body dropped |
|
If DB is down at startup the service auto-downgrades to **observe** until healthy.

## Metrics to Watch

* `ps_rulepack_activations_total` – increments on every activation (expect +1)
* `ps_rulepack_validation_failures_total` – should remain 0 after validation step
* `ps_policy_bypass_total{reason="no_rules"}` – any spike after activation indicates regressions

## Checklist

- [ ] RulePack YAML passes `rules:validate`
- [ ] Activation returns 200 and new ETag
- [ ] Canary period completes without error SLO alerts
- [ ] `ps_rulepack_activations_total` increased exactly once
- [ ] Rollback plan tested in staging
