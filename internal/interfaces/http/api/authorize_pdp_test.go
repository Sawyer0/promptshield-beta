package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/pdp"
)

type stubPDPClient struct {
	resp       pdp.Response
	err        error
	lastReq    *pdp.Request
	sleepBefore time.Duration
}

func (s *stubPDPClient) Evaluate(ctx context.Context, req pdp.Request) (pdp.Response, error) {
	if s.sleepBefore > 0 {
		t := time.NewTimer(s.sleepBefore)
		select {
		case <-ctx.Done():
			return pdp.Response{}, ctx.Err()
		case <-t.C:
		}
	}
	s.lastReq = &req
	return s.resp, s.err
}

func TestAuthorizePDP_Matrix(t *testing.T) {
	mkReq := func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/x", bytes.NewReader([]byte("{}")))
		req.Header.Set("X-PS-User-ID", "u1")
		req.Header.Set("X-PS-Tenant-ID", uuid.New().String())
		return req
	}
	t.Run("no_pdp_configured_allows", func(t *testing.T) {
		req := mkReq()
		ok, reason := authorizePDP(req, "a", "res", "id", nil, true)
		if !ok || reason != "pdp_not_configured" {
			t.Fatalf("expected allow with reason pdp_not_configured, got ok=%v reason=%q", ok, reason)
		}
	})
	t.Run("error_fail_closed_denies", func(t *testing.T) {
		st := &stubPDPClient{err: context.DeadlineExceeded}
		ctx := context.WithValue(mkReq().Context(), pdpCtxKey{}, st)
		req := mkReq().WithContext(ctx)
		ok, reason := authorizePDP(req, "a", "res", "id", nil, true)
		if ok || reason != "pdp_error" {
			t.Fatalf("expected deny with pdp_error, got ok=%v reason=%q", ok, reason)
		}
	})
	t.Run("error_fail_open_allows", func(t *testing.T) {
		st := &stubPDPClient{err: context.DeadlineExceeded}
		ctx := context.WithValue(mkReq().Context(), pdpCtxKey{}, st)
		req := mkReq().WithContext(ctx)
		ok, reason := authorizePDP(req, "a", "res", "id", nil, false)
		if !ok || reason != "pdp_error_allow" {
			t.Fatalf("expected allow with pdp_error_allow, got ok=%v reason=%q", ok, reason)
		}
	})
	t.Run("permit_allows", func(t *testing.T) {
		st := &stubPDPClient{resp: pdp.Response{Decision: pdp.Permit, Reason: "permit"}}
		ctx := context.WithValue(mkReq().Context(), pdpCtxKey{}, st)
		req := mkReq().WithContext(ctx)
		ok, reason := authorizePDP(req, "a", "res", "id", nil, true)
		if !ok || reason != "permit" {
			t.Fatalf("expected allow with reason 'permit', got ok=%v reason=%q", ok, reason)
		}
	})
	t.Run("not_applicable_allows", func(t *testing.T) {
		st := &stubPDPClient{resp: pdp.Response{Decision: pdp.NotApplicable, Reason: "na"}}
		ctx := context.WithValue(mkReq().Context(), pdpCtxKey{}, st)
		req := mkReq().WithContext(ctx)
		ok, _ := authorizePDP(req, "a", "res", "id", nil, true)
		if !ok { t.Fatalf("expected allow for NotApplicable") }
	})
	t.Run("deny_denies", func(t *testing.T) {
		st := &stubPDPClient{resp: pdp.Response{Decision: pdp.Deny, Reason: "nope"}}
		ctx := context.WithValue(mkReq().Context(), pdpCtxKey{}, st)
		req := mkReq().WithContext(ctx)
		ok, _ := authorizePDP(req, "a", "res", "id", nil, true)
		if ok { t.Fatalf("expected deny for Deny") }
	})
	t.Run("indeterminate_denies", func(t *testing.T) {
		st := &stubPDPClient{resp: pdp.Response{Decision: pdp.Indeterminate, Reason: "err"}}
		ctx := context.WithValue(mkReq().Context(), pdpCtxKey{}, st)
		req := mkReq().WithContext(ctx)
		ok, _ := authorizePDP(req, "a", "res", "id", nil, true)
		if ok { t.Fatalf("expected deny for Indeterminate") }
	})
	t.Run("invalid_decision_fail_closed_denies", func(t *testing.T) {
		st := &stubPDPClient{resp: pdp.Response{Decision: pdp.Decision("UNKNOWN")}}
		ctx := context.WithValue(mkReq().Context(), pdpCtxKey{}, st)
		req := mkReq().WithContext(ctx)
		ok, reason := authorizePDP(req, "a", "res", "id", nil, true)
		if ok || reason != "pdp_invalid_decision" {
			t.Fatalf("expected deny with pdp_invalid_decision, got ok=%v reason=%q", ok, reason)
		}
	})
	t.Run("invalid_decision_fail_open_allows", func(t *testing.T) {
		st := &stubPDPClient{resp: pdp.Response{Decision: pdp.Decision("UNKNOWN")}}
		ctx := context.WithValue(mkReq().Context(), pdpCtxKey{}, st)
		req := mkReq().WithContext(ctx)
		ok, reason := authorizePDP(req, "a", "res", "id", nil, false)
		if !ok || reason != "pdp_invalid_decision_allow" {
			t.Fatalf("expected allow with pdp_invalid_decision_allow, got ok=%v reason=%q", ok, reason)
		}
	})
}

func TestAuthorizePDP_PIPAttributesMerged(t *testing.T) {
	st := &stubPDPClient{resp: pdp.Response{Decision: pdp.Permit, Reason: "ok"}}
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-PS-User-ID", "u1")
	req.Header.Set("X-PS-Tenant-ID", uuid.New().String())
	req.Header.Set("X-PS-Device", "ios-abc")
	req.Header.Set("X-Forwarded-For", "203.0.113.10")
	ctx := context.WithValue(req.Context(), pdpCtxKey{}, st)
	req = req.WithContext(ctx)

	ok, _ := authorizePDP(req, "foo", "bar", "id", map[string]any{"k":"v"}, true)
	if !ok { t.Fatalf("expected allow") }
	if st.lastReq == nil {
		t.Fatalf("expected stub to capture request")
	}
	attrs := st.lastReq.Environment.Attributes
	if attrs["device"] != "ios-abc" { t.Fatalf("missing device attribute in env: %#v", attrs) }
	if attrs["client_ip"] != "203.0.113.10" { t.Fatalf("missing client_ip attribute in env: %#v", attrs) }
}

func TestAuthorizePDP_PIPAttributesMissingAreIgnored(t *testing.T) {
	st := &stubPDPClient{resp: pdp.Response{Decision: pdp.Permit}}
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-PS-User-ID", "u1")
	req.Header.Set("X-PS-Tenant-ID", uuid.New().String())
	ctx := context.WithValue(req.Context(), pdpCtxKey{}, st)
	req = req.WithContext(ctx)
	ok, _ := authorizePDP(req, "a", "r", "id", nil, true)
	if !ok { t.Fatalf("expected allow") }
	if st.lastReq == nil { t.Fatalf("expected capture") }
	attrs := st.lastReq.Environment.Attributes
	if _, ok := attrs["device"]; ok { t.Fatalf("unexpected device attribute present: %#v", attrs) }
	if _, ok := attrs["client_ip"]; ok { t.Fatalf("unexpected client_ip attribute present: %#v", attrs) }
}
