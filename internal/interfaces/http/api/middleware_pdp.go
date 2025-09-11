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
