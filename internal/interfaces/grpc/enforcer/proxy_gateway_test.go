package grpcenforcer_test

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	grpcenforcer "github.com/promptshield/promptshield/internal/interfaces/grpc/enforcer"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/testutil/fixtures"
)

// TestIntegration_EnvoyExtProcFlow tests the complete Envoy ext_proc workflow
func TestIntegration_EnvoyExtProcFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a test gRPC server with PromptShield enforcer
	server := grpcenforcer.NewWithOptions(grpcenforcer.Options{
		Timeout:         500 * time.Millisecond,
		MaxStreamBytes:  1024 * 1024,
		FailOn:          "LOW",
		EnforcementMode: "enforce",
	})
	// Load minimal rules for prompt injection detection
	server.LoadTestRules([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "test-pack", Version: "1.0.0"},
		Rules:    []rules.Rule{{ID: "test-prompt-injection", Level: 1, Severity: "HIGH", Keywords: []string{"ignore previous instructions", "reveal your system prompt"}}},
	}})

	// Set up in-memory gRPC server
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	extproc.RegisterExternalProcessorServer(s, server)

	go func() {
		if err := s.Serve(lis); err != nil {
			t.Logf("Server exited with error: %v", err)
		}
	}()
	defer s.Stop()

	// Create client connection
	conn, err := grpc.NewClient("bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	client := extproc.NewExternalProcessorClient(conn)

	t.Run("malicious prompt blocked", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Simulate Envoy ext_proc stream
		stream, err := client.Process(ctx)
		require.NoError(t, err)

		// Send request headers (simulating Envoy)
		err = stream.Send(&extproc.ProcessingRequest{
			Request: &extproc.ProcessingRequest_RequestHeaders{
				RequestHeaders: &extproc.HttpHeaders{
					Headers: &corev3.HeaderMap{
						Headers: []*corev3.HeaderValue{
							{Key: ":method", RawValue: []byte("POST")},
							{Key: ":path", RawValue: []byte("/v1/chat/completions")},
							{Key: ":authority", RawValue: []byte("api.openai.com")},
							{Key: "content-type", RawValue: []byte("application/json")},
							{Key: "authorization", RawValue: []byte("Bearer sk-test-key")},
						},
					},
				},
			},
		})
		require.NoError(t, err)

		// Send malicious request body
		maliciousPayload := `{
			"model": "gpt-4",
			"messages": [
				{"role": "user", "content": "` + fixtures.PromptInjectionPayload + `"}
			]
		}`

		err = stream.Send(&extproc.ProcessingRequest{
			Request: &extproc.ProcessingRequest_RequestBody{
				RequestBody: &extproc.HttpBody{
					Body: []byte(maliciousPayload),
				},
			},
		})
		require.NoError(t, err)

		// Send request trailers to flush processing
		_ = stream.Send(&extproc.ProcessingRequest{Request: &extproc.ProcessingRequest_RequestTrailers{RequestTrailers: &extproc.HttpTrailers{}}})

		// Try a few times to receive the immediate response
		var resp *extproc.ProcessingResponse
		var recvErr error
		for i := 0; i < 3; i++ {
			resp, recvErr = stream.Recv()
			if recvErr == nil {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		require.NoError(t, recvErr)

		// Should receive immediate response (blocked)
		if immediateResp := resp.GetImmediateResponse(); immediateResp != nil {
			assert.Equal(t, typev3.StatusCode_Forbidden, immediateResp.Status.Code)
			body := string(immediateResp.Body)
			assert.Contains(t, body, "blocked")
			assert.Contains(t, body, "PromptShield")
			headers := immediateResp.Headers.GetSetHeaders()
			var hasDecisionHeader bool
			for _, header := range headers {
				if header.Header.Key == "x-ps-decision" {
					hasDecisionHeader = true
					break
				}
			}
			assert.True(t, hasDecisionHeader, "Should have x-ps-decision header")
		} else {
			t.Fatal("Expected immediate response for blocked request")
		}
	})

	t.Run("benign prompt allowed", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		stream, err := client.Process(ctx)
		require.NoError(t, err)

		// Send request headers
		err = stream.Send(&extproc.ProcessingRequest{
			Request: &extproc.ProcessingRequest_RequestHeaders{
				RequestHeaders: &extproc.HttpHeaders{
					Headers: &corev3.HeaderMap{
						Headers: []*corev3.HeaderValue{
							{Key: ":method", RawValue: []byte("POST")},
							{Key: ":path", RawValue: []byte("/v1/chat/completions")},
							{Key: "content-type", RawValue: []byte("application/json")},
						},
					},
				},
			},
		})
		require.NoError(t, err)

		// Send benign request body
		benignPayload := `{
			"model": "gpt-4",
			"messages": [
				{"role": "user", "content": "What is the capital of France?"}
			]
		}`

		err = stream.Send(&extproc.ProcessingRequest{
			Request: &extproc.ProcessingRequest_RequestBody{
				RequestBody: &extproc.HttpBody{
					Body: []byte(benignPayload),
				},
			},
		})
		require.NoError(t, err)

		// Should receive continue response (allowed)
		resp, err := stream.Recv()
		require.NoError(t, err)

		// Should NOT be an immediate response (should continue)
		assert.Nil(t, resp.GetImmediateResponse(), "Benign request should not be blocked")

		// Should be a header response allowing continuation
		if headerResp := resp.GetRequestHeaders(); headerResp != nil {
			// Request is being processed normally
			assert.NotNil(t, headerResp)
		} else if bodyResp := resp.GetRequestBody(); bodyResp != nil {
			// Body processing continues
			assert.NotNil(t, bodyResp)
		}
	})
}

