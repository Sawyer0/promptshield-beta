package contracts

import (
	"context"
	"encoding/json"
	"time"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// JobManager defines the interface for managing asynchronous jobs
type JobManager interface {
	// SubmitJob submits a new job for processing
	SubmitJob(ctx context.Context, job *types.Job) error
	
	// GetJob retrieves a job by ID
	GetJob(ctx context.Context, jobID string) (*types.Job, error)
	
	// ListJobs lists jobs with optional filtering
	ListJobs(ctx context.Context, filter *types.JobFilter) ([]*types.Job, error)
	
	// CancelJob cancels a running or pending job
	CancelJob(ctx context.Context, jobID string) error
	
	// RetryJob retries a failed job
	RetryJob(ctx context.Context, jobID string) error
	
	// GetJobStatus gets the current status of a job
	GetJobStatus(ctx context.Context, jobID string) (types.JobStatus, error)
	
	// WaitForJob waits for a job to complete with timeout
	WaitForJob(ctx context.Context, jobID string, timeout time.Duration) (*types.Job, error)
	
	// Start starts the job manager
	Start(ctx context.Context) error
	
	// Stop gracefully stops the job manager
	Stop(ctx context.Context) error
}

// JobProcessor defines the interface for processing specific job types
type JobProcessor interface {
	// Process processes a job
	Process(ctx context.Context, job *Job) error
	
	// Type returns the job type this processor handles
	Type() string
	
	// CanProcess returns true if this processor can handle the job
	CanProcess(job *Job) bool
	
	// ValidateInput validates job input before processing
	ValidateInput(input []byte) error
}

// JobQueue defines the interface for job queue operations
type JobQueue interface {
	// Enqueue adds a job to the queue
	Enqueue(ctx context.Context, job *Job) error
	
	// Dequeue retrieves the next job from the queue
	Dequeue(ctx context.Context, timeout time.Duration) (*Job, error)
	
	// Complete marks a job as completed
	Complete(ctx context.Context, jobID string, result []byte) error
	
	// Fail marks a job as failed
	Fail(ctx context.Context, jobID string, err error) error
	
	// Retry marks a job for retry
	Retry(ctx context.Context, jobID string, delay time.Duration) error
	
	// GetQueueSize returns the current queue size
	GetQueueSize(ctx context.Context) (int64, error)
	
	// GetQueueStats returns queue statistics
	GetQueueStats(ctx context.Context) (*JobQueueStats, error)
	
	// Purge removes all jobs from the queue
	Purge(ctx context.Context) error
}

// JobScheduler defines the interface for scheduling recurring jobs
type JobScheduler interface {
	// ScheduleJob schedules a job to run at a specific time
	ScheduleJob(ctx context.Context, job *Job, at time.Time) error
	
	// ScheduleRecurringJob schedules a job to run on a recurring basis
	ScheduleRecurringJob(ctx context.Context, job *Job, schedule string) error
	
	// CancelScheduledJob cancels a scheduled job
	CancelScheduledJob(ctx context.Context, jobID string) error
	
	// ListScheduledJobs lists all scheduled jobs
	ListScheduledJobs(ctx context.Context) ([]*ScheduledJob, error)
	
	// Start starts the job scheduler
	Start(ctx context.Context) error
	
	// Stop stops the job scheduler
	Stop(ctx context.Context) error
}

// JobMonitor defines the interface for monitoring job execution
type JobMonitor interface {
	// RecordJobStart records when a job starts
	RecordJobStart(ctx context.Context, jobID string, jobType string) error
	
	// RecordJobComplete records when a job completes
	RecordJobComplete(ctx context.Context, jobID string, duration time.Duration) error
	
	// RecordJobError records when a job fails
	RecordJobError(ctx context.Context, jobID string, err error) error
	
	// GetJobMetrics returns job execution metrics
	GetJobMetrics(ctx context.Context, timeWindow time.Duration) (*JobMetrics, error)
	
	// GetJobHistory returns execution history for a job
	GetJobHistory(ctx context.Context, jobID string) ([]*JobExecution, error)
}

// Job represents an asynchronous job
type Job struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Status      JobStatus              `json:"status"`
	Priority    int                    `json:"priority"`
	Input       []byte                 `json:"input,omitempty"`
	Result      json.RawMessage        `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Progress    int                    `json:"progress"`
	MaxRetries  int                    `json:"max_retries"`
	Attempts    int                    `json:"attempts"`
	Timeout     time.Duration          `json:"timeout,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
}

// JobStatus represents the current state of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
	JobStatusRetrying  JobStatus = "retrying"
	JobStatusScheduled JobStatus = "scheduled"
)

// JobFilter represents filtering options for listing jobs
type JobFilter struct {
	Status    []JobStatus `json:"status,omitempty"`
	Type      []string    `json:"type,omitempty"`
	Tags      []string    `json:"tags,omitempty"`
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	Limit     int         `json:"limit,omitempty"`
	Offset    int         `json:"offset,omitempty"`
}

