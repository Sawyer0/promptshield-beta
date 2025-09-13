**psctl CLI (stub)**

Purpose
- Load compliance seed bundles (frameworks/controls)
- Push OPA policy bundles (arbitration/binding/validation)

Commands
- Load seed bundle
  - psctl library load -f seeds/compliance/eu-uk-controls.v1.json
  - Options:
    - -f, --file: path to JSON bundle matching `docs/schemas/seed_bundle.schema.json`
    - --validate-only: schema-validate without loading

- Push OPA policy bundle
  - psctl policy push --bundle policy/opa
  - Options:
    - --bundle: directory containing .rego and optional data/*.json
    - --dry-run: validate and print summary without upload
    - --tag: bundle tag/version to record in policy_version

Config
- Env vars
  - PS_API_URL: base URL for PromptShield API
  - PS_API_TOKEN: admin or scoped token for library/policy actions

Examples
- Validate and load EU/UK seeds
  - psctl library load -f seeds/compliance/eu-uk-controls.v1.json

- Validate only
  - psctl library load -f seeds/compliance/eu-uk-controls.v1.json --validate-only

- Push OPA policies (defaults)
  - psctl policy push --bundle policy/opa --tag v1

Notes
- Seed bundle schema: see `docs/schemas/seed_bundle.schema.json`
- OPA policy pack: see `policy/opa/README.md`
