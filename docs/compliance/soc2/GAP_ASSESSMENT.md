### SOC 2 Gap Assessment (PromptShield)

Scope
- System: PromptShield Enforcer (HTTP/gRPC), Envoy integration, Gateway API v1
- Environments: Dev, Staging, Production (multi‑region)
- Trust Services Criteria: Security (required), Availability, Confidentiality (target)

Current State (technical)
- Transport security: TLS/mTLS supported for HTTP and gRPC
- AuthZ/Z: Admin token; user token; optional OIDC; per‑tenant quotas
- Audit/Observability: NDJSON audit logger; Prometheus metrics; tracing hooks; health/ready endpoints
- Secrets: externalized via env/volumes; OS keyring helpers
- Config: versioned API, rulepack validation + reload

Organizational/Process Controls – Gaps and Actions
- Access Control
  - Gap: Central SSO/MFA, quarterly access reviews, joiner/mover/leaver
  - Action: Enforce SSO/MFA across services; implement JML process; quarterly reviews; record evidence
- Change Management
  - Gap: Formal change policy, approvals, CAB, change logs retained
  - Action: Adopt change policy; PR reviews required; change tickets linked; weekly CAB notes stored
- Secure SDLC
  - Gap: Documented SDLC, threat modeling, code scanning, dependency management
  - Action: Define SDLC; add SAST/secret scan; Renovate/Dependabot; threat model records per major change
- Incident Response
  - Gap: IR policy, RACI, on‑call, comms plan, postmortems
  - Action: Publish IR plan; set paging/on‑call; run quarterly exercises; store PMs
- Risk Management
  - Gap: Formal risk register, annual assessment, vendor risk
  - Action: Maintain register; annual review; vendor questionnaires + DPAs
- Vulnerability Management
  - Gap: Patch cadence, vuln scanning, SLA, penetration tests
  - Action: Weekly scans; 30/7 day SLAs (high/critical); annual pen test; evidence retained
- Business Continuity/DR
  - Gap: BCP/DR plan, RTO/RPO, backup/restore tests
  - Action: Define RTO/RPO; quarterly restores; annual DR exercise with evidence
- Logging/Monitoring
  - Gap: Centralized retention policy, alerts for security events
  - Action: SIEM or logs bucket with 1‑year retention; alert playbooks + evidence
- Data Classification/Retention
  - Gap: Classification policy, retention schedules
  - Action: Document data classes; define retention + purge procedures
- Asset/Configuration Management
  - Gap: Inventory, baseline configs, hardened images
  - Action: Maintain inventory; CIS baselines; golden images; evidence

Remediation Plan (assign owners/dates)
- Access control: Owner ____, Due ____
- Change management: Owner ____, Due ____
- SDLC/security scanning: Owner ____, Due ____
- Incident response: Owner ____, Due ____
- Risk/vendor: Owner ____, Due ____
- Vulnerability mgmt: Owner ____, Due ____
- BCP/DR/backups: Owner ____, Due ____
- Logging/alerts: Owner ____, Due ____
- Data classification/retention: Owner ____, Due ____
- Asset/config mgmt: Owner ____, Due ____

Evidence Collection (Type II)
- Tickets: access requests, approvals, change records
- Reviews: quarterly access reviews, CAB minutes
- Exercises: IR tabletop, DR test results, restore logs
- Scans/patches: vulnerability reports, remediation tickets
- Monitoring: alert history, metrics SLOs

