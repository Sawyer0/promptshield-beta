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
- Install: Docker/Kubernetes deployment for Gateway + Enforcer
- Envoy integration examples and `/v1/*` API docs
- Release notes: highlights + breaking changes

Highlights (since v0.2.x):
- Structured logging with request correlation IDs across Gateway and audit
- Configurable scanner buffers and overlap (performance.buffer_bytes/chunk_overlap)
- Global regex cache for Level 2 patterns
- RulePack composition strategies: all_matches (default), first_match, priority_order; extends/overrides merge support
- Redaction verifiers (Luhn) and expanded token patterns
- Envoy integration: HTTP `/v1/check`, `/healthz`, `/metrics` and gRPC ext_proc streaming

Optional (pre‑GA):
- Self‑update flow with signature verification


