package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/promptshield/promptshield/internal/jobs"
	"github.com/promptshield/promptshield/internal/jobs/processors"
	"github.com/promptshield/promptshield/internal/license"
	"github.com/promptshield/promptshield/internal/scanner"
	"github.com/promptshield/promptshield/internal/usage"
	"github.com/promptshield/promptshield/internal/version"
)

// Router

func NewMux(opt Options) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Timeout(10 * time.Second))
	r.Use(versionHeader("1"))
	// bytes in/out accounting
	r.Use(captureBytesMiddleware)

	// Ensure defaults for optional dependencies
	if opt.ConfigStore == nil {
		opt.ConfigStore = NewRuntimeConfigStoreFromEnv()
	}
	if opt.RulepackManager == nil {
		opt.RulepackManager = NewRulepackManager()
	}
	if opt.Events == nil {
		opt.Events = NewEventHub()
	}
	// usage store is optional; if unset, usage endpoint will report zeroes
	// Quota store default: derive from entitlements/env if not provided
	if opt.QuotaStore == nil {
		var rps float64
		var burst int
		if v := strings.TrimSpace(os.Getenv("PS_ENFORCER_RPS")); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
				rps = f
			}
		}
		if rps > 0 {
			if v := strings.TrimSpace(os.Getenv("PS_ENFORCER_RPS_BURST")); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n > 0 {
					burst = n
				}
			}
			opt.QuotaStore = usage.NewInMemoryQuota(rps, burst)
		}
	}
	if opt.JobManager == nil {
		// Create default job manager with 2 workers
		opt.JobManager = jobs.NewManager(2)

		// Create and register scan processor with basic scanner
		sc := scanner.New(0)
		scanProcessor := processors.NewScanProcessor(sc)
		opt.JobManager.RegisterProcessor(scanProcessor)

		// Start the job manager
		go opt.JobManager.Start(context.Background())
	}

	// Health & info
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("PS_ENFORCER_RULEPACK") == "" {
			if _, err := os.Stat("/rules/basic-security.yaml"); err != nil {
				if _, err := os.Stat("rules/basic-security.yaml"); err != nil {
					http.Error(w, "not ready: rulepack not loaded", http.StatusServiceUnavailable)
					return
				}
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	// Version
	r.Get("/version", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"version":    version.Version,
			"commit":     version.Commit,
			"build_date": version.BuildDate,
		})
	})

	// Rulepacks
	mountRulepacks(r, opt)

	// Config
	mountConfig(r, opt)

	// Admin
	r.Group(func(a chi.Router) {
		a.Use(adminAuth(opt))
		a.Post("/admin/drain", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			if opt.OnDrain != nil {
				go func() { _ = opt.OnDrain(r.Context()) }()
			}
		})
		a.Post("/admin/shutdown", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			if opt.OnShutdown != nil {
				// optional delay in seconds
				delay := 0 * time.Second
				if v := r.URL.Query().Get("delay"); v != "" {
					if n, err := strconv.Atoi(v); err == nil && n >= 0 {
						delay = time.Duration(n) * time.Second
					}
				}
				go func() { _ = opt.OnShutdown(r.Context(), delay) }()
			}
		})
	})

	// License (GET is public; POST rotates license and is admin-protected)
	r.Get("/license", func(w http.ResponseWriter, r *http.Request) {
		l := license.Info()
		ent, _ := license.Entitlement()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"org":          l.Organization,
			"tier":         l.Tier,
			"expires_at":   l.ExpiresAt,
			"licensed":     license.IsLicensed(),
			"entitlements": ent,
		})
	})
	r.Group(func(a chi.Router) {
		a.Use(adminAuth(opt))
		a.Post("/license", func(w http.ResponseWriter, r *http.Request) {
			key := r.FormValue("key")
			if key == "" {
				var body struct {
					Key string `json:"key"`
				}
				_ = json.NewDecoder(r.Body).Decode(&body)
				key = body.Key
			}
			if key == "" {
				writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "missing key", nil)
				return
			}
			_ = os.Setenv("PROMPTSHIELD_LICENSE_KEY", key)
			w.WriteHeader(http.StatusNoContent)
		})
	})

	// Decision endpoints (protected with user auth)
	r.Group(func(g chi.Router) {
		// legacy token-based user auth
		g.Use(userAuth(opt))
		// optional OIDC
		if opt.OIDC.Issuer != "" {
			g.Use(oidcAuth(opt))
		}
		// optional per-tenant quota
		if opt.QuotaStore != nil {
			g.Use(tenantQuota(opt))
		}
		g.Post("/check", checkHandlerVersioned(opt))
		g.Post("/scan", scanHandler(opt))
	})
	r.Post("/scan/async", func(w http.ResponseWriter, r *http.Request) {
		// Ensure license state is loaded before feature check
		_ = license.IsLicensed()
		if !license.HasFeature("async_jobs") {
			writeError(w, http.StatusForbidden, "UNAUTHORIZED", "feature not available: async_jobs", nil)
			return
		}

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
	})

	// Alias to support legacy colon-style path
	r.Post("/scan:async", func(w http.ResponseWriter, r *http.Request) {
		// Forward to /scan/async handler by rewriting and re-serving
		r.URL.Path = "/scan/async"
		r.RequestURI = ""
		// Serve the rewritten request on this router
		r = r.WithContext(r.Context())
		// Re-enter the router stack
		chi.NewRouter().ServeHTTP(w, r)
	})

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

	// Observability endpoints (admin-protected)
	r.Group(func(a chi.Router) {
		a.Use(adminAuth(opt))
		a.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
			// Summarize limited stats from Prometheus default gatherer
			mf, _ := prometheus.DefaultGatherer.Gather()
			stats := map[string]any{
				"decisions_total": map[string]int64{},
				"p95_latency_ms":  0.0,
			}
			// decisions
			for _, m := range mf {
				if m.GetName() == "ps_enforcer_decisions_total" {
					for _, mm := range m.Metric {
						var decision string
						for _, lp := range mm.Label {
							if lp.GetName() == "decision" {
								decision = lp.GetValue()
								break
							}
						}
						if decision != "" && mm.GetCounter() != nil {
							dt := stats["decisions_total"].(map[string]int64)
							dt[decision] += int64(mm.GetCounter().GetValue())
						}
					}
				}
				if m.GetName() == "ps_enforcer_request_duration_seconds" && m.GetType() == dto.MetricType_HISTOGRAM {
					// crude p95 from buckets
					for _, mm := range m.Metric {
						if mm.GetHistogram() != nil {
							var total uint64
							for _, b := range mm.GetHistogram().Bucket {
								total += b.GetCumulativeCount()
							}
							if total == 0 {
								continue
							}
							var threshold = float64(total) * 0.95
							var cum uint64
							var p95 float64
							for _, b := range mm.GetHistogram().Bucket {
								cum += b.GetCumulativeCount()
								if float64(cum) >= threshold {
									p95 = b.GetUpperBound() * 1000.0 // seconds -> ms
									break
								}
							}
							if p95 > 0 {
								stats["p95_latency_ms"] = p95
								break
							}
						}
					}
				}
			}
			_ = json.NewEncoder(w).Encode(stats)
		})
		a.Get("/events", func(w http.ResponseWriter, r *http.Request) {
			var flusher http.Flusher
			if f, ok := w.(http.Flusher); ok {
				flusher = f
			}
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			ch := opt.Events.Subscribe(r.Context().Done())
			defer opt.Events.Unsubscribe(ch)
			// Optional type filtering
			var filters map[string]struct{}
			if v := strings.TrimSpace(r.URL.Query().Get("types")); v != "" {
				filters = make(map[string]struct{})
				for _, t := range strings.Split(v, ",") {
					if t = strings.TrimSpace(t); t != "" {
						filters[t] = struct{}{}
					}
				}
			}
			// initial
			_, _ = w.Write([]byte("event: ready\ndata: {\"status\":\"ok\"}\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
			notify := r.Context().Done()
			for {
				select {
				case <-notify:
					return
				case ev := <-ch:
					if ev == nil {
						return
					}
					if len(filters) > 0 {
						if _, ok := filters[ev.Type]; !ok {
							continue
						}
					}
					_, _ = w.Write(ev.SSE())
					if flusher != nil {
						flusher.Flush()
					}
				}
			}
		})
		a.Get("/usage", func(w http.ResponseWriter, r *http.Request) {
			now := time.Now().UTC()
			windowStart := now.Add(-1 * time.Hour)
			// Best-effort usage summary; computing precise usage should be offloaded to metrics pipeline
			mf, _ := prometheus.DefaultGatherer.Gather()
			var decisions int64
			var bytes int64
			for _, m := range mf {
				if m.GetName() == "ps_enforcer_decisions_total" {
					for _, mm := range m.Metric {
						if mm.GetCounter() != nil {
							decisions += int64(mm.GetCounter().GetValue())
						}
					}
				}
				if m.GetName() == "ps_extproc_bytes_total" {
					if len(m.Metric) > 0 && m.Metric[0].GetCounter() != nil {
						bytes += int64(m.Metric[0].GetCounter().GetValue())
					}
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"window_start": windowStart.Format(time.RFC3339),
				"window_end":   now.Format(time.RFC3339),
				"counts":       decisions,
				"bytes":        bytes,
			})
		})
	})

	// Expose usage store via context for handlers that may record usage
	if opt.UsageStore != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := withUsageStore(req.Context(), opt.UsageStore)
			r.ServeHTTP(w, req.WithContext(ctx))
		})
	}
	return r
}
