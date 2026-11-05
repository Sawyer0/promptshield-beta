package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/promptshield/promptshield/internal/jobs"
)

// mountJobs registers async scan submission and job management.
func mountJobs(r chi.Router, opt Options) {
	scanAsync := func(w http.ResponseWriter, r *http.Request) {
		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "failed to read request body", nil)
			return
		}

		// Parse metadata from headers and query params
		metadata := map[string]interface{}{
			"output_format": r.URL.Query().Get("format"),
			"fail_on":       r.URL.Query().Get("fail_on"),
			"input_name":    r.URL.Query().Get("input_name"),
		}

		// Set defaults
		if metadata["output_format"] == "" {
			metadata["output_format"] = "json"
		}
		if metadata["input_name"] == "" {
			metadata["input_name"] = "http-request"
		}

		// Submit the job
		jobID, err := opt.JobManager.Submit("scan", body, metadata)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "JOB_SUBMISSION_FAILED", err.Error(), nil)
			return
		}

		// Return job ID
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"job_id": jobID,
			"status": "pending",
		})
	}
	r.Post("/scan/async", scanAsync)
	// Legacy colon-style alias. Registering the same handler avoids re-entering
	// the router, which looped forever because chi kept the original route path.
	r.Post("/scan:async", scanAsync)

	// Job management endpoints
	r.Get("/jobs", func(w http.ResponseWriter, r *http.Request) {
		status := r.URL.Query().Get("status")
		jobs := opt.JobManager.List(jobs.JobStatus(status))
		w.Header().Set("content-type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"jobs": jobs,
		})
	})

	r.Get("/jobs/{jobID}", func(w http.ResponseWriter, r *http.Request) {
		jobID := chi.URLParam(r, "jobID")
		job, err := opt.JobManager.Get(jobID)
		if err != nil {
			writeError(w, http.StatusNotFound, "JOB_NOT_FOUND", err.Error(), nil)
			return
		}
		w.Header().Set("content-type", "application/json")
		_ = json.NewEncoder(w).Encode(job)
	})

	r.Delete("/jobs/{jobID}", func(w http.ResponseWriter, r *http.Request) {
		jobID := chi.URLParam(r, "jobID")
		err := opt.JobManager.Cancel(jobID)
		if err != nil {
			writeError(w, http.StatusNotFound, "JOB_NOT_FOUND", err.Error(), nil)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
