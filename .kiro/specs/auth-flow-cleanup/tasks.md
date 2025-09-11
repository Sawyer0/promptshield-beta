# Implementation Plan

- [x] 1. Fix JWT private key handling in frontend BFF


  - Add private key format validation and auto-correction in jwtAuth.ts
  - Implement startup configuration validation for JWT settings
  - Add detailed error logging for JWT generation failures
  - _Requirements: 1.1, 1.2, 4.2, 5.1_



- [x] 2. Improve JWT validation middleware in Go backend


  - Enhance parseRSAPublicKeyFromPEM function with better error handling
  - Add structured error responses with specific failure reasons in middleware_jwt.go

  - Implement request correlation ID support for debugging
  - _Requirements: 1.2, 1.3, 4.1, 4.3_

- [x] 3. Fix tenant context middleware chain

  - Modify tenantValidationMiddleware to prioritize JWT claims for tenant ID
  - Add specific error codes for different tenant validation failures
  - Ensure proper middleware ordering in router.go
  - _Requirements: 2.1, 2.3, 3.1, 3.2_

- [x] 4. Standardize dev bypass mode handling



  - Ensure PS_DEV_BYPASS_AUTH works consistently across frontend and backend
  - Add consistent dev user injection in both components
  - Implement unified dev tenant handling
  - _Requirements: 3.3, 5.2, 5.3_

- [x] 5. Add comprehensive error response structure


  - Create standardized error response format with codes and details
  - Implement error code constants for different failure scenarios
  - Add request correlation IDs to all error responses
  - _Requirements: 1.3, 4.1, 4.3_

- [x] 6. Improve JWT token generation robustness


  - Add validation for required JWT configuration on startup
  - Implement better error handling for missing or invalid private keys
  - Add token expiration and claim validation
  - _Requirements: 1.1, 4.2, 5.4_

- [x] 7. Fix tenant membership validation


  - Improve tenant membership checking with specific error messages
  - Add proper RLS context setting with error handling
  - Implement tenant status validation (active/inactive)
  - _Requirements: 2.2, 2.3, 4.3_

- [x] 8. Add authentication debugging endpoints


  - Create debug endpoint to validate JWT configuration
  - Add endpoint to check current user and tenant context
  - Implement auth health check endpoint
  - _Requirements: 4.1, 4.4_

- [x] 9. Update environment variable handling


  - Standardize JWT-related environment variable names
  - Add validation for required environment variables
  - Implement consistent fallback values for development
  - _Requirements: 5.1, 5.2, 5.4_

- [x] 10. Add comprehensive logging for auth flow


  - Implement structured logging for JWT validation steps
  - Add tenant context logging with correlation IDs
  - Create auth flow tracing for debugging
  - _Requirements: 4.1, 4.4_

- [x] 11. Create integration tests for auth flow


  - Write tests for complete Clerk → JWT → Backend flow
  - Add tests for tenant selection and validation
  - Implement error scenario testing
  - _Requirements: 1.1, 1.2, 2.1, 2.2_

- [x] 12. Update documentation and error messages



  - Create clear error message documentation
  - Add troubleshooting guide for common auth issues
  - Update environment variable documentation
  - _Requirements: 4.1, 4.4, 5.4_