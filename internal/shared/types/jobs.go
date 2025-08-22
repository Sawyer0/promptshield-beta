package types

import (
	"encoding/json"
	"time"
)

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
	Status        []JobStatus `json:"status,omitempty"`
	Type          []string    `json:"type,omitempty"`
	Tags          []string    `json:"tags,omitempty"`
	CreatedAfter  *time.Time  `json:"created_after,omitempty"`
	CreatedBefore *time.Time  `json:"created_before,omitempty"`
	Limit         int         `json:"limit,omitempty"`
	Offset        int         `json:"offset,omitempty"`
}

// JobQueueStats represents job queue statistics
type JobQueueStats struct {
	TotalJobs             int64               `json:"total_jobs"`
	PendingJobs           int64               `json:"pending_jobs"`
	RunningJobs           int64               `json:"running_jobs"`
	CompletedJobs         int64               `json:"completed_jobs"`
	FailedJobs            int64               `json:"failed_jobs"`
	JobsByType            map[string]int64    `json:"jobs_by_type"`
	JobsByStatus          map[JobStatus]int64 `json:"jobs_by_status"`
	AverageWaitTime       time.Duration       `json:"average_wait_time"`
	AverageProcessingTime time.Duration       `json:"average_processing_time"`
	LastUpdated           time.Time           `json:"last_updated"`
}

// ScheduledJob represents a scheduled job
type ScheduledJob struct {
	ID        string     `json:"id"`
	Job       *Job       `json:"job"`
	Schedule  string     `json:"schedule"`    // Cron expression or "once"
	NextRun   time.Time  `json:"next_run"`
	LastRun   *time.Time `json:"last_run,omitempty"`
	Enabled   bool       `json:"enabled"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// JobMetrics represents job execution metrics
type JobMetrics struct {
	TotalJobs            int64                      `json:"total_jobs"`
	SuccessfulJobs       int64                      `json:"successful_jobs"`
	FailedJobs           int64                      `json:"failed_jobs"`
	AverageExecutionTime time.Duration              `json:"average_execution_time"`
	MedianExecutionTime  time.Duration              `json:"median_execution_time"`
	P95ExecutionTime     time.Duration              `json:"p95_execution_time"`
	JobThroughput        float64                    `json:"job_throughput"` // jobs per second
	ErrorRate            float64                    `json:"error_rate"`
	RetryRate            float64                    `json:"retry_rate"`
	JobsByType           map[string]*JobTypeMetrics `json:"jobs_by_type"`
	TimeWindow           time.Duration              `json:"time_window"`
	GeneratedAt          time.Time                  `json:"generated_at"`
}

// JobTypeMetrics represents metrics for a specific job type
type JobTypeMetrics struct {
	Type                 string        `json:"type"`
	TotalJobs            int64         `json:"total_jobs"`
	SuccessfulJobs       int64         `json:"successful_jobs"`
	FailedJobs           int64         `json:"failed_jobs"`
	AverageExecutionTime time.Duration `json:"average_execution_time"`
	ErrorRate            float64       `json:"error_rate"`
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
	Provider    Provider               `json:"provider,omitempty"`
	Model       string                 `json:"model,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// ScanJobResult represents the result of a scan job
type ScanJobResult struct {
	Result      ScanResult             `json:"result"`
	RequestID   string                 `json:"request_id"`
	ProcessedAt time.Time              `json:"processed_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// WorkerStats represents statistics for a worker
type WorkerStats struct {
	WorkerID       string       `json:"worker_id"`
	Status         WorkerStatus `json:"status"`
	JobsProcessed  int64        `json:"jobs_processed"`
	JobsSuccessful int64        `json:"jobs_successful"`
	JobsFailed     int64        `json:"jobs_failed"`
	StartedAt      time.Time    `json:"started_at"`
	LastJobAt      *time.Time   `json:"last_job_at,omitempty"`
	CurrentJob     *string      `json:"current_job,omitempty"`
}

// WorkerStatus represents the status of a worker
type WorkerStatus string

const (
	WorkerStatusIdle    WorkerStatus = "idle"
	WorkerStatusBusy    WorkerStatus = "busy"
	WorkerStatusStopped WorkerStatus = "stopped"
	WorkerStatusError   WorkerStatus = "error"
)