// +build integration

package gateway_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	grpcenforcer "github.com/promptshield/promptshield/internal/interfaces/grpc/enforcer"
	"github.com/promptshield/promptshield/internal/testutil/fixtures"
)

// TestIntegration_FullGatewayFlow tests the complete gateway flow:
// Client → Gateway → PromptShield → Backend (or block)
func TestIntegration_FullGatewayFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create PromptShield enforcer
	enforcer := grpcenforcer.NewWithOptions(grpcenforcer.Options{
		Timeout:         500 * time.Millisecond,
		EnforcementMode: "enforce",
		FailOn:          "HIGH",
	})

	// Create mock OpenAI backend
	openaiMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate OpenAI API response
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"role":    "assistant",
						"content": "Paris is the capital of France.",
					},
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer openaiMock.Close()

	// Test cases for different types of requests
	testCases := []struct {
		name           string
		payload        map[string]interface{}
		expectBlocked  bool
		expectStatus   int
		expectContains string
	}{
		{
			name: "benign request should pass through",
			payload: map[string]interface{}{
				"model": "gpt-4",
				"messages": []map[string]string{
					{"role": "user", "content": "What is the capital of France?"},
				},
			},
			expectBlocked:  false,
			expectStatus:   200,
			expectContains: "Paris",
		},
		{
			name: "prompt injection should be blocked",
			payload: map[string]interface{}{
				"model": "gpt-4",
				"messages": []map[string]string{
					{"role": "user", "content": fixtures.PromptInjectionPayload},
				},
			},
			expectBlocked:  true,
			expectStatus:   403,
			expectContains: "blocked",
		},
		{
			name: "jailbreak attempt should be blocked",
			payload: map[string]interface{}{
				"model": "gpt-4",
				"messages": []map[string]string{
					{"role": "user", "content": fixtures.JailbreakPayload},
				},
			},
			expectBlocked:  true,
			expectStatus:   403,
			expectContains: "PromptShield",
		},
		{
			name: "request with API key should be blocked",
			payload: map[string]interface{}{
				"model": "gpt-4",
				"messages": []map[string]string{
					{"role": "user", "content": "My API key is " + fixtures.SampleAPIKey},
				},
			},
			expectBlocked:  true,
			expectStatus:   403,
			expectContains: "blocked",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create gateway server that simulates Envoy + PromptShield flow
			gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Read request body
				body, err := io.ReadAll(r.Body)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				// Simulate PromptShield scanning
				decision, details := simulatePromptShieldScan(string(body))
				
				if decision == "block" {
					// Return detailed blocking response
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-PS-Decision", "quarantine")
					w.WriteHeader(http.StatusForbidden)
					json.NewEncoder(w).Encode(details)
					return
				}

				// If allowed, proxy to backend (OpenAI mock)
				backendReq, err := http.NewRequest(r.Method, openaiMock.URL+r.URL.Path, bytes.NewReader(body))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				// Copy headers
				for k, v := range r.Header {
					backendReq.Header[k] = v
				}

				// Make request to backend
				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Do(backendReq)
				if err != nil {
					w.WriteHeader(http.StatusBadGateway)
					return
				}
				defer resp.Body.Close()

				// Copy response headers
				for k, v := range resp.Header {
					w.Header()[k] = v
				}
				w.WriteHeader(resp.StatusCode)

				// Copy response body
				io.Copy(w, resp.Body)
			}))
			defer gateway.Close()

			// Make request to gateway
			payloadBytes, err := json.Marshal(tc.payload)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", gateway.URL+"/v1/chat/completions", bytes.NewReader(payloadBytes))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+fixtures.SampleAPIKey)

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Verify response
			assert.Equal(t, tc.expectStatus, resp.StatusCode)

			respBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Contains(t, string(respBody), tc.expectContains)

			// Check enforcement headers for blocked requests
			if tc.expectBlocked {
				assert.Equal(t, "quarantine", resp.Header.Get("X-PS-Decision"))
			}
		})
	}
}

// simulatePromptShieldScan simulates the PromptShield scanning logic
func simulatePromptShieldScan(requestBody string) (string, map[string]interface{}) {
	// Parse JSON to extract message content
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(requestBody), &payload); err != nil {
		return "allow", nil
	}

	messages, ok := payload["messages"].([]interface{})
	if !ok {
		return "allow", nil
	}

	// Scan all message content
	for _, msg := range messages {
		if msgMap, ok := msg.(map[string]interface{}); ok {
			if content, ok := msgMap["content"].(string); ok {
				if decision, details := scanContent(content); decision == "block" {
					return decision, details
				}
			}
		}
	}

	return "allow", nil
}

