package services

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/promptshield/promptshield/internal/audit"
	nats "github.com/promptshield/promptshield/internal/infrastructure/messaging/nats"
	"github.com/promptshield/promptshield/internal/repository"
	"github.com/promptshield/promptshield/internal/shared/types"
)

func TestRulepackServiceFromFactory(t *testing.T) {
	// Set test mode to ensure we get a test factory
	os.Setenv("PS_TEST_MODE", "true")
	defer os.Unsetenv("PS_TEST_MODE")

	// Set audit sink to none to ensure we get a no-op logger
	os.Setenv("PS_AUDIT_SINK", "none")
	defer os.Unsetenv("PS_AUDIT_SINK")

	ctx := context.Background()
	
	// Create repository factory
	repoFactory, err := repository.BuildWithFallback(ctx)
	require.NoError(t, err, "Failed to create repository factory")
	defer repoFactory.Close()

	// Create NATS publisher
	publisher, err := nats.NewPublisher("")
	require.NoError(t, err, "Failed to create NATS publisher")
	defer publisher.Close()

	// Test RulepackServiceFromFactory
	rulepackService := RulepackServiceFromFactory(repoFactory, publisher)
	assert.NotNil(t, rulepackService, "RulepackService should not be nil")
	assert.NotNil(t, rulepackService.repo, "Repository should not be nil")
	assert.NotNil(t, rulepackService.pub, "Publisher should not be nil")
	assert.NotNil(t, rulepackService.audit, "Audit logger should not be nil (should be no-op logger)")
}

func TestPolicyServiceFromFactory(t *testing.T) {
	// Set test mode to ensure we get a test factory
	os.Setenv("PS_TEST_MODE", "true")
	defer os.Unsetenv("PS_TEST_MODE")

	ctx := context.Background()
	
	// Create repository factory
	repoFactory, err := repository.BuildWithFallback(ctx)
	require.NoError(t, err, "Failed to create repository factory")
	defer repoFactory.Close()

	// Test PolicyServiceFromFactory
	policyService := PolicyServiceFromFactory(repoFactory)
	assert.NotNil(t, policyService, "PolicyService should not be nil")
	
	// Note: The policy service will have nil repositories since PolicyRepository
	// is not yet implemented in the factory, but the service should still be created
}

func TestPolicyScannerServiceFromFactory(t *testing.T) {
	// Set test mode to ensure we get a test factory
	os.Setenv("PS_TEST_MODE", "true")
	defer os.Unsetenv("PS_TEST_MODE")

	ctx := context.Background()
	
	// Create repository factory
	repoFactory, err := repository.BuildWithFallback(ctx)
	require.NoError(t, err, "Failed to create repository factory")
	defer repoFactory.Close()

	// Test PolicyScannerServiceFromFactory
	scannerService := PolicyScannerServiceFromFactory(repoFactory)
	assert.NotNil(t, scannerService, "PolicyScannerService should not be nil")
	assert.NotNil(t, scannerService.scanner, "Scanner should not be nil")
	
	// Note: The scanner service will have nil policy repository since PolicyRepository
	// is not yet implemented in the factory, but the service should still be created
}

func TestNewServicesFromFactory(t *testing.T) {
	// Set test mode to ensure we get a test factory
	os.Setenv("PS_TEST_MODE", "true")
	defer os.Unsetenv("PS_TEST_MODE")

	ctx := context.Background()
	
	// Create repository factory
	repoFactory, err := repository.BuildWithFallback(ctx)
	require.NoError(t, err, "Failed to create repository factory")
	defer repoFactory.Close()

	// Create NATS publisher
	publisher, err := nats.NewPublisher("")
	require.NoError(t, err, "Failed to create NATS publisher")
	defer publisher.Close()

	// Test NewServicesFromFactory
	services := NewServicesFromFactory(repoFactory, publisher)
	assert.NotNil(t, services, "Services should not be nil")
	assert.NotNil(t, services.Rulepack, "Rulepack service should not be nil")
	assert.NotNil(t, services.Policy, "Policy service should not be nil")
	assert.NotNil(t, services.PolicyScanner, "PolicyScanner service should not be nil")
}

func TestNoOpAuditLogger(t *testing.T) {
	logger := &noOpAuditLogger{}
	
	// Test audit.Logger interface
	err := logger.Log(audit.Event{Type: "test"})
	assert.NoError(t, err, "Log should not return error")
	
	// Test contracts.AuditLogger interface
	ctx := context.Background()
	err = logger.LogWithContext(ctx, types.AuditEvent{Action: "test"})
	assert.NoError(t, err, "LogWithContext should not return error")
	
	err = logger.Flush()
	assert.NoError(t, err, "Flush should not return error")
	
	err = logger.Close()
	assert.NoError(t, err, "Close should not return error")
}

func TestAuditLoggerAdapter(t *testing.T) {
	// Create a mock audit logger
	mockLogger := &mockAuditLogger{}
	adapter := &auditLoggerAdapter{
		logger: mockLogger,
		closeFunc: func() error { return nil },
	}
	
	ctx := context.Background()
	event := types.AuditEvent{
		Action:     "test.action",
		ObjectType: "test",
		ObjectID:   uuid.New(),
	}
	
	err := adapter.LogWithContext(ctx, event)
	assert.NoError(t, err, "LogWithContext should not return error")
	assert.True(t, mockLogger.logCalled, "Log should have been called on underlying logger")
	
	err = adapter.Flush()
	assert.NoError(t, err, "Flush should not return error")
	
	err = adapter.Close()
	assert.NoError(t, err, "Close should not return error")
}

// mockAuditLogger for testing
type mockAuditLogger struct {
	logCalled bool
}

func (m *mockAuditLogger) Log(event audit.Event) error {
	m.logCalled = true
	return nil
}