package api

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// MockServiceManager provides a mock implementation for testing
type MockServiceManager struct {
	services map[string]ServiceStatus
}

// NewMockServiceManager creates a new mock service manager
func NewMockServiceManager() *MockServiceManager {
	return &MockServiceManager{
		services: make(map[string]ServiceStatus),
	}
}

// StartService simulates starting a service
func (m *MockServiceManager) StartService(ctx context.Context, tenantID uuid.UUID, config ServiceConfig) error {
	serviceName := fmt.Sprintf("ps-%s-%s", tenantID.String()[:8], config.ServiceName)
	
	slog.Info("Starting service", "service", serviceName, "tenant_id", tenantID)
	
	// Simulate startup time
	time.Sleep(2 * time.Second)
	
	m.services[serviceName] = ServiceStatusRunning
	return nil
}

// StopService simulates stopping a service
func (m *MockServiceManager) StopService(ctx context.Context, tenantID uuid.UUID, serviceName string) error {
	fullServiceName := fmt.Sprintf("ps-%s-%s", tenantID.String()[:8], serviceName)
	
	slog.Info("Stopping service", "service", fullServiceName, "tenant_id", tenantID)
	
	// Simulate shutdown time
	time.Sleep(1 * time.Second)
	
	m.services[fullServiceName] = ServiceStatusStopped
	return nil
}

// GetServiceStatus returns the current status of a service
func (m *MockServiceManager) GetServiceStatus(ctx context.Context, tenantID uuid.UUID, serviceName string) (ServiceStatus, error) {
	fullServiceName := fmt.Sprintf("ps-%s-%s", tenantID.String()[:8], serviceName)
	
	if status, exists := m.services[fullServiceName]; exists {
		return status, nil
	}
	
	return ServiceStatusStopped, nil
}

// RestartService simulates restarting a service
func (m *MockServiceManager) RestartService(ctx context.Context, tenantID uuid.UUID, serviceName string) error {
	fullServiceName := fmt.Sprintf("ps-%s-%s", tenantID.String()[:8], serviceName)
	
	slog.Info("Restarting service", "service", fullServiceName, "tenant_id", tenantID)
	
	// Simulate restart time
	time.Sleep(3 * time.Second)
	
	m.services[fullServiceName] = ServiceStatusRunning
	return nil
}

// ScaleService simulates scaling a service
func (m *MockServiceManager) ScaleService(ctx context.Context, tenantID uuid.UUID, serviceName string, replicas int) error {
	fullServiceName := fmt.Sprintf("ps-%s-%s", tenantID.String()[:8], serviceName)
	
	slog.Info("Scaling service", "service", fullServiceName, "tenant_id", tenantID, "replicas", replicas)
	
	// Simulate scaling time
	time.Sleep(1 * time.Second)
	
	if replicas == 0 {
		m.services[fullServiceName] = ServiceStatusStopped
	} else {
		m.services[fullServiceName] = ServiceStatusRunning
	}
	
	return nil
}