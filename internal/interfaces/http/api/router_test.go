package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/application/services"
)

// mockRulepackRepo is a minimal mock for testing
type mockRulepackRepo struct{}

func (m *mockRulepackRepo) Create(ctx context.Context, tenantID uuid.UUID, name, desc string) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *mockRulepackRepo) CreateVersion(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, status string, createdBy uuid.UUID) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *mockRulepackRepo) GetActive(ctx context.Context, packID uuid.UUID) (json.RawMessage, int, error) {
	return json.RawMessage("{}"), 1, nil
}

func (m *mockRulepackRepo) Activate(ctx context.Context, packID, versionID uuid.UUID) error {
	return nil
}

func (m *mockRulepackRepo) GetVersion(ctx context.Context, packID uuid.UUID, version int) (json.RawMessage, string, error) {
	return json.RawMessage("{}"), "approved", nil
}

func (m *mockRulepackRepo) GetLatestVersion(ctx context.Context, packID uuid.UUID) (uuid.UUID, int, error) {
	return uuid.New(), 1, nil
}

func (m *mockRulepackRepo) ActivateLatest(ctx context.Context, packID uuid.UUID) error {
	return nil
}

func (m *mockRulepackRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]services.RulepackInfo, error) {
	return []services.RulepackInfo{}, nil
}

func (m *mockRulepackRepo) Delete(ctx context.Context, packID uuid.UUID) error {
	return nil
}

func (m *mockRulepackRepo) CreateVersionActivateTx(ctx context.Context, packID uuid.UUID, version int, dsl json.RawMessage, createdBy uuid.UUID) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *mockRulepackRepo) PurgeOldVersions(ctx context.Context, packID uuid.UUID, keep int) error {
	return nil
}

func TestVersion(t *testing.T) {
	mockRepo := &mockRulepackRepo{}
	svc := services.NewRulepackService(mockRepo, nil)
	srv := httptest.NewServer(NewMux(Options{AllowInsecureAdmin: true, RulepackService: svc}))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/version")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
}

func TestRulepacksListEmpty(t *testing.T) {
	// Create a mock service that returns empty list
	mockRepo := &mockRulepackRepo{}
	svc := &services.RulepackService{}
	svc = services.NewRulepackService(mockRepo, nil)

	srv := httptest.NewServer(NewMux(Options{AllowInsecureAdmin: true, RulepackService: svc}))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/rulepacks/")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
}

func TestConfigGetPutReset(t *testing.T) {
	mockRepo := &mockRulepackRepo{}
	svc := services.NewRulepackService(mockRepo, nil)
	srv := httptest.NewServer(NewMux(Options{AllowInsecureAdmin: true, AdminToken: "x", RulepackService: svc}))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/config/")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	req, _ := json.Marshal(map[string]any{"enforcement_mode": "enforce"})
	reqResp, err := http.NewRequest(http.MethodPut, srv.URL+"/config/", bytes.NewReader(req))
	if err != nil {
		t.Fatal(err)
	}
	reqResp.Header.Set("Authorization", "Bearer x")
	res2, err := http.DefaultClient.Do(reqResp)
	if err != nil {
		t.Fatal(err)
	}
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", res2.StatusCode)
	}
	resetReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/config/reset", nil)
	resetReq.Header.Set("Authorization", "Bearer x")
	res3, err := http.DefaultClient.Do(resetReq)
	if err != nil {
		t.Fatal(err)
	}
	if res3.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", res3.StatusCode)
	}
}

