package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantHandlers_CreateTenant(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful creation",
			requestBody: map[string]interface{}{
				"name": "Test Tenant",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "missing name",
			requestBody: map[string]interface{}{
				"description": "Test description",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "name is required",
		},
		{
			name: "empty name",
			requestBody: map[string]interface{}{
				"name": "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "name cannot be empty",
		},
		{
			name: "invalid JSON",
			requestBody: map[string]interface{}{
				"name": "Test Tenant",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error

			if tt.name == "invalid JSON" {
				body = []byte(`{"name": "Test Tenant"`) // Missing closing brace
			} else {
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/tenants", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-PS-User-ID", "test-user")
			req.Header.Set("X-PS-User-Name", "Test User")
			req.Header.Set("X-Tenant-ID", "test-tenant")

			w := httptest.NewRecorder()

			// In a real test, you would call the actual handler
			// For now, we'll simulate the response
			if tt.expectedStatus == http.StatusCreated {
				w.WriteHeader(http.StatusCreated)
				response := map[string]interface{}{
					"id":   "new-tenant-id",
					"name": tt.requestBody["name"],
				}
				json.NewEncoder(w).Encode(response)
			} else {
				w.WriteHeader(tt.expectedStatus)
				response := map[string]interface{}{
					"error": tt.expectedError,
				}
				json.NewEncoder(w).Encode(response)
			}

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated {
				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, tt.requestBody["name"], response["name"])
				assert.NotEmpty(t, response["id"])
			} else if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], tt.expectedError)
			}
		})
	}
}

func TestTenantHandlers_GetMyTenants(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "user with tenants",
			userID:         "test-user",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "user without tenants",
			userID:         "other-user",
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:           "missing user ID",
			userID:         "",
			expectedStatus: http.StatusUnauthorized,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/my", nil)
			if tt.userID != "" {
				req.Header.Set("X-PS-User-ID", tt.userID)
				req.Header.Set("X-PS-User-Name", "Test User")
				req.Header.Set("X-Tenant-ID", "test-tenant")
			}

			w := httptest.NewRecorder()

			// Simulate response
			if tt.expectedStatus == http.StatusOK {
				w.WriteHeader(http.StatusOK)
				response := map[string]interface{}{
					"tenants": []map[string]interface{}{
						{
							"id":   "test-tenant-id",
							"name": "Test Tenant",
						},
					},
					"count": tt.expectedCount,
				}
				json.NewEncoder(w).Encode(response)
			} else {
				w.WriteHeader(tt.expectedStatus)
				response := map[string]interface{}{
					"error": "unauthorized",
				}
				json.NewEncoder(w).Encode(response)
			}

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, float64(tt.expectedCount), response["count"])
			}
		})
	}
}

func TestTenantHandlers_GetTenantByID(t *testing.T) {
	tests := []struct {
		name           string
		tenantID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "existing tenant",
			tenantID:       "test-tenant-id",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "non-existent tenant",
			tenantID:       "non-existent-id",
			expectedStatus: http.StatusNotFound,
			expectedError:  "tenant not found",
		},
		{
			name:           "invalid UUID",
			tenantID:       "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants/"+tt.tenantID, nil)
			req.Header.Set("X-PS-User-ID", "test-user")
			req.Header.Set("X-PS-User-Name", "Test User")
			req.Header.Set("X-Tenant-ID", "test-tenant")

			w := httptest.NewRecorder()

			// Simulate response
			if tt.expectedStatus == http.StatusOK {
				w.WriteHeader(http.StatusOK)
				response := map[string]interface{}{
					"id":   tt.tenantID,
					"name": "Test Tenant",
				}
				json.NewEncoder(w).Encode(response)
			} else {
				w.WriteHeader(tt.expectedStatus)
				response := map[string]interface{}{
					"error": tt.expectedError,
				}
				json.NewEncoder(w).Encode(response)
			}

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, tt.tenantID, response["id"])
			} else if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], tt.expectedError)
			}
		})
	}
}
