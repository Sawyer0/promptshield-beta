# Implementation Plan

- [x] 1. Create core factory interfaces and configuration structures




  - Define RepositoryFactory interface with methods for all repository types
  - Create RepositoryConfig struct with database, Redis, and environment settings
  - Implement factory builder pattern with configuration validation
  - Write unit tests for configuration validation and interface contracts
  - _Requirements: 1.1, 4.1, 4.2, 4.4_

- [x] 2. Implement connection management system


  - Create ConnectionManager struct to handle PostgreSQL and Redis connections
  - Implement connection pooling with configurable limits and timeouts
  - Add health check methods for database and Redis connectivity
  - Write unit tests for connection lifecycle and error scenarios
  - _Requirements: 3.1, 3.2, 3.4, 2.3_

- [x] 3. Create production repository factory implementation



  - Implement ProductionRepositoryFactory with PostgreSQL backing
  - Add Redis caching layer for tenant, assignment, and token repositories
  - Configure appropriate cache TTL values for each repository type
  - Write integration tests with real PostgreSQL and Redis connections
  - _Requirements: 2.1, 1.2, 3.1_

- [x] 4. Implement development repository factory




  - Create DevelopmentRepositoryFactory using PostgreSQL without caching
  - Add detailed logging and error reporting for debugging
  - Implement graceful fallback when Redis is unavailable
  - Write unit tests for fallback scenarios and error handling
  - _Requirements: 2.4, 3.4, 1.2_

- [x] 5. Create test repository factory and utilities




  - Implement TestRepositoryFactory with in-memory repositories
  - Create standardized mock implementations for all repository interfaces
  - Add test utility functions for factory creation and cleanup
  - Write comprehensive test suite for test factory functionality
  - _Requirements: 5.1, 5.2, 5.3, 5.4_

- [x] 6. Add environment detection and auto-configuration




  - Implement automatic environment detection based on configuration
  - Create factory selection logic based on available dependencies
  - Add configuration validation with clear error messages
  - Write tests for environment detection and factory selection
  - _Requirements: 2.1, 2.2, 2.3, 4.4_

- [x] 7. Migrate gateway main.go to use repository factory





  - Replace individual repository initialization with factory usage
  - Update service constructors to accept repositories from factory
  - Add proper error handling and graceful shutdown
  - Write integration tests to verify gateway functionality
  - _Requirements: 1.1, 1.3, 3.2, 3.3_

- [x] 8. Update test files to use shared test factory



  - Replace duplicate mock repository creation in test files
  - Update integration tests to use TestRepositoryFactory
  - Migrate fuzz tests and property tests to use factory
  - Remove duplicate mock repository implementations
  - _Requirements: 5.1, 5.2, 1.1, 1.3_

- [x] 9. Migrate service constructors and bootstrap code





  - Update policy service bootstrap to use repository factory
  - Modify service constructors to accept factory-created repositories
  - Replace memory.NewRulepackRepository() calls with factory usage
  - Write unit tests for updated service constructors
  - _Requirements: 1.1, 1.3, 4.3_

- [x] 10. Add comprehensive error handling and monitoring



  - Implement detailed error messages for connection failures
  - Add health check endpoints for repository connectivity
  - Create metrics and logging for repository operations
  - Write tests for error scenarios and recovery mechanisms
  - _Requirements: 3.4, 4.4, 2.3, 2.4_

- [x] 11. Clean up deprecated repository initialization patterns





  - Remove duplicate repository creation code from existing files
  - Delete unused mock repository implementations
  - Update documentation to reflect new factory usage
  - Run full test suite to ensure no regressions
  - _Requirements: 1.1, 1.3, 4.3_

- [x] 12. Add factory lifecycle management and resource cleanup



  - Implement proper Close() methods for all factory implementations
  - Add graceful shutdown handling for database connections
  - Create resource cleanup utilities for tests
  - Write tests for resource cleanup and connection management
  - _Requirements: 3.2, 3.3, 5.4_