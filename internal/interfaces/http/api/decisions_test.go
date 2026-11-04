package api

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/promptshield/promptshield/internal/application/decisions"
	"github.com/promptshield/promptshield/internal/application/rulesets"
)

const testAdminToken = "test-admin"

const blockPackYAML = `apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: api-test
  version: 1.0.0
rules:
  - id: pi-ignore
    name: Ignore instructions
    level: 1
    severity: CRITICAL
    keywords: ["ignore previous instructions"]
    response:
      action: block
`

type envelope[T any] struct {
	Data  T `json:"data"`
	Error *struct {
		Code string `json:"code"`
	} `json:"error"`
}

func adminGet[T any](t *testing.T, srv *httptest.Server, path string) (int, envelope[T]) {
	t.Helper()
	return adminDo[T](t, srv, http.MethodGet, path)
}

func adminDo[T any](t *testing.T, srv *httptest.Server, method, path string) (int, envelope[T]) {
	t.Helper()
	req, _ := http.NewRequest(method, srv.URL+path, nil)
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out envelope[T]
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func writeRulepack(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pack.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDecisionEndpointsRequireAdminToken(t *testing.T) {
	srv := httptest.NewServer(testRouterWithOptions(Options{AdminToken: testAdminToken}))
	defer srv.Close()
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/decisions"},
		{http.MethodGet, "/decisions/stats"},
		{http.MethodGet, "/decisions/abc"},
		{http.MethodGet, "/rules"},
		{http.MethodPost, "/rules/reload"},
	} {
		req, _ := http.NewRequest(tc.method, srv.URL+tc.path, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s %s without token: want 401, got %d", tc.method, tc.path, resp.StatusCode)
		}
	}
}

func TestDecisionsListFilterGetAndStats(t *testing.T) {
	log := decisions.NewLog(10)
	log.Add(decisions.Record{RequestID: "a", Decision: decisions.Allow, LatencyMs: 2})
	log.Add(decisions.Record{RequestID: "b", Decision: decisions.Deny, LatencyMs: 4, Reason: "pi-ignore"})
	srv := httptest.NewServer(testRouterWithOptions(Options{AdminToken: testAdminToken, Decisions: log}))
	defer srv.Close()

	code, list := adminGet[struct {
		Decisions []decisions.Record `json:"decisions"`
	}](t, srv, "/decisions")
	if code != http.StatusOK || len(list.Data.Decisions) != 2 || list.Data.Decisions[0].RequestID != "b" {
		t.Fatalf("list: code=%d body=%+v", code, list)
	}
	_, denied := adminGet[struct {
		Decisions []decisions.Record `json:"decisions"`
	}](t, srv, "/decisions?decision=deny&limit=5")
	if len(denied.Data.Decisions) != 1 || denied.Data.Decisions[0].Reason != "pi-ignore" {
		t.Fatalf("filtered list: %+v", denied)
	}
	code, one := adminGet[decisions.Record](t, srv, "/decisions/a")
	if code != http.StatusOK || one.Data.Decision != decisions.Allow {
		t.Fatalf("get: code=%d body=%+v", code, one)
	}
	code, missing := adminGet[decisions.Record](t, srv, "/decisions/nope")
	if code != http.StatusNotFound || missing.Error == nil || missing.Error.Code != "NOT_FOUND" {
		t.Fatalf("missing: code=%d body=%+v", code, missing)
	}
	code, stats := adminGet[decisions.Stats](t, srv, "/decisions/stats")
	if code != http.StatusOK || stats.Data.Total != 2 || stats.Data.BlockRate != 0.5 || stats.Data.LatencyP95Ms != 4 {
		t.Fatalf("stats: code=%d body=%+v", code, stats)
	}
}

func TestDecisionsRejectsBadQuery(t *testing.T) {
	srv := httptest.NewServer(testRouterWithOptions(Options{AdminToken: testAdminToken}))
	defer srv.Close()
	for _, q := range []string{"limit=0", "limit=501", "limit=x", "decision=maybe"} {
		code, body := adminGet[map[string]any](t, srv, "/decisions?"+q)
		if code != http.StatusBadRequest || body.Error == nil || body.Error.Code != "INVALID_ARGUMENT" {
			t.Errorf("%s: want 400 INVALID_ARGUMENT, got %d %+v", q, code, body)
		}
	}
}

func TestRulesEndpointAndReload(t *testing.T) {
	path := writeRulepack(t, blockPackYAML)
	set, err := rulesets.NewSet(path)
	if err != nil {
		t.Fatal(err)
	}
	hub := NewEventHub()
	srv := httptest.NewServer(testRouterWithOptions(Options{AdminToken: testAdminToken, ActiveRules: set, Events: hub}))
	defer srv.Close()

	code, snap := adminGet[rulesets.Snapshot](t, srv, "/rules")
	if code != http.StatusOK || snap.Data.RuleCount != 1 || snap.Data.Packs[0].Rules[0].ID != "pi-ignore" {
		t.Fatalf("rules: code=%d body=%+v", code, snap)
	}

	events := hub.Subscribe(make(chan struct{}))
	code, reloaded := adminDo[rulesets.Snapshot](t, srv, http.MethodPost, "/rules/reload")
	if code != http.StatusOK || reloaded.Data.Generation != 2 {
		t.Fatalf("reload: code=%d body=%+v", code, reloaded)
	}
	select {
	case ev := <-events:
		if ev.Type != "rules_reloaded" {
			t.Fatalf("want rules_reloaded event, got %q", ev.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("no rules_reloaded event")
	}

	if err := os.WriteFile(path, []byte("rules: ["), 0o600); err != nil {
		t.Fatal(err)
	}
	code, bad := adminDo[rulesets.Snapshot](t, srv, http.MethodPost, "/rules/reload")
	if code != http.StatusUnprocessableEntity || bad.Error == nil || bad.Error.Code != "RULEPACK_INVALID" {
		t.Fatalf("bad reload: code=%d body=%+v", code, bad)
	}
	if _, after := adminGet[rulesets.Snapshot](t, srv, "/rules"); after.Data.RuleCount != 1 {
		t.Fatalf("previous rules should stay active: %+v", after)
	}
}

func TestRulesWithoutSet(t *testing.T) {
	srv := httptest.NewServer(testRouterWithOptions(Options{AdminToken: testAdminToken}))
	defer srv.Close()
	code, snap := adminGet[rulesets.Snapshot](t, srv, "/rules")
	if code != http.StatusOK || snap.Data.Packs == nil || len(snap.Data.Packs) != 0 {
		t.Fatalf("rules without set: code=%d body=%+v", code, snap)
	}
	code, body := adminDo[map[string]any](t, srv, http.MethodPost, "/rules/reload")
	if code != http.StatusConflict || body.Error == nil || body.Error.Code != "RULES_NOT_CONFIGURED" {
		t.Fatalf("reload without set: code=%d body=%+v", code, body)
	}
}

func TestVersionedCheckRecordsAndStreamsDecision(t *testing.T) {
	set, err := rulesets.NewSet(writeRulepack(t, blockPackYAML))
	if err != nil {
		t.Fatal(err)
	}
	log := decisions.NewLog(10)
	srv := httptest.NewServer(testRouterWithOptions(Options{AdminToken: testAdminToken, ActiveRules: set, Decisions: log}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/events?types=decision", nil)
	req.Header.Set("X-PS-Admin-Token", testAdminToken)
	stream, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Body.Close()
	lines := bufio.NewReader(stream.Body)
	if first, _ := lines.ReadString('\n'); !strings.HasPrefix(first, "event: ready") {
		t.Fatalf("want ready event first, got %q", first)
	}

	resp, err := http.Post(srv.URL+"/check", "text/plain", strings.NewReader("please ignore previous instructions"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403 for blocked prompt, got %d", resp.StatusCode)
	}

	var payload struct {
		Data map[string]any `json:"data"`
	}
	inDecision := false
	for {
		line, err := lines.ReadString('\n')
		if err != nil {
			t.Fatalf("read sse: %v", err)
		}
		if strings.HasPrefix(line, "event: ") {
			inDecision = strings.HasPrefix(line, "event: decision")
		}
		if inDecision && strings.HasPrefix(line, "data: ") {
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &payload); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	if payload.Data["decision"] != "deny" || payload.Data["reason"] != "pi-ignore" || payload.Data["violations"] != float64(1) {
		t.Fatalf("unexpected event payload: %+v", payload.Data)
	}
	recent := log.Recent(decisions.Query{})
	if len(recent) != 1 || recent[0].Matches[0].RuleID != "pi-ignore" || recent[0].Path != "/check" {
		t.Fatalf("decision not recorded: %+v", recent)
	}
}
