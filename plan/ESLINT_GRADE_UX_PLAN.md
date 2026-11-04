### ESLint‑grade UX Upgrade Plan for PromptShield

This plan evolves PromptShield’s CLI and developer experience to match or exceed ESLint’s UX while preserving stability. It defines phased deliverables, concrete file‑level edits, and acceptance criteria.

Goals
- Match ESLint’s ergonomics: config cascade, ignore behavior, globs, deterministic output, formatter ecosystem, exit codes suitable for CI, and caching.
- Keep the current core scanning logic intact; add UX around it.
- Maintain modularity: ports/adapters, DI container, and `Result<T,E>` across handlers.

Summary of the most impactful improvements
- Discovery and selection: `--rules-dir` (default `rulepacks/`) + deterministic rulepack selection.
- Console hygiene: `--plain` for CI‑clean output; deterministic non‑emoji, non‑color output by default when printing to console.
- CI semantics: distinct exit codes for findings vs operational errors.
- Input ergonomics: globs, `.promptshieldignore`, `--ext`, `--stdin-filename`.
- Formatter parity: default “stylish” formatter, with `--format-options`.
- Config cascade with `--print-config`.
- Caching and parallelism defaults.

Phased roadmap

Phase 1: Immediate UX + CI semantics (low risk)
- Add `--rules-dir` (default: `rulepacks/`) and selection rules:
  - If `--rulepack` provided: use it.
  - Else if `--rules-dir` has exactly one pack: use it.
  - Else if multiple: error listing candidates and instruct to pass `--rulepack`.
- Add `--plain` to suppress special characters (emojis, progress bars) and normalize to ASCII; implies `--no-color` when to console. File outputs unaffected.
- Exit codes:
  - 0: success (no violations at/above threshold)
  - 1: runtime/config/CLI errors
  - 2: violations met/exceeded `--fail-on` (and/or `--max-warnings` in future)

File‑level edits (Phase 1)
- `src/cli/index.ts`
  - Add options: `--rules-dir <dir>` (defaults to `rulepacks/`), `--plain`.
  - In scan action: accept either `--rulepack` or discovery via `--rules-dir` using a service.
  - Map fail‑on threshold to exit code 2 (operational errors remain 1).
- `src/cli/bootstrap.ts`
  - Register `RulePackDiscovery` service in the container.
  - Wire `--plain` into `Report` creation (e.g., add `plain` to `ReportOptions`).
- New: `src/domains/rules/adapters/RulePackDiscovery.ts`
  - Port/service that lists YAML rulepacks in a directory, resolves to one or errors with helpful context.
- `src/application/commands/scan/ScanCommand.ts`
  - Extend `ScanCommandOptions` with `rulesDir?: string; plain?: boolean`.
  - Ensure `toScanConfig()` carries `plain` for renderers to consume (either as `noColor` + “no special chars” or an explicit flag).
- `src/application/commands/scan/ScanCommandHandler.ts`
  - Adjust validation: allow discovery path (either `rulepack` or `rulesDir`).
  - When returning error for fail‑on, include a typed marker to enable exit code 2 in CLI.
- `src/domains/reporting/core/entities/Report.ts`
  - Add `plain?: boolean` in `ReportOptions` to signal renderers to suppress non‑ASCII embellishments.
- `src/domains/reporting/adapters/renderers/MarkdownRenderer.ts`
  - Respect `report.options.plain` to force non‑emoji, non‑chalk output (use the “no‑color markdown” code path and avoid glyphs).
- `tests/e2e/cli/cli-commands.test.ts`
  - Add tests for `--rules-dir` single/multiple selection behavior.
  - Add tests for `--plain` ensuring no emojis/special characters in console output.
  - Add exit code 2 test for `--fail-on` with findings.
- `docs/CLI_REFERENCE.md`
  - Document `--rules-dir` (default), discovery rules, `--plain`, and exit codes 0/1/2.
  - Clarify that file outputs are unaffected by `--plain`.

Acceptance criteria (Phase 1)
- Running `promptshield scan <path> --rules-dir rulepacks/` uses the single rulepack found; if multiple, shows a clear error with candidates.
- `--plain` yields ASCII‑only console output; file outputs unchanged.
- Fail‑on with violations exits 2; malformed CLI/config errors exit 1.
- All updated tests pass; docs reflect behavior precisely.

