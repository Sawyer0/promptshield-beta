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
	c := buildPDPClientForAdmin()
	if c != nil { pdpClient = c }
}

func buildPDPClientForAdmin() pdp.Client {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("PS_PDP_MODE")))
	apiKey := strings.TrimSpace(os.Getenv("PS_PDP_API_KEY"))
	to := 2000 * time.Millisecond
	if v := strings.TrimSpace(os.Getenv("PS_PDP_TIMEOUT_MS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			to = time.Duration(n) * time.Millisecond
		}
	}

	// Try in-process mode first if requested
if mode == "inprocess" {
		cfg := pdp.InprocessConfig{
			PolicyPath: strings.TrimSpace(os.Getenv("PS_PDP_REGO_PATH")),
			DataPath:   strings.TrimSpace(os.Getenv("PS_PDP_DATA_PATH")),
			EntryPoint: strings.TrimSpace(os.Getenv("PS_PDP_ENTRYPOINT")),
			Timeout:    to,
		}
		if c, err := pdp.NewInprocessClient(cfg); err == nil && c != nil {
			client := pdp.NewCached(c)
			slog.Info("PDP enabled (in-process)", "entrypoint", cfg.EntryPoint, "policy", cfg.PolicyPath)
			return client
		} else if err != nil {
			slog.Warn("In-process PDP not available, falling back to HTTP", "error", err)
		}
	}

	// HTTP/sidecar mode
	endpoint := strings.TrimSpace(os.Getenv("PS_PDP_ENDPOINT"))
if endpoint == "" {
		slog.Info("PDP disabled: PS_PDP_ENDPOINT not set")
		return nil
	}
	base := pdphttp.New(pdphttp.Config{Endpoint: endpoint, APIKey: apiKey, Timeout: to})
	client := pdp.NewCached(base)
	slog.Info("PDP enabled (http)", "endpoint", endpoint, "timeout_ms", to.Milliseconds())
	return client
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

// resolvePIPAttributes collects additional environment attributes (PIP) from the request.
// This is a lightweight hook; extend as needed (e.g., org roles, device posture) with caching/timeouts.
func resolvePIPAttributes(r *http.Request) map[string]any {
	attrs := map[string]any{}
	if v := strings.TrimSpace(r.Header.Get("X-PS-Device")); v != "" {
		attrs["device"] = v
	}
	// client IP (best-effort)
	if ip := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); ip != "" {
		attrs["client_ip"] = ip
	}
	return attrs
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
	// Merge additional PIP-derived attributes
	if extra := resolvePIPAttributes(r); len(extra) > 0 {
		if env.Attributes == nil { env.Attributes = map[string]any{} }
		for k, v := range extra { env.Attributes[k] = v }
	}
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
