package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestAsyncScanEndpoint(t *testing.T) {
	cleanup := withAsyncJobsLicense(t)
	defer cleanup()
	// Create test router with default job manager
	router := NewMux(Options{})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test async scan submission
	body := bytes.NewBufferString("test input with potential issues")
	req := httptest.NewRequest("POST", "/scan/async?format=json&input_name=test", body)
	req.Header.Set("Content-Type", "text/plain")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req.WithContext(ctx))

	if rec.Code != http.StatusAccepted {
		t.Fatalf("Expected status 202, got %d: %s", rec.Code, rec.Body.String())
	}

	// Parse response to get job ID
	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	jobID, ok := response["job_id"].(string)
	if !ok || jobID == "" {
		t.Fatal("Job ID not found in response")
	}

	if response["status"] != "pending" {
		t.Errorf("Expected status 'pending', got %v", response["status"])
	}

	// Test job status endpoint
	req = httptest.NewRequest("GET", "/jobs/"+jobID, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req.WithContext(ctx))

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200 for job status, got %d: %s", rec.Code, rec.Body.String())
	}

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Check final job status
	req = httptest.NewRequest("GET", "/jobs/"+jobID, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req.WithContext(ctx))

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200 for final job status, got %d", rec.Code)
	}

	var job map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &job); err != nil {
		t.Fatalf("Failed to parse job response: %v", err)
	}

	// Job should be completed or still running
	status := job["status"].(string)
	if status != "completed" && status != "running" && status != "pending" {
		t.Errorf("Unexpected job status: %s", status)
	}
}

func TestJobListEndpoint(t *testing.T) {
	cleanup := withAsyncJobsLicense(t)
	defer cleanup()
	router := NewMux(Options{})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Submit a test job first
	body := bytes.NewBufferString("test input")
	req := httptest.NewRequest("POST", "/scan/async", body)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req.WithContext(ctx))

	// Test jobs list endpoint
	req = httptest.NewRequest("GET", "/jobs", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req.WithContext(ctx))

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200 for jobs list, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse jobs list response: %v", err)
	}

	jobs, ok := response["jobs"].([]interface{})
	if !ok {
		t.Fatal("Jobs list not found in response")
	}

	if len(jobs) == 0 {
		t.Error("Expected at least one job in the list")
	}
}

func TestJobNotFound(t *testing.T) {
	cleanup := withAsyncJobsLicense(t)
	defer cleanup()
	router := NewMux(Options{})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := httptest.NewRequest("GET", "/jobs/nonexistent", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req.WithContext(ctx))

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for nonexistent job, got %d", rec.Code)
	}
}

func TestJobCancellation(t *testing.T) {
	cleanup := withAsyncJobsLicense(t)
	defer cleanup()
	router := NewMux(Options{})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Submit a job
	body := bytes.NewBufferString("test input")
	req := httptest.NewRequest("POST", "/scan/async", body)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req.WithContext(ctx))

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)
	jobID := response["job_id"].(string)

	// Cancel the job
	req = httptest.NewRequest("DELETE", "/jobs/"+jobID, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req.WithContext(ctx))

	if rec.Code != http.StatusNoContent {
		t.Errorf("Expected status 204 for job cancellation, got %d", rec.Code)
	}

	// Verify job is cancelled
	req = httptest.NewRequest("GET", "/jobs/"+jobID, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req.WithContext(ctx))

	if rec.Code == http.StatusOK {
		var job map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &job)

		if job["status"] != "cancelled" {
			t.Errorf("Expected cancelled job status, got %v", job["status"])
		}
	}
}

// withAsyncJobsLicense sets a temporary license enabling the async_jobs feature.
func withAsyncJobsLicense(t *testing.T) func() {
	t.Helper()
	payload := map[string]any{
		"org":        "test",
		"expires_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"tier":       "enterprise",
		"entitlements": map[string]any{
			"max_rps":  1000,
			"features": map[string]bool{"async_jobs": true},
		},
	}
	b, _ := json.Marshal(payload)
	token := base64.RawURLEncoding.EncodeToString(b) + "." + base64.RawURLEncoding.EncodeToString([]byte("sig"))
	prev := os.Getenv("PROMPTSHIELD_LICENSE_KEY")
	_ = os.Setenv("PROMPTSHIELD_LICENSE_KEY", token)
	return func() {
		if prev == "" {
			_ = os.Unsetenv("PROMPTSHIELD_LICENSE_KEY")
		} else {
			_ = os.Setenv("PROMPTSHIELD_LICENSE_KEY", prev)
		}
	}
}
