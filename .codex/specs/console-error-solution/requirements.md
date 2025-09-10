# Requirements Document

## Introduction

This effort will resolve the current TypeScript compilation failures in the RulepackManager frontend, align UI types with backend/BFF responses, and stabilize the test/build toolchain so `npm run check` and `npm run test` pass reliably. The work focuses on unifying entity models (especially `RulePack`), correcting API hook signatures, normalizing health/readiness data types, fixing component prop contracts, and tightening the testing setup (Vitest + MSW + utilities) to eliminate type errors and brittle mocks.

## Requirements

### Requirement 1: Unify RulePack Model Across UI

**User Story:** As a developer, I want a single, consistent `RulePack` shape across shared types, components, and tests, so that the UI compiles cleanly and behavior is predictable.

#### Acceptance Criteria

1. WHEN a `RulePack` is used in UI components or tests THEN the system SHALL use a consistent shape from `shared/apiTypes.ts`
2. WHEN components expect “active state” THEN the system SHALL use `isActive: boolean` (not `enabled`)
3. WHEN components expect a version identifier THEN the system SHALL use `currentVersionId?: string` (not `version`)
4. WHEN missing/legacy fields are referenced (e.g., `userId`) THEN the system SHALL either add optional fields to the type or adapt components to omit them
5. WHEN tests create mock rule packs THEN the system SHALL include required properties (e.g., `isActive`) to satisfy the shared type

### Requirement 2: Align API Hooks With lib/api Contracts

**User Story:** As a developer, I want `useAuthenticatedApi` to match the exported functions in `@/lib/api`, so that type-checking passes and calls don’t provide wrong arguments.

#### Acceptance Criteria

1. WHEN calling `tenantApi` THEN the hook SHALL use the available methods (`getMine`, `get`, `create`, `update`, `delete`) and SHALL NOT pass `userContext` where not accepted
2. WHEN calling `policyAssignmentApi` THEN the hook SHALL use the current signatures (`getAll()`, `get(id)`, `create(data)`, `update(id, patch)`, `delete(id)`) without extra args
3. WHEN calling `auditApi` THEN the hook SHALL call the existing functions with the correct argument counts and shapes
4. WHEN calling `rulePackApi` or `systemApi` THEN the hook SHALL pass arguments and user context consistent with their definitions

### Requirement 3: Normalize Health/Readiness Data for SystemHealth Page

**User Story:** As a user, I want the health dashboard to render correctly, regardless of whether endpoints return JSON or plain text, so that status indicators work.

#### Acceptance Criteria

1. WHEN `systemApi.getHealth()` returns a string THEN the page SHALL normalize it to `{ status: 'ok' | 'error' | ... }`
2. WHEN `systemApi.getReadiness()` returns a string or object THEN the page SHALL safely access readiness checks using guards
3. WHEN displaying component badges (API, DB, Auth, Storage) THEN the page SHALL handle string vs boolean statuses without type errors

### Requirement 4: Fix Component Prop Contracts (Layout, Modal, ConfirmDialog)

**User Story:** As a developer, I want component props to be correct and enforced, so pages don’t fail type checks.

#### Acceptance Criteria

1. WHEN using `Layout` THEN pages SHALL provide required `title` and `description` props
2. WHEN using `ConfirmDialog` THEN `Modal` SHALL receive a valid `children` node OR `Modal` SHALL accept optional `children`
3. WHEN updating components THEN the system SHALL keep the API consistent across callers (no breaking changes without fixes in all usages)

### Requirement 5: Update RulePackDetailsModal To Shared Shape

**User Story:** As a user, I want the RulePack details modal to display correct metadata, so that information reflects real data.

#### Acceptance Criteria

1. WHEN showing active/inactive state THEN the component SHALL use `isActive` (and not `enabled`)
2. WHEN showing current version THEN the component SHALL use `currentVersionId` (and not `version`)
3. WHEN creator/owner is not available THEN the component SHALL handle it gracefully (e.g., display Unknown) without accessing unknown fields
4. WHEN passing props to nested components (e.g., `VersionManager`) THEN the system SHALL pass the normalized fields

### Requirement 6: Fix AssignmentModal Tests And Mocks

**User Story:** As a developer, I want AssignmentModal tests to compile and run, so that CI can verify behavior.

#### Acceptance Criteria

1. WHEN constructing `mockRulePacks` in tests THEN each item SHALL include `isActive: boolean`
2. WHEN submitting the form THEN the test SHALL validate the expected payload using normalized field names
3. WHEN interacting with selects and inputs THEN tests SHALL avoid flakiness by using user-event with keyboard interactions as needed

