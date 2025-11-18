package grpcenforcer

import (
	"context"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/contracts"
	nats "github.com/promptshield/promptshield/internal/infrastructure/messaging/nats"
	"github.com/promptshield/promptshield/internal/license"
	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/internal/scanner"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	health "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"gopkg.in/yaml.v3"
)

// New returns a Server with default Options using the provided timeout.
func New(timeout time.Duration) *Server {
	return NewWithOptions(Options{Timeout: timeout, MaxStreamBytes: 5_000_000, FailOn: "HIGH", RulepackPath: defaultRulepackPathFromEnv()})
}

// NewWithOptions builds a Server with the supplied Options, wiring scanners, health, limits, etc.
func NewWithOptions(opt Options) *Server {
	if opt.Timeout <= 0 {
		opt.Timeout = 300 * time.Millisecond
	}
	if opt.MaxStreamBytes <= 0 {
		opt.MaxStreamBytes = 5_000_000
	}
	if opt.FailOn == "" {
		opt.FailOn = "HIGH"
	}
	// allow env-based rulepack when not explicitly provided
	if strings.TrimSpace(opt.RulepackPath) == "" {
		opt.RulepackPath = defaultRulepackPathFromEnv()
	}

	mode := strings.ToLower(strings.TrimSpace(opt.EnforcementMode))
	if mode == "" {
		mode = strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_ENFORCEMENT_MODE")))
	}
	if mode == "" {
		mode = "observe" // observe|redact|quarantine|enforce
	}

	s := &Server{
		timeout:         opt.Timeout,
		maxStreamBytes:  opt.MaxStreamBytes,
		failOn:          opt.FailOn,
		rulepackPath:    opt.RulepackPath,
		telemetry:       opt.Telemetry,
		enforcementMode: mode,
		rulepackRepo:    opt.RulepackRepo,
		tenantID:        opt.TenantID,
	}

	// scanners per direction
	scReq := scanner.ScanEngineCstor(0)
	scResp := scanner.ScanEngineCstor(0)
	scReq.SetMaxStreamBytes(opt.MaxStreamBytes)
	scResp.SetMaxStreamBytes(opt.MaxStreamBytes)
	scReq.SetQuarantineOnTimeout(true)
	scReq.SetQuarantineOnError(true)
	scResp.SetQuarantineOnTimeout(true)
	scResp.SetQuarantineOnError(true)

	loaded := false

	// Try loading from database first (if repo and tenantID provided)
	if opt.RulepackRepo != nil && opt.TenantID != uuid.Nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if packs, err := LoadRulesFromDatabase(ctx, opt.RulepackRepo, opt.TenantID); err == nil && len(packs) > 0 {
			scReq.LoadRulePacks(packs)
			scResp.LoadRulePacks(packs)
			loaded = true
		}
	}

	// Fallback to file-based loading if database loading failed or no rules found
	if !loaded && s.rulepackPath != "" {
		if packs, err := rules.LoadPacks(s.rulepackPath); err == nil {
			scReq.LoadRulePacks(packs)
			scResp.LoadRulePacks(packs)
			loaded = true
		}
	}

	// Check environment variable for fail-closed behavior (opt-in)
	failClosed := strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_FAIL_CLOSED"))) == "true"

	scReq.SetRuntimeContext(map[string]string{"direction": "request"})
	scResp.SetRuntimeContext(map[string]string{"direction": "response"})
	s.scReq = scReq
	s.scResp = scResp

	// health reporter
	// Fail-open by default: ready if we can accept traffic even without rules
	// Fail-closed only if explicitly requested via PS_ENFORCER_FAIL_CLOSED=true
	s.ready = loaded || !failClosed
	redEnv := strings.ToLower(strings.TrimSpace(os.Getenv("PS_ENFORCER_REDACTION_MUTATION")))
	redOn := !(redEnv == "0" || redEnv == "false")
	hs := health.NewServer()
	if s.ready {
		hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	} else {
		hs.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	}
	svcName := "envoy.service.ext_proc.v3.ExternalProcessor"
	if s.ready {
		hs.SetServingStatus(svcName, healthpb.HealthCheckResponse_SERVING)
	} else {
		hs.SetServingStatus(svcName, healthpb.HealthCheckResponse_NOT_SERVING)
	}
	if redOn {
		hs.SetServingStatus("promptshield.features.redaction", healthpb.HealthCheckResponse_SERVING)
	}
	s.healthSrv = hs

	// defaults for sliding windows
	s.windowLimit = 64 * 1024
	s.overlap = 4096

	// streaming tunables via env
	if v := strings.TrimSpace(os.Getenv("PS_ENFORCER_STREAM_WINDOW")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 8192 {
			s.windowLimit = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("PS_ENFORCER_STREAM_OVERLAP")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 && n < s.windowLimit {
			s.overlap = n
		}
	}

	// rate‐limiter based on license or env
	if ent, ok := license.Entitlement(); ok && ent.MaxRPS > 0 {
		burst := 1
		if b := strings.TrimSpace(os.Getenv("PS_ENFORCER_RPS_BURST")); b != "" {
			if n, err := strconv.Atoi(b); err == nil && n > 0 {
				burst = n
			}
		}
		s.limiter = rate.NewLimiter(rate.Limit(ent.MaxRPS), burst)
	} else if rpsStr := strings.TrimSpace(os.Getenv("PS_ENFORCER_RPS")); rpsStr != "" {
		if r, err := strconv.ParseFloat(rpsStr, 64); err == nil && r > 0 {
			burst := 1
			if b := strings.TrimSpace(os.Getenv("PS_ENFORCER_RPS_BURST")); b != "" {
				if n, err := strconv.Atoi(b); err == nil && n > 0 {
					burst = n
				}
			}
			s.limiter = rate.NewLimiter(rate.Limit(r), burst)
		}
	}

	// optional global stream slots + inflight ceilings
	if v := strings.TrimSpace(os.Getenv("PS_ENFORCER_MAX_STREAMS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			s.streamSlots = make(chan struct{}, n)
		}
	}
	if v := strings.TrimSpace(os.Getenv("PS_ENFORCER_INFLIGHT_LIMIT_BYTES")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			s.inflightLimit = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("PS_ENFORCER_INFLIGHT_BACKOFF_MS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			s.inflightBackoff = time.Duration(n) * time.Millisecond
		}
	}

	// Set up live rule updates subscriber (if Redis configured)
	if opt.RedisAddr != "" && opt.TenantID != uuid.Nil {
		consumerGroup := "ps-enforcers"
		consumerName := "enforcer-" + uuid.New().String()[:8] // Unique consumer name

		// Create rule update handler
		handler := func(ctx context.Context, update nats.RuleUpdate) error {
			// Reload rules when we receive an update for our tenant
			return s.ReloadRules(ctx)
		}

		if sub, err := nats.NewSubscriber(opt.RedisAddr, consumerGroup, consumerName, opt.TenantID.String(), handler); err == nil {
			s.subscriber = sub
			// Start subscriber in background
			go func() {
				if err := sub.Start(context.Background()); err != nil {
					logger := slog.With("component", "grpc-enforcer")
					logger.Error("Rule update subscriber stopped", "error", err)
				}
			}()
		}
	}

	return s
}

