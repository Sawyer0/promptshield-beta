# Product Improvement Plan (PIP): Compliance Mapping & Evidence Export

Last updated: 2025-09-11
Owner: Product/Engineering
Status: Proposed

## Summary
We currently claim that “Compliance evidence [is] mapped to OWASP Top 10 for LLM, SOC 2, HIPAA, GDPR, NIST AI RMF,” but there is no machine-readable mapping or exportable evidence report implemented. This PIP scopes and delivers a truthful, auditable implementation for control mapping and evidence exports.

## Problem Statement
- Marketing/UI copy asserts control mappings exist, but there is no authoritative mapping registry nor tagging in RulePacks or audit events to support exports.
- Without mappings, auditors and admins cannot trace evidence (configs, diffs, audit events) to specific controls across frameworks.

## Goals
- Provide a single source of truth for compliance mappings across:
  - OWASP Top 10 for LLM (LLM01–LLM10)
  - SOC 2 (AICPA TSC, selected CC controls)
  - HIPAA (Security/Privacy Rule citations, e.g., 45 CFR 164.*)
  - GDPR (Articles/Recitals, e.g., 5, 25, 30, 32, 33)
  - NIST AI RMF (Govern/Map/Measure/Manage subcategories)
- Tag product artifacts (RulePacks, rules, policies, events) with control IDs.
- Enable exportable compliance evidence reports by framework/control and time range.

## Non-Goals
- Achieve or represent formal certification (e.g., SOC 2 report) within the product.
- Full legal/compliance coverage beyond the scoped controls in this PIP.

## Deliverables
1) Compliance Mapping Registry (machine-readable)
- File: compliance/mappings/mapping.json (versioned)
- Defines frameworks, control IDs, names, descriptions, and evidence sources:
  - Referenced rule IDs and/or rule tags
  - Audit action filters and metadata filters
  - Config snapshot sources (active rulepacks, assignments, enforcement modes)

2) Rule/Policy Control Tagging
- DSL extension to support per-rule control tags: controls: ["OWASP_LLM.LLM01", "SOC2.CC7.2", ...]
- UI support in RulePack creation/edit forms to add/remove control tags.

3) Evidence Export API (BFF)
- Endpoint: GET /api/compliance/export?framework=OWASP_LLM&controls=LLM01,LLM02&from=ISO8601&to=ISO8601&format=json|csv
- Assembles:
  - Control definitions
  - Matching rules/rulepacks (including versions and diffs where available)
  - Relevant audit events (bounded by time range and filters)
  - Config snapshots and settings relevant to the control

4) Compliance UI Page
- “Compliance” page listing:
  - Framework coverage (% controls referenced by at least one artifact)
  - Per-control status: Mapped, Partially Mapped, Not Mapped, With Evidence (in range)
  - “Export Report” button (JSON/CSV) per framework or per-control selection

5) Documentation & Disclaimers
- Clarify that mappings provide evidence support and traceability, not certifications.
- Document mapping methodology and how evidence is collected/retained.

## Architecture Overview (MVP)
- Registry: JSON file stored in repo (can be migrated to DB later).
- Tagging: RulePack DSL extended with controls array; tags persisted in RulePack metadata and surfaced in UI.
- Evidence Sources:
  - Audit API (existing): actions ["request.decision", "scan.decision"], metadata.reason/category filters
  - Config API: rulepacks/active, assignments, enforcement mode, preferences (egress allowlist, approvals)
- Exporter: BFF composes report by reading registry and querying sources within [from, to].

## Implementation Plan
Phase 1: Registry + UI skeleton (1 week)
- Create compliance/mappings/mapping.json with schema and initial entries
- Add “Compliance” page (framework list, per-control grid, mock export)
- Add docs/disclaimers

Phase 2: Tagging + Export (1–1.5 weeks)
- Extend RulePack DSL and frontend to add controls: string[] per rule
- Update RulePackModal to include controls in the mapped DSL payload
- Implement GET /api/compliance/export in BFF
- Wire Compliance page to live export and evidence counts

Phase 3: Coverage & Quality (0.5–1 week)
- Populate mappings for:
  - OWASP LLM: LLM01–LLM10 baseline
  - SOC 2: CC7.2, CC7.3, CC6.1, CC6.6, CC8.1 (initial set)
  - HIPAA: 164.312(b), 164.308(a)(1)(ii)(D), 164.308(a)(3)
  - GDPR: Art 5, 25, 32, 33
  - NIST AI RMF: MAN.3, GOV.2, MAP.1, MEA.2 (initial set)
