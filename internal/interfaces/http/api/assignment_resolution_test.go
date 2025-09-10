package api

import "testing"

func TestMatchesEndpointPattern(t *testing.T) {
    cases := []struct{
        path string
        pattern string
        want bool
    }{
        {"/v1/chat/completions", "/", true},
        {"/v1/chat/completions", "*", true},
        {"/v1/chat/completions", "/v1/*", true},
        {"/v1", "/v1/*", true},
        {"/v1/", "/v1/*", true},
        {"/api/orders/123", "/api/orders/*", true},
        {"/api/orders", "/api/orders/*", true},
        {"/api/orders", "/api/orders", true},
        {"/api/payments/refund", "/api/payments/refund", true},
        {"/api/payments/refund/extra", "/api/payments/refund", false},
        {"/v2/chat/completions", "/v1/*", false},
        {"v1/chat/completions", "/v1/*", true},
    }
    for _, tc := range cases {
        if got := matchesEndpointPattern(tc.path, tc.pattern); got != tc.want {
            t.Fatalf("matchesEndpointPattern(%q, %q) = %v; want %v", tc.path, tc.pattern, got, tc.want)
        }
    }
}