Phase 2: Input targeting & ignore (medium risk)
- Globbing: Accept multiple inputs and globs (e.g., `src/**/*.{json,ndjson,txt}`). Expand to file list before scanning.
- Ignore: Support `.promptshieldignore` (gitignore syntax) and `--ignore-path`.
- `--no-ignore` to disable ignore rules.
- `--ext` to include additional extensions.
- `--stdin-filename` to infer processor and apply ignore/config to stdin content.

File‑level edits (Phase 2)
- `src/cli/index.ts`: allow multiple `<inputs...>` and parse globs; add `--ignore-path`, `--no-ignore`, `--ext`, `--stdin-filename`.
- New: `src/domains/scanning/adapters/IgnoreMatcher.ts` implementing gitignore semantics.
- `src/domains/scanning/adapters/LocalFileReader.ts`: utility to expand glob patterns and apply ignore.
- `src/application/commands/scan/ScanCommand.ts`: carry new flags in options.
- Tests: new e2e/integration tests for globs, ignore path, and stdin filename inference.
- Docs: add sections mirroring ESLint’s input targeting and ignore behavior.

Acceptance criteria (Phase 2)
- Globs expand to expected file sets; ignore rules exclude matches; `--no-ignore` disables.
- Stdin can be processed with an inferred filename.

Phase 3: Formatter parity (medium risk)
- Add `stylish` formatter as default for console; keep `json`, `markdown`, `table`, `html`, `ndjson`.
- `--format-options <json>` for formatter‑specific settings.
- Deterministic ordering of messages.

File‑level edits (Phase 3)
- New: `src/domains/reporting/adapters/renderers/StylishRenderer.ts`.
- `src/cli/bootstrap.ts`: register new renderer as default.
- `src/domains/reporting/core/ports/Renderer.ts`: optionally extend to accept `options`.
- Tests: formatter output snapshots for `stylish`; ordering invariants.
- Docs: update `--format` list and describe `stylish` as default.

Acceptance criteria (Phase 3)
- `promptshield scan <input>` defaults to stylish output; `--format` switches renderers; `--format-options` honored when supported.

Phase 4: Config cascade (higher scope, gated)
- Support: `promptshield.config.{js,cjs,json,yaml}`, `.promptshieldrc.{json,yaml}` and `package.json:{ "promptshield": {...} }`.
- Precedence: CLI > config > defaults.
- `--print-config <path>` to show resolved config.

File‑level edits (Phase 4)
- New: `src/infrastructure/config/ConfigLoader.ts` with schema, merging logic, and validation (`zod`).
- `src/cli/index.ts`: add `--config` and `--print-config`.
- Tests: resolution precedence; schema validation; `--print-config` output.
- Docs: configuration guide mirroring ESLint behavior.

Acceptance criteria (Phase 4)
- Resolved config matches precedence; `--print-config` prints deterministic JSON.

Phase 5: Caching and performance (optional, later)
- `--cache`, `--cache-location` with a cache key (file hash + rulepack version + config hash).
- Parallel by default (num cores); `--no-parallel` to opt‑out.

File‑level edits (Phase 5)
- New: `src/infrastructure/cache/CacheService.ts`.
- `src/cli/index.ts`: add `--cache`, `--cache-location`, default to parallel.
- Tests: cache hits/misses; performance tests; correctness under parallel.
- Docs: cache semantics and invalidation.

Acceptance criteria (Phase 5)
- Re‑runs skip unchanged files with `--cache`; invalidated correctly on rulepack/config changes.

Phase 6: Init enhancements (optional)
- Add interactive `--init` to scaffold config (like ESLint’s `--init`).
- Keep `init <filename>` for rulepack authoring; consider `init rulepack <filename>` and `init config` split.

Risks and mitigations
- Behavior drift: Gate new semantics behind additive flags; keep defaults stable where possible (stylish default is a notable change; announce in CHANGELOG).
- Test churn: Update E2E and integration tests alongside implementation to keep CI green.
- Backward compat: Preserve `--output` formats; ensure `.md`/`.html` file outputs remain unaffected by `--plain`.

Versioning & communications
- Release as minor version (e.g., 1.1.0) for Phase 1–3.
- Add CHANGELOG with migration notes (exit code 2 addition; new flags; new default formatter if changed).
- Update `README.md`, `docs/CLI_REFERENCE.md`, and add a quick “From 1.0.x to 1.1.x” section.

Appendix: Deterministic selection rules for `--rules-dir`
1) Gather `*.yaml` files under the directory (non‑recursive initially).
2) If none found → error with guidance to pass `--rulepack`.
3) If one found → use it.
4) If multiple → error with a numbered list of candidates and an example command to pass `--rulepack`.


