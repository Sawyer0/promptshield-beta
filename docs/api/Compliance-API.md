**Compliance API**

Compliance Report
- GET `/api/compliance/{tenantId}/report?standard=SOC2&start=RFC3339&end=RFC3339`
- Auth: admin (BFF/JWT)
- Response: summary of audit events, incidents, policy violations, integrity status, access controls, data retention, and security measures.

Example:

- GET `/api/compliance/9b5c.../report?standard=SOC2&start=2025-01-01T00:00:00Z&end=2025-03-31T23:59:59Z`
  Returns `{ "tenant_id":"...","compliance_standard":"SOC2", ... }`

Audit Search & Export (Admin)
- List: GET `/v1/admin/audits?tenant={uuid}&limit=100`
- Search: POST `/v1/admin/audits/search` with body:
  - `{ "tenant_id":"...", "object_types":["policy"], "start_time":"RFC3339", "end_time":"RFC3339", "limit":500 }`
- Export: GET `/v1/admin/audits/export?tenant={uuid}&format=json|csv`
- By Object: GET `/v1/admin/audits/object/{type}/{objectId}?limit=200`

Notes
- Audit trails are tamper‑evident (hash‑chained with daily Merkle roots).
- Evidence exports include integrity hashes suitable for auditor hand‑off.

See also
- docs/COMPLIANCE_ORCHESTRATION.md
- docs/policies/Compliance-Mapping.md
