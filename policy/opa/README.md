Policy packs for PromptShield (OPA/Rego)

- ps.detectors.arb.rego: Arbitration of multiple detector events into a single action
- ps.compliance.bind.rego: Requirement binding validation
- ps.detectors.validate.rego: Ingestion guard for detector_event/v1

Run tests with opa:
- opa test -v policy/opa

Bundle layout (example):
- data/
  - thresholds.json
  - bindings.json
  - detectors.json
  - precedence.json
