**PromptShield Post‑Pivot Architecture Spec**

This document consolidates the post‑pivot design: a detector‑agnostic Security & Compliance Orchestration Platform for LLM applications. It captures the universal detector event contract, OPA‑driven policy orchestration, compliance framework library, evidence ledger, reporting, integrations, and minimal APIs.

**Core Mission**
- Not a detector: do not invent detection.
- Orchestrate: unify vendor/open‑source/custom detectors across endpoints.
- Evidence: map detector outputs to compliance frameworks automatically.
- Audit: produce tamper‑evident, regulator‑ready reports.

**Architecture Layers**
- Detector Orchestration
  - Universal detector event schema (adapter contract)
  - Multiple ingestion modes (webhook, SDK, sidecar; inline later)
  - Per‑tenant detector catalog; binding model (requirement → detectors → endpoints)
  - Version pinning, OPA‑driven arbitration and obligations
- Compliance Framework Library
  - Frameworks (GDPR, EU AI Act, SOC2, HIPAA, ISO, OWASP LLM Top‑10)
  - Controls as normalized units; requirements curated by compliance
  - Cross‑mapping across frameworks to reduce redundancy
- Evidence Ledger
  - Event ingestion (`POST /api/detector-events`) and enrichment
  - Tamper‑evident hash chaining + daily Merkle roots
  - Retention (per plan), SIEM export
- Compliance Reporting
  - Coverage Matrix and Incident Timelines
  - Report Generator (JSON/CSV/PDF), localized; auditor portal
- Integration Fabric
  - Jira/GitHub/Slack webhooks; IaC exports (Terraform/Helm); SSO/SCIM
- Inline Enforcement (Phase 2)
  - Envoy `ext_proc` → fan‑out to detectors → OPA adjudication → single action back

---

**Universal Detector Event Schema (Adapter Contract)**
- Required
  - `schema_version`: `detector_event/v1`
  - `detector_id`: string (stable vendor+model slug)
  - `version`: string (semver/commit)
  - `endpoint`: string (route or logical service, e.g., `/v1/chat`, `chatbot-prod`)
  - `direction`: `request | response | tool | model`
  - `decision`: `allow | alert | redact | quarantine | block`
  - `reason`: string (machine code, e.g., `pii.email`, `pi_direct_ignore`)
  - `event_ts`: RFC3339 UTC
  - `correlation_id`: propagated request/trace ID
- Recommended
  - `confidence` (0..1), `severity` (`LOW|MEDIUM|HIGH|CRITICAL`)
  - `rule_id`, `labels` (category/subtype/model), `latency_ms`, `cost`
  - `source` (`webhook|sdk|sidecar|inline|provider`), `tenant_id`
  - `controls` (framework → [control_ids])
  - `trace_id`, `span_id`, `event_id` (idempotency), `extensions` (vendor), `signature` (optional)
- Example
```json
{
  "schema_version": "detector_event/v1",
  "detector_id": "promptguard_v2",
  "version": "2.3.1",
  "endpoint": "chatbot-prod",
  "direction": "response",
  "decision": "block",
  "reason": "pii.email",
  "confidence": 0.92,
  "severity": "HIGH",
  "controls": { "SOC2": ["CC6.6"], "GDPR": ["Art. 32"] },
  "event_ts": "2025-09-13T14:02:55Z",
  "correlation_id": "req_abc123",
  "labels": { "category": "pii", "subtype": "email" },
  "latency_ms": 7,
  "source": "webhook"
}
```

**Integration Modes**
- Webhook: `POST /api/detector-events` (single) and `POST /api/detector-events:batch` (array)
  - Headers: `X-PS-Tenant-ID`, `Authorization` (JWT/mTLS), `X-Idempotency-Key`
  - `Content-Type`: `application/vnd.promptshield.detector-event+json`
- SDK: thin wrappers (Python/Node) wrap detector invocation and emit events; auto inject `correlation_id` and retry 5xx with jitter.
- Sidecar: NDJSON ingestion from stdout/file tail; each line is one event in schema.
- Inline (Phase 2): Envoy `ext_proc` → PromptShield → fan‑out → merge → single action back.

