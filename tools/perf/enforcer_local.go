//go:build toolsperf
// +build toolsperf

package main

import (
	"log"
	"net/http"
	"os"
	"time"

	enforcerhttp "github.com/promptshield/promptshield/internal/interfaces/http/enforcer"

	_ "net/http/pprof"
)

// enforcer_local starts the enforcer HTTP mux on localhost:8080 and pprof on :6060.
// It requires no database and defaults to fast, dev-friendly settings.
func main() {
	addr := getenv("PERF_ADDR", "localhost:8080")

	// Perf-friendly defaults
	os.Setenv("PS_ENFORCER_FAST", getenv("PS_ENFORCER_FAST", "1"))
	os.Setenv("PS_ENFORCER_DISABLE_TRACING", getenv("PS_ENFORCER_DISABLE_TRACING", "1"))
	// Optional: preload a rulepack via PS_ENFORCER_RULEPACK=<path>
	// Optional: cap streaming bytes via PS_ENFORCER_MAX_STREAM_BYTES

	// pprof on :6060
	go func() {
		log.Printf("pprof listening on http://localhost:6060/debug/pprof/")
		_ = http.ListenAndServe("localhost:6060", nil)
	}()

	h := enforcerhttp.NewMux()
	srv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("enforcer-local listening on http://%s (endpoints: /check, /scan, /healthz, /readyz)", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

