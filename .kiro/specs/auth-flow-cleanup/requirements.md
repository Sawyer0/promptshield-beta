# Requirements Document

## Introduction

The current multi-tenant authentication system using Clerk has several issues causing 401 errors and inconsistent auth flow. The system needs to be cleaned up to ensure proper authentication and tenant context flow between the frontend BFF (Backend for Frontend) and the Go gateway backend.

## Requirements

### Requirement 1: Fix Authentication Flow Consistency

**User Story:** As a developer, I want the authentication flow to work consistently between the frontend and backend, so that authenticated users don't receive 401 errors for valid requests.

#### Acceptance Criteria

1. WHEN a user is authenticated via Clerk THEN the frontend BFF SHALL generate valid JWT tokens for backend requests
2. WHEN the backend receives a JWT token THEN it SHALL properly validate the token and extract user context
3. WHEN authentication fails THEN the system SHALL provide clear error messages indicating the specific failure reason
4. WHEN a user accesses `/api/auth/user` THEN it SHALL return user information if authenticated or a proper 401 with details if not

### Requirement 2: Streamline Tenant Context Handling

**User Story:** As a user, I want my tenant selection to persist properly across requests, so that I can access my organization's resources without repeated authentication failures.

#### Acceptance Criteria

1. WHEN a user selects a tenant THEN the tenant ID SHALL be stored in signed cookies and included in JWT tokens
2. WHEN the backend validates tenant context THEN it SHALL accept tenant ID from JWT claims as the primary source
3. WHEN tenant validation fails THEN the system SHALL return specific error codes (TENANT_NOT_FOUND, TENANT_INACTIVE, etc.)
4. WHEN a user accesses `/api/session/tenant` THEN it SHALL return the current tenant context from cookies

### Requirement 3: Clean Up Middleware Chain

**User Story:** As a developer, I want the middleware chain to be clean and predictable, so that authentication and authorization logic is easy to understand and debug.

#### Acceptance Criteria

1. WHEN the backend processes a request THEN it SHALL apply middleware in a consistent order: JWT auth, tenant validation, then business logic
2. WHEN JWT validation fails THEN the middleware SHALL not proceed to tenant validation
3. WHEN dev bypass mode is enabled THEN the system SHALL consistently bypass auth across all components
4. WHEN public endpoints are accessed THEN they SHALL bypass authentication without errors

### Requirement 4: Improve Error Handling and Debugging

**User Story:** As a developer, I want clear error messages and logging, so that I can quickly identify and fix authentication issues.

#### Acceptance Criteria

1. WHEN authentication fails THEN the system SHALL log the specific failure reason (invalid token, expired, missing tenant, etc.)
2. WHEN JWT generation fails THEN the BFF SHALL return a 500 error with details about the configuration issue
3. WHEN tenant validation fails THEN the backend SHALL return structured error responses with error codes
4. WHEN in development mode THEN the system SHALL provide additional debugging information in error responses

### Requirement 5: Standardize Environment Configuration

**User Story:** As a developer, I want consistent environment variable usage across frontend and backend, so that configuration is predictable and deployment is reliable.

#### Acceptance Criteria

1. WHEN configuring JWT authentication THEN both frontend and backend SHALL use the same environment variable names for keys and settings
2. WHEN enabling dev bypass mode THEN the PS_DEV_BYPASS_AUTH flag SHALL work consistently across all components
3. WHEN setting tenant defaults THEN the PS_DEV_TENANT_ID SHALL be used consistently for development
4. WHEN JWT keys are missing THEN the system SHALL fail fast with clear error messages during startup