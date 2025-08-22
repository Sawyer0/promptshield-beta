package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/contracts"
	nats "github.com/promptshield/promptshield/internal/infrastructure/messaging/nats"
)

// MockRulepackRepository for testing
type MockRulepackRepository struct {
	data map[uuid.UUID][]byte
}

func NewMockRulepackRepository() *MockRulepackRepository {
	return &MockRulepackRepository{
		data: make(map[uuid.UUID][]byte),
	}
}

func (m *MockRulepackRepository) Create(ctx context.Context, tenantID uuid.UUID, name, desc string) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *MockRulepackRepository) CreateVersion(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, status string, createdBy uuid.UUID) (uuid.UUID, error) {
	m.data[packID] = dsl
	return uuid.New(), nil
}

func (m *MockRulepackRepository) CreateVersionActivateTx(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, createdBy uuid.UUID) (uuid.UUID, error) {
	m.data[packID] = dsl
	return uuid.New(), nil
}

func (m *MockRulepackRepository) GetActive(ctx context.Context, packID uuid.UUID) (json.RawMessage, int, error) {
	if data, exists := m.data[packID]; exists {
		return data, 1, nil
	}
	return nil, 0, assert.AnError
}

func (m *MockRulepackRepository) Activate(ctx context.Context, packID, versionID uuid.UUID) error {
	return nil
}

func (m *MockRulepackRepository) GetVersion(ctx context.Context, packID uuid.UUID, version int) (json.RawMessage, string, error) {
	if data, exists := m.data[packID]; exists {
		return data, "approved", nil
	}
	return nil, "", assert.AnError
}

func (m *MockRulepackRepository) GetLatestVersion(ctx context.Context, packID uuid.UUID) (uuid.UUID, int, error) {
	return uuid.New(), 1, nil
}

func (m *MockRulepackRepository) ActivateLatest(ctx context.Context, packID uuid.UUID) error {
	return nil
}

func (m *MockRulepackRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]contracts.RulepackInfo, error) {
	return []contracts.RulepackInfo{}, nil
}

func (m *MockRulepackRepository) Delete(ctx context.Context, packID uuid.UUID) error {
	delete(m.data, packID)
	return nil
}

func (m *MockRulepackRepository) PurgeOldVersions(ctx context.Context, packID uuid.UUID, keep int) error {
	return nil
}

// FuzzHTTPRulepackEndpoints tests HTTP endpoints against malicious inputs
func FuzzHTTPRulepackEndpoints(f *testing.F) {
	// Setup test environment
	setupTestServer := func() *httptest.Server {
		repo := NewMockRulepackRepository()
		publisher, _ := nats.NewPublisher("")
		service := services.RulepackServiceCstor(repo, publisher)

		options := Options{
			RulepackService: service,
			AdminToken:      "test-admin-token",
		}

		router := NewMux(options)
		return httptest.NewServer(router)
	}

	// Seed corpus with various attack vectors
	seeds := [][]byte{
		// Valid baseline
		[]byte(`{"metadata": {"name": "test"}, "rules": []}`),

		// HTTP parameter pollution
		[]byte(`{"metadata": {"name": "test&name=evil"}, "rules": []}`),

		// Header injection attempts
		[]byte("GET /rulepacks HTTP/1.1\r\nHost: evil.com\r\n\r\n"),

		// JSON injection
		[]byte(`{"metadata": {"name": "test\"; DROP TABLE rulepacks; --"}, "rules": []}`),

		// XSS payloads
		[]byte(`{"metadata": {"name": "<script>alert('xss')</script>"}, "rules": []}`),

		// SQL injection patterns
		[]byte(`{"metadata": {"name": "'; DELETE FROM rulepacks; --"}, "rules": []}`),

		// Path traversal
		[]byte(`{"metadata": {"name": "../../../etc/passwd"}, "rules": []}`),

		// Large payloads
		[]byte(`{"metadata": {"name": "` + strings.Repeat("A", 100000) + `"}, "rules": []}`),

		// Binary content
		{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD, 0xFC},

		// Unicode/encoding attacks
		[]byte(`{"metadata": {"name": "test\u0000null"}, "rules": []}`),
		[]byte(`{"metadata": {"name": "test\u202e"}, "rules": []}`),  // RTL override
		[]byte(`{"metadata": {"name": "test\uFEFF"}, "rules": []}`),  // BOM

		// Content-Type confusion
		[]byte("metadata:\n  name: test\nrules: []"), // YAML content

		// Malformed JSON
		[]byte(`{"metadata": {"name": "test"`), // Truncated
		[]byte(`{metadata: {name: "test"}}`),   // Unquoted keys
		[]byte(`{"metadata": {"name": test}}`), // Unquoted values

		// Empty/minimal
		[]byte(""),
		[]byte("{}"),
		[]byte("null"),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, payload []byte) {
		server := setupTestServer()
		defer server.Close()

		// Test POST /rulepacks endpoint
		testPOSTRulepacks := func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("POST /rulepacks panicked: %v", r)
				}
			}()

			req, err := http.NewRequest("POST", server.URL+"/rulepacks", bytes.NewReader(payload))
			if err != nil {
				return
			}

			req.Header.Set("Authorization", "Bearer test-admin-token")
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", uuid.New().String())

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			// Read response
			body, _ := io.ReadAll(resp.Body)

			// Should return valid HTTP status
			if resp.StatusCode < 100 || resp.StatusCode >= 600 {
				t.Errorf("Invalid HTTP status: %d", resp.StatusCode)
			}

			// Response should be valid JSON (for non-error cases)
			if resp.StatusCode == 200 || resp.StatusCode == 201 {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Errorf("Response not valid JSON: %v", err)
				}
			}

			// Error responses should not expose internal details
			if resp.StatusCode >= 400 {
				bodyStr := string(body)
				if strings.Contains(bodyStr, "panic") ||
					strings.Contains(bodyStr, "runtime error") ||
					strings.Contains(bodyStr, "goroutine") {
					t.Errorf("Error response exposes internals: %s", bodyStr)
				}
			}
		}

		// Test POST /rulepacks/validate endpoint
		testPOSTValidate := func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("POST /rulepacks/validate panicked: %v", r)
				}
			}()

			req, err := http.NewRequest("POST", server.URL+"/rulepacks/validate", bytes.NewReader(payload))
			if err != nil {
				return
			}

			req.Header.Set("Authorization", "Bearer test-admin-token")
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)

			// Validate endpoint should always return 200 with validation result
			if resp.StatusCode == 200 {
				var result map[string]interface{}
				if err := json.Unmarshal(body, &result); err != nil {
					t.Errorf("Validation response not valid JSON: %v", err)
				}

				// Should have required fields
				if _, exists := result["valid"]; !exists {
					t.Error("Validation response missing 'valid' field")
				}
			}
		}

		testPOSTRulepacks()
		testPOSTValidate()
	})
}