func defaultRulepackPathFromEnv() string {
	if v := os.Getenv("PS_ENFORCER_RULEPACK"); v != "" {
		return v
	}
	return ""
}

// LoadRulesFromDatabase loads all active rulepacks for a tenant from the database
func LoadRulesFromDatabase(ctx context.Context, repo contracts.RulepackRepository, tenantID uuid.UUID) ([]rules.RulePack, error) {
	if repo == nil {
		return nil, nil
	}

	// Get list of all rulepacks for tenant
	rulepackInfos, err := repo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var packs []rules.RulePack
	for _, info := range rulepackInfos {
		if !info.Active {
			continue // Skip inactive rulepacks
		}

		// Get the active DSL for this rulepack
		dslBytes, _, err := repo.GetActive(ctx, info.ID)
		if err != nil {
			continue // Skip rulepacks that can't be loaded
		}

		// Parse the DSL into a RulePack
		var pack rules.RulePack
		if err := yaml.Unmarshal(dslBytes, &pack); err != nil {
			continue // Skip invalid rulepacks
		}

		packs = append(packs, pack)
	}

	return packs, nil
}

// Build creates a gRPC server with the enforcer registered and starts listening on the given address.
// This is a convenience function that combines gRPC server creation, enforcer registration, and listener setup.
func Build(addr string, opts Options, serverOpts ...grpc.ServerOption) (*grpc.Server, error) {
	return buildWithServerOptions(addr, opts, serverOpts...)
}

func buildWithServerOptions(addr string, opts Options, serverOpts ...grpc.ServerOption) (*grpc.Server, error) {
	// Create the enforcer server
	server := NewWithOptions(opts)

	// Create gRPC server
	grpcServer := grpc.NewServer(serverOpts...)

	// Register the external processor service
	extproc.RegisterExternalProcessorServer(grpcServer, server)

	// Register health service
	healthpb.RegisterHealthServer(grpcServer, server.healthSrv)

	// Start listening
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	// Start serving in a goroutine
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			logger := slog.With("component", "grpc-enforcer")
			logger.Error("gRPC server error", "error", err)
		}
	}()

	return grpcServer, nil
}
