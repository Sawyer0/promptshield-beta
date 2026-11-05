package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/promptshield/promptshield/internal/application/decisions"
)

type decisionPage struct {
	Decisions []decisions.Record `json:"decisions"`
	Next      *int64             `json:"next"`
}

func fillLog(log *decisions.Log, n int, decide func(i int) string) {
	for i := 1; i <= n; i++ {
		log.Add(decisions.Record{RequestID: fmt.Sprintf("req-%05d", i), Decision: decide(i)})
	}
}

func allowAll(int) string { return decisions.Allow }

// pageThrough follows next cursors from the first page until next is null,
// returning every page. It fails the test if a listing never ends.
func pageThrough(t *testing.T, srv *httptest.Server, query string) []decisionPage {
	t.Helper()
	var pages []decisionPage
	path := "/decisions?" + query
	for range 1000 {
		code, body := adminGet[decisionPage](t, srv, path)
		if code != http.StatusOK {
			t.Fatalf("GET %s: code=%d error=%+v", path, code, body.Error)
		}
		pages = append(pages, body.Data)
		if body.Data.Next == nil {
			return pages
		}
		path = fmt.Sprintf("/decisions?%s&before=%d", query, *body.Data.Next)
	}
	t.Fatal("paging did not terminate")
	return nil
}

func flatten(pages []decisionPage) []decisions.Record {
	var out []decisions.Record
	for _, p := range pages {
		out = append(out, p.Decisions...)
	}
	return out
}

// assertDescendingRun checks records are exactly seq from..to, newest first:
// no gaps, no duplicates, and request ids that belong to those seqs.
func assertDescendingRun(t *testing.T, got []decisions.Record, from, to int64) {
	t.Helper()
	if want := int(from - to + 1); len(got) != want {
		t.Fatalf("got %d records, want %d", len(got), want)
	}
	seen := make(map[string]bool, len(got))
	for i, r := range got {
		want := from - int64(i)
		if r.Seq != want || r.RequestID != fmt.Sprintf("req-%05d", want) {
			t.Fatalf("record %d: seq=%d id=%s, want seq %d", i, r.Seq, r.RequestID, want)
		}
		if seen[r.RequestID] {
			t.Fatalf("duplicate %s", r.RequestID)
		}
		seen[r.RequestID] = true
	}
}

func TestDecisionsPageThroughLogLargerThanOneRequest(t *testing.T) {
	log := decisions.NewLog(1200)
	fillLog(log, 1200, allowAll)
	srv := httptest.NewServer(testRouterWithOptions(Options{AdminToken: testAdminToken, Decisions: log}))
	defer srv.Close()

	pages := pageThrough(t, srv, "limit=500")
	if len(pages) != 3 || len(pages[0].Decisions) != 500 || len(pages[1].Decisions) != 500 || len(pages[2].Decisions) != 200 {
		t.Fatalf("want pages of 500, 500, 200; got %d pages", len(pages))
	}
	if *pages[0].Next != 701 || *pages[1].Next != 201 {
		t.Fatalf("next cursors: %d, %d", *pages[0].Next, *pages[1].Next)
	}
	assertDescendingRun(t, flatten(pages), 1200, 1)
}

func TestDecisionsLastPageHasNullNextOnExactMultiple(t *testing.T) {
	log := decisions.NewLog(1000)
	fillLog(log, 1000, allowAll)
	srv := httptest.NewServer(testRouterWithOptions(Options{AdminToken: testAdminToken, Decisions: log}))
	defer srv.Close()

	pages := pageThrough(t, srv, "limit=500")
	if len(pages) != 2 || len(pages[1].Decisions) != 500 {
		t.Fatalf("want exactly 2 full pages and no empty third page, got %d", len(pages))
	}
	assertDescendingRun(t, flatten(pages), 1000, 1)

	_, raw := adminGet[map[string]any](t, srv, "/decisions?limit=500&before=501")
	if v, ok := raw.Data["next"]; !ok || v != nil {
		t.Fatalf("last page must carry \"next\": null, got %v (present=%v)", v, ok)
	}
}

