package api

import (
    "context"
    "math"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
    "time"

    pkg "github.com/promptshield/promptshield/pkg/types"
)

func TestComputeFinalScore_Weights(t *testing.T) {
    t.Setenv("PS_ALPHA", "0.7")
    t.Setenv("PS_BETA", "0.3")

    r := pkg.ScanResult{
        Violations: []pkg.Violation{
            {Level: 3, Confidence: 0.80}, // L3 risk
            {Level: 2},                   // L2 pattern
        },
    }
    final, risk, pattern := computeFinalScore(&r)
    if risk != 0.80 {
        t.Fatalf("risk = %v, want 0.80", risk)
    }
    if pattern != 1.0 {
        t.Fatalf("pattern = %v, want 1.0", pattern)
    }
    want := 0.7*0.80 + 0.3*1.0 // 0.86
    if math.Abs(final-want) > 1e-9 {
        t.Fatalf("final = %v, want %v", final, want)
    }

    // Only L1 pattern
    r2 := pkg.ScanResult{Violations: []pkg.Violation{{Level: 1}}}
    final2, risk2, pattern2 := computeFinalScore(&r2)
    if risk2 != 0 {
        t.Fatalf("risk2 = %v, want 0", risk2)
    }
    if pattern2 != 0.5 {
        t.Fatalf("pattern2 = %v, want 0.5", pattern2)
    }
    want2 := 0.7*0 + 0.3*0.5
    if math.Abs(final2-want2) > 1e-9 {
        t.Fatalf("final2 = %v, want %v", final2, want2)
    }

    // Only L2 pattern
    r3 := pkg.ScanResult{Violations: []pkg.Violation{{Level: 2}}}
    _, risk3, pattern3 := computeFinalScore(&r3)
    if risk3 != 0 || pattern3 != 1.0 {
        t.Fatalf("expected risk=0, pattern=1.0; got risk=%v pattern=%v", risk3, pattern3)
    }

    // Multiple L3; risk = max confidence
    r4 := pkg.ScanResult{Violations: []pkg.Violation{{Level: 3, Confidence: 0.2}, {Level: 3, Confidence: 0.9}, {Level: 3, Confidence: 0.7}}}
    _, risk4, pattern4 := computeFinalScore(&r4)
    if risk4 != 0.9 || pattern4 != 0.0 {
        t.Fatalf("expected risk=0.9, pattern=0.0; got risk=%v pattern=%v", risk4, pattern4)
    }

    // Different weights
    t.Setenv("PS_ALPHA", "0.5")
    t.Setenv("PS_BETA", "0.5")
    final5, _, _ := computeFinalScore(&r)
    if math.Abs(final5-(0.5*0.80+0.5*1.0)) > 1e-9 {
        t.Fatalf("final5 mismatch: %v", final5)
    }
}

func TestRunScanLine_NoRationaleLeak(t *testing.T) {
    // Ensure semantic disabled for a clean run
    t.Setenv("PS_SEMANTIC_ENABLED", "false")
m := runScanLine(context.Background(), []byte("hello"), "conv-test", Options{}, "", http.MethodPost)
    if _, ok := m["rationale"]; ok {
        t.Fatalf("rationale should not be present in response: %v", m)
    }
    if _, ok := m["decision"]; !ok {
        t.Fatalf("decision missing in response")
    }
    if _, ok := m["reason"]; !ok {
        t.Fatalf("reason missing in response")
    }
    if _, ok := m["violations"]; !ok {
        t.Fatalf("violations missing in response")
    }
}

// Stub scanner manager for /check policy bridge tests
type stubScannerMgr struct{ res pkg.ScanResult }
func (s stubScannerMgr) HasActivePolicies() bool { return true }
func (s stubScannerMgr) ScanReader(ctx context.Context, reader interface{}, inputName string) (pkg.ScanResult, error) {
    return s.res, nil
}
func (s stubScannerMgr) ReloadRulepacks() error { return nil }

func TestPolicyBridge_CheckRoute_TriggersQuarantine(t *testing.T) {
    // α=0.7, β=0.3, threshold 0.75; risk=0.9 => final=0.7*0.9=0.63; need pattern contribution
    // set to α=0.9 β=0.1 so final=0.9*0.9=0.81 >= 0.8 threshold
    t.Setenv("PS_ALPHA", "0.9")
    t.Setenv("PS_BETA", "0.1")
    t.Setenv("PS_BLOCK_THRESHOLD", "0.80")

    // No response actions that would already block; only L3 confidence
    res := pkg.ScanResult{Violations: []pkg.Violation{{Level: 3, Confidence: 0.90}}}

    opt := Options{ScannerManager: stubScannerMgr{res: res}}
    h := checkHandlerVersioned(opt)

    req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader("{}"))
    req.Header.Set("X-PS-Tenant-ID", "00000000-0000-0000-0000-000000000001")
    rr := httptest.NewRecorder()

    h.ServeHTTP(rr, req)

    if rr.Code != http.StatusForbidden {
        t.Fatalf("expected 403, got %d", rr.Code)
    }
    // Confirm decision and reason headers
    if got := rr.Header().Get("x-ps-decision"); got != "quarantine" {
        t.Fatalf("expected x-ps-decision=quarantine, got %s", got)
    }
    if got := rr.Header().Get("x-ps-reason"); got != "policy_bridge_threshold" {
        t.Fatalf("expected x-ps-reason=policy_bridge_threshold, got %s", got)
    }
}

func TestPolicyBridge_CheckRoute_DoesNotOverrideBlock(t *testing.T) {
    t.Setenv("PS_ALPHA", "0.9")
    t.Setenv("PS_BETA", "0.1")
    t.Setenv("PS_BLOCK_THRESHOLD", "0.50")

    // ResponseAction="deny" should already block irrespective of policy bridge
    res := pkg.ScanResult{Violations: []pkg.Violation{{Level: 1, ResponseAction: "deny"}}}
    opt := Options{ScannerManager: stubScannerMgr{res: res}}
    h := checkHandlerVersioned(opt)

    req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader("{}"))
    req.Header.Set("X-PS-Tenant-ID", "00000000-0000-0000-0000-000000000001")
    rr := httptest.NewRecorder()

    h.ServeHTTP(rr, req)
    if rr.Code != http.StatusForbidden {
        t.Fatalf("expected 403 due to explicit deny, got %d", rr.Code)
    }
    if got := rr.Header().Get("x-ps-reason"); got == "policy_bridge_threshold" {
        t.Fatalf("policy bridge should not override explicit deny")
    }
}

func TestConversationPruneTTL(t *testing.T) {
    // Insert an expired conversation entry
    convStore.Store("old", &convState{LastText: "x", UpdatedAt: time.Now().Add(-time.Hour)})

    t.Setenv("PS_CONV_TTL", "1ns")
_ = runScanLine(context.Background(), []byte("hi"), "new", Options{}, "", http.MethodPost)

    // Give a tiny moment
    time.Sleep(1 * time.Millisecond)

    if _, ok := convStore.Load("old"); ok {
        t.Fatalf("expected expired conversation 'old' to be pruned")
    }
    if _, ok := convStore.Load("new"); !ok {
        t.Fatalf("expected 'new' conversation to be stored")
    }
}

