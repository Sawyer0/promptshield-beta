package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

// MockSettingsRepository provides a simple in-memory implementation
type MockSettingsRepository struct {
	mu       sync.RWMutex
	settings *domain.PlatformSettings
	history  []*domain.PlatformSettings
}

// NewMockSettingsRepository creates a new mock settings repository
func NewMockSettingsRepository() *MockSettingsRepository {
	return &MockSettingsRepository{
		settings: nil,
		history:  make([]*domain.PlatformSettings, 0),
	}
}

// Get retrieves the current platform settings
func (r *MockSettingsRepository) Get(ctx context.Context) (*domain.PlatformSettings, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if r.settings == nil {
		return nil, fmt.Errorf("settings not found")
	}
	return r.settings, nil
}

// Update updates the platform settings
func (r *MockSettingsRepository) Update(ctx context.Context, settings interface{}) (*domain.PlatformSettings, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Store old settings in history if they exist
	if r.settings != nil {
		r.history = append(r.history, r.settings)
	}
	
	// Create new settings (simplified for mock)
	newSettings := &domain.PlatformSettings{
		ID:       uuid.New(),
		Settings: []byte(fmt.Sprintf("%v", settings)),
	}
	
	r.settings = newSettings
	return newSettings, nil
}

// GetHistory retrieves settings history with pagination
func (r *MockSettingsRepository) GetHistory(ctx context.Context, limit int, offset int) ([]*domain.PlatformSettings, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	total := len(r.history)
	
	// Apply pagination
	if offset >= total {
		return []*domain.PlatformSettings{}, total, nil
	}
	
	end := offset + limit
	if end > total {
		end = total
	}
	
	return r.history[offset:end], total, nil
}

// Delete deletes the current settings
func (r *MockSettingsRepository) Delete(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.settings = nil
	return nil
}

// Backup creates a backup of current settings
func (r *MockSettingsRepository) Backup(ctx context.Context) ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if r.settings == nil {
		return nil, fmt.Errorf("no settings to backup")
	}
	
	return r.settings.Settings, nil
}

// Restore restores settings from backup data
func (r *MockSettingsRepository) Restore(ctx context.Context, backupData []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Store current settings in history if they exist
	if r.settings != nil {
		r.history = append(r.history, r.settings)
	}
	
	// Create new settings from backup
	r.settings = &domain.PlatformSettings{
		ID:       uuid.New(),
		Settings: backupData,
	}
	
	return nil
}

// ValidateConnection validates the repository connection
func (r *MockSettingsRepository) ValidateConnection(ctx context.Context) error {
	// Mock implementation always returns success
	return nil
}

// Reset clears all data (for testing)
func (r *MockSettingsRepository) Reset() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.settings = nil
	r.history = make([]*domain.PlatformSettings, 0)
	return nil
}