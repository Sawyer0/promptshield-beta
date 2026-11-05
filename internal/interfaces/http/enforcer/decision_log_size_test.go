package enforcerhttp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// TestDecisionLogSizeFromEnv drives the gateway past a PS_DECISION_LOG_SIZE
// of 10 and checks the API retains exactly the newest 10 while counting all.
func TestDecisionLogSizeFromEnv(t *testing.T) {
	t.Setenv("PS_DECISION_LOG_SIZE", "10")
	srv, _ := newFlowServer(t)

	var ids []string
	for i := 0; i < 13; i++ {
		_, id := postCheck(t, srv, fmt.Sprintf("hello %d", i))
		ids = append(ids, id)
	}

	resp := adminRequest(t, srv, http.MethodGet, "/v1/decisions")
	var list decisionList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	got := list.Data.Decisions
	if len(got) != 10 || got[0].RequestID != ids[12] || got[9].RequestID != ids[3] {
		t.Fatalf("retained %d decisions, newest %q oldest %q", len(got), got[0].RequestID, got[len(got)-1].RequestID)
	}

	resp = adminRequest(t, srv, http.MethodGet, "/v1/decisions/stats")
	var stats struct {
		Data struct {
			Total    int64 `json:"total"`
			Retained int   `json:"retained"`
			Capacity int   `json:"capacity"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if stats.Data.Capacity != 10 || stats.Data.Retained != 10 || stats.Data.Total != 13 {
		t.Fatalf("stats = %+v", stats.Data)
	}

	// The evicted oldest decision is gone from the detail lookup too.
	resp = adminRequest(t, srv, http.MethodGet, "/v1/decisions/"+ids[0])
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("evicted decision lookup: %d", resp.StatusCode)
	}
}
