package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/domain"
)

// MockAPITokenRepository provides a simple in-memory implementation
type MockAPITokenRepository struct {
	mu     sync.RWMutex
	tokens map[uuid.UUID]*domain.APIToken
}

// NewMockAPITokenRepository creates a new mock API token repository
func NewMockAPITokenRepository() *MockAPITokenRepository {
	return &MockAPITokenRepository{
		tokens: make(map[uuid.UUID]*domain.APIToken),
	}
}

// Create creates a new API token
func (r *MockAPITokenRepository) Create(ctx context.Context, token *domain.APIToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens[token.ID] = token
	return nil
}

// Get retrieves a token by ID
func (r *MockAPITokenRepository) Get(ctx context.Context, id uuid.UUID) (*domain.APIToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if token, exists := r.tokens[id]; exists {
		return token, nil
	}
	return nil, fmt.Errorf("token not found")
}

// GetByHash retrieves a token by its hash
func (r *MockAPITokenRepository) GetByHash(ctx context.Context, hashedToken string) (*domain.APIToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	for _, token := range r.tokens {
		if token.TokenHash == hashedToken {
			return token, nil
		}
	}
	return nil, fmt.Errorf("token not found")
}

// ListByTenant retrieves tokens by tenant ID
func (r *MockAPITokenRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.APIToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	tokens := make([]*domain.APIToken, 0)
	for _, token := range r.tokens {
		if token.TenantID == tenantID {
			tokens = append(tokens, token)
		}
	}
	return tokens, nil
}

// Update updates an existing token
func (r *MockAPITokenRepository) Update(ctx context.Context, token *domain.APIToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens[token.ID] = token
	return nil
}

// Delete deletes a token
func (r *MockAPITokenRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tokens, id)
	return nil
}

// Rotate rotates a token (generates new token value)
func (r *MockAPITokenRepository) Rotate(ctx context.Context, id uuid.UUID) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if token, exists := r.tokens[id]; exists {
		// Generate a new token value (simplified for mock)
		newToken := fmt.Sprintf("ps_token_%d", time.Now().UnixNano())
		token.TokenHash = fmt.Sprintf("hashed_%s", newToken)
		// Note: APIToken doesn't have UpdatedAt field, so we skip that
		return newToken, nil
	}
	return "", fmt.Errorf("token not found")
}

// UpdateLastUsed updates the last used timestamp
func (r *MockAPITokenRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if token, exists := r.tokens[id]; exists {
		now := time.Now()
		token.LastUsed = &now
		return nil
	}
	return fmt.Errorf("token not found")
}

// Revoke revokes a token
func (r *MockAPITokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if token, exists := r.tokens[id]; exists {
		token.RevokedAt = &time.Time{}
		now := time.Now()
		token.RevokedAt = &now
		return nil
	}
	return fmt.Errorf("token not found")
}

// DeleteExpired deletes expired tokens
func (r *MockAPITokenRepository) DeleteExpired(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	now := time.Now()
	for id, token := range r.tokens {
		if token.ExpiresAt != nil && token.ExpiresAt.Before(now) {
			delete(r.tokens, id)
		}
	}
	return nil
}

// Reset clears all data (for testing)
func (r *MockAPITokenRepository) Reset() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens = make(map[uuid.UUID]*domain.APIToken)
	return nil
}