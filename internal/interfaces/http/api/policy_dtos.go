package api

import (
	"time"
	"github.com/google/uuid"
)

// PolicyCreateRequest represents a request to create a new policy
type PolicyCreateRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=100"`
	Description string `json:"description,omitempty" validate:"max=500"`
	Content     string `json:"content" validate:"required"`
	Type        string `json:"type" validate:"required,oneof=builtin custom managed"`
	Tags        []string `json:"tags,omitempty"`
}

// PolicyUpdateRequest represents a request to update an existing policy  
type PolicyUpdateRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=100"`
	Description string `json:"description,omitempty" validate:"max=500"`
	Content     string `json:"content" validate:"required"`
	Tags        []string `json:"tags,omitempty"`
}

// PolicyResponse represents a policy in API responses
type PolicyResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Version     int       `json:"version"`
	Type        string    `json:"type"`
	RulesCount  int       `json:"rules_count,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedBy   *string   `json:"created_by,omitempty"`
}

// PolicyTestRequest represents a request to test content against a policy
type PolicyTestRequest struct {
	Content  string                 `json:"content" validate:"required"`
	Context  map[string]interface{} `json:"context,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// PolicyValidateRequest represents a request to validate a policy
type PolicyValidateRequest struct {
	Name    string `json:"name" validate:"required"`
	Content string `json:"content" validate:"required"`
	Type    string `json:"type" validate:"required,oneof=builtin custom managed"`
}

// PolicyListResponse represents a paginated list of policies
type PolicyListResponse struct {
	Policies   []PolicyResponse `json:"policies"`
	Count      int              `json:"count"`
	TotalCount int              `json:"total_count"`
	Pagination *Pagination      `json:"pagination,omitempty"`
}


// PolicyActionResponse represents the result of a policy action (activate/deactivate/delete)
type PolicyActionResponse struct {
	PolicyID  string    `json:"policy_id"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// PolicyValidationResponse represents the result of policy validation
type PolicyValidationResponse struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// Helper functions for DTO conversions

// parseUUID safely parses a UUID string
func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// stringPtr returns a pointer to a string (for optional fields)
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// intPtr returns a pointer to an int (for optional fields)
func intPtr(i int) *int {
	return &i
}