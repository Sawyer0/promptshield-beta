# Design Document

## Overview

This design eliminates the TypeScript build/test errors in the RulepackManager frontend by aligning shared types, normalizing API contracts, tightening component props, and hardening the testing toolchain. The approach is incremental and non‑disruptive: normalize data at boundaries, keep UI components consistent with the shared model, and make tests resilient and type‑safe.

## Architecture

### Current State Analysis

- UI uses `shared/apiTypes.ts` but some components/tests assume different shapes (e.g., `enabled` vs `isActive`, `version` vs `currentVersionId`).
- Hooks in `useAuthenticatedApi` pass arguments not supported by functions in `client/src/lib/api.ts` (extra `userContext` params, missing methods like `getAll`).
- System health endpoints return plain text in some environments, but the page expects objects with `.status` and `.checks`.
- Several components have prop mismatches (e.g., `Modal` always requiring `children`; pages using `Layout` without required props).
- Tests and MSW handlers access untyped `request.json()` fields, and test utilities use `vi` without an explicit import in some contexts.
- Vite config uses a `test` block in `vite.config.ts` that TypeScript flags unless imported from `vitest/config` or handled as `UserConfigExport` correctly.

### Issues Identified

1. Divergent RulePack shape across components/tests
2. Hook ↔ API signature mismatches
3. Health/readiness data shape mismatch (string vs object)
4. Component contract gaps (required props, optional children)
5. Test utilities and MSW handlers lack strict typing; stray `@ts-expect-error`
6. Sidebar sign‑out references `signOut` without import/guard
7. Vite/Vitest config typing
8. Server tests: missing Express Request augmentation for `req.auth`

## Components and Interfaces

### 1) Type Model Normalization

- Source of truth: `shared/apiTypes.ts`.
- Normalization at API boundary (client/src/lib/api.ts) maps backend/BFF responses to shared shapes.

RulePack mapping policy at boundary:
- active state → `isActive: boolean`
- version id → `currentVersionId?: string`
- optional metadata fields allowed; components must tolerate absence

Updates in UI components:
- Replace `details.enabled` with `details.isActive`
- Replace `details.version` with `details.currentVersionId`
- Remove reliance on fields not present in model (e.g., `userId`) or make them optional and guarded

### 2) Hooks/API Alignment

- `useAuthenticatedApi` returns methods that directly mirror `lib/api.ts` exports.
- Remove extra `userContext` args where not accepted.
- For tenants list, use `tenantApi.getMine()` (or implement `getAll()` in api if truly required and used elsewhere).

### 3) Health/Readiness Normalization

- `systemApi.getHealth()` may return plain text. UI normalizes to `{ status: string }`.
- `systemApi.getReadiness()` may return text or object; UI guards with `typeof === 'object'` and optional chaining.

### 4) Component Prop Contracts

- `Layout` remains: `title: string; description: string; children: ReactNode`.
- Pages must pass required props; fix usages where missing.
- `Modal` accepts optional `children?: ReactNode` to support confirm dialogs with only description/footer.

### 5) Testing Tooling and MSW

- Import `{ vi }` from `vitest` in utilities using `vi`.
- MSW handlers cast `await request.json()` to a narrow type before property access to avoid “possibly null/unknown”.
- Remove unused `@ts-expect-error` directives; replace with safe casts/mocks.
- Ensure `server/test` Request augmentation for `req.auth` via `server/types/express.d.ts` is included by tsconfig.

### 6) Sidebar Sign‑Out

- Either import `signOut` from `@clerk/clerk-react` or feature‑detect and fallback to setting `window.location.href = '/sign-in'`.

### 7) Vite/Vitest Config

- Use `defineConfig` from `vitest/config` or adapt the config so `test` block is typed correctly.
- Keep existing plugins; no runtime behavior change.

### 8) Auth Logger Context

- Extend `AuthLogContext` to include `component?: string` or move it to `details` when logging configuration issues.

## Data Models

### RulePack (shared)
```ts
export interface RulePack {
  id: string;
  tenantId?: string | null;
  name: string;
  description?: string | null;
  currentVersionId?: string | null;
  createdAt?: string;
  updatedAt?: string;
  yamlContent?: string | null;
  rules?: any;
  isActive: boolean;
  status?: string;
  enforcementMode?: 'monitor' | 'enforce' | 'redact' | string;
  failOnSeverity?: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL' | string;
  priority?: number;
  metadata?: any;
}
```

### Health/Readiness
- Health: string | `{ status: string }` → normalized in UI to object with `status`.
- Readiness: string | `{ status: string; checks?: { database?: boolean; auth?: boolean; storage?: boolean } }` → UI uses guards and optional chaining.

## Error Handling

- Type mismatches: enforced at boundaries with mapping in `lib/api.ts` and explicit guards in components.
- Header construction: build concrete `HeadersInit` objects instead of unions with `{}`.
- Tests: MSW handlers cast body to narrow shapes; utilities import `{ vi }` and avoid unused `@ts-expect-error`.
- Config: vitest config typed via `vitest/config` to remove “No overload matches” errors.

## Testing Strategy

### Unit Tests
- Components updated to use `isActive`/`currentVersionId` render correctly.
- AssignmentModal tests pass with mock rule packs including `isActive`.
- Sidebar sign‑out path guarded.
- SystemHealth handles string/object responses.

### Integration/Mock Tests
- MSW handlers return shapes expected by UI list/detail endpoints: `{ data, total }` where applicable.
- Server tests compile due to Express Request augmentation for `req.auth`.

### Tooling
- Ensure `vitest` globals via config; utilities import `{ vi }` directly when used in module scope.

## Implementation Plan

### Phase 1: Types and Boundary Mapping
- Normalize RulePack mapping in `client/src/lib/api.ts` to shared shape.
- Update components (`RulePackDetailsModal`, children props to VersionManager) to use `isActive`/`currentVersionId`.
- Fix AssignmentModal tests to include `isActive` in mocks.

### Phase 2: Hook/API Signature Alignment
- Update `client/src/hooks/useAuthenticatedApi.ts` to match `lib/api.ts` method signatures.
- Tenants page to use `tenantApi.getMine()` for list.

### Phase 3: Health/Readiness + Components
- Normalize health/readiness data handling in `SystemHealth.tsx`.
- Make `Modal` children optional; update `ConfirmDialog` usage accordingly.
- Ensure pages provide `Layout` props.

### Phase 4: Testing and Config
- Import `{ vi }` where needed (test utils); remove unused `@ts-expect-error` with safe shims.
- Type MSW request bodies in handlers before access.
- Adjust Vite/Vitest config typing using `vitest/config`.

### Phase 5: Misc Fixes
- Sidebar sign‑out guard (import or fallback logic).
- AuthLogger context typing for `component`.
- Verify Express Request augmentation is picked by tsconfig `types`/`include`.

## Migration Strategy

- Changes are internal to the frontend; no backend API change required.
- Boundary mapping preserves backward compatibility by adapting backend/BFF responses to the shared UI model.
- Rollout: commit in small, testable steps; ensure `npm run check` and `npm run test` are green locally.

## Risks and Mitigations

- Hidden consumers of old fields (`enabled`, `version`): mitigate by centralized mapping and a quick search for references; add optional aliases only if strictly necessary.
- Test fragility with overlay components: continue using user‑event with pointer checks disabled where appropriate.
- Config typing drift: keep vitest config minimal and typed via `vitest/config`.

## Success Criteria

- `npm run check` returns zero TypeScript errors.
- `npm run test` passes with no type/runtime failures.
- UI components render using the normalized RulePack model without regressions.

