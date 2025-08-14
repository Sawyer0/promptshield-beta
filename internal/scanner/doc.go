package scanner

// Package scanner implements the streaming scanning engine for PromptShield.
//
// Design highlights:
//   - Streaming-first: scans readers line-by-line with bounded memory
//   - Three-tier rule evaluation: keywords (L1), regex (L2), semantic (L3)
//   - Deterministic, parallel multi-file orchestration with backpressure
//   - Plug-in semantic analyzers with per-rule timeouts and fallbacks
//   - Structured tracing/logging injected from callers; no prints here
//
// Key files:
//   - io.go: file/reader streaming and very-long-line chunking
//   - evaluate.go: per-line rule evaluation (L1/L2/L3) and helpers
//   - loader.go: rulepack compilation, composition, perf gating (Aho/Bloom)
//   - orchestrator.go: worker pool over paths with deterministic ordering
//   - aho.go, bloom.go, regex_literals.go: fast matchers and gates
//   - semantic.go: L3 evaluation with budgets and fallbacks
//   - keywords.go: built-in keyword rules for first-run experience
//   - util.go, time_helpers.go: small utilities
