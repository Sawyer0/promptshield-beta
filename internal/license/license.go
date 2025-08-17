package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type Entitlements struct {
	MaxRPS   float64         `json:"max_rps"`
	Features map[string]bool `json:"features"`
}

type License struct {
	Organization string       `json:"org"`
	ExpiresAt    time.Time    `json:"expires_at"`
	Tier         string       `json:"tier"`
	Entitlements Entitlements `json:"entitlements"`
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
	
	// Development bypass
	if strings.ToLower(os.Getenv("PS_DEV_MODE")) == "true" || os.Getenv("PS_DISABLE_LICENSE") == "1" {
		fmt.Print("🛠️  PromptShield Pro - Development Mode (No License Required)\n")
		devLicense := License{
			Organization: "Development",
			ExpiresAt:    time.Now().Add(365 * 24 * time.Hour),
			Tier:         "enterprise",
			Entitlements: Entitlements{
				MaxRPS:   1000.0,
				Features: map[string]bool{"l3_semantic": true, "enterprise": true},
			},
		}
		setLicensed(true, devLicense)
		return
	}
	
	key := strings.TrimSpace(os.Getenv("PROMPTSHIELD_LICENSE_KEY"))
	if key == "" {
		fmt.Print("⚠️  PromptShield Pro - Unlicensed\n\nThis is commercial software requiring a paid license.\n- 30-day trial: sales@promptshield.io\n- Community version: npm install @promptshield/cli\n\nStarting in EVALUATION MODE (watermarked output, 10 req/min limit)\n")
		time.Sleep(5 * time.Second)
		setLicensed(false, License{})
		return
	}
	lic, err := validate(key)
	if err != nil {
		fmt.Printf("⚠️  Invalid license key: %v\n\nFalling back to EVALUATION MODE (watermarked output, 10 req/min limit)\n", err)
		time.Sleep(3 * time.Second)
		setLicensed(false, License{})
		return
	}
	setLicensed(true, *lic)
	fmt.Printf("✅ PromptShield Pro licensed to %s (Expires: %s)\n", lic.Organization, lic.ExpiresAt.Format("2006-01-02"))
}

func IsLicensed() bool {
	ensureLoaded()
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

// HasFeature reports whether the current license permits a specific feature.
func HasFeature(name string) bool {
	ensureLoaded()
	mu.RLock()
	defer mu.RUnlock()
	if !licensed {
		return false
	}
	if name == "" {
		return false
	}
	n := strings.ToLower(name)
	if licenseInfo.Entitlements.Features == nil {
		return false
	}
	return licenseInfo.Entitlements.Features[n]
}

// Entitlement returns entitlements and whether the process is licensed.
func Entitlement() (Entitlements, bool) {
	ensureLoaded()
	mu.RLock()
	defer mu.RUnlock()
	return licenseInfo.Entitlements, licensed
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

// ensureLoaded lazily initializes license state from environment without printing or sleeping.
func ensureLoaded() {
	mu.RLock()
	if licensed {
		mu.RUnlock()
		return
	}
	mu.RUnlock()
	key := strings.TrimSpace(os.Getenv("PROMPTSHIELD_LICENSE_KEY"))
	if key == "" {
		return
	}
	if lic, err := validate(key); err == nil {
		setLicensed(true, *lic)
	}
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