**Detector Catalog & Binding**
- Catalog (per tenant): allowed detectors, versions, health, scopes.
- Binding model: requirement → detectors → endpoints; validated before publish.
- Version tracking: detectors pinned; versions recorded in audit logs and reports.

**OPA Policy Engine Integration**
- Use OPA for:
  - Arbitration PDP (`ps.detectors.arb`): combine multiple detector events by correlation into one final action and obligations.
  - Requirement Binding (`ps.compliance.bind`): ensure endpoints satisfy frameworks (detectors present, thresholds met).
  - Ingestion Guard (`ps.detectors.validate`): schema/version/tenant/signature/idempotency checks.
  - Evidence Mapping (`ps.compliance.evidence`): coverage by control and reporting attributes.
- Inputs: tenant_id, endpoint, direction, events[], context, config.
- Data: frameworks/controls, requirements/intents, bindings, detector thresholds/weights, precedence.
- Outputs: action, reasons[], obligations[], controls{}, decision_id, policy_version.
- Deployment: OPA sidecar (REST) and/or WASM in‑process for hot paths; bundles for data/policy distribution.

**Arbitration Policy (Default)**
- Precedence: `block > quarantine > redact > alert > allow`.
- Thresholds: per detector/reason; `confidence >= threshold`.
- Optional quorum/weights: N‑of‑M or weighted score across detectors.
- Tie‑breakers: severity → confidence → deterministic detector order.
- Time window: aggregate by `correlation_id` within request window (+grace).

---

**Compliance Framework Library (Spec)**
- Framework object (YAML)
```yaml
framework:
  key: gdpr
  name: "General Data Protection Regulation"
  jurisdiction: ["EU","UK"]
  publisher: "EU"
  version: "2024.12"
  license_type: public_domain   # public_domain | summary_only | custom
  default_locale: "en-GB"
  references:
    - "https://gdpr.eu"
```

- Control object (YAML)
```yaml
control:
  framework_key: gdpr
  control_key: "GDPR-ART-32-1"
  title: "Security of Processing"
  summary_plain: "Implement appropriate technical/organizational measures to ensure confidentiality, integrity, availability of personal data."
  obligations:
    - "Prevent unauthorized disclosure in outputs."
    - "Maintain tamper-evident records of processing security events."
  data_categories: ["PII","Sensitive"]
  scope_conditions:
    - "Endpoints that process or may output personal data"
  suggested_intents: ["block_outbound_pii","log_all_decisions"]
  required_evidence:
    - "tamper_evident_logs"
    - "violation_counts"
    - "sample_evidence_with_snippets_or_hashes"
    - "change_history_of_rules"
  risk_category: "security_of_processing"
  cross_maps:
    - { framework_key: "iso_27001", control_key: "A.12" }
    - { framework_key: "eu_ai_act", control_key: "AI-ACT-ART-9" }
  locales:
    - { locale: "en-GB", title: "Security of Processing", summary: "" }
  version: "1.0.0"
  status: "active"          # active | superseded
```

- Requirement object (YAML)
```yaml
requirement:
  id: "req_1234"
  control_ref: { framework_key: "gdpr", control_key: "GDPR-ART-32-1" }
  intent: block                 # block | alert | redact | log_only
  internal_name: "GDPR-32 Outbound PII Control"
  notes: "Applies to customer-facing chat surfaces in EU region"
  status: published             # draft | approved | published | archived
  version: "1.0.0"
```

- Binding object (JSON/YAML)
```yaml
binding:
  requirement_id: "req_1234"
  detectors:
    - id: "presidio_phi"
      config: { threshold: 0.8, mode: "block" }
    - id: "promptguard_v2"
      config: { threshold: 0.7, mode: "block" }
  endpoints: ["chatbot-prod","rag-api"]
```

**Control Registry**
- Source of truth for frameworks, controls, versions, locales, and cross‑maps.
- Requirement lifecycle: `draft → approved → published → archived`.
- Webhooks on publish/update: create Jira tickets/PR stubs; emit IaC exports.

---

