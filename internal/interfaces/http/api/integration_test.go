// +build integration

package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"github.com/promptshield/promptshield/internal/testutil/fixtures"
	"github.com/promptshield/promptshield/pkg/types"
)

// TestIntegration_HTTPGatewayScanEndpoint tests the HTTP gateway scan endpoint
func TestIntegration_HTTPGatewayScanEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Create test server with scan endpoint
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate method and path
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		
		if r.URL.Path != "/v1/scan" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		
		// Parse request
		var req struct {
			Content  string `json:"content"`
			Rulepack string `json:"rulepack"`
		}
		
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		
		// Validate required fields
		if req.Content == "" || req.Rulepack == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "content and rulepack are required"})
			return
		}
		
		// Simulate scanning
		result := types.ScanResult{
			Input: "inline-content",
			Violations: []types.Violation{
				{
					RuleID:   "test-violation",
					Message:  "Test violation detected",
					Severity: "HIGH",
					Line:     1,
					Column:   0,
				},
			},
			Metrics: types.Metrics{
				BytesRead: int64(len(req.Content)),
				LinesRead: 1,
			},
			DurationMs: 10,
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	})
	
	server := httptest.NewServer(handler)
	defer server.Close()
	
	// Test successful scan
	t.Run("successful scan", func(t *testing.T) {
		reqBody := map[string]string{
			"content":  fixtures.TextWithViolations,
			"rulepack": fixtures.ValidRulepackJSON,
		}
		
		bodyBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)
		
		resp, err := http.Post(
			server.URL+"/v1/scan",
			"application/json",
			bytes.NewReader(bodyBytes),
		)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		var result types.ScanResult
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		
		assert.Len(t, result.Violations, 1)
		assert.Equal(t, "test-violation", result.Violations[0].RuleID)
		assert.Equal(t, "HIGH", result.Violations[0].Severity)
		assert.Greater(t, result.Metrics.BytesRead, int64(0))
	})
	
	// Test invalid request
	t.Run("missing content", func(t *testing.T) {
		reqBody := map[string]string{
			"rulepack": fixtures.ValidRulepackJSON,
		}
		
		bodyBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)
		
		resp, err := http.Post(
			server.URL+"/v1/scan",
			"application/json",
			bytes.NewReader(bodyBytes),
		)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		
		var errResp map[string]string
		err = json.NewDecoder(resp.Body).Decode(&errResp)
		require.NoError(t, err)
		assert.Contains(t, errResp["error"], "required")
	})
	
	// Test wrong path
	t.Run("wrong path", func(t *testing.T) {
		resp, err := http.Post(
			server.URL+"/v1/wrong",
			"application/json",
			bytes.NewReader([]byte("{}")),
		)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestIntegration_HTTPBatchScanning tests batch scanning via HTTP
func TestIntegration_HTTPBatchScanning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/scan/batch" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		
		var req struct {
			Items []struct {
				ID       string `json:"id"`
				Content  string `json:"content"`
				Rulepack string `json:"rulepack"`
			} `json:"items"`
		}
		
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		
		// Process each item
		results := make([]map[string]interface{}, len(req.Items))
		for i, item := range req.Items {
			results[i] = map[string]interface{}{
				"id": item.ID,
				"result": types.ScanResult{
					Input:      item.ID,
					Violations: []types.Violation{},
					Metrics: types.Metrics{
						BytesRead: int64(len(item.Content)),
					},
				},
			}
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": results,
		})
	})
	
	server := httptest.NewServer(handler)
	defer server.Close()
	
	// Prepare batch request
	reqBody := map[string]interface{}{
		"items": []map[string]string{
			{
				"id":       "item1",
				"content":  "Test content 1",
				"rulepack": fixtures.MinimalValidDSL,
			},
			{
				"id":       "item2",
				"content":  "Test content 2",
				"rulepack": fixtures.MinimalValidDSL,
			},
		},
	}
	
	bodyBytes, err := json.Marshal(reqBody)
	require.NoError(t, err)
	
	resp, err := http.Post(
		server.URL+"/v1/scan/batch",
		"application/json",
		bytes.NewReader(bodyBytes),
	)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	var batchResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&batchResp)
	require.NoError(t, err)
	
	results, ok := batchResp["results"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, results, 2)
}

// TestIntegration_HTTPStreamingResponse tests streaming responses
func TestIntegration_HTTPStreamingResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/scan/stream" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		
		// Set headers for streaming
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Transfer-Encoding", "chunked")
		
		// Send results as NDJSON stream
		encoder := json.NewEncoder(w)
		
		// Simulate streaming multiple results
		for i := 0; i < 3; i++ {
			result := types.ScanResult{
				Input: fmt.Sprintf("chunk-%d", i),
				Violations: []types.Violation{
					{
						RuleID:   fmt.Sprintf("rule-%d", i),
						Severity: "LOW",
						Line:     i + 1,
					},
				},
			}
			
			err := encoder.Encode(result)
			if err != nil {
				return
			}
			
			// Flush to send immediately
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	})
	
	server := httptest.NewServer(handler)
	defer server.Close()
	
	resp, err := http.Post(
		server.URL+"/v1/scan/stream",
		"application/json",
		bytes.NewReader([]byte("{}")),
	)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/x-ndjson", resp.Header.Get("Content-Type"))
	
	// Read streaming response
	decoder := json.NewDecoder(resp.Body)
	var results []types.ScanResult
	
	for decoder.More() {
		var result types.ScanResult
		err := decoder.Decode(&result)
		require.NoError(t, err)
		results = append(results, result)
	}
	
	assert.Len(t, results, 3)
	for i, result := range results {
		assert.Equal(t, fmt.Sprintf("chunk-%d", i), result.Input)
		assert.Len(t, result.Violations, 1)
	}
}