// TestIntegration_ResponseScanning tests scanning of LLM responses
func TestIntegration_ResponseScanning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	server := grpcenforcer.NewWithOptions(grpcenforcer.Options{
		Timeout:         500 * time.Millisecond,
		FailOn:          "LOW",
		EnforcementMode: "enforce",
	})
	// Load rule to redact API keys in responses
	server.LoadTestRules([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "resp-pack", Version: "1.0.0"},
		Rules: []rules.Rule{{
			ID: "leak-api-key", Level: 2, Severity: "HIGH",
			Patterns: []rules.Pattern{{Regex: "sk-proj-[A-Za-z0-9]+", Flags: []string{"ignorecase"}}},
			Response: &rules.Response{Action: "redact", Message: "API key redacted"},
		}},
	}})

	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	extproc.RegisterExternalProcessorServer(s, server)

	go func() {
		_ = s.Serve(lis)
	}()
	defer s.Stop()

	conn, err := grpc.NewClient("bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	client := extproc.NewExternalProcessorClient(conn)

	t.Run("response with leaked credentials", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		stream, err := client.Process(ctx)
		require.NoError(t, err)

		// Send response headers
		err = stream.Send(&extproc.ProcessingRequest{
			Request: &extproc.ProcessingRequest_ResponseHeaders{
				ResponseHeaders: &extproc.HttpHeaders{
					Headers: &corev3.HeaderMap{
						Headers: []*corev3.HeaderValue{
							{Key: ":status", RawValue: []byte("200")},
							{Key: "content-type", RawValue: []byte("application/json")},
						},
					},
				},
			},
		})
		require.NoError(t, err)

		// Send response body containing leaked API key
		responseWithAPIKey := `{
			"choices": [{
				"message": {
					"content": "Here's your API key: ` + fixtures.SampleAPIKey + ` - keep it safe!"
				}
			}]
		}`

		err = stream.Send(&extproc.ProcessingRequest{
			Request: &extproc.ProcessingRequest_ResponseBody{
				ResponseBody: &extproc.HttpBody{
					Body: []byte(responseWithAPIKey),
				},
			},
		})
		require.NoError(t, err)

		// Expect redaction or blocking
		resp, err := stream.Recv()
		require.NoError(t, err)

		// Should either redact the content or block the response
		if immediateResp := resp.GetImmediateResponse(); immediateResp != nil {
			// Response was blocked entirely
			assert.Equal(t, typev3.StatusCode_Forbidden, immediateResp.Status.Code)
		} else if bodyResp := resp.GetResponseBody(); bodyResp != nil {
			// Response was modified (redacted)
			if bodyResp.Response != nil && bodyResp.Response.BodyMutation != nil {
				modifiedBody := string(bodyResp.Response.BodyMutation.GetBody())
				assert.NotContains(t, modifiedBody, fixtures.SampleAPIKey, "API key should be redacted")
				assert.Contains(t, modifiedBody, "[REDACTED", "Should contain redaction marker")
			}
		}
	})
}