**Evidence Ledger (Tamper‑Evident)**
- Ingestion: `POST /api/detector-events` (and batch). Correlate via `correlation_id` across request/response.
- Append‑only events with per‑row `row_hash` and `prev_hash`; daily Merkle roots.
- PII‑safe defaults: store hashes/snippets; full payload optional and encrypted with per‑tenant KMS.
- Retention: per plan (90d/365d/custom). SIEM export (Splunk/Datadog/ELK) of masked subset.

**Reporting & Forensics**
- Coverage Matrix: framework × requirement × endpoints × detector versions × gaps.
- Incident Timelines: session view keyed by `correlation_id` (detector decisions, actions, playbooks).
- Exports: JSON/CSV/PDF (locale‑aware). Auditor portal with expirable read‑only links.

**Integrations**
- Slack/Jira/GitHub: tickets/PR stubs on publish; alerts on violations; attach evidence packets.
- IaC: Terraform/Helm snippets for requirement bindings and runtime policies.
- SSO/SCIM: enterprise IAM integration (optional phase).

**Playbooks (Optional v1)**
- Conditions over detector results/requirements/endpoint tags.
- Actions: block/redact/quarantine (when inline), create ticket, notify Slack, attach packet.
- Dry‑run/replay on recent traffic before publish.

**Inline Policy Adjudication (Phase 2)**
- Envoy `ext_proc` → Policy Adjudication API merges detector verdicts according to requirement intents.
- Modes: alert‑only, canary, block; fail‑open/closed; latency budget with circuit breakers.

**Platform Non‑Functionals**
- RBAC: Compliance, Dev/Sec, Auditor roles; approval workflows for publishing.
- Multitenancy: isolated boundaries; mTLS/TLS1.3 everywhere.
- Observability: OpenTelemetry traces; metrics for event/sec, lag, coverage, error rates.
- SLOs: e.g., 99.9% event intake; arbitration P95 ≤ 3 ms; inline end‑to‑end P95 ≤ 100 ms.
- Versioning: frameworks/controls and detector versions visible in reports.
- Data Residency: EU region option (GDPR‑first GTM).

---

**UI: Selection & Creation Flow (Compliance)**
- Step 1: Pick framework & filter (jurisdiction, topic, data category, status).
- Step 2: Inspect control (title, summary, obligations, suggested intents, required evidence, cross‑maps, version history).
- Step 3: Create requirement (choose intent; internal name + notes; Publish triggers Jira/PR/IaC export).
- Step 4: Track mapping status (unmapped/mapped/enforced, last event time, detector versions). Export coverage matrix any time.

**Minimal APIs**
- Library
  - `GET /api/frameworks`
  - `GET /api/controls?framework=gdpr&query=processing`
- Requirements
  - `POST /api/requirements`
  - `GET /api/requirements?status=published`
- Dev/Sec
  - `POST /api/endpoints`
  - `POST /api/detectors`
  - `POST /api/bindings` (requirement ↔ detectors ↔ endpoints)
- Events
  - `POST /api/detector-events`
  - `GET /api/evidence?requirement_id=...&from=...&to=...`
- Reports
  - `POST /api/reports` (type, timeframe, locale)
  - `GET /api/reports/{id}`

**Guardrails & Policy Notes**
- Legal: GDPR & AI Act are public; ISO is `summary_only` (no verbatim text).
- Security: per‑tenant encryption for optional payload storage; default to hashes/snippets.
- Latency: evidence‑only path is async; adjudication adds latency later (phase two).
- Privacy: tenant option to store no raw content.

---

**Starter EU/UK Control List (Seed)**
- GDPR (public domain; include summaries, not legal advice)
  1. GDPR‑ART‑5 (Principles) — Intents: `log_all_decisions`, `redact_optional`
  2. GDPR‑ART‑25 (Privacy by design/default) — Intents: `log_all_decisions`; Evidence: config snapshots, change logs
  3. GDPR‑ART‑32‑1 (Security of processing) — Intents: `block_outbound_pii`, `log_all_decisions`
  4. GDPR‑ART‑35 (DPIA for high‑risk processing) — Intents: `log_all_decisions`; Evidence: DPIA link, event sampling
