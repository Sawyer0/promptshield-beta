package api

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/promptshield/promptshield/internal/pdp"
	pdphttp "github.com/promptshield/promptshield/internal/pdp/httpclient"
)

type pdpCtxKey struct{}

var (
	pdpOnce   sync.Once
	pdpClient pdp.Client
)

func initPDP() {
	endpoint := strings.TrimSpace(os.Getenv("PS_PDP_ENDPOINT"))
	apiKey := strings.TrimSpace(os.Getenv("PS_PDP_API_KEY"))
	to := 2000 * time.Millisecond
	if v := strings.TrimSpace(os.Getenv("PS_PDP_TIMEOUT_MS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			to = time.Duration(n) * time.Millisecond
		}
	}
	if endpoint == "" {
		slog.Info("PDP disabled: PS_PDP_ENDPOINT not set")
		return
	}
	pdpClient = pdphttp.New(pdphttp.Config{Endpoint: endpoint, APIKey: apiKey, Timeout: to})
	slog.Info("PDP enabled", "endpoint", endpoint, "timeout_ms", to.Milliseconds())
}

// pdpMiddleware attaches a pdp.Client to the request context if configured.
func pdpMiddleware(next http.Handler) http.Handler {
	pdpOnce.Do(initPDP)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if pdpClient == nil {
			next.ServeHTTP(w, r)
			return
		}
		ctx := context.WithValue(r.Context(), pdpCtxKey{}, pdpClient)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetPDPClient extracts a pdp.Client from context.
func GetPDPClient(ctx context.Context) (pdp.Client, bool) {
	c, ok := ctx.Value(pdpCtxKey{}).(pdp.Client)
	return c, ok
}

// authorizePDP performs a PDP check for the given action/resource. If PDP is not configured
// it allows by default. On PDP errors, it will deny when failClosed is true, otherwise allow.
func authorizePDP(r *http.Request, action, resourceType, resourceID string, resourceAttrs map[string]any, failClosed bool) (bool, string) {
	client, ok := GetPDPClient(r.Context())
	if !ok || client == nil {
		return true, "pdp_not_configured"
	}
	// Build subject and environment
	sub := pdp.Subject{UserID: r.Header.Get("X-PS-User-ID"), TenantID: r.Header.Get("X-PS-Tenant-ID"), Roles: parseCSV(r.Header.Get("X-PS-User-Roles"))}
	env := pdp.Environment{CorrelationID: getCorrelationID(r), Time: time.Now(), Attributes: map[string]any{"path": r.URL.Path, "method": r.Method}}
	res := pdp.Resource{Type: resourceType, ID: strings.TrimSpace(resourceID), Attributes: resourceAttrs}
	ctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
	defer cancel()
	resp, err := client.Evaluate(ctx, pdp.Request{Subject: sub, Action: action, Resource: res, Environment: env})
	if err != nil {
		if failClosed {
			return false, "pdp_error"
		}
		return true, "pdp_error_allow"
	}
	switch resp.Decision {
	case pdp.Permit, pdp.NotApplicable:
		return true, resp.Reason
	case pdp.Deny, pdp.Indeterminate:
		return false, resp.Reason
	default:
		if failClosed { return false, "pdp_invalid_decision" }
		return true, "pdp_invalid_decision_allow"
	}
}
