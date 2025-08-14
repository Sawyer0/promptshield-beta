package bootstrap

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strconv"
	"time"

	cfg "github.com/promptshield/promptshield/internal/config"
	"github.com/promptshield/promptshield/internal/logging"
	"github.com/promptshield/promptshield/internal/scanner"
	"github.com/promptshield/promptshield/internal/security/cred"
	semanthropic "github.com/promptshield/promptshield/internal/semantic/anthropic"
	semopenai "github.com/promptshield/promptshield/internal/semantic/openai"
	"go.opentelemetry.io/otel"
)

// Deps bundles application-wide dependencies.
type Deps struct {
	Logger  *slog.Logger
	Scanner *scanner.Scanner
	Config  cfg.Config
	// Telemetry collector (optional)
	Telemetry TelemetryCollector
}

type Options struct {
	Debug         bool
	Quiet         bool
	LogJSON       bool
	LogWriter     io.Writer
	MaxTokenBytes int
	Semantic      scanner.SemanticAnalyzer
}

// Build constructs dependencies with sensible defaults.
func Build(opts Options) *Deps {
	logger := logging.New(logging.Options{
		Debug:  opts.Debug,
		Quiet:  opts.Quiet,
		Format: ternary(opts.LogJSON, "json", "text"),
		Writer: opts.LogWriter,
	})
	sc := scanner.New(opts.MaxTokenBytes)
	// Attach a default OpenTelemetry tracer; callers can override
	tr := otel.Tracer("promptshield/scanner")
	sc.SetTracer(tr)
	// Wire logger into scanner for optional debug output
	sc.SetLogger(logger)
	if opts.Semantic != nil {
		sc.SetSemanticAnalyzer(opts.Semantic)
	} else {
		// Env-driven semantic wiring (opt-in)
		// PS_SEMANTIC_ENABLED=true and appropriate API key
		if isTrue(os.Getenv("PS_SEMANTIC_ENABLED")) {
			maxConc := atoiDefault(os.Getenv("PS_SEMANTIC_MAX_CONCURRENCY"), 2)
			cacheSize := atoiDefault(os.Getenv("PS_SEMANTIC_CACHE_SIZE"), 1000)
			cacheTTL := parseDurDefault(os.Getenv("PS_SEMANTIC_CACHE_TTL"), 15*time.Minute)

			// Provider must be explicitly selected via PS_SEMANTIC_PROVIDER
			provider := os.Getenv("PS_SEMANTIC_PROVIDER")

			switch provider {
			case "anthropic":
				// Prefer OS keyring, fallback to env
				key := ""
				if k, err := cred.GetProviderAPIKey(context.TODO(), cred.ProviderAnthropic); err == nil && k != "" {
					key = k
				} else {
					key = os.Getenv("ANTHROPIC_API_KEY")
					if key == "" {
						key = os.Getenv("PS_ANTHROPIC_API_KEY")
					}
				}
				if key != "" {
					analyzer := semanthropic.New(semanthropic.Options{
						APIKey:         key,
						MaxConcurrency: maxConc,
						CacheSize:      cacheSize,
						CacheTTL:       cacheTTL,
						Logger:         logger,
					})
					sc.SetSemanticAnalyzer(analyzer)
				}
			case "openai":
				// Prefer OS keyring, fallback to env
				key := ""
				if k, err := cred.GetProviderAPIKey(context.TODO(), cred.ProviderOpenAI); err == nil && k != "" {
					key = k
				} else {
					key = os.Getenv("OPENAI_API_KEY")
					if key == "" {
						key = os.Getenv("PS_OPENAI_API_KEY")
					}
				}
				if key != "" {
					analyzer := semopenai.New(semopenai.Options{
						APIKey:         key,
						MaxConcurrency: maxConc,
						CacheSize:      cacheSize,
						CacheTTL:       cacheTTL,
						Logger:         logger,
					})
					sc.SetSemanticAnalyzer(analyzer)
				}
			default:
				// Unknown or empty provider: do not wire an analyzer; service layer will error if L3 is enabled without valid provider
			}
		}
	}
	return &Deps{Logger: logger, Scanner: sc, Config: cfg.Defaults()}
}

// TelemetryCollector is a minimal interface for emitting privacy-safe events.
type TelemetryCollector interface {
	Collect(eventType string, payload map[string]any)
	Shutdown(context.Context) error
}

func ternary[T any](cond bool, a T, b T) T {
	if cond {
		return a
	}
	return b
}

func isTrue(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

func parseDurDefault(s string, def time.Duration) time.Duration {
	if s == "" {
		return def
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	return def
}
