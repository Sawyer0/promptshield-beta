package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// JobStatus represents the current state of a job
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusRunning    JobStatus = "running"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCancelled  JobStatus = "cancelled"
)

// Job represents an asynchronous scan job
type Job struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Status      JobStatus              `json:"status"`
	Input       []byte                 `json:"input,omitempty"`
	Result      json.RawMessage        `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Progress    int                    `json:"progress"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// JobProcessor defines the interface for processing different job types
type JobProcessor interface {
	Process(ctx context.Context, job *Job) error
	Type() string
}

// Manager handles asynchronous job execution
type Manager struct {
	mu         sync.RWMutex
	jobs       map[string]*Job
	queue      chan string
	processors map[string]JobProcessor
	workers    int
	shutdown   chan struct{}
	done       chan struct{}
}

// NewManager creates a new job manager with the specified number of workers
func NewManager(workers int) *Manager {
	if workers <= 0 {
		workers = 2
	}
	return &Manager{
		jobs:       make(map[string]*Job),
		queue:      make(chan string, 1000),
		processors: make(map[string]JobProcessor),
		workers:    workers,
		shutdown:   make(chan struct{}),
		done:       make(chan struct{}),
	}
}

// RegisterProcessor registers a job processor for a specific job type
func (m *Manager) RegisterProcessor(processor JobProcessor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processors[processor.Type()] = processor
}

// Start begins processing jobs with the configured number of workers
func (m *Manager) Start(ctx context.Context) {
	for i := 0; i < m.workers; i++ {
		go m.worker(ctx, i)
	}
}

// Stop gracefully shuts down the job manager
func (m *Manager) Stop() {
	close(m.shutdown)
	<-m.done
}

// Submit creates and queues a new job for processing
func (m *Manager) Submit(jobType string, input []byte, metadata map[string]interface{}) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.processors[jobType]; !exists {
		return "", fmt.Errorf("no processor registered for job type: %s", jobType)
	}

	jobID := uuid.New().String()
	job := &Job{
		ID:        jobID,
		Type:      jobType,
		Status:    JobStatusPending,
		Input:     input,
		CreatedAt: time.Now().UTC(),
		Progress:  0,
		Metadata:  metadata,
	}

	m.jobs[jobID] = job
	
	// Queue the job for processing
	select {
	case m.queue <- jobID:
		return jobID, nil
	default:
		job.Status = JobStatusFailed
		job.Error = "job queue is full"
		return jobID, fmt.Errorf("job queue is full")
	}
}

// Get retrieves a job by ID
func (m *Manager) Get(jobID string) (*Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	job, exists := m.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	
	// Return a copy to prevent external modification
	jobCopy := *job
	return &jobCopy, nil
}

// List returns all jobs, optionally filtered by status
func (m *Manager) List(status JobStatus) []*Job {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var results []*Job
	for _, job := range m.jobs {
		if status == "" || job.Status == status {
			jobCopy := *job
			results = append(results, &jobCopy)
		}
	}
	return results
}

// Cancel cancels a pending or running job
func (m *Manager) Cancel(jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	job, exists := m.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}
	
	if job.Status == JobStatusCompleted || job.Status == JobStatusFailed {
		return fmt.Errorf("cannot cancel job in status: %s", job.Status)
	}
	
	job.Status = JobStatusCancelled
	now := time.Now().UTC()
	job.CompletedAt = &now
	job.Error = "cancelled by user"
	
	return nil
}

// worker processes jobs from the queue
func (m *Manager) worker(ctx context.Context, workerID int) {
	defer func() {
		if workerID == 0 { // Only the first worker signals completion
			close(m.done)
		}
	}()

	for {
		select {
		case <-m.shutdown:
			log.Printf("Job worker %d shutting down", workerID)
			return
		case <-ctx.Done():
			log.Printf("Job worker %d cancelled", workerID)
			return
		case jobID := <-m.queue:
			m.processJob(ctx, jobID, workerID)
		}
	}
}

// processJob executes a single job
func (m *Manager) processJob(ctx context.Context, jobID string, workerID int) {
	m.mu.Lock()
	job, exists := m.jobs[jobID]
	if !exists {
		m.mu.Unlock()
		log.Printf("Worker %d: job %s not found", workerID, jobID)
		return
	}
	
	if job.Status != JobStatusPending {
		m.mu.Unlock()
		log.Printf("Worker %d: job %s not pending (status: %s)", workerID, jobID, job.Status)
		return
	}
	
	processor, processorExists := m.processors[job.Type]
	if !processorExists {
		job.Status = JobStatusFailed
		job.Error = fmt.Sprintf("no processor for job type: %s", job.Type)
		now := time.Now().UTC()
		job.CompletedAt = &now
		m.mu.Unlock()
		return
	}
	
	// Mark job as running
	job.Status = JobStatusRunning
	now := time.Now().UTC()
	job.StartedAt = &now
	m.mu.Unlock()
	
	log.Printf("Worker %d: processing job %s (type: %s)", workerID, jobID, job.Type)
	
	// Process the job
	err := processor.Process(ctx, job)
	
	// Update job status
	m.mu.Lock()
	defer m.mu.Unlock()
	
	completedAt := time.Now().UTC()
	job.CompletedAt = &completedAt
	
	if err != nil {
		job.Status = JobStatusFailed
		job.Error = err.Error()
		log.Printf("Worker %d: job %s failed: %v", workerID, jobID, err)
	} else {
		job.Status = JobStatusCompleted
		job.Progress = 100
		log.Printf("Worker %d: job %s completed", workerID, jobID)
	}
}

// CleanupCompleted removes completed jobs older than the specified duration
func (m *Manager) CleanupCompleted(maxAge time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	cutoff := time.Now().UTC().Add(-maxAge)
	var removed int
	
	for jobID, job := range m.jobs {
		if (job.Status == JobStatusCompleted || job.Status == JobStatusFailed || job.Status == JobStatusCancelled) &&
			job.CompletedAt != nil && job.CompletedAt.Before(cutoff) {
			delete(m.jobs, jobID)
			removed++
		}
	}
	
	log.Printf("Cleaned up %d completed jobs older than %v", removed, maxAge)
	return removed
}