func TestDecisionsPagingAfterEvictionStopsAtOldestRetained(t *testing.T) {
	log := decisions.NewLog(600)
	fillLog(log, 1000, allowAll)
	srv := httptest.NewServer(testRouterWithOptions(Options{AdminToken: testAdminToken, Decisions: log}))
	defer srv.Close()

	assertDescendingRun(t, flatten(pageThrough(t, srv, "limit=250")), 1000, 401)

	code, stale := adminGet[decisionPage](t, srv, "/decisions?before=401")
	if code != http.StatusOK || len(stale.Data.Decisions) != 0 || stale.Data.Next != nil {
		t.Fatalf("cursor past retention: code=%d body=%+v", code, stale.Data)
	}
}

func TestDecisionsPagingIsStableWhileNewDecisionsArrive(t *testing.T) {
	log := decisions.NewLog(5000)
	fillLog(log, 900, allowAll)
	srv := httptest.NewServer(testRouterWithOptions(Options{AdminToken: testAdminToken, Decisions: log}))
	defer srv.Close()

	_, first := adminGet[decisionPage](t, srv, "/decisions?limit=300")
	got := first.Data.Decisions
	next := first.Data.Next
	for next != nil {
		// Traffic keeps flowing between page requests.
		for i := range 50 {
			log.Add(decisions.Record{RequestID: fmt.Sprintf("late-%d-%d", *next, i), Decision: decisions.Allow})
		}
		_, page := adminGet[decisionPage](t, srv, fmt.Sprintf("/decisions?limit=300&before=%d", *next))
		got = append(got, page.Data.Decisions...)
		next = page.Data.Next
	}
	assertDescendingRun(t, got, 900, 1)
}

func TestDecisionsPagingWithOutcomeFilter(t *testing.T) {
	log := decisions.NewLog(2000)
	denyEveryThird := func(i int) string {
		if i%3 == 0 {
			return decisions.Deny
		}
		return decisions.Allow
	}
	fillLog(log, 1500, denyEveryThird)
	srv := httptest.NewServer(testRouterWithOptions(Options{AdminToken: testAdminToken, Decisions: log}))
	defer srv.Close()

	pages := pageThrough(t, srv, "limit=200&decision=deny")
	got := flatten(pages)
	if len(pages) != 3 || len(got) != 500 {
		t.Fatalf("want 500 denies over 3 pages, got %d over %d", len(got), len(pages))
	}
	for i, r := range got {
		if want := int64(1500 - 3*i); r.Seq != want || r.Decision != decisions.Deny {
			t.Fatalf("record %d: seq=%d decision=%s, want deny seq %d", i, r.Seq, r.Decision, want)
		}
	}
}

func TestDecisionsRejectsBadCursor(t *testing.T) {
	log := decisions.NewLog(10)
	fillLog(log, 5, allowAll)
	srv := httptest.NewServer(testRouterWithOptions(Options{AdminToken: testAdminToken, Decisions: log}))
	defer srv.Close()
	for _, q := range []string{"before=abc", "before=0", "before=-3", "before=1.5", "before=9223372036854775808", "before=req-00001"} {
		code, body := adminGet[map[string]any](t, srv, "/decisions?"+q)
		if code != http.StatusBadRequest || body.Error == nil || body.Error.Code != "INVALID_ARGUMENT" {
			t.Errorf("%s: want 400 INVALID_ARGUMENT, got %d %+v", q, code, body)
		}
	}
}

func TestDecisionsPagingRequiresAdminToken(t *testing.T) {
	log := decisions.NewLog(10)
	fillLog(log, 5, allowAll)
	srv := httptest.NewServer(testRouterWithOptions(Options{AdminToken: testAdminToken, Decisions: log}))
	defer srv.Close()
	for _, header := range []string{"", "Bearer wrong"} {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/decisions?limit=2&before=4", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("auth %q: want 401, got %d", header, resp.StatusCode)
		}
	}
	code, ok := adminGet[decisionPage](t, srv, "/decisions?limit=2&before=4")
	if code != http.StatusOK || len(ok.Data.Decisions) != 2 || ok.Data.Decisions[0].Seq != 3 || *ok.Data.Next != 2 {
		t.Fatalf("authorized page: code=%d body=%+v", code, ok.Data)
	}
}
