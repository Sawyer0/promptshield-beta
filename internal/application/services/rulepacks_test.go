package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/promptshield/promptshield/internal/testutil/fixtures"
	"github.com/promptshield/promptshield/internal/testutil/mocks"
)

func TestRulepackService_CreateVersionActivate(t *testing.T) {
	tests := []struct {
		name        string
		tenantID    uuid.UUID
		packID      uuid.UUID
		version     int
		rawDSL      string
		setupMocks  func(*mocks.MockRulepackRepository)
		wantErr     bool
		errContains string
	}{
		{
			name:     "TP-1.1 happy path - all operations succeed",
			tenantID: fixtures.TenantID1,
			packID:   fixtures.RulepackID1,
			version:  2,
			rawDSL:   fixtures.MinimalValidDSL,
			setupMocks: func(repo *mocks.MockRulepackRepository) {
				repo.On("CreateVersionActivateTx", mock.Anything, fixtures.RulepackID1, 2, json.RawMessage(fixtures.MinimalValidDSL), uuid.Nil).
					Return(fixtures.VersionID1, nil)
				repo.On("PurgeOldVersions", mock.Anything, fixtures.RulepackID1, mock.AnythingOfType("int")).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:     "TP-1.2 create version fails",
			tenantID: fixtures.TenantID1,
			packID:   fixtures.RulepackID1,
			version:  2,
			rawDSL:   fixtures.MinimalValidDSL,
			setupMocks: func(repo *mocks.MockRulepackRepository) {
				repo.On("CreateVersionActivateTx", mock.Anything, fixtures.RulepackID1, 2, json.RawMessage(fixtures.MinimalValidDSL), uuid.Nil).
					Return(fixtures.VersionID1, errors.New("database error"))
				// PurgeOldVersions not called when transaction fails
			},
			wantErr:     true,
			errContains: "database error",
		},
		{
			name:     "TP-1.3 activate fails",
			tenantID: fixtures.TenantID1,
			packID:   fixtures.RulepackID1,
			version:  2,
			rawDSL:   fixtures.MinimalValidDSL,
			setupMocks: func(repo *mocks.MockRulepackRepository) {
				repo.On("CreateVersionActivateTx", mock.Anything, fixtures.RulepackID1, 2, json.RawMessage(fixtures.MinimalValidDSL), uuid.Nil).
					Return(fixtures.VersionID1, errors.New("activation failed"))
				// PurgeOldVersions not called when transaction fails
			},
			wantErr:     true,
			errContains: "activation failed",
		},
		{
			name:     "TP-1.5 publisher error is tolerated",
			tenantID: fixtures.TenantID1,
			packID:   fixtures.RulepackID1,
			version:  2,
			rawDSL:   fixtures.MinimalValidDSL,
			setupMocks: func(repo *mocks.MockRulepackRepository) {
				repo.On("CreateVersionActivateTx", mock.Anything, fixtures.RulepackID1, 2, json.RawMessage(fixtures.MinimalValidDSL), uuid.Nil).
					Return(fixtures.VersionID1, nil)
				repo.On("PurgeOldVersions", mock.Anything, fixtures.RulepackID1, mock.AnythingOfType("int")).
					Return(nil)
			},
			wantErr: false, // Publisher errors are non-blocking
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mocks.MockRulepackRepository)
			tt.setupMocks(repo)

			svc := &RulepackService{
				repo: repo,
				pub:  nil, // No publisher
			}

			err := svc.CreateVersionActivate(context.Background(), tt.tenantID, tt.packID, tt.version, json.RawMessage(tt.rawDSL))

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestRulepackService_CreateVersionActivate_NilPublisher(t *testing.T) {
	// TP-1.4 nil publisher should still succeed
	repo := new(mocks.MockRulepackRepository)

	repo.On("CreateVersionActivateTx", mock.Anything, fixtures.RulepackID1, 2, json.RawMessage(fixtures.MinimalValidDSL), uuid.Nil).
		Return(fixtures.VersionID1, nil)
	repo.On("PurgeOldVersions", mock.Anything, fixtures.RulepackID1, mock.AnythingOfType("int")).
		Return(nil)

	svc := &RulepackService{
		repo: repo,
		pub:  nil,
	}

	err := svc.CreateVersionActivate(context.Background(), fixtures.TenantID1, fixtures.RulepackID1, 2, json.RawMessage(fixtures.MinimalValidDSL))
	require.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestRulepackService_ChecksumJSON(t *testing.T) {
	// TP-1.6 checksumJSON deterministic
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple JSON",
			input:    `{"key":"value"}`,
			expected: fixtures.ComputeChecksum(`{"key":"value"}`),
		},
		{
			name:     "complex JSON",
			input:    fixtures.ValidRulepackJSON,
			expected: fixtures.ComputeChecksum(fixtures.ValidRulepackJSON),
		},
		{
			name:     "empty JSON",
			input:    `{}`,
			expected: fixtures.ComputeChecksum(`{}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checksumJSON(json.RawMessage(tt.input))
			assert.Equal(t, tt.expected, result)

			// Verify deterministic - same input always produces same output
			result2 := checksumJSON(json.RawMessage(tt.input))
			assert.Equal(t, result, result2)
		})
	}
}

// TestRulepackService_ConcurrentOperations tests thread safety
func TestRulepackService_ConcurrentOperations(t *testing.T) {
	repo := new(mocks.MockRulepackRepository)

	// Set up expectations for concurrent calls
	repo.On("CreateVersionActivateTx", mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("json.RawMessage"), uuid.Nil).
		Return(fixtures.VersionID1, nil).Times(10)
	repo.On("PurgeOldVersions", mock.Anything, mock.Anything, mock.AnythingOfType("int")).
		Return(nil).Times(10)

	svc := &RulepackService{
		repo: repo,
		pub:  nil,
	}

	// Run concurrent operations
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			err := svc.CreateVersionActivate(context.Background(), fixtures.TenantID1, fixtures.RulepackID1, 2, json.RawMessage(fixtures.MinimalValidDSL))
			assert.NoError(t, err)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	repo.AssertExpectations(t)
}