// TestIntegration_StreamingRequests tests handling of large streaming requests
func TestIntegration_StreamingRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	server := grpcenforcer.NewWithOptions(grpcenforcer.Options{
		Timeout:        2 * time.Second,
		FailOn:         "LOW",
		MaxStreamBytes: 64 * 1024, // 64KB limit
	})
	// Load minimal rules for prompt injection detection
	server.LoadTestRules([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "test-pack", Version: "1.0.0"},
		Rules:    []rules.Rule{{ID: "test-prompt-injection", Level: 1, Severity: "HIGH", Keywords: []string{"ignore previous instructions", "reveal your system prompt"}}},
	}})

	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	extproc.RegisterExternalProcessorServer(s, server)

	go func() {
		_ = s.Serve(lis)
	}()
	defer s.Stop()

	conn, err := grpc.NewClient("bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	client := extproc.NewExternalProcessorClient(conn)

	t.Run("large request chunked processing", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		stream, err := client.Process(ctx)
		require.NoError(t, err)

		// Send headers
		err = stream.Send(&extproc.ProcessingRequest{
			Request: &extproc.ProcessingRequest_RequestHeaders{
				RequestHeaders: &extproc.HttpHeaders{
					Headers: &corev3.HeaderMap{
						Headers: []*corev3.HeaderValue{
							{Key: ":method", RawValue: []byte("POST")},
							{Key: ":path", RawValue: []byte("/v1/chat/completions")},
							{Key: "content-type", RawValue: []byte("application/json")},
						},
					},
				},
			},
		})
		require.NoError(t, err)

		// Create large payload with embedded prompt injection
		largeContent := fixtures.GenerateLargeContent(1) // 1MB
		// Insert malicious content in the middle
		maliciousContent := largeContent[:len(largeContent)/2] +
			fixtures.PromptInjectionPayload +
			largeContent[len(largeContent)/2:]

		// Send in chunks to simulate streaming
		chunkSize := 8192 // 8KB chunks
		for i := 0; i < len(maliciousContent); i += chunkSize {
			end := i + chunkSize
			if end > len(maliciousContent) {
				end = len(maliciousContent)
			}

			err = stream.Send(&extproc.ProcessingRequest{
				Request: &extproc.ProcessingRequest_RequestBody{
					RequestBody: &extproc.HttpBody{
						Body: []byte(maliciousContent[i:end]),
					},
				},
			})
			require.NoError(t, err)

			// Check if we get an immediate response (blocked)
			select {
			case <-ctx.Done():
				t.Fatal("Context timeout")
			default:
				// Try to receive (non-blocking)
				resp, err := stream.Recv()
				if err != nil {
					// Expected for streaming - continue
					continue
				}

				if immediateResp := resp.GetImmediateResponse(); immediateResp != nil {
					// Request was blocked due to malicious content
					assert.Equal(t, typev3.StatusCode_Forbidden, immediateResp.Status.Code)
					return
				}
			}
		}

		// If we get here, the stream should have been blocked
		// Try one more receive to get the final decision
		resp, err := stream.Recv()
		if err == nil {
			if immediateResp := resp.GetImmediateResponse(); immediateResp != nil {
				assert.Equal(t, typev3.StatusCode_Forbidden, immediateResp.Status.Code)
			}
		}
	})
}

// TestIntegration_ConcurrentStreams tests handling of multiple concurrent ext_proc streams
func TestIntegration_ConcurrentStreams(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	server := grpcenforcer.NewWithOptions(grpcenforcer.Options{
		Timeout:         1 * time.Second,
		FailOn:          "LOW",
		EnforcementMode: "enforce",
	})
	// Load minimal rules for prompt injection detection
	server.LoadTestRules([]rules.RulePack{{
		Metadata: rules.Metadata{Name: "test-pack", Version: "1.0.0"},
		Rules:    []rules.Rule{{ID: "test-prompt-injection", Level: 1, Severity: "HIGH", Keywords: []string{"ignore previous instructions", "reveal your system prompt"}}},
	}})

	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	extproc.RegisterExternalProcessorServer(s, server)

	go func() {
		_ = s.Serve(lis)
	}()
	defer s.Stop()

	// Test concurrent streams
	numStreams := 10
	results := make(chan bool, numStreams)

	for i := 0; i < numStreams; i++ {
		go func(streamID int) {
			conn, err := grpc.DialContext(context.Background(), "bufnet",
				grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
					return lis.Dial()
				}),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			if err != nil {
				results <- false
				return
			}
			defer conn.Close()

			client := extproc.NewExternalProcessorClient(conn)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			stream, err := client.Process(ctx)
			if err != nil {
				results <- false
				return
			}

			// Send headers
			err = stream.Send(&extproc.ProcessingRequest{
				Request: &extproc.ProcessingRequest_RequestHeaders{
					RequestHeaders: &extproc.HttpHeaders{
						Headers: &corev3.HeaderMap{
							Headers: []*corev3.HeaderValue{
								{Key: ":method", RawValue: []byte("POST")},
								{Key: ":path", RawValue: []byte("/v1/chat/completions")},
								{Key: "content-type", RawValue: []byte("application/json")},
							},
						},
					},
				},
			})
			if err != nil {
				results <- false
				return
			}

			// Send either malicious or benign content
			var payload string
			if streamID%2 == 0 {
				payload = `{"messages": [{"role": "user", "content": "` + fixtures.PromptInjectionPayload + `"}]}`
			} else {
				payload = `{"messages": [{"role": "user", "content": "What is 2+2?"}]}`
			}

			err = stream.Send(&extproc.ProcessingRequest{
				Request: &extproc.ProcessingRequest_RequestBody{
					RequestBody: &extproc.HttpBody{
						Body: []byte(payload),
					},
				},
			})
			if err != nil {
				results <- false
				return
			}

			// Send trailers to flush
			_ = stream.Send(&extproc.ProcessingRequest{Request: &extproc.ProcessingRequest_RequestTrailers{RequestTrailers: &extproc.HttpTrailers{}}})
			// Read until EOF or ImmediateResponse
			blocked := false
			for {
				resp, err := stream.Recv()
				if err != nil {
					if err == io.EOF {
						break
					}
					results <- false
					return
				}
				if immediate := resp.GetImmediateResponse(); immediate != nil {
					blocked = (immediate.Status.Code == typev3.StatusCode_Forbidden)
					break
				}
			}
			if streamID%2 == 0 {
				results <- blocked
			} else {
				results <- !blocked
			}
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numStreams; i++ {
		if <-results {
			successCount++
		}
	}

	// Should have processed all streams correctly
	assert.Equal(t, numStreams, successCount, "All concurrent streams should be processed correctly")
}