- EU AI Act (public; high‑level duties)
  5. AI‑ACT‑ART‑9 (Risk management) — Intents: `log_all_decisions`; Evidence: runtime logs, risk register linkage
  6. AI‑ACT‑ART‑12 (Record‑keeping/logs) — Intents: `log_all_decisions`; Evidence: tamper‑evident logs, retention policy
  7. AI‑ACT‑ART‑13 (Transparency & instructions) — Intents: `log_all_decisions`; Evidence: detector/policy documentation
- ISO (summary_only; identifiers + our summaries)
  8. ISO27001‑A.9 (Access control) — Intents: `log_all_decisions`; Evidence: RBAC changes, approvals
  9. ISO27001‑A.12 (Operations security) — Intents: `log_all_decisions`; Evidence: health/coverage metrics
- OWASP LLM Top‑10 (community)
  10. OWASP‑LLM‑01 (Prompt Injection) — Intents: `alert_or_block_on_injection`; Evidence: detector verdicts, incident timelines
  11. OWASP‑LLM‑02 (Sensitive Info Disclosure) — Intents: `block_outbound_pii/secrets`; Evidence: masked snippets, counts
  12. OWASP‑LLM‑07 (Model Misuse/Overreach) — Intents: `alert`; playbooks for quarantine/escalation

---

**Seed Bundle (Docs‑First)**
- File(s): `seeds/compliance/eu-uk-controls.v1.json` (or split into `frameworks.json`, `controls.json`).
- Bundle shape:
  - `{"frameworks": [ {framework}... ], "controls": [ {control}... ]}`
- JSON Schemas (for validation): framework/control/requirement/binding payloads.

**Phased Delivery (No Code in This Doc)**
- Phase 1: Event schema + ingestion (webhook/SDK/sidecar), OPA arbitration, evidence ledger, basic coverage report.
- Phase 2: Framework library + cross‑mapping + auditor portal + SIEM exports.
- Phase 3: Integrations (Slack/Jira/GitHub), IaC exports, SSO/SCIM.
- Phase 4: Inline Envoy `ext_proc` fan‑out; policy adjudication API.

**Open Items for Confirmation**
- Intent enum names (e.g., `alert_or_block_on_injection` vs `alert_then_block`).
- Single seed bundle vs split; default locale `en-GB` for EU/UK seeds.
- Evidence export locales/wording templates for EU/US auditors.

---

**Production-Ready Starter Package (DDL, UI, Seeds, API/IaC)**

