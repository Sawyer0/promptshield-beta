# PromptShield Post‑Pivot Ship Plan

This plan turns the pivot spec into a first‑useable, production‑credible package. Items below are checklists with explicit DoD criteria.

## 1) Overview & Success Criteria

- [ ] Compliance officer can select GDPR Art. 32 → publish a requirement → export a Coverage Matrix showing Enforced on ≥1 endpoint.
- [ ] Developer receives a Jira ticket or GitHub PR stub → binds two detectors to an endpoint via YAML/API → normalized events flow within minutes (SDKs).
- [ ] Auditor opens an expirable link → views Coverage Matrix + Incident Timeline → verifies hash chain for sampled events.

## 2) Deliverables (Ship Checklist)

### A. Seed Bundle (EU/UK)
- [ ] Seed bundle created: `seeds/compliance/eu-uk-controls.v1.json` (or split files)
- [ ] JSON Schemas: `docs/schemas/{framework,control,requirement,binding}.json`
- [ ] CLI ingest: `psctl library load -f seeds/compliance/eu-uk-controls.v1.json`
- [ ] API returns seeded items: `GET /api/frameworks`, `GET /api/controls?framework=gdpr`
- [ ] Docs added: how to load, browse, and create requirements from seeds

### B. SDKs (Detector Event Emitters)
- [ ] Python SDK: `sdks/python/promptshield_sdk` (emit_event, correlation_id middleware)
- [ ] Node SDK: `sdks/node/promptshield-sdk`
- [ ] Idempotency + retries + backoff implemented
- [ ] Example integrations and quickstarts committed
- [ ] Optional: CI publish to PyPI/npm (credentials stubbed for now)

### C. Report Templates
- [ ] Coverage Matrix template (en‑GB PDF/HTML) + JSON/CSV schema
- [ ] Incident Timeline template (en‑GB PDF/HTML) + JSON/CSV schema
- [ ] `POST /api/reports` generates PDF + JSON/CSV; `GET /api/reports/{id}` serves
- [ ] Auditor links: expirable signed URLs
- [ ] EU/UK auditor wording reviewed (tone, disclaimers)

### D. Policy Packs (OPA)
- [ ] Rego: arbitration (`any‑block‑blocks`, quorum, weights)
- [ ] Rego: binding validation (requirements ↔ detectors ↔ endpoints)
- [ ] Rego: ingestion validation (schema/enums/idempotency)
- [ ] Tests: `policy/opa/tests/*_test.rego` with golden I/O
- [ ] Bundle packaging + hot reload documented

### E. Integrations (DevSec Loop)
- [ ] Jira ticket template on requirement publish
- [ ] GitHub PR stub for IaC YAML (`promptshield.enforcement.yaml`)
- [ ] Slack alert Block Kit for violations
- [ ] SIEM field mapping for Splunk/Datadog/ELK
- [ ] Webhook configuration guide (endpoints, secrets, retries)

### F. Security Hardening
- [ ] mTLS between components (enable flag + CA mount + config examples)
- [ ] Per‑tenant KMS keys for optional raw payload encryption
- [ ] Default privacy: store hashes/snippets only (`PS_EVIDENCE_STORE_RAW=false`)
- [ ] Key rotation runbook + test
- [ ] E2E test: encrypted at rest + mTLS enforced

### G. Ops/SLOs
- [ ] Metrics: ingest success/errors, evidence lag, OPA eval latency
- [ ] Grafana dashboard + alert rules committed
- [ ] Evidence lag alert (threshold‑based)
- [ ] Daily Merkle root sealing job + success metric
- [ ] Synthetic event pipeline monitor

### H. GTM Package
- [ ] `docs/GTM.md`: tiers, EU/UK positioning, detector‑agnostic partner page
- [ ] `docs/partners/detectors.md`: adapter program + certification steps
- [ ] Messaging copy: detector‑agnostic orchestration + audit‑ready evidence
- [ ] Optional: simple ROI calculator walkthrough

## 3) API & IaC Readiness

- [ ] Minimal APIs implemented (or stubbed with docs):
  - Library: `GET /api/frameworks`, `GET /api/controls?framework=...`
  - Requirements: `POST /api/requirements`, `GET /api/requirements?status=published`
  - Dev/Sec: `POST /api/endpoints`, `POST /api/detectors`, `POST /api/bindings`
  - Events: `POST /api/detector-events`, `GET /api/evidence?requirement_id=...&from=...&to=...`
  - Reports: `POST /api/reports`, `GET /api/reports/{id}`
- [ ] IaC example: `promptshield.enforcement.yaml` + `psctl apply -f ...`
- [ ] Adapter Guide: `detector_event/v1` schema, headers, idempotency, signatures

## 4) Data & Privacy Controls

- [ ] Tenant toggle: no raw content storage (hash/snippet only)
- [ ] Redaction/masking tested in SDK examples
- [ ] Evidence retention policies configurable by plan
- [ ] Auditor verification guide for hash chain

## 5) Timeline & Ownership (Editable)

- Week 1–2: Seed bundle + SDKs + OPA defaults
- Week 2–3: Reports + Integrations (Jira/PR/Slack/SIEM)
- Week 3–4: Security hardening + Ops/SLOs
- Week 4–6: UI polish + auditor links + GTM docs

Owners (TBD):
- Seed bundle: ____  | SDKs: ____  | OPA: ____  | Reports: ____
- Integrations: ____ | Security: ____ | Ops/SLOs: ____ | GTM: ____

## 6) Risks & Mitigations

- Detector variance → SDKs + OPA `validate` policy; strict JSON Schema
- Latency concerns → evidence path async now; inline comes as phase 2
- Legal wording → `summary_plain` + locale review; avoid verbatim ISO text
- Adoption friction → Jira/PR/Slack templates + IaC quickstarts to reduce time‑to‑value

## 7) Release Gates (Definition of Done)

- [ ] Compliance officer scenario validated (demo tenant)
- [ ] Developer scenario validated (YAML/API + events flowing)
- [ ] Auditor scenario validated (expirable link + hash chain verification)
- [ ] Security review passed (mTLS/KMS/privacy toggles)
- [ ] Ingest load test at target EPS; SLO dashboards green

