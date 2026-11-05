package enforcerhttp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/promptshield/promptshield/internal/audit"
	pg "github.com/promptshield/promptshield/internal/infrastructure/persistence/postgres"
	"github.com/promptshield/promptshield/internal/security/paths"
	"github.com/promptshield/promptshield/internal/shared/types"
	"github.com/promptshield/promptshield/internal/usage"
	redis "github.com/redis/go-redis/v9"
)

// Serve starts an HTTP server on addr with sane timeouts.
func Serve(addr string) *http.Server {
	var srv *http.Server
	// Initialize database pool if configured
	var dbPool *pg.Pool
	if dsn := os.Getenv("PS_PG_DSN"); dsn != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if pool, err := pg.NewPool(ctx, dsn); err == nil {
			pingCtx, cancelPing := context.WithTimeout(ctx, 2*time.Second)
			if _, err := pool.Raw().Exec(pingCtx, "SELECT 1"); err == nil {
				dbPool = pool
			} else {
				pool.Close()
			}
			cancelPing()
		}
	}
	// Inject shutdown hooks into API mux with database pool
	options := getAPIOptionsWithDB(dbPool)
	// Wire Redis-backed UsageStore when configured
	if options.UsageStore == nil {
		if addr := strings.TrimSpace(os.Getenv("PS_USAGE_REDIS_ADDR")); addr != "" {
			// Normalize prefix: default to PS_REGION or "ps"
			prefix := strings.TrimSpace(os.Getenv("PS_USAGE_PREFIX"))
			if prefix == "" {
				prefix = strings.TrimSpace(os.Getenv("PS_REGION"))
			}
			if prefix == "" {
				prefix = "ps"
			}
			// Optional TTL in days
			ttlDays := 35
			if v := strings.TrimSpace(os.Getenv("PS_USAGE_TTL_DAYS")); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n > 0 {
					ttlDays = n
				}
			}
			rdb := redis.NewClient(&redis.Options{Addr: addr})
			options.UsageStore = usage.NewRedisUsageStore(rdb, prefix, time.Duration(ttlDays)*24*time.Hour)
		}
	}
	// Construct durable audit logger from environment and adapt to API interface
	var auditClose func() error
	if lgr, closeFn, err := audit.NewLoggerFromEnv(); err == nil && lgr != nil {
		options.AuditLogger = auditAdapter{inner: lgr}
		auditClose = closeFn
	}
	options.OnDrain = func(ctx context.Context) error { return nil }
	options.OnShutdown = func(ctx context.Context, delay time.Duration) error {
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if auditClose != nil {
			_ = auditClose()
		}
		if options.UsageStore != nil {
			_ = options.UsageStore.Close(ctx)
		}
		if dbPool != nil {
			dbPool.Close()
		}
		if srv != nil {
			return srv.Shutdown(ctx)
		}
		return nil
	}
	apiMux := NewMuxWithOptions(options)
	srv = newHTTPServer(addr, apiMux)

	// TLS policy: require on non-loopback unless explicitly disabled; auto-detect certs when not provided
	mode := tlsMode()
	certFile, keyFile, havePair := findTLSPair()
	clientCA := os.Getenv("PS_ENFORCER_TLS_CLIENT_CA")
	nonLoop := !isLoopbackAddr(addr)

	startHTTPS := func(certFile, keyFile string) {
		tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
		if clientCA != "" {
			logger := slog.With("component", "enforcer-http")
			if err := paths.ValidateCAFilePath(clientCA); err != nil {
				logger.Error("invalid client CA file path", "error", err)
			} else if caPEM, err := os.ReadFile(clientCA); err == nil {
				pool := x509.NewCertPool()
				if pool.AppendCertsFromPEM(caPEM) {
					tlsCfg.ClientCAs = pool
					tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
				} else {
					logger.Error("failed to parse client CA certificate")
				}
			} else {
				logger.Error("failed to read client CA file", "error", err)
			}
		}
		srv.TLSConfig = tlsCfg
		go func() {
			if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
				logger := slog.With("component", "enforcer-http")
				logger.Error("enforcer https server error", "error", err)
			}
		}()
	}

	switch mode {
	case "disable":
		// Explicit insecure mode
		go func() {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger := slog.With("component", "enforcer-http")
				logger.Error("enforcer http server error", "error", err)
			}
		}()
		return srv
	case "require":
		if !havePair {
			logger := slog.With("component", "enforcer-http")
			logger.Error("TLS required but no certificate configured", "address", addr)
			os.Exit(1)
		}
		startHTTPS(certFile, keyFile)
		return srv
	default: // auto
		if nonLoop {
			if !havePair {
				logger := slog.With("component", "enforcer-http")
				logger.Error("Refusing to listen on non-loopback address without TLS", "address", addr)
				os.Exit(1)
			}
			startHTTPS(certFile, keyFile)
			return srv
		}
		// loopback: prefer TLS if available, else plain HTTP
		if havePair {
			startHTTPS(certFile, keyFile)
		} else {
			go func() {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger := slog.With("component", "enforcer-http")
					logger.Error("enforcer http server error", "error", err)
				}
			}()
		}
	}
	return srv
}

// auditAdapter adapts internal/audit.Logger to contracts.AuditLogger
type auditAdapter struct{ inner audit.Logger }

func (a auditAdapter) LogWithContext(ctx context.Context, ev types.AuditEvent) error {
	return a.inner.Log(audit.Event{Type: ev.Action, Data: ev.Metadata, Hash: "", PrevHash: ""})
}

func (a auditAdapter) Flush() error {
	return nil
}

func (a auditAdapter) Close() error {
	return nil
}

// newHTTPServer applies the gateway's connection timeouts. The SSE stream at
// /v1/events lifts the read and write deadlines for its own connection.
func newHTTPServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
