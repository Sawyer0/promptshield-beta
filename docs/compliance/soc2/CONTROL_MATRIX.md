### SOC 2 Control Matrix (Mapping → Evidence → Gaps)

Legend
- Implemented (Tech): ✓
- Process/Evidence Needed: ◻

Access Control
- Admin/API auth (tokens, OIDC) → ✓ Code: `internal/interfaces/http/api` auth middleware; Evidence ◻ access reviews, SSO/MFA
- Least privilege/segregation → ✓ per‑tenant quotas; ◻ role model, approvals

Change Management
- Version control, approvals → ✓ PR reviews; ◻ formal change tickets/CAB logs
- Rollback/DR → ✓ K8s manifests; ◻ DR runbooks, exercise evidence

System Operations
- Health/metrics/tracing → ✓ `/healthz`, `/metrics`, tracing hooks; ◻ alert runbooks/evidence
- Backup/restore → ◻ policy, schedules, test logs

Security
- Transport encryption → ✓ TLS/mTLS envs; ◻ certificate rotation evidence
- Vulnerability mgmt → ◻ weekly scans, remediation SLAs, reports
- Incident response → ◻ IR policy, on‑call, PMs

Confidentiality
- Secrets mgmt → ✓ env/volumes/keyring; ◻ rotation logs, access reviews
- Data retention → ◻ policy, purge evidence

Availability
- SLO/monitoring → ✓ metrics; ◻ documented SLOs, reports

Privacy (if in scope)
- Data minimization/redaction → ✓ redaction utils; ◻ DPIA, notices

Evidence Register
- Policies: security, access, change, IR, BCP/DR, vendor, data classification
- Reviews/logs: access quarterly, CAB minutes, DR/IR exercises, vulnerability reports