### Requirement 7: Stabilize Test Utilities And MSW Handlers

**User Story:** As a developer, I want robust mocks and utilities, so that tests do not fail from type changes in libraries.

#### Acceptance Criteria

1. WHEN using `vi` in test utilities THEN the file SHALL import `{ vi }` from `vitest` or have `vitest` types included via tsconfig
2. WHEN parsing `request.json()` in MSW handlers THEN code SHALL cast to a known shape before property access (to avoid “possibly null/unknown” errors)
3. WHEN handlers construct responses THEN they SHALL match the shapes expected by the UI code (e.g., list endpoints returning `{ data, total }` when used that way)
4. WHEN using `@ts-expect-error` in setup THEN directives SHALL be removed or replaced with safe casts to avoid unused directive errors

### Requirement 8: Sidebar Sign-Out Flow

**User Story:** As a user, I want sign-out to work reliably, so that I’m redirected to the sign-in screen.

#### Acceptance Criteria

1. WHEN invoking sign-out THEN the system SHALL either import and call `signOut` from `@clerk/clerk-react` or handle absence gracefully
2. WHEN sign-out fails THEN the UI SHALL fall back to `window.location.href = '/sign-in'`

### Requirement 9: Organization Page Header Typing

**User Story:** As a developer, I want the organization page to fetch data with correctly typed headers, so TypeScript does not error.

#### Acceptance Criteria

1. WHEN constructing headers with optional Authorization THEN code SHALL build a `HeadersInit` using a concrete object (no union `{}` vs `{ Authorization }` ambiguity)
2. WHEN refreshing invitations THEN the same typed header builder SHALL be reused

### Requirement 10: Tenants Page API Method

**User Story:** As a developer, I want the tenants list to load, so that the page compiles and functions.

#### Acceptance Criteria

1. WHEN fetching tenants THEN the page SHALL call `tenantApi.getMine()` or an implemented `tenantApi.getAll()` consistent with backend support
2. WHEN mutations complete THEN the page SHALL invalidate the correct query keys

### Requirement 11: Users and Dashboard Implicit Any Fixes

**User Story:** As a developer, I want strict TypeScript compliance, so that there are no implicit `any` errors.

#### Acceptance Criteria

1. WHEN mapping or filtering over arrays in pages THEN callback params SHALL be explicitly typed
2. WHEN UI state or derived values are used THEN types SHALL be annotated to avoid implicit `any`

### Requirement 12: Auth Logger Context Typing

**User Story:** As a developer, I want `AuthLogger` methods to be type-safe, so that server logging compiles cleanly.

#### Acceptance Criteria

1. WHEN calling `logConfigurationIssue` THEN the `component` field SHALL be part of the context type or moved into `details`
2. WHEN formatting log messages THEN the logger SHALL accept the updated context type without error

### Requirement 13: Vitest + Vite Configuration

**User Story:** As a developer, I want the test config to be recognized by TypeScript and Vite, so CLI commands work.

#### Acceptance Criteria

1. WHEN defining `test` configuration THEN the project SHALL use `defineConfig` from `vitest/config` (or an equivalent supported pattern)
2. WHEN running `npm run test` THEN vitest SHALL pick up `test` options from the config without type errors

### Requirement 14: Express Request Augmentation For Server Tests

**User Story:** As a developer, I want server integration tests to compile, so the test suite runs.

#### Acceptance Criteria

1. WHEN accessing `req.auth` in server code/tests THEN TypeScript SHALL recognize the augmented field via a `.d.ts` module augmentation
2. WHEN building the project THEN no `Property 'auth' does not exist on type Request` errors SHALL occur

### Requirement 15: Keep Backward-Compatible API Shapes Where Practical

**User Story:** As a maintainer, I want minimal churn, so existing call sites don’t break unnecessarily.

#### Acceptance Criteria

1. WHEN renaming or normalizing fields (e.g., `enabled` → `isActive`) THEN the UI SHALL either map at boundaries or add optional aliases to types temporarily
2. WHEN normalizing server responses in `lib/api` THEN mapping SHALL ensure UI receives the shared shapes

## Out of Scope

- Backend behavior changes beyond type/shape normalization already used by the frontend
- Adding new UI features beyond what’s needed to remove the reported type errors

## Completion Criteria

- `npm run check` completes with zero TypeScript errors
- `npm run test` passes locally with no type or runtime failures
- No regressions in major views (Dashboard, Rulepacks, Tenants, Users, System Health)

