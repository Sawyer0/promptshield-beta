package jobs

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// MockProcessor for testing
type MockProcessor struct {
	processFunc func(ctx context.Context, job *Job) error
}

func (m *MockProcessor) Process(ctx context.Context, job *Job) error {
	if m.processFunc != nil {
		return m.processFunc(ctx, job)
	}
	
	// Default: simple success
	result := map[string]interface{}{
		"message": "processed successfully",
		"input_length": len(job.Input),
	}
	
	job.Result, _ = json.Marshal(result)
	return nil
}

func (m *MockProcessor) Type() string {
	return "test"
}

func TestManager_BasicOperations(t *testing.T) {
	manager := NewManager(1)
	processor := &MockProcessor{}
	manager.RegisterProcessor(processor)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	go manager.Start(ctx)
	defer manager.Stop()
	
	// Test job submission
	jobID, err := manager.Submit("test", []byte("test input"), nil)
	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}
	
	if jobID == "" {
		t.Fatal("Job ID should not be empty")
	}
	
	// Test job retrieval
	job, err := manager.Get(jobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}
	
	if job.ID != jobID {
		t.Errorf("Expected job ID %s, got %s", jobID, job.ID)
	}
	
	if job.Type != "test" {
		t.Errorf("Expected job type 'test', got %s", job.Type)
	}
	
	// Wait for processing
	time.Sleep(100 * time.Millisecond)
	
	// Check final state
	job, err = manager.Get(jobID)
	if err != nil {
		t.Fatalf("Failed to get job after processing: %v", err)
	}
	
	if job.Status != JobStatusCompleted {
		t.Errorf("Expected job status %s, got %s", JobStatusCompleted, job.Status)
	}
	
	if job.Progress != 100 {
		t.Errorf("Expected progress 100, got %d", job.Progress)
	}
}

func TestManager_JobNotFound(t *testing.T) {
	manager := NewManager(1)
	
	_, err := manager.Get("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent job")
	}
}

func TestManager_List(t *testing.T) {
	manager := NewManager(1)
	processor := &MockProcessor{}
	manager.RegisterProcessor(processor)
	
	// Submit multiple jobs
	jobID1, _ := manager.Submit("test", []byte("input1"), nil)
	jobID2, _ := manager.Submit("test", []byte("input2"), nil)
	
	// List all jobs
	jobs := manager.List("")
	if len(jobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(jobs))
	}
	
	// Verify job IDs are present
	found := make(map[string]bool)
	for _, job := range jobs {
		found[job.ID] = true
	}
	
	if !found[jobID1] || !found[jobID2] {
		t.Error("Not all submitted jobs found in list")
	}
}

func TestManager_Cancel(t *testing.T) {
	manager := NewManager(1)
	processor := &MockProcessor{
		processFunc: func(ctx context.Context, job *Job) error {
			// Simulate long-running job
			time.Sleep(200 * time.Millisecond)
			return nil
		},
	}
	manager.RegisterProcessor(processor)
	
	jobID, _ := manager.Submit("test", []byte("input"), nil)
	
	// Cancel the job
	err := manager.Cancel(jobID)
	if err != nil {
		t.Fatalf("Failed to cancel job: %v", err)
	}
	
	job, _ := manager.Get(jobID)
	if job.Status != JobStatusCancelled {
		t.Errorf("Expected job status %s, got %s", JobStatusCancelled, job.Status)
	}
}

func TestManager_UnknownJobType(t *testing.T) {
	manager := NewManager(1)
	
	_, err := manager.Submit("unknown", []byte("input"), nil)
	if err == nil {
		t.Fatal("Expected error for unknown job type")
	}
}