Database schema (PostgreSQL DDL)
```sql
-- =========================
--  Core: Tenancy & Users
-- =========================
CREATE TABLE tenants (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name            TEXT NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  email           CITEXT NOT NULL,
  full_name       TEXT,
  role            TEXT NOT NULL CHECK (role IN ('compliance','devsec','admin','auditor_view')),
  sso_subject     TEXT,           -- for OIDC/SAML
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, email)
);

-- =========================
--  Framework Library (read-only, vendor-managed)
-- =========================
CREATE TABLE frameworks (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  key             TEXT NOT NULL UNIQUE,         -- e.g., 'gdpr', 'eu_ai_act', 'iso_27001', 'owasp_llm_top10'
  name            TEXT NOT NULL,                -- 'GDPR'
  publisher       TEXT NOT NULL,                -- 'EU', 'ISO', 'OWASP'
  version         TEXT NOT NULL,                -- '2024.12'
  license_type    TEXT NOT NULL CHECK (license_type IN ('public_domain','summary_only','custom')),
  locale_default  TEXT NOT NULL DEFAULT 'en-GB',
  meta            JSONB NOT NULL DEFAULT '{}'   -- links, notes
);

CREATE TABLE controls (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  framework_id        UUID NOT NULL REFERENCES frameworks(id) ON DELETE CASCADE,
  control_key         TEXT NOT NULL,        -- 'GDPR-ART-32-1'
  title               TEXT NOT NULL,
  summary_plain       TEXT NOT NULL,        -- legally safe summary
  canonical_text_ref  TEXT,                 -- URL/citation, not full ISO text
  tags                JSONB NOT NULL DEFAULT '[]',
  jurisdictions       JSONB NOT NULL DEFAULT '["EU","UK"]',
  cross_maps          JSONB NOT NULL DEFAULT '[]',  -- [{framework_key, control_key}]
  evidence_expectations JSONB NOT NULL DEFAULT '[]',-- ['tamper_evident_logs','runtime_block_counts']
  suggested_intents   JSONB NOT NULL DEFAULT '[]',  -- ['block_outbound_pii','log_all_decisions']
  version             TEXT NOT NULL,        -- semver for this control
  status              TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','superseded')),
  superseded_by       UUID REFERENCES controls(id),
  UNIQUE (framework_id, control_key)
);

CREATE INDEX idx_controls_framework ON controls(framework_id);
CREATE INDEX idx_controls_tags_gin ON controls USING GIN(tags);
CREATE INDEX idx_controls_cross_maps_gin ON controls USING GIN(cross_maps);

CREATE TABLE control_localizations (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  control_id        UUID NOT NULL REFERENCES controls(id) ON DELETE CASCADE,
  locale            TEXT NOT NULL,            -- 'en-GB', 'de-DE'
  title_localized   TEXT NOT NULL,
  summary_localized TEXT NOT NULL,
  UNIQUE (control_id, locale)
);

-- ==================================
--  Tenant Requirements (Compliance)
-- ==================================
CREATE TABLE policy_requirements (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  control_id      UUID NOT NULL REFERENCES controls(id),
  intent          TEXT NOT NULL CHECK (intent IN ('block','alert','redact','log_only')),
  status          TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','approved','published','archived')),
  internal_name   TEXT NOT NULL,     -- human-friendly label used in reports
  owner_user_id   UUID NOT NULL REFERENCES users(id),
  notes           TEXT,
  version         TEXT NOT NULL DEFAULT '1.0.0',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_policy_requirements_tenant ON policy_requirements(tenant_id);
CREATE INDEX idx_policy_requirements_status ON policy_requirements(status);

-- ==================================
--  Dev/Sec: Endpoints & Detectors
-- ==================================
CREATE TABLE endpoints (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  endpoint_id     TEXT NOT NULL,        -- e.g., 'chatbot-prod', 'rag-api'
  tags            JSONB NOT NULL DEFAULT '[]',  -- ['handles_phi','us_region']
  owner_team      TEXT,
  UNIQUE (tenant_id, endpoint_id)
);

CREATE TABLE detectors_catalog (
  -- catalog of detector identifiers the tenant intends to use (BYO/open-source/vendor)
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  detector_id     TEXT NOT NULL,           -- 'promptguard_v2', 'injecguard_v1'
  vendor          TEXT NOT NULL,           -- 'open_source','vendor','internal'
  description     TEXT,
  UNIQUE (tenant_id, detector_id)
);

CREATE TABLE detector_bindings (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  requirement_id      UUID NOT NULL REFERENCES policy_requirements(id) ON DELETE CASCADE,
  detector_catalog_id UUID NOT NULL REFERENCES detectors_catalog(id) ON DELETE CASCADE,
  config_json         JSONB NOT NULL DEFAULT '{}', -- thresholds, modes
  status              TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','pending','deprecated')),
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Which endpoints are in scope for a given requirement (devs bind "where")
CREATE TABLE requirement_endpoints (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  requirement_id  UUID NOT NULL REFERENCES policy_requirements(id) ON DELETE CASCADE,
  endpoint_ref_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
  mode            TEXT NOT NULL CHECK (mode IN ('block','alert','redact','log_only')),
  UNIQUE (tenant_id, requirement_id, endpoint_ref_id)
);

-- =========================
--  Evidence & Audit Ledger
-- =========================
-- Tamper-evident chain: each row hashes prior row.
CREATE TABLE evidence_events (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id            UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  requirement_id       UUID REFERENCES policy_requirements(id),
  endpoint_id          UUID REFERENCES endpoints(id),
  correlation_id       TEXT,            -- from app/proxy to tie req/resp
  direction            TEXT NOT NULL CHECK (direction IN ('request','response')),
  final_action         TEXT NOT NULL CHECK (final_action IN ('allow','block','alert','redact')),
  detector_results     JSONB NOT NULL DEFAULT '[]',  -- array of {detector_id, version, decision, reason, confidence}
  policy_trace         JSONB NOT NULL DEFAULT '[]',  -- framework/control refs used
  snippet_hash         TEXT,            -- SHA-256 of normalized snippet (no raw PII in DB by default)
  created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  prev_hash            TEXT,            -- hash of previous row (lexicographic by created_at,id)
  row_hash             TEXT             -- hash over (id,tenant_id,created_at,prev_hash,final_action,detector_results,...)
);
CREATE INDEX idx_evidence_events_tenant_time ON evidence_events(tenant_id, created_at);
CREATE INDEX idx_evidence_events_requirement ON evidence_events(requirement_id);

-- Chain integrity view (optional)
-- Compute row_hash in application code when inserting.

-- =========================
--  Reports & Exports
-- =========================
CREATE TABLE reports (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  title           TEXT NOT NULL,
  format          TEXT NOT NULL CHECK (format IN ('pdf','csv','json')),
  params          JSONB NOT NULL DEFAULT '{}', -- timeframe, frameworks included, locales
  storage_uri     TEXT,                        -- where the generated file lives
  created_by      UUID REFERENCES users(id),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- =========================
--  RBAC helpers (materialized views are optional)
-- =========================
-- Enforce in app layer: only 'compliance' can write policy_requirements,
-- only 'devsec' can write detector_bindings/requirement_endpoints, 'auditor_view' is read-only.
```

