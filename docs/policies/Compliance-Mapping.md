**Compliance Mapping in RulePacks**

Add compliance metadata at the RulePack and rule level to map policies to frameworks.

Top‑level mapping (applies to the pack):

- Key: `metadata.compliance`
- Type: object of framework name → array of control IDs
- Example:

  metadata:
    compliance:
      SOC2: ["CC6.6", "CC7.2"]
      HIPAA: ["164.312(b)"]

Rule‑level mapping (fine‑grained controls per rule):

- Key: `controls`
- Type: object of framework name → array of control IDs
- Example rule:

  - id: pii-leakage-detection
    level: 2
    patterns:
      - regex: "..."
    controls:
      SOC2: ["CC6.6"]
      GDPR: ["Art. 32"]

Validation
- The YAML schema enforces the new keys. See `internal/rules/schema.json`.
- These fields are optional and backward‑compatible.

Usage
- Evidence and compliance services can attribute enforcement to specific controls.
- UI can filter/group rules by framework/control for auditor workflows.
