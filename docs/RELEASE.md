### Release checklist (v0.2.x → v0.3.0)

CI & quality gates:
- GitHub Actions: `ci.yml` runs `make fmt`, `make lint`, `make test`, govulncheck, gosec
- Security workflow: dependency scan (OSV/Snyk optional)

Packaging:
- GoReleaser config for multi‑OS/arch (darwin/linux/windows; amd64/arm64)
- Checksums (SHA256) and cosign signatures
- SBOM via Syft; vulnerability scan via Grype/Trivy

Versioning:
- Bump `cmd/version.go` version
- Update `CHANGELOG.md` with features and fixes
- Verify Makefile ldflags set version/commit/date

Docs:
- Install: Homebrew, Scoop, curl‑bash
- Shell completions: `promptshield completion [bash|zsh|fish|powershell]`
- Release notes: highlights + breaking changes

Highlights (since v0.2.x):
- Structured logging with request correlation IDs across CLI and audit
- Configurable scanner buffers and overlap (performance.buffer_bytes/chunk_overlap)
- Global regex cache for Level 2 patterns
- RulePack composition strategies: all_matches (default), first_match, priority_order; extends/overrides merge support
- Redaction verifiers (Luhn) and expanded token patterns
- Discovery allow/deny path controls (PS_ALLOW_PATHS/PS_DENY_PATHS) and glob breadth guard
- ps-enforcer: experimental HTTP `/healthz`, `/check`, `/metrics` and gRPC ext_proc streaming (not production-ready)

Optional (pre‑GA):
- Self‑update flow with signature verification


