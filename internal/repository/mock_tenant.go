package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

// MockTenantRepository provides a simple in-memory implementation
type MockTenantRepository struct {
	mu      sync.RWMutex
	tenants map[uuid.UUID]*domain.Tenant
}

// NewMockTenantRepository creates a new mock tenant repository
func NewMockTenantRepository() *MockTenantRepository {
	return &MockTenantRepository{
		tenants: make(map[uuid.UUID]*domain.Tenant),
	}
}

// Create creates a new tenant
func (r *MockTenantRepository) Create(ctx context.Context, tenant *domain.Tenant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tenants[tenant.ID] = tenant
	return nil
}

// Get retrieves a tenant by ID
func (r *MockTenantRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if tenant, exists := r.tenants[id]; exists {
		return tenant, nil
	}
	return nil, fmt.Errorf("tenant not found")
}

// GetByName retrieves a tenant by name
func (r *MockTenantRepository) GetByName(ctx context.Context, name string) (*domain.Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	for _, tenant := range r.tenants {
		if tenant.Name == name {
			return tenant, nil
		}
	}
	return nil, fmt.Errorf("tenant not found")
}

// List lists tenants with pagination
func (r *MockTenantRepository) List(ctx context.Context, offset, limit int) ([]*domain.Tenant, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	tenants := make([]*domain.Tenant, 0, len(r.tenants))
	for _, tenant := range r.tenants {
		tenants = append(tenants, tenant)
	}
	
	total := len(tenants)
	
	// Apply pagination
	if offset >= total {
		return []*domain.Tenant{}, total, nil
	}
	
	end := offset + limit
	if end > total {
		end = total
	}
	
	return tenants[offset:end], total, nil
}

// GetByExternalOrg retrieves a tenant by external organization mapping
func (r *MockTenantRepository) GetByExternalOrg(ctx context.Context, provider string, externalOrgID string) (*domain.Tenant, error) {
	// For mock implementation, we'll just return not found
	return nil, fmt.Errorf("tenant not found for external org")
}

// LinkExternalOrg links an external organization to a tenant
func (r *MockTenantRepository) LinkExternalOrg(ctx context.Context, provider string, externalOrgID string, tenantID uuid.UUID) error {
	// For mock implementation, we'll just return success
	return nil
}

// Update updates an existing tenant
func (r *MockTenantRepository) Update(ctx context.Context, tenant *domain.Tenant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tenants[tenant.ID] = tenant
	return nil
}

// Delete deletes a tenant
func (r *MockTenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tenants, id)
	return nil
}



// Reset clears all data (for testing)
func (r *MockTenantRepository) Reset() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tenants = make(map[uuid.UUID]*domain.Tenant)
	return nil
}