Hash-chain insert (evidence) — application logic
- On insert into `evidence_events`:
  - `prev_hash` = `row_hash` of latest prior event for the tenant (by `created_at`, `id`).
  - `row_hash` = SHA‑256 over a canonical serialization of key fields.
- Auditors can recompute and verify the chain; daily Merkle roots can be added as an optimization.

Seed data (EU/UK library — examples)
```sql
-- Frameworks
INSERT INTO frameworks (key, name, publisher, version, license_type, locale_default, meta)
VALUES
('gdpr','General Data Protection Regulation','EU','2024.12','public_domain','en-GB','{"site":"https://gdpr.eu"}'),
('eu_ai_act','EU AI Act','EU','2025.01','public_domain','en-GB','{"status":"phased rollout"}'),
('owasp_llm_top10','OWASP LLM Top 10','OWASP','2024.1','public_domain','en-GB','{}'),
('iso_27001','ISO/IEC 27001','ISO','2022','summary_only','en-GB','{}');

-- Controls (GDPR examples)
INSERT INTO controls (framework_id, control_key, title, summary_plain, canonical_text_ref, tags, evidence_expectations, suggested_intents, version)
SELECT id, 'GDPR-ART-32-1', 'Security of Processing',
       'Ensure appropriate technical and organisational measures to protect personal data (confidentiality, integrity, availability).',
       'https://gdpr.eu/article-32-security-of-processing/',
       '["security","logging","access_control"]'::jsonb,
       '["tamper_evident_logs","runtime_block_counts","retention_policy"]'::jsonb,
       '["block_outbound_pii","log_all_decisions"]'::jsonb,
       '1.0.0'
FROM frameworks WHERE key='gdpr';

INSERT INTO controls (framework_id, control_key, title, summary_plain, canonical_text_ref, tags, evidence_expectations, suggested_intents, version)
SELECT id, 'GDPR-ART-25', 'Data Protection by Design and by Default',
       'Integrate data protection into processing activities and systems by default.',
       'https://gdpr.eu/article-25-data-protection-by-design/',
       '["privacy","minimization"]'::jsonb,
       '["configuration_snapshots","change_logs"]'::jsonb,
       '["log_all_decisions"]'::jsonb,
       '1.0.0'
FROM frameworks WHERE key='gdpr';

-- AI Act (high-level)
INSERT INTO controls (framework_id, control_key, title, summary_plain, canonical_text_ref, tags, evidence_expectations, suggested_intents, version)
SELECT id, 'AI-ACT-ART-9', 'Risk Management System',
       'Establish, implement, document and maintain a risk management system for high-risk AI systems.',
       'https://eur-lex.europa.eu/eli/reg/ai_act/2024',
       '["ai_risk","monitoring"]'::jsonb,
       '["runtime_event_logs","risk_register_link"]'::jsonb,
       '["log_all_decisions"]'::jsonb,
       '1.0.0'
FROM frameworks WHERE key='eu_ai_act';
```

