package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

type License struct {
	Organization string    `json:"org"`
	ExpiresAt    time.Time `json:"expires_at"`
	Tier         string    `json:"tier"`
}

var (
	mu          sync.RWMutex
	licensed    bool
	licenseInfo License
	evalLimiter *tokenBucket
	initOnce    sync.Once
)

const evalRatePerMinute = 10

func Check() {
	initOnce.Do(func() {
		evalLimiter = newTokenBucket(evalRatePerMinute, time.Minute)
	})
	key := strings.TrimSpace(os.Getenv("PROMPTSHIELD_LICENSE_KEY"))
	if key == "" {
		fmt.Print("⚠️  PromptShield Pro - Unlicensed\n\nThis is commercial software requiring a paid license.\n- 30-day trial: sales@promptshield.io\n- Community version: npm install @promptshield/cli\n\nStarting in EVALUATION MODE (watermarked output, 10 req/min limit)\n")
		time.Sleep(5 * time.Second)
		setLicensed(false, License{})
		return
	}
	lic, err := validate(key)
	if err != nil {
		log.Fatal("Invalid license key")
		return
	}
	setLicensed(true, *lic)
	fmt.Printf("✅ PromptShield Pro licensed to %s (Expires: %s)\n", lic.Organization, lic.ExpiresAt.Format("2006-01-02"))
}

func IsLicensed() bool {
	mu.RLock()
	v := licensed
	mu.RUnlock()
	return v
}

func Info() License {
	mu.RLock()
	v := licenseInfo
	mu.RUnlock()
	return v
}

func AllowEvalRequest() bool {
	if IsLicensed() {
		return true
	}
	// Lazy init to avoid nil deref in tests/HTTP server when Check() hasn't run
	if evalLimiter == nil {
		initOnce.Do(func() {
			evalLimiter = newTokenBucket(evalRatePerMinute, time.Minute)
		})
		if evalLimiter == nil {
			evalLimiter = newTokenBucket(evalRatePerMinute, time.Minute)
		}
	}
	return evalLimiter.Allow()
}

func setLicensed(v bool, info License) {
	mu.Lock()
	licensed = v
	licenseInfo = info
	mu.Unlock()
}

func validate(token string) (*License, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid token format")
	}
	payloadB, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	sigB, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}
	var lic License
	if err := json.Unmarshal(payloadB, &lic); err != nil {
		return nil, fmt.Errorf("decode license: %w", err)
	}
	if lic.Organization == "" {
		return nil, errors.New("missing org")
	}
	switch strings.ToLower(lic.Tier) {
	case "startup", "business", "enterprise":
	default:
		return nil, errors.New("invalid tier")
	}
	if time.Now().After(lic.ExpiresAt) {
		return nil, errors.New("license expired")
	}
	pubB64 := strings.TrimSpace(os.Getenv("PROMPTSHIELD_LICENSE_PUBLIC_KEY"))
	if pubB64 != "" {
		pubKey, err := base64.RawURLEncoding.DecodeString(pubB64)
		if err != nil {
			return nil, fmt.Errorf("invalid public key: %w", err)
		}
		if len(pubKey) != ed25519.PublicKeySize {
			return nil, errors.New("invalid public key length")
		}
		if !ed25519.Verify(ed25519.PublicKey(pubKey), payloadB, sigB) {
			return nil, errors.New("signature verification failed")
		}
	}
	return &lic, nil
}

type tokenBucket struct {
	mu       sync.Mutex
	capacity float64
	tokens   float64
	ratePerS float64
	lastFill time.Time
}

func newTokenBucket(tokensPerWindow int, window time.Duration) *tokenBucket {
	if tokensPerWindow <= 0 {
		tokensPerWindow = 10
	}
	if window <= 0 {
		window = time.Minute
	}
	return &tokenBucket{
		capacity: float64(tokensPerWindow),
		tokens:   float64(tokensPerWindow),
		ratePerS: float64(tokensPerWindow) / window.Seconds(),
		lastFill: time.Now(),
	}
}

func (b *tokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(b.lastFill).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * b.ratePerS
		if b.tokens > b.capacity {
			b.tokens = b.capacity
		}
		b.lastFill = now
	}
	if b.tokens >= 1 {
		b.tokens -= 1
		return true
	}
	return false
}
