package main

import (
	"context"
	"testing"
	"time"

	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	grpcenforcer "github.com/promptshield/promptshield/internal/interfaces/grpc/enforcer"
)

// BenchmarkGatewayGRPCExtProc_SetupOnly measures overhead of establishing the ext_proc server and a minimal stream.
func BenchmarkGatewayGRPCExtProc_SetupOnly(b *testing.B) {
	// Create an enforcer server configured for speed with small limits
	s := grpcenforcer.NewWithOptions(grpcenforcer.Options{Timeout: 50 * time.Millisecond, MaxStreamBytes: 256 * 1024})

	// Use the in-process fake stream from tests to avoid network overhead
	fs := &fakeStream{ctx: context.Background(), recvQ: []*extproc.ProcessingRequest{
		{Request: &extproc.ProcessingRequest_RequestHeaders{RequestHeaders: &extproc.HttpHeaders{}}},
		{Request: &extproc.ProcessingRequest_RequestBody{RequestBody: &extproc.HttpBody{Body: []byte("hello world")}}},
		{Request: &extproc.ProcessingRequest_RequestTrailers{RequestTrailers: &extproc.HttpTrailers{}}},
	}}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fs.recvIdx = 0
		fs.sendQ = nil
		_ = s.Process(fs)
	}
}
