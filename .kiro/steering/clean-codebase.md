---
inclusion: always
---
File and folder layout:

Keep cmd/ thin: cmd/<app>/main.go does only setup and delegates; each command in its own file (root.go, scan.go, validate.go, version.go). No business logic here.
Put business logic in internal/: e.g., internal/scanner/, internal/rules/, internal/discovery/, internal/report/. The CLI should call these packages, not implement logic.
Public types/APIs in pkg/: Only export what external programs might import, e.g., pkg/types.
Rules/config/examples: Keep user assets under rules/, examples/, testdata/.
Tools and CI: Makefile, tools/, .github/workflows/ for CI, release, lint.

Cobra command layer (delivery):

One command = one file: Keeps help, flags, and completion close to the command.
Use RunE and return errors: Let main handle printing and exit codes; avoid os.Exit outside main.
Persistent vs local flags: Global flags (e.g., --output-format, --config) on rootCmd; command-specific flags on the command.
Shell completion: Provide ValidArgsFunction and RegisterFlagCompletionFunc; expose completion subcommand.
Help text: Clear Use, Short, Long, and examples.

Separation of concerns:

Commands orchestrate, don’t compute: Parse inputs, build options, call into internal/* services, render results.
Pure packages: internal/* must not print, parse flags, or read env; accept explicit params and context.Context.
No global state: Thread dependencies via constructors; prefer small, focused structs over globals/singletons.

Configuration (Viper):

Precedence: Flags > env (PS_*) > config file > defaults.
Binding: Bind flags to keys; set defaults in one place; validate early in PersistentPreRunE.
Deterministic read: Support --config and project/home locations; handle “file not found” gracefully.

Output and UX:

Two modes: Human-readable default plus --json machine output. Keep JSON schema stable.
Deterministic ordering: Sort inputs; ensure stable output order in parallel scans.
Progress for long runs: Optional progress reporting behind a flag; never spam stdout in --json mode.

Error handling:

Wrap with context: fmt.Errorf("opening %s: %w", path, err).
User-friendly CLI errors: Validate inputs early; print actionable messages; non-zero exit codes on failure.
Library boundaries: Return typed/sentinel errors from internal/*; mapping to CLI messages happens in cmd/*.

Concurrency and streaming:

Streaming-first: Never load entire files; use bounded buffers; handle long lines explicitly.
Worker pools with backpressure: Size with --workers; keep deterministic rendering by indexing results.
Context everywhere: Accept and honor context.Context for cancellation and timeouts.

Testing:

Unit (table-driven): Core packages (internal/*) with property tests for tricky parts.
CLI integration: testscript/golden files for flags, env, completion, and output formats.
Performance benches: Critical paths (scanning, rule evaluation); protect budgets.

Observability and logging:

Structured logging: Libraries return data; CLI decides when to log. No fmt.Print in libraries.
Metrics/tracing hooks: Pluggable, behind interfaces; do not contaminate business code.

Build, release, and versioning:

ldflags for version: Inject version, commit, buildDate.
Makefile: build, test, lint, fmt, release.
Releaser: Automate cross-platform builds, checksums, and signatures.

Code style and APIs:

Effective Go: Clear names, no stutter, small interfaces, explicit error flows, early returns.
Small interfaces at boundaries: Accept interfaces, return concrete types.
Avoid over-abstraction: Introduce interfaces only where substitution is needed.

Security & supply chain:

Enforce govulncheck, gosec, golangci-lint in CI; block on critical findings
Generate SBOM (CycloneDX/Syft) and scan (Grype/Trivy) on every release
Reproducible, signed releases (GoReleaser + checksums + Cosign); publish provenance (SLSA)
Secrets hygiene: never log inputs; redact tokens; keyring for API keys; zero secrets in configs
Validate/limit inputs: max size/line length, safe path handling, no symlink traversal without opt-in

Compatibility & contracts:

Stable JSON schema with versioning; document breaking changes
Stable exit codes (0 ok, 1 operational error, 2 usage/config error)
Deprecation policy and feature flags; semantic versioning with clear CHANGELOG

Observability & diagnostics:

Structured logs to stderr only; never mix logs with stdout results
OpenTelemetry hooks (metrics + tracing); emit timings per stage; correlation IDs
--debug and --quiet that do not affect stdout JSON payloads

Performance & resilience:

Handle very long lines (JSONL) beyond Scanner buffer; chunked reader strategy
Bound memory explicitly; size worker pool; backpressure when ordering buffer grows
Timeouts and cancellation via context; per-file/per-rule budgets; retries where appropriate

UX & output discipline:

Zero noise in --json mode; progress/errors to stderr only
Deterministic ordering guaranteed; include file counts and timing summaries in stylish mode
Helpful errors with suggested fixes and links to docs

Testing strategy:

Expand testscript coverage for error paths, large files, glob + recursive discovery, Windows paths
Fuzz tests for rule parsing and regex flags; race detector in CI (-race)
Benchmarks with size/latency budgets; performance regression guardrails

Configuration governance:

Strict validation for config keys and values; reject unknown keys in config files
Document precedence; support environment-only operation for containerized use
Config/RulePack schema JSON/YAML schemas with a validate subcommand

Release & distro:

Cross-platform artifacts (Windows/macOS/Linux; ARM/AMD64); Homebrew/Scoop packages
Auto-update with signature verification; air-gapped install docs

Docs & operations:

Threat model, security.md (disclosure policy), runbooks (logs, tracing, profiling, common failures)
Migration guides between versions; JSON schema reference with examples

Team workflows:

CODEOWNERS, Renovate/Dependabot, required reviews, protected branches, release checklist

Future rule engine hardening:

Sandboxed regex (avoid catastrophic backtracking), rule timeouts
LLM integration with rate limits, caching, and audit logs; tenant-aware controls

Also very important:
Color auto-detect TTY; explicit --no-color
Telemetry opt-in with clear privacy stance; coarse metrics only

File structure (aligns with plan/, scales cleanly) :

promptshield/
├── cmd/
│   ├── promptshield/           # main.go only
│   ├── root.go                 # global flags, config binding
│   ├── scan.go                 # orchestrate scan
│   ├── validate.go             # rulepack validation
│   └── version.go
├── internal/
│   ├── bootstrap/              # dependency wiring (construct services)
│   │   └── deps.go
│   ├── config/                 # Viper wiring, defaults, validation
│   │   └── config.go
│   ├── discovery/              # path/glob/dir discovery
│   ├── rules/                  # rule domain + loaders
│   │   ├── loader.go
│   │   ├── types.go
│   │   ├── validator.go        # strict schema checks
│   │   └── engine/             # matching strategies
│   │       ├── keyword.go
│   │       └── regex.go
│   ├── scanning/               # scan orchestrator (streaming + workers)
│   │   ├── scanner.go          # line/JSONL streaming
│   │   └── rule_eval.go
│   ├── reporting/              # renderers and selection
│   │   ├── report.go
│   │   ├── stylish.go
│   │   └── json.go
│   ├── logging/                # slog/zap factory; honors --debug/--quiet
│   ├── observability/          # metrics/tracing hooks (OTel-ready)
│   ├── security/               # keyring, redaction utils (future)
│   └── update/                 # self-update/check (future)
├── pkg/
│   └── types/                  # stable public ScanResult, Violation
├── rules/                      # built-in example packs
├── docs/
│   ├── CLI.md
│   ├── RulePacks.md
│   ├── Output.md
│   ├── Architecture.md
│   └── adr/                    # architecture decision records
├── testdata/                   # shared fixtures (if needed)
├── tools/                      # dev helpers, scripts
├── .github/workflows/          # ci.yml, release.yml, security.yml
├── Makefile
└── go.mod

Responsibilities:

cmd/: flags, help, arg parsing, call services; no business logic.
internal/config: one place for defaults, env mapping, validation; expose a typed Config.
internal/scanning: streaming and worker pool orchestration; deterministic merge.
internal/rules: schema types, YAML loader, strict validator, matching strategies in engine/.
internal/reporting: “stylish” and “json” renderers; select by config.
internal/bootstrap: build dependency graph (config, logger, rule engine, scanner, reporter).
pkg/types: only stable types external tooling may import.

Testing layout:

Unit: alongside packages, or consolidate under tests/unit/{rules,scanning,reporting}.
CLI integration: keep cmd/testdata/scripts (good); add more cases for long lines, globs, Windows paths.
Benchmarks: internal/scanning and internal/rules/engine critical paths.

enterprise evolution:

If you adopt the DDD split in the plan, mirror it as:
internal/domain/{rules,scanning} (pure models/logic)
internal/application/{commands,queries,services}
internal/infrastructure/{persistence,messaging,auth,monitoring}
internal/interfaces/{cli,grpc,http}

Notes for production polish:

Lock JSON schema stability: pkg/types versioning when breaking.
Activate logging/observability to honor --debug without polluting stdout in --json.
Add security/keyring and redaction helpers to support API key guidance from plan/.