// FuzzHTTPMultipartUpload tests multipart form uploads
func FuzzHTTPMultipartUpload(f *testing.F) {
	setupTestServer := func() *httptest.Server {
		repo := NewMockRulepackRepository()
		publisher, _ := nats.NewPublisher("")
		service := services.RulepackServiceCstor(repo, publisher)

		options := Options{
			RulepackService: service,
			AdminToken:      "test-admin-token",
		}

		router := NewMux(options)
		return httptest.NewServer(router)
	}

	// Seed with various payloads
	seeds := [][]byte{
		[]byte(`{"metadata": {"name": "test"}, "rules": []}`),
		[]byte("metadata:\n  name: test\nrules: []"),
		[]byte(`malformed content`),
		[]byte(strings.Repeat("A", 10000)),
		{0x00, 0x01, 0x02, 0x03},
		[]byte(""),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, fileContent []byte) {
		server := setupTestServer()
		defer server.Close()

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Multipart upload panicked: %v", r)
			}
		}()

		// Create multipart form
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		// Add file field
		fileWriter, err := writer.CreateFormFile("file", "test.yaml")
		if err != nil {
			return
		}

		_, err = fileWriter.Write(fileContent)
		if err != nil {
			return
		}

		writer.Close()

		// Create request
		req, err := http.NewRequest("POST", server.URL+"/rulepacks", &buf)
		if err != nil {
			return
		}

		req.Header.Set("Authorization", "Bearer test-admin-token")
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Idempotency-Key", uuid.New().String())

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		// Should handle gracefully
		if resp.StatusCode < 100 || resp.StatusCode >= 600 {
			t.Errorf("Invalid HTTP status for multipart: %d", resp.StatusCode)
		}

		// Error messages should not expose internals
		if resp.StatusCode >= 400 {
			bodyStr := string(body)
			if strings.Contains(bodyStr, "panic") ||
				strings.Contains(bodyStr, "runtime error") {
				t.Errorf("Multipart error exposes internals: %s", bodyStr)
			}
		}
	})
}

