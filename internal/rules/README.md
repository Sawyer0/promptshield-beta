Rules package

Defines the RulePack schema and pure domain logic:
- types.go: schema types (Rule, Pattern, Semantic, Condition, RulePack)
- loader.go: load and resolve imports (higher layers handle source discovery)
- merge.go: merge multiple packs; supports priority order
- validate.go: strict schema validation with helpful messages

Guidelines:
- Pure package: no printing, no env reads, no network access
- Return typed errors; callers format messages for CLI/API

User RulePacks
---------------

End users should keep their own packs under project `rules/`. The repository ships a few small example packs in `rules/` for quick starts (unmaintained), and heavier industry samples in `docs/samples/`.

Quickstart:

```bash
promptshield rules:create --template l1|l2|l3|pii|prompt-injection|industry --dest rules/my-pack.yaml
promptshield rules:validate --path rules/my-pack.yaml
promptshield scan --rulepack rules/my-pack.yaml path/
```

