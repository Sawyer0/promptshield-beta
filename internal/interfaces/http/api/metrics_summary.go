package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// statsHandler serves GET /stats from the Prometheus default gatherer.
func statsHandler(w http.ResponseWriter, r *http.Request) {
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
}

// usageHandler serves GET /usage from the Prometheus default gatherer.
func usageHandler(w http.ResponseWriter, r *http.Request) {
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
}
