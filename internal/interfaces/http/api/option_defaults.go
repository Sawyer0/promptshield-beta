package api

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/promptshield/promptshield/internal/application/decisions"
	"github.com/promptshield/promptshield/internal/jobs"
	"github.com/promptshield/promptshield/internal/jobs/processors"
	q "github.com/promptshield/promptshield/internal/jobs/queue"
	"github.com/promptshield/promptshield/internal/scanner"
	"github.com/promptshield/promptshield/internal/usage"
)

// withDefaults fills the optional dependencies NewMux needs and starts the
// default job manager. It panics when RulepackService is missing.
func withDefaults(opt Options) Options {
	// Ensure defaults for optional dependencies
	if opt.ConfigStore == nil {
		opt.ConfigStore = NewRuntimeConfigStoreFromEnv()
	}
	// RulepackService is required and must be provided by caller
	if opt.RulepackService == nil {
		panic("RulepackService is required")
	}
	if opt.Events == nil {
		opt.Events = NewEventHub()
	}
	if opt.Decisions == nil {
		opt.Decisions = decisions.NewLogFromEnv()
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
		sc := scanner.ScanEngineCstor(0)
		scanProcessor := processors.NewScanProcessor(sc)
		opt.JobManager.RegisterProcessor(scanProcessor)

		// Optional durable queue via Redis
		if strings.EqualFold(os.Getenv("PS_ASYNC_QUEUE"), "redis") {
			addr := os.Getenv("PS_REDIS_ADDR")
			if addr == "" {
				addr = "127.0.0.1:6379"
			}
			stream := os.Getenv("PS_REDIS_STREAM")
			if stream == "" {
				stream = "ps.jobs"
			}
			group := os.Getenv("PS_REDIS_GROUP")
			if group == "" {
				group = "ps.workers"
			}
			consumer := os.Getenv("PS_REDIS_CONSUMER")
			if consumer == "" {
				consumer = "ps-enforcer"
			}
			vis := 30 * time.Second
			if v := os.Getenv("PS_REDIS_VISIBILITY_TTL"); v != "" {
				if d, err := time.ParseDuration(v); err == nil {
					vis = d
				}
			}
			if rq, err := q.NewRedisQueue(addr, os.Getenv("PS_REDIS_PASSWORD"), 0, stream, group, consumer, vis); err == nil {
				opt.JobManager = opt.JobManager.WithDurable(rq)
			}
		}

		// Start the job manager
		go opt.JobManager.Start(context.Background())
	}
	return opt
}