// JobQueueStats represents job queue statistics
type JobQueueStats struct {
	TotalJobs     int64              `json:"total_jobs"`
	PendingJobs   int64              `json:"pending_jobs"`
	RunningJobs   int64              `json:"running_jobs"`
	CompletedJobs int64              `json:"completed_jobs"`
	FailedJobs    int64              `json:"failed_jobs"`
	JobsByType    map[string]int64   `json:"jobs_by_type"`
	JobsByStatus  map[JobStatus]int64 `json:"jobs_by_status"`
	AverageWaitTime time.Duration    `json:"average_wait_time"`
	AverageProcessingTime time.Duration `json:"average_processing_time"`
	LastUpdated   time.Time          `json:"last_updated"`
}

// ScheduledJob represents a scheduled job
type ScheduledJob struct {
	ID          string    `json:"id"`
	Job         *Job      `json:"job"`
	Schedule    string    `json:"schedule"`    // Cron expression or "once"
	NextRun     time.Time `json:"next_run"`
	LastRun     *time.Time `json:"last_run,omitempty"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// JobMetrics represents job execution metrics
type JobMetrics struct {
	TotalJobs         int64         `json:"total_jobs"`
	SuccessfulJobs    int64         `json:"successful_jobs"`
	FailedJobs        int64         `json:"failed_jobs"`
	AverageExecutionTime time.Duration `json:"average_execution_time"`
	MedianExecutionTime  time.Duration `json:"median_execution_time"`
	P95ExecutionTime     time.Duration `json:"p95_execution_time"`
	JobThroughput     float64       `json:"job_throughput"` // jobs per second
	ErrorRate         float64       `json:"error_rate"`
	RetryRate         float64       `json:"retry_rate"`
	JobsByType        map[string]*JobTypeMetrics `json:"jobs_by_type"`
	TimeWindow        time.Duration `json:"time_window"`
	GeneratedAt       time.Time     `json:"generated_at"`
}

// JobTypeMetrics represents metrics for a specific job type
type JobTypeMetrics struct {
	Type              string        `json:"type"`
	TotalJobs         int64         `json:"total_jobs"`
	SuccessfulJobs    int64         `json:"successful_jobs"`
	FailedJobs        int64         `json:"failed_jobs"`
	AverageExecutionTime time.Duration `json:"average_execution_time"`
	ErrorRate         float64       `json:"error_rate"`
}

// JobExecution represents a single job execution record
type JobExecution struct {
	JobID       string        `json:"job_id"`
	Attempt     int           `json:"attempt"`
	Status      JobStatus     `json:"status"`
	StartedAt   time.Time     `json:"started_at"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
	Duration    time.Duration `json:"duration"`
	Error       string        `json:"error,omitempty"`
	WorkerID    string        `json:"worker_id,omitempty"`
}

// ScanJobInput represents input for a scan job
type ScanJobInput struct {
	Content     string                 `json:"content"`
	RulepackID  string                 `json:"rulepack_id"`
	TenantID    string                 `json:"tenant_id"`
	RequestID   string                 `json:"request_id"`
	Provider    types.Provider         `json:"provider,omitempty"`
	Model       string                 `json:"model,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// ScanJobResult represents the result of a scan job
type ScanJobResult struct {
	Result     types.ScanResult       `json:"result"`
	RequestID  string                 `json:"request_id"`
	ProcessedAt time.Time             `json:"processed_at"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// JobWorker defines the interface for job workers
type JobWorker interface {
	// Start starts the worker
	Start(ctx context.Context) error
	
	// Stop stops the worker gracefully
	Stop(ctx context.Context) error
	
	// IsRunning returns true if the worker is running
	IsRunning() bool
	
	// GetWorkerID returns the unique worker ID
	GetWorkerID() string
	
	// GetProcessedCount returns the number of jobs processed
	GetProcessedCount() int64
}

// JobWorkerPool defines the interface for managing job workers
type JobWorkerPool interface {
	// Start starts the worker pool with the specified number of workers
	Start(ctx context.Context, numWorkers int) error
	
	// Stop stops all workers gracefully
	Stop(ctx context.Context) error
	
	// Scale scales the worker pool to the specified number of workers
	Scale(ctx context.Context, numWorkers int) error
	
	// GetWorkerCount returns the current number of workers
	GetWorkerCount() int
	
	// GetActiveWorkers returns the number of active workers
	GetActiveWorkers() int
	
	// GetWorkerStats returns statistics for all workers
	GetWorkerStats() map[string]*WorkerStats
}

// WorkerStats represents statistics for a worker
type WorkerStats struct {
	WorkerID       string        `json:"worker_id"`
	Status         WorkerStatus  `json:"status"`
	JobsProcessed  int64         `json:"jobs_processed"`
	JobsSuccessful int64         `json:"jobs_successful"`
	JobsFailed     int64         `json:"jobs_failed"`
	StartedAt      time.Time     `json:"started_at"`
	LastJobAt      *time.Time    `json:"last_job_at,omitempty"`
	CurrentJob     *string       `json:"current_job,omitempty"`
}

// WorkerStatus represents the status of a worker
type WorkerStatus string

const (
	WorkerStatusIdle    WorkerStatus = "idle"
	WorkerStatusBusy    WorkerStatus = "busy"
	WorkerStatusStopped WorkerStatus = "stopped"
	WorkerStatusError   WorkerStatus = "error"
)