- Add unit tests
- Validate exports with sample tenants

## Acceptance Criteria
Global
- A single mapping file (mapping.json) exists, validated against internal schema.
- Each mapping entry references at least one evidence source (rule IDs/tags, audit filters, configs).
- Compliance page renders coverage by framework and lists controls with statuses.
- Export endpoint returns JSON with:
  - Framework, controls
  - Control definitions and mapped artifacts
  - Evidence bundles (audit events sampled or summarized; config snapshots; rulepack metadata)
- CSV export available with a minimum useful subset (framework, control, artifact, timestamp, summary).

OWASP Top 10 for LLM
- LLM01–LLM10 mapped to at least one rule/tag or config.
- Export returns at least one evidence artifact per mapped control over a test period.

SOC 2 (selected CC controls)
- CC7.2 (monitor/detect anomalies), CC7.3 (respond), CC6.1 (logical access baseline) mapped.
- Export demonstrates audit trails and relevant configurations (enforcement mode, approvals, logs).
- Disclaimer present regarding certification.

HIPAA (selected)
- 164.312(b) (Audit controls) mapped to audit events and retention settings.
- 164.308(a)(1)(ii)(D) (Information system activity review) mapped to dashboards/exports.
- Disclaimer present regarding covered entity responsibilities and scope.

GDPR (selected)
- Art 32 (Security of processing) mapped to PII/secrets redaction rules + enforcement settings.
- Art 25 (Privacy by design) mapped to context minimization/tool constraints where applicable.
- Disclaimer present regarding controller/processor obligations.

NIST AI RMF (initial set)
- MAN.3, GOV.2, MAP.1, MEA.2 mapped to controls and evidence.
- Export demonstrates traceability to risk management activities (rulepacks, audits).

## Risks & Mitigations
- Risk: Over-claiming coverage beyond implemented controls.
  - Mitigation: Limit mappings to controls with tangible evidence; add disclaimers.
- Risk: Performance regressions for large audit exports.
  - Mitigation: Paginate/limit exports; provide summaries and links for full exports.
- Risk: Schema drift between UI and DSL for controls tags.
  - Mitigation: Type-safe interfaces; validation tests.

## Dependencies
- Existing audit/search endpoints reliability
- RulePack DSL versioning and validation pipeline

## Interim Copy Change (until GA)
- Update SolutionSection claim to: “Compliance mapping: in progress (OWASP LLM, SOC 2, HIPAA, GDPR, NIST AI RMF).”
- Keep Trust badges as “in progress” where applicable.

## Mapping Schema (proposed)
- File: compliance/mappings/mapping.json
- Example entry:

```json
{
  "version": "0.1.0",
  "frameworks": {
    "OWASP_LLM": {
      "LLM01": {
        "name": "Prompt Injection",
        "description": "Detection and prevention of prompt injection and tool abuse.",
        "evidence": {
          "rules": ["rule-001", "rule-017"],
          "rule_tags": ["prompt-injection", "tool-constraint"],
          "audits": {
            "actions": ["scan.decision", "request.decision"],
            "filters": { "reason": ["prompt_injection", "tool_abuse"] }
          },
          "configs": ["rulepacks/active", "assignments", "enforcement_mode"]
        }
      }
    }
  }
}
```

## Test Plan
- Unit tests: mapping schema validation; export assembler with mocked audit data.
- UI tests: Compliance page renders coverage; export triggers download.
- Manual: Create sample rulepacks with controls tags; generate 24h audit; verify evidence per control.

## Timeline
- Phase 1: 1 week
- Phase 2: 1–1.5 weeks
- Phase 3: 0.5–1 week
Total: ~2.5–3.5 weeks to GA (subject to scope).

## Design Decisions (Resolved)
- Registry location & precedence
  - Baseline mapping is versioned in-repo at compliance/mappings/mapping.json for auditability and code review.
  - Support overlays later (DB-backed). Precedence: Tenant override (DB) > Global override (DB) > Repo default (file).
- Per-tenant overrides
  - Not required; supported as additive overlays. A control is considered covered if either default or overlay references at least one evidence artifact. No subtractive removals in v1.
- CSV export format
  - One row per control–artifact tuple with bounded time context. Columns:
    - framework, control_id, control_name, tenant_id, coverage_status (mapped|partial|unmapped|evidence_available),
      artifact_type (rule|rulepack|config|audit_summary|audit_event), artifact_id, artifact_name, artifact_version,
      evidence_period_start, evidence_period_end, evidence_count, sample_reference, generated_at, notes.

