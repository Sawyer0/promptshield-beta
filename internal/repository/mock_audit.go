package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

// MockAuditRepository provides a simple in-memory implementation
type MockAuditRepository struct {
	mu      sync.RWMutex
	entries map[uuid.UUID]*domain.AuditEntry
}

// NewMockAuditRepository creates a new mock audit repository
func NewMockAuditRepository() *MockAuditRepository {
	return &MockAuditRepository{
		entries: make(map[uuid.UUID]*domain.AuditEntry),
	}
}

// Create creates a new audit entry
func (r *MockAuditRepository) Create(ctx context.Context, entry *domain.AuditEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[entry.ID] = entry
	return nil
}

// Get retrieves an audit entry by ID
func (r *MockAuditRepository) Get(ctx context.Context, id uuid.UUID) (*domain.AuditEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if entry, exists := r.entries[id]; exists {
		return entry, nil
	}
	return nil, fmt.Errorf("audit entry not found")
}

// ListByTenant lists audit entries by tenant with pagination
func (r *MockAuditRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*domain.AuditEntry, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	entries := make([]*domain.AuditEntry, 0)
	for _, entry := range r.entries {
		if entry.TenantID != nil && *entry.TenantID == tenantID {
			entries = append(entries, entry)
		}
	}
	
	total := len(entries)
	
	// Apply pagination
	if offset >= total {
		return []*domain.AuditEntry{}, total, nil
	}
	
	end := offset + limit
	if end > total {
		end = total
	}
	
	return entries[offset:end], total, nil
}

// ListByObject lists audit entries by object with pagination
func (r *MockAuditRepository) ListByObject(ctx context.Context, objectType string, objectID uuid.UUID, offset, limit int) ([]*domain.AuditEntry, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	entries := make([]*domain.AuditEntry, 0)
	for _, entry := range r.entries {
		if entry.ObjectType == objectType && entry.ObjectID == objectID {
			entries = append(entries, entry)
		}
	}
	
	total := len(entries)
	
	// Apply pagination
	if offset >= total {
		return []*domain.AuditEntry{}, total, nil
	}
	
	end := offset + limit
	if end > total {
		end = total
	}
	
	return entries[offset:end], total, nil
}

// ListByAction lists audit entries by action with pagination
func (r *MockAuditRepository) ListByAction(ctx context.Context, action string, offset, limit int) ([]*domain.AuditEntry, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	entries := make([]*domain.AuditEntry, 0)
	for _, entry := range r.entries {
		if entry.Action == action {
			entries = append(entries, entry)
		}
	}
	
	total := len(entries)
	
	// Apply pagination
	if offset >= total {
		return []*domain.AuditEntry{}, total, nil
	}
	
	end := offset + limit
	if end > total {
		end = total
	}
	
	return entries[offset:end], total, nil
}

// Reset clears all data (for testing)
func (r *MockAuditRepository) Reset() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = make(map[uuid.UUID]*domain.AuditEntry)
	return nil
}