package main

import (
	"context"
	"io"
	"testing"
	"time"

	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	grpcenforcer "github.com/promptshield/promptshield/internal/interfaces/grpc/enforcer"
)

// local fake stream for testing
type fakeStream struct {
	extproc.ExternalProcessor_ProcessServer
	ctx     context.Context
	recvQ   []*extproc.ProcessingRequest
	sendQ   []*extproc.ProcessingResponse
	recvIdx int
}

func (f *fakeStream) Context() context.Context { return f.ctx }
func (f *fakeStream) Recv() (*extproc.ProcessingRequest, error) {
	if f.recvIdx >= len(f.recvQ) {
		return nil, io.EOF
	}
	r := f.recvQ[f.recvIdx]
	f.recvIdx++
	return r, nil
}
func (f *fakeStream) Send(resp *extproc.ProcessingResponse) error {
	f.sendQ = append(f.sendQ, resp)
	return nil
}

func TestExtProc_Process_Responds(t *testing.T) {
	s := grpcenforcer.NewWithOptions(grpcenforcer.Options{Timeout: 200 * time.Millisecond, MaxStreamBytes: 1_000_000, FailOn: "HIGH"})
	st := &fakeStream{ctx: context.Background(), recvQ: []*extproc.ProcessingRequest{
		{Request: &extproc.ProcessingRequest_RequestHeaders{RequestHeaders: &extproc.HttpHeaders{}}},
		{Request: &extproc.ProcessingRequest_ResponseBody{ResponseBody: &extproc.HttpBody{Body: []byte("api_key=123")}}},
		{Request: &extproc.ProcessingRequest_ResponseTrailers{ResponseTrailers: &extproc.HttpTrailers{}}},
	}}
	if err := s.Process(st); err != nil && err != context.DeadlineExceeded {
		t.Fatalf("Process error: %v", err)
	}
	if len(st.sendQ) == 0 {
		t.Fatalf("no responses sent")
	}
}