package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequestTimeoutExemptsOnlyTheEventStream(t *testing.T) {
	cases := []struct {
		method, path string
		wantDeadline bool
	}{
		{http.MethodGet, "/v1/events", false},
		{http.MethodGet, "/events", false},
		{http.MethodGet, "/v1/decisions", true},
		{http.MethodGet, "/v1/stats", true},
		{http.MethodPost, "/v1/events", true},
		{http.MethodGet, "/v1/events/x", true},
	}
	for _, tc := range cases {
		var got bool
		h := RequestTimeout(10 * time.Second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, got = r.Context().Deadline()
		}))
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(tc.method, tc.path, nil))
		if got != tc.wantDeadline {
			t.Errorf("%s %s: deadline=%v, want %v", tc.method, tc.path, got, tc.wantDeadline)
		}
	}
}

func TestRequestTimeoutAnswers504OnSlowRoutes(t *testing.T) {
	h := RequestTimeout(20 * time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/decisions", nil))
	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("status %d, want 504", rec.Code)
	}
}
