**Compliance Orchestration**
- Purpose: Bridge security enforcement and regulatory evidence.
- Orchestrate detectors: Wrap any filter/guardrail (PromptGuard, Jatmo, provider filters) under endpoint policies.
- Map to frameworks: SOC 2, HIPAA, GDPR, NIST AI RMF, EU AI Act.
- Evidence: Hash‑chained, tamper‑evident audit trails; export JSON/CSV/PDF.

**How It Fits**
- Gateway + Envoy ext_proc: zero/low‑code runtime enforcement and telemetry.
- RulePacks: declarative policies with compliance mappings.
- Audit services: generate status, validation, and evidence artifacts.

**Roles**
- Developers: assign RulePacks to endpoints, integrate via proxy/API.
- Security Engineers: curate detectors and policies, set fail/rewrite/observe modes.
- Compliance Officers: run reports, export evidence, validate frameworks.
- Auditors: read‑only evidence exports and integrity verification.

**Key Endpoints**
- Compliance: `GET /api/compliance/{tenantId}/report?standard=SOC2&start=...&end=...`
- Audit admin: `GET /v1/admin/audits`, `POST /v1/admin/audits/search`, `GET /v1/admin/audits/export`, `GET /v1/admin/audits/object/{type}/{id}`

**Detector Orchestration Patterns**
- Inline: use provider moderation (OpenAI/Anthropic/Gemini) with RulePack obligations.
- Sidecar: Envoy ext_proc for stream inspection and redaction.
- Hybrid: combine keywords/regex + provider filters + third‑party guards under one decision.

**Evidence Model**
- Audit entries are hash‑chained per day; daily Merkle roots recorded (see migrations/0004_audit_merkle_roots.sql).
- Evidence exports include integrity hashes for hand‑off.

**Next Steps**
- Use `rules/soc2-bridge-example.yaml` as a template.
- Assign RulePacks to endpoints via admin API.
- Query `GET /api/compliance/{tenantId}/report` for compliance snapshots.
