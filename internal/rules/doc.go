package rules

// Package rules contains the RulePack domain model, YAML loader, and validation.
//
// Responsibilities:
//   - Define the schema for rules, patterns, semantic config, and composition
//   - Load RulePacks from YAML (local files, globs, URLs handled at higher layers)
//   - Merge multiple packs with either additive or priority-order strategies
//   - Validate RulePacks strictly with helpful error messages
//
// This package holds pure domain concerns. It does not read env vars, print,
// or access the network. Callers provide inputs and receive data/typed errors.
