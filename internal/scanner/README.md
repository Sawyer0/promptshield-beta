Scanner package layout

- scanner.go: public struct, constructors, and configurators only
- io.go: streaming scan over files/readers; long-line chunking
- evaluate.go: per-line rule evaluation (keywords/regex/semantic)
- loader.go: RulePack compilation, composition, perf settings (Aho/Bloom)
- orchestrator.go: deterministic multi-file scanning with worker pool
- aho.go, bloom.go: fast matchers/gates
- regex_literals.go: literal token extraction from regex for gating
- semantic.go: L3 analyzer interface + timeout/fallback handling
- keywords.go: built-in keyword set for first run (opt-in)
- time_helpers.go, util.go: small utilities

Principles:
- Streaming-first, bounded memory
- No printing inside the package; structured logs and tracing injected
- Deterministic output ordering
- Pure functions where possible; clear separation of concerns

