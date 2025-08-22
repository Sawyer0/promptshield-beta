package main

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	grpcenforcer "github.com/promptshield/promptshield/internal/interfaces/grpc/enforcer"
)

// TestGRPCExtProc_SLA checks that processing a small 3-message stream meets a latency SLA.
// Skipped unless PS_ENFORCE_SLA=1 to avoid flakiness across hosts.
func TestGRPCExtProc_SLA(t *testing.T) {
	if os.Getenv("PS_ENFORCE_SLA") != "1" {
		t.Skip("set PS_ENFORCE_SLA=1 to enforce gateway gRPC SLA")
	}
	maxMs := 25.0
	if v := os.Getenv("PS_SLA_GRPC_MS_MAX"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			maxMs = f
		}
	}
	s := grpcenforcer.NewWithOptions(grpcenforcer.Options{Timeout: 100 * time.Millisecond, MaxStreamBytes: 256 * 1024})
	fs := &fakeStream{ctx: context.Background(), recvQ: []*extproc.ProcessingRequest{
		{Request: &extproc.ProcessingRequest_RequestHeaders{RequestHeaders: &extproc.HttpHeaders{}}},
		{Request: &extproc.ProcessingRequest_RequestBody{RequestBody: &extproc.HttpBody{Body: []byte("hello world")}}},
		{Request: &extproc.ProcessingRequest_RequestTrailers{RequestTrailers: &extproc.HttpTrailers{}}},
	}}
	start := time.Now()
	if err := s.Process(fs); err != nil && err != context.DeadlineExceeded {
		t.Fatalf("Process error: %v", err)
	}
	elapsedMs := float64(time.Since(start).Milliseconds())
	if elapsedMs > maxMs {
		t.Fatalf("ext_proc stream latency %.1fms exceeds SLA (max %.1fms)", elapsedMs, maxMs)
	}
}