func TestCheckAllow(t *testing.T) {
	mockRepo := &mockRulepackRepo{}
	svc := services.NewRulepackService(mockRepo, nil)
	srv := httptest.NewServer(NewMux(Options{AllowInsecureAdmin: true, RulepackService: svc}))
	defer srv.Close()
	resp, err := http.Post(srv.URL+"/check", "text/plain", bytes.NewBufferString("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
}

func TestScanAggregateJSONAndNDJSON(t *testing.T) {
	mockRepo := &mockRulepackRepo{}
	svc := services.NewRulepackService(mockRepo, nil)
	srv := httptest.NewServer(NewMux(Options{AllowInsecureAdmin: true, RulepackService: svc}))
	defer srv.Close()

	// Aggregate JSON array input
	arr := []string{"hello", "world"}
	b, _ := json.Marshal(arr)
	resp, err := http.Post(srv.URL+"/scan", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("aggregate: want 200, got %d", resp.StatusCode)
	}
	var agg map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&agg); err != nil {
		t.Fatalf("decode aggregate: %v", err)
	}
	dec, _ := agg["decisions"].([]any)
	if len(dec) != 2 {
		t.Fatalf("expected 2 decisions, got %d", len(dec))
	}
	if _, ok := agg["summary"].(map[string]any); !ok {
		t.Fatalf("expected summary object in response")
	}

	// NDJSON streaming with aggregate=false
	nd := "one\nsecond\n"
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/scan?aggregate=false", strings.NewReader(nd))
	req.Header.Set("Content-Type", "application/x-ndjson")
	sresp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer sresp.Body.Close()
	if ct := sresp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/x-ndjson") {
		t.Fatalf("expected content-type application/x-ndjson, got %q", ct)
	}
	rd := bufio.NewReader(sresp.Body)
	// read two JSON lines
	for i := 0; i < 2; i++ {
		line, err := rd.ReadString('\n')
		if err != nil {
			t.Fatalf("reading ndjson line %d: %v", i+1, err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			t.Fatalf("empty ndjson line %d", i+1)
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Fatalf("invalid json line %d: %v", i+1, err)
		}
	}
}

func TestSSEEventStreamingAndFiltering(t *testing.T) {
	hub := NewEventHub()
	mockRepo := &mockRulepackRepo{}
	svc := services.NewRulepackService(mockRepo, nil)
	srv := httptest.NewServer(NewMux(Options{AllowInsecureAdmin: true, Events: hub, RulepackService: svc}))
	defer srv.Close()

	// Start SSE client
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/events?types=decision", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	br := bufio.NewReader(resp.Body)

	// Wait a bit to ensure subscription is active, then publish events
	time.AfterFunc(50*time.Millisecond, func() {
		hub.Publish(Event{Type: "other", Data: map[string]any{"k": "v"}})
		hub.Publish(Event{Type: "decision", Data: map[string]any{"decision": "allow"}})
	})

	deadline := time.After(2 * time.Second)
	sawReady := false
	sawDecision := false
	for !sawDecision {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for decision event (sawReady=%v)", sawReady)
		default:
			line, err := br.ReadString('\n')
			if err != nil {
				t.Fatalf("read sse: %v", err)
			}
			if strings.HasPrefix(line, "event: ready") {
				sawReady = true
			}
			if strings.HasPrefix(line, "event: decision") {
				sawDecision = true
			}
		}
	}
}

func TestRulepacksUploadMultipart(t *testing.T) {
	// Create a mock service
	mockRepo := &mockRulepackRepo{}
	svc := services.NewRulepackService(mockRepo, nil)

	srv := httptest.NewServer(NewMux(Options{AllowInsecureAdmin: true, RulepackService: svc}))
	defer srv.Close()

	// load sample rulepack
	data, err := os.ReadFile("rules/basic-security.yaml")
	if err != nil {
		t.Skipf("sample rulepack not found: %v", err)
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "basic-security.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(fw, bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	_ = mw.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/rulepacks/?activate=true", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("want 201, got %d", resp.StatusCode)
	}
	var meta struct {
		ID     string `json:"id"`
		Active bool   `json:"active"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		t.Fatalf("decode meta: %v", err)
	}
	if meta.ID == "" || !meta.Active {
		t.Fatalf("unexpected meta: %+v", meta)
	}
}

func TestAdminShutdownDelayHandling(t *testing.T) {
	delays := make(chan time.Duration, 1)
	mockRepo := &mockRulepackRepo{}
	svc := services.NewRulepackService(mockRepo, nil)
	srv := httptest.NewServer(NewMux(Options{
		AllowInsecureAdmin: true,
		RulepackService:    svc,
		OnShutdown: func(ctx context.Context, d time.Duration) error {
			delays <- d
			return nil
		},
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/admin/shutdown?delay=2", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("want 202, got %d", resp.StatusCode)
	}
	select {
	case d := <-delays:
		if d != 2*time.Second {
			t.Fatalf("expected delay 2s, got %v", d)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for shutdown callback")
	}
}
