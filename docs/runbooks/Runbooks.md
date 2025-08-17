### Operational Runbooks & Evidence Checklists

Access Reviews (Quarterly)
- Export user list per system; validate least privilege; record approvals; remediate gaps; store minutes

Change Management (Weekly CAB)
- Review deployed changes; link PRs/tickets; risks/rollbacks; store notes

Incident Response
- Triage flow; comms templates; escalation; PM template; evidence checklist

BCP/DR
- Backup schedule; restore procedure; verification steps; quarterly restore test log; annual DR exercise steps

Vulnerability Management
- Weekly scan steps; triage; remediation SLAs; exceptions; report archive

Vendor Review
- Initial security questionnaire; DPA checklist; annual reassessment steps; evidence

Logging & Monitoring
- Alert catalog; on‑call playbooks; evidence of alert tests; dashboard snapshots

Messaging Outages
- Monitor stream_lag_seconds > 60s alert.
- For outages: Check Redis health, restart consumers if needed.
- DLQ replay: Use redis-cli to read from rulepacks.dlq, inspect, re-publish to original stream if valid.

Redis Failover
- Use Cluster mode. Test by killing master, ensure slaves promote.
- Enable AOF persistence on server with appendonly yes, appendfsync everysec.
- Failover test: Simulate by stopping Redis, verify app retries and recovers.

