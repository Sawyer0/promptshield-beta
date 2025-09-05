# Repository Guidelines

## Project Structure & Module Organization
- Backend (Go): `gateway/` (HTTP APIs, auth), `internal/` (domain, scanner, persistence), `enforcer/` (Envoy ext_proc), `cmd/` (CLIs), binary output in `bin/`.
- Frontend (TypeScript/React + Express BFF): `frontend/RulepackManager/` (`server/`, `client/`, `shared/`).
- Config & Ops: `docs/`, `rules/`, `migrations_consolidated/`, `deployments/`, `charts/`, `docker-compose.yaml`, `Makefile`.
- Examples: `examples/`, `scripts/`, API docs in `docs/api/`.

## Build, Test, and Development Commands
- `make build`: Build Go gateway to `bin/ps-gateway` (ldflags versioning).
- `make run`: Build and run the gateway locally.
- `make dev`: Run gateway and frontend together with auth bypass (uses `.env.dev`).
- `docker compose up -d --build`: Full demo stack (Envoy + Enforcer + backend).
- `make test`: Run backend and frontend tests. Windows backend: `make test-backend-win`.
- `make fmt` / `make format`: Format Go and frontend code. `make lint`: Lint both.

## Coding Style & Naming Conventions
- Go: Use `go fmt`; run `make fmt`. Packages lower-case, no underscores. Exported identifiers `CamelCase`; tests live next to code.
- Frontend: TypeScript strict; `npm run check` for types. Components `PascalCase`, variables `camelCase`. Run `npm run format` in `frontend/RulepackManager`.
- Commit small, focused changes; keep files and dirs descriptive (e.g., `internal/scanner/aho/…`).

## Testing Guidelines
- Backend: Standard `go test`; prefer table-driven tests; race checks enabled by default on Unix via `make test-backend`. DB tests: set `PS_TEST_PG_DSN` then `make test-backend-db`.
- Frontend: Vitest. `npm run test` or `make test-frontend`; coverage via `npm run test:coverage`.
- Test names: Go `*_test.go` with `TestXxx`; frontend `*.test.ts(x)` colocated with sources.

## Commit & Pull Request Guidelines
- Use Conventional Commits where possible: `feat(gateway): …`, `fix(scanner): …`, `perf: …`, `docs: …`.
- PRs: clear description, scope, and reasoning; link issues; include tests and updated docs; attach UI screenshots for frontend changes; ensure `make lint` and `make test` pass.

## Security & Configuration Tips
- Never commit secrets; use `.env.dev` for local only. Required for dev: `PS_PG_DSN`. Dev bypass: `PS_DEV_BYPASS_AUTH=true`.
- Generate JWT keys for BFF↔Gateway: `make jwt-keys` and `make jwt-export-env`.
- See `docs/RulePacks.md`, `docs/ENVOY_INTEGRATION.md`, and `docs/Runtime-Architecture.md` for deeper context.

