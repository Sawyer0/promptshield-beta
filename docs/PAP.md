# PAP (Policy Administration Point)

This document describes the Policy Administration Point (PAP) responsibilities, typical cost model, and strategies to keep costs low. PAP is the administrative plane for policy authoring, versioning, activation, and auditing; it complements the data-plane PDP (Policy Decision Point).

## What PAP does

- Authoring
  - Tenants define their RulePacks (keywords, regexes, semantic categories, tool allowlists)
- Versioning
  - Policies are stored, signed, and bundled for distribution to PDPs
- Activation
  - Tenants can activate or rollback bundles safely
- Audit trail
  - Who changed what, and when (for compliance and forensics)

## PAP cost drivers

- Storage
  - RulePacks are YAML and small (KB–MB per tenant). Even at enterprise scale (e.g., 1,000 tenants × 100 versions), total storage is in the hundreds of MBs
  - Use S3/GCS with versioning → pennies per month
- Database
  - Metadata only (tenant, policy_id, version, active/inactive, author)
  - Postgres with row-level security (RLS) fits well with existing multi-tenancy
  - DB size stays small; cost is uptime/HA, not capacity
- UI / API hosting
  - A web console or admin API to upload/activate bundles
  - One small service/pod per environment in most deployments
- Signing & distribution
  - Sign bundles and distribute securely to PDPs
  - Costs are minimal (cryptographic ops + CDN/S3 bandwidth); bundles are small
- Audit log storage
  - Every policy change should be durable and retained per compliance
  - Costs grow with tenants × changes × retention; CloudWatch/ELK/Splunk can add up without compression/expiration

## Why PAP isn’t a big runtime cost

- PDP evaluates thousands of decisions per second
- PAP changes dozens of times per day per tenant (admin-plane vs data-plane)
- Cost is dominated by storage and audit logging, not CPU

## Cost control strategies

- Keep RulePacks small and modular (avoid excessively large regex sets)
- Tiered retention for audit logs (e.g., hot 90 days, cold 1 year, archive beyond)
- Precompute policy bundles in CI/CD so PDP does not recompile per deployment

## Order-of-magnitude costs (illustrative)

- Storage + signing + distribution: ~$5–$20/month at cloud scale
- Audit logs (by backend): ~$100–$500/month if logs are verbose and tenants are active
- UI/API hosting: roughly one small container per environment

## HTTP endpoints (admin, PDP-gated)

- GET /v1/rulepacks/{id}/versions/{ver}/bundle
  - Export a freshly signed bundle (not persisted)
  - PDP action: rulepack.bundle.export
- POST /v1/rulepacks/{id}/versions/{ver}/publish
  - Sign and persist bundle to store (filesystem by default)
  - PDP action: rulepack.bundle.publish
  - Response: { status, path, checksum, key_id }
- GET /v1/rulepacks/{id}/bundles
  - List stored bundles for a rulepack and tenant
  - PDP action: rulepack.bundle.list
- GET /v1/rulepacks/{id}/bundles/{ver}
  - Retrieve a stored bundle; server verifies signature before returning
  - PDP action: rulepack.bundle.get
- POST /v1/rulepacks/{id}/bundles/{ver}/activate
  - Load stored bundle, verify, then activate (create-and-activate or fallback to activate existing version)
  - PDP action: rulepack.bundle.activate
- POST /v1/rulepacks/{id}/bundles/verify
  - Verify posted bundle JSON without persisting/activating
  - PDP action: rulepack.bundle.verify

## Configuration

- PS_RULEPACK_HMAC_KEY (base64): HMAC signing key required for bundle signing
- PS_RULEPACK_HMAC_KEY_ID (optional): logical identifier for the key used (for rotation and audit)
- PS_BUNDLE_DIR (optional): filesystem root for bundle store (default: ./bundles)

## Implementation notes for PromptShield

- RulePack authoring already uses YAML; extend RulepackService to:
  - Validate and sign bundles during version creation
  - Store signatures and provenance metadata in the DB
- Add endpoints and UI flows to:
  - Upload a RulePack version and mark it as candidate
  - Activate/rollback bundles per-tenant (with ETag preconditions)
  - Export signed bundles for PDPs
- Enforce auditing for all mutating PAP actions:
  - Write who/what/when and PDP decision metadata into audit logs
- Distribution
  - Host bundles in a private bucket with signed URLs or use a small CDN; PDP pulls with integrity verification
- Security
  - Verify signatures in PDP before accepting/activating a bundle
  - Use RLS in Postgres for tenant isolation; enforce PAP actions via PDP policies