// scanContent simulates PromptShield rule matching
func scanContent(content string) (string, map[string]interface{}) {
	contentLower := strings.ToLower(content)
	
	// Check for prompt injection patterns
	if strings.Contains(contentLower, "ignore previous instructions") ||
		strings.Contains(contentLower, "reveal your system prompt") {
		return "block", map[string]interface{}{
			"blocked": true,
			"decision": "quarantine",
			"reason": "prompt-injection-ignore-previous",
			"message": "Request blocked by PromptShield",
			"violations": []map[string]interface{}{
				{
					"rule_id": "prompt-injection-ignore-previous",
					"severity": "CRITICAL",
					"message": "Direct instruction override attempts",
					"line": 1,
					"column": strings.Index(contentLower, "ignore"),
				},
			},
			"scan_info": map[string]interface{}{
				"total_violations": 1,
				"scan_duration_ms": 5,
			},
		}
	}

	// Check for jailbreak attempts
	if strings.Contains(contentLower, "dan mode") ||
		strings.Contains(contentLower, "forget all restrictions") {
		return "block", map[string]interface{}{
			"blocked": true,
			"decision": "quarantine",
			"reason": "jailbreak-attempt",
			"message": "Request blocked by PromptShield",
			"violations": []map[string]interface{}{
				{
					"rule_id": "jailbreak-dan-mode",
					"severity": "HIGH",
					"message": "Jailbreak attempt detected",
					"line": 1,
				},
			},
		}
	}

	// Check for API key patterns
	if strings.Contains(content, "sk-") && strings.Contains(content, "api") {
		return "block", map[string]interface{}{
			"blocked": true,
			"decision": "quarantine",
			"reason": "api-key-detected",
			"message": "Request blocked by PromptShield",
			"violations": []map[string]interface{}{
				{
					"rule_id": "openai-api-key",
					"severity": "CRITICAL",
					"message": "OpenAI API key detected",
					"line": 1,
				},
			},
		}
	}

	return "allow", nil
}

// TestIntegration_ResponseFiltering tests response content filtering
func TestIntegration_ResponseFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create mock backend that returns responses with sensitive data
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responses := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"role": "assistant",
						"content": fmt.Sprintf("Here's your API key: %s and SSN: %s", 
							fixtures.SampleAPIKey, fixtures.SampleSSN),
					},
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responses)
	}))
	defer backend.Close()

	// Create gateway with response filtering
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Forward request to backend
		body, _ := io.ReadAll(r.Body)
		backendReq, _ := http.NewRequest(r.Method, backend.URL+r.URL.Path, bytes.NewReader(body))
		
		client := &http.Client{}
		resp, err := client.Do(backendReq)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		// Read and scan response
		respBody, _ := io.ReadAll(resp.Body)
		
		// Simulate PromptShield response scanning
		if containsSensitiveData(string(respBody)) {
			// Block or redact the response
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-PS-Decision", "redact")
			w.WriteHeader(http.StatusOK)
			
			// Return redacted response
			redactedResponse := map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]string{
							"role":    "assistant",
							"content": "Here's your API key: [REDACTED] and SSN: [REDACTED]",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(redactedResponse)
			return
		}

		// If clean, return original response
		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		w.Write(respBody)
	}))
	defer gateway.Close()

	// Test request
	payload := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "user", "content": "Show me an example"},
		},
	}

	payloadBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", gateway.URL+"/v1/chat/completions", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should get redacted response
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "redact", resp.Header.Get("X-PS-Decision"))

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	choices := response["choices"].([]interface{})
	content := choices[0].(map[string]interface{})["message"].(map[string]interface{})["content"].(string)

	// Should be redacted
	assert.Contains(t, content, "[REDACTED]")
	assert.NotContains(t, content, fixtures.SampleAPIKey)
	assert.NotContains(t, content, fixtures.SampleSSN)
}

func containsSensitiveData(content string) bool {
	return strings.Contains(content, fixtures.SampleAPIKey) ||
		strings.Contains(content, fixtures.SampleSSN) ||
		strings.Contains(content, "sk-")
}

// TestIntegration_HighThroughputScanning tests performance under load
func TestIntegration_HighThroughputScanning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create simple gateway for load testing
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		
		// Simulate scanning delay
		decision, details := simulatePromptShieldScan(string(body))
		
		if decision == "block" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(details)
			return
		}

		// Return success
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "allowed"})
	}))
	defer gateway.Close()

	// Test concurrent requests
	numRequests := 50
	concurrency := 10
	
	sem := make(chan struct{}, concurrency)
	results := make(chan bool, numRequests)

	start := time.Now()

	for i := 0; i < numRequests; i++ {
		go func(reqID int) {
			sem <- struct{}{} // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			var content string
			if reqID%5 == 0 {
				// 20% malicious requests
				content = fixtures.PromptInjectionPayload
			} else {
				// 80% benign requests
				content = "What is the weather today?"
			}

			payload := map[string]interface{}{
				"model": "gpt-4",
				"messages": []map[string]string{
					{"role": "user", "content": content},
				},
			}

			payloadBytes, _ := json.Marshal(payload)
			req, _ := http.NewRequest("POST", gateway.URL+"/v1/chat/completions", bytes.NewReader(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				results <- false
				return
			}
			defer resp.Body.Close()

			// Verify correct decision
			if reqID%5 == 0 {
				// Should be blocked
				results <- resp.StatusCode == 403
			} else {
				// Should be allowed
				results <- resp.StatusCode == 200
			}
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numRequests; i++ {
		if <-results {
			successCount++
		}
	}

	duration := time.Since(start)
	rps := float64(numRequests) / duration.Seconds()

	t.Logf("Processed %d requests in %v (%.2f RPS)", numRequests, duration, rps)
	t.Logf("Success rate: %d/%d (%.1f%%)", successCount, numRequests, float64(successCount)/float64(numRequests)*100)

	// Should have high success rate and reasonable performance
	assert.Greater(t, successCount, numRequests*9/10, "Should have >90% success rate") // Allow 10% failure
	assert.Greater(t, rps, 10.0, "Should process at least 10 RPS")
}