// FuzzHTTPHeaders tests various HTTP header attacks
func FuzzHTTPHeaders(f *testing.F) {
	setupTestServer := func() *httptest.Server {
		repo := NewMockRulepackRepository()
		publisher, _ := nats.NewPublisher("")
		service := services.RulepackServiceCstor(repo, publisher)

		options := Options{
			RulepackService: service,
			AdminToken:      "test-admin-token",
		}

		router := NewMux(options)
		return httptest.NewServer(router)
	}

	// Seed with header injection attacks
	seeds := []string{
		// Normal headers
		"application/json",
		"Bearer test-admin-token",
		"gzip, deflate",

		// Header injection attempts
		"application/json\r\nX-Injected: evil",
		"Bearer token\r\n\r\nGET /evil HTTP/1.1",
		"application/json\nSet-Cookie: evil=true",

		// Encoding attacks
		"application/json; charset=utf-8",
		"application/json; charset=iso-8859-1",
		"application/json; charset=windows-1252",

		// Large headers
		strings.Repeat("A", 10000),
		strings.Repeat("X", 100000),

		// Unicode in headers
		"application/json 🔥",
		"Bearer test\u0000token",
		"gzip\u202e",

		// Binary data
		string([]byte{0x00, 0x01, 0x02}),
		string([]byte{0xFF, 0xFE, 0xFD}),

		// Empty/whitespace
		"",
		"   ",
		"\t\t",
		"\r\n",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, headerValue string) {
		server := setupTestServer()
		defer server.Close()

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Header fuzzing panicked: %v", r)
			}
		}()

		// Test various headers
		headers := map[string]string{
			"Content-Type":    headerValue,
			"Authorization":   "Bearer " + headerValue,
			"Accept":          headerValue,
			"User-Agent":      headerValue,
			"X-Custom":        headerValue,
			"Idempotency-Key": headerValue,
		}

		payload := []byte(`{"metadata": {"name": "test"}, "rules": []}`)

		for headerName, headerVal := range headers {
			req, err := http.NewRequest("POST", server.URL+"/rulepacks", bytes.NewReader(payload))
			if err != nil {
				continue
			}

			// Set standard required headers
			req.Header.Set("Authorization", "Bearer test-admin-token")
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", uuid.New().String())

			// Override with fuzzed header
			if isValidHeaderValue(headerVal) {
				req.Header.Set(headerName, headerVal)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				continue
			}

			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			// Should not expose header injection in response
			bodyStr := string(body)
			if strings.Contains(bodyStr, "\r\n") ||
				strings.Contains(bodyStr, "X-Injected") ||
				strings.Contains(bodyStr, "Set-Cookie") {
				t.Errorf("Response may contain header injection: %s", bodyStr)
			}
		}
	})
}

// PropertyTest_HTTPSecurity tests HTTP security properties
func TestProperty_HTTPSecurity(t *testing.T) {
	repo := NewMockRulepackRepository()
	publisher, _ := nats.NewPublisher("")
	service := services.RulepackServiceCstor(repo, publisher)

	options := Options{
		RulepackService: service,
		AdminToken:      "test-admin-token",
	}

	router := NewMux(options)
	server := httptest.NewServer(router)
	defer server.Close()

	t.Run("RequiresAuthentication", func(t *testing.T) {
		// Requests without auth should be rejected
		req, _ := http.NewRequest("POST", server.URL+"/rulepacks", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("RejectsInvalidContentType", func(t *testing.T) {
		req, _ := http.NewRequest("POST", server.URL+"/rulepacks", strings.NewReader("test"))
		req.Header.Set("Authorization", "Bearer test-admin-token")
		req.Header.Set("Content-Type", "text/plain")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("IdempotencyWorks", func(t *testing.T) {
		payload := `{"metadata": {"name": "idempotent-test"}, "rules": []}`
		idempotencyKey := uuid.New().String()

		// First request
		req1, _ := http.NewRequest("POST", server.URL+"/rulepacks", strings.NewReader(payload))
		req1.Header.Set("Authorization", "Bearer test-admin-token")
		req1.Header.Set("Content-Type", "application/json")
		req1.Header.Set("Idempotency-Key", idempotencyKey)

		resp1, err := http.DefaultClient.Do(req1)
		assert.NoError(t, err)
		defer resp1.Body.Close()

		// Second request with same key
		req2, _ := http.NewRequest("POST", server.URL+"/rulepacks", strings.NewReader(payload))
		req2.Header.Set("Authorization", "Bearer test-admin-token")
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Idempotency-Key", idempotencyKey)

		resp2, err := http.DefaultClient.Do(req2)
		assert.NoError(t, err)
		defer resp2.Body.Close()

		// Should have idempotency cache header
		assert.Equal(t, "HIT", resp2.Header.Get("X-Idempotency-Cache"))
		assert.Equal(t, http.StatusOK, resp2.StatusCode) // Changed to 200 for cached response
	})

	t.Run("HandlesLargePayloads", func(t *testing.T) {
		// Test size limits
		largePayload := `{"metadata": {"name": "` + strings.Repeat("A", 2*1024*1024) + `"}, "rules": []}`

		req, _ := http.NewRequest("POST", server.URL+"/rulepacks", strings.NewReader(largePayload))
		req.Header.Set("Authorization", "Bearer test-admin-token")
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Should reject large payloads
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// Helper functions

func isValidHeaderValue(value string) bool {
	// Basic validation - header values should not contain control characters
	for _, r := range value {
		if r < ' ' || r == 0x7F {
			if r != '\t' { // Tab is allowed
				return false
			}
		}
		if !unicode.IsPrint(r) && r != '\t' {
			return false
		}
	}
	return true
}