API & IaC contracts (dev-friendly)
```text
Pull requirements (Dev/Sec)
GET /api/requirements?status=published
-> 200 [{ id, control: {framework:"GDPR", key:"GDPR-ART-32-1"}, intent:"block", internal_name, version }]

Register endpoint
POST /api/endpoints
{ "endpoint_id": "chatbot-prod", "tags": ["handles_pii","eu_region"], "owner_team": "ml-platform" }

Declare detector catalog entry
POST /api/detectors
{ "detector_id": "promptguard_v2", "vendor": "open_source", "description": "Prompt injection & PII guardrail" }

Bind requirement ↔ detectors ↔ endpoints (IaC YAML example)
# promptshield.enforcement.yaml (checked into Git; applied via CLI)
requirements:
  - id: "req_1234"             # from /api/requirements
    detectors:
      - id: "promptguard_v2"
        config:
          threshold: 0.80
          action: "block"
    endpoints:
      - "chatbot-prod"
      - "rag-api"

CLI:

psctl apply -f promptshield.enforcement.yaml
```

Compliance UI — screen-by-screen
- Library Browser
  - Filters: Framework (GDPR, AI Act, ISO 27001, OWASP LLM Top‑10), Jurisdiction (EU/UK), Topic tags.
  - List: Control ID, Title, Summary snippet, Version, Status (active/superseded).
  - Action: View Control
- Control Detail
  - Header: Framework / Control ID / Title / Version.
  - Tabs: Overview (summary, canonical link, tags, jurisdictions); Suggested Intents; Evidence expectations; Cross‑maps; Changelog.
  - CTA: Create Requirement
- Create Requirement (Wizard)
  - Step 1: Confirm control (e.g., GDPR Art. 32(1))
  - Step 2: Choose Intent (Block / Alert / Redact / Log only) with tooltips
  - Step 3: Internal name + notes
  - Step 4: Status: Draft → Approve → Publish
  - Success: shows Requirement ID (copy)
- Requirements List (Home)
  - Columns: Requirement, Framework/Control, Intent, Status, Mapping Status (Unmapped / Mapped / Enforced), Owner, Updated.
  - Row actions: View; Export summary (CSV/JSON); Archive (if Draft/Approved)
- Requirement Detail (read‑only Dev mapping view)
  - Compliance tab: requirement metadata, intent, notes.
  - Implementation tab: Detectors bound (id, version/config, heartbeat); Endpoints bound (list, mode); Evidence health (events/day, violations blocked, 30‑day coverage).
  - Reports tab: quick export (PDF/CSV/JSON) scoped to this requirement.
- Reports
  - Templates: Coverage Matrix, Event Digest, Change Log.
  - Filters: timeframe, frameworks, locales, endpoints.
  - Export: PDF (signed), CSV/JSON (machine‑readable). Link: generate expirable auditor link.

Minimal Dev dashboard (read‑only)
- Coverage: Requirements (published) + mapping state (unmapped/mapped/enforced).
- Violations feed: recent evidence events with chips (endpoint, detector, action).
- Health: detector heartbeat status; last event time per endpoint.

Ops notes
- RBAC: enforce in API — only `compliance` mutates policy_requirements; only `devsec` mutates detector_bindings and requirement_endpoints; `auditor_view` is read‑only.
- PII safety: store only hashes/snippets in evidence_events by default; full payloads optional & encrypted per tenant with rotation.
- Localization: use control_localizations and a UI locale toggle; exports respect selected locale.
- Library updates: signed JSON bundles with diff preview; mark superseded controls, never hard‑delete.
