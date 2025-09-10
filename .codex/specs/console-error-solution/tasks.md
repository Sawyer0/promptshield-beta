# Implementation Plan

- [x] 1. Normalize RulePack model decisively

  - Map backend/BFF fields to the canonical shape in `client/src/lib/api.ts` with a single mapping path: `isActive` (from `active`), `currentVersionId` (from `version`). Do not introduce alternate field names in UI components.
  - Ensure all RulePack list/detail functions return `RulePack` per `shared/apiTypes.ts`.
  - Add compile-time checks by running `npm run check` after changes.
  - _Requirements: 1.1, 1.2, 1.3, 15.2_

- [x] 2. Refactor components to canonical fields

  - Update `RulePackDetailsModal` and any consumers to exclusively use `isActive` and `currentVersionId`. Remove direct references to `enabled`, `version`, `userId` on RulePack.
  - Pass normalized props to `VersionManager`.
  - Guard `rules` rendering with safe checks.
  - _Requirements: 1.1, 5.1, 5.2, 5.3_

- [x] 3. Fix AssignmentModal tests and data

  - Update `mockRulePacks` to include `isActive` in `client/src/components/__tests__/AssignmentModal.test.tsx`.
  - Keep user-event flows; verify submission assertions match normalized field names.
  - _Requirements: 6.1, 6.2_

- [x] 4. Make hooks match API (no optional paths)

  - Update `useAuthenticatedApi` to call functions with the exact signatures in `client/src/lib/api.ts` (no `userContext` where not accepted).
  - Implement `tenantApi.getAll()` in `client/src/lib/api.ts` to return the same shape as `getMine()` for UI simplicity; remove `getMine()` usage from pages and standardize on `getAll()`.
  - Implement `policyAssignmentApi.batchCreate()` (sequential client-side fallback) and `getByEndpoint()` (filter client-side) to satisfy callers.
  - _Requirements: 2.1, 2.2, 2.3, 10.1_

- [x] 5. Normalize health/readiness usage

  - Update `client/src/pages/SystemHealth.tsx` to normalize string health to `{ status }` and guard readiness checks with runtime type checks.
  - Ensure badges and icons rely only on normalized values.
  - _Requirements: 3.1, 3.2, 3.3_

- [x] 6. Enforce component contracts

  - Ensure all pages pass `title` and `description` to `Layout`.
  - Make `Modal` `children` optional in props to satisfy `ConfirmDialog` without hacks.
  - _Requirements: 4.1, 4.2_

- [x] 7. Robust sign‑out flow

  - Import and use `signOut` from `@clerk/clerk-react`; if unavailable at runtime, fall back to `window.location.href = '/sign-in'` after clearing sessions.
  - _Requirements: 8.1, 8.2_

- [x] 8. Strongly typed fetch headers in Organization

  - Add a `buildAuthHeaders(): Promise<HeadersInit>` helper and use it for both member and invitation requests to eliminate union header types.
  - _Requirements: 9.1, 9.2_

- [x] 9. Tenants and Users typing fixes

  - Standardize tenants fetching on `tenantApi.getAll()`; update `Tenants.tsx` query accordingly.
  - Add explicit types to map/filter callbacks in Tenants/Users/Dashboard.
  - _Requirements: 10.1, 11.1, 11.2_

- [x] 10. Test utilities and MSW hardening

  - Import `{ vi }` from `vitest` in `client/src/test/utils/test-utils.tsx`.
  - Replace unused `@ts-expect-error` in `client/src/test/setup.ts` with safe shims.
  - In MSW handlers, cast `await request.json()` and add null checks before property access; align response shapes with UI expectations.
  - _Requirements: 7.1, 7.2, 7.3, 7.4_

- [x] 11. Vite/Vitest config typing (decisive)

  - Switch to `defineConfig` from `vitest/config` and export a typed config with a `test` block; remove patterns that rely on top-level `await` arrays.
  - _Requirements: 13.1, 13.2_

- [x] 12. AuthLogger context alignment

  - Extend `AuthLogContext` with `component?: string` and update `logConfigurationIssue` accordingly.
  - _Requirements: 12.1_

- [x] 13. Server test typing

  - Confirm `server/types/express.d.ts` augmentation is included; fix any remaining `req.auth` type errors.
  - _Requirements: 14.1, 14.2_

- [ ] 14. Validate and document

  - Run `npm run check` to zero errors.
  - Run `npm run test` to green.
  - Add a concise Developer Notes section summarizing normalization points and invariants.
  - _Requirements: Completion Criteria_
