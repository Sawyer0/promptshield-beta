package chaos

import (
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

// Controller injects faults and delays when enabled via environment variables.
//
// Env variables:
//
//	PS_CHAOS=1 (enable)
//	PS_CHAOS_FAIL_PCT=5 (percentage 0..100 of operations to fail)
//	PS_CHAOS_DELAY_MS_AVG=10 (average delay in milliseconds; actual is +/-50%)
type Controller struct {
	enabled    bool
	failPct    float64
	delayAvgMs int64
}

// NewFromEnv constructs a chaos controller based on environment variables.
func NewFromEnv() *Controller {
	enabled := strings.TrimSpace(os.Getenv("PS_CHAOS")) == "1"
	fp := 0.0
	if v := strings.TrimSpace(os.Getenv("PS_CHAOS_FAIL_PCT")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 && f <= 100 {
			fp = f
		}
	}
	delay := int64(0)
	if v := strings.TrimSpace(os.Getenv("PS_CHAOS_DELAY_MS_AVG")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 {
			delay = n
		}
	}
	return &Controller{enabled: enabled, failPct: fp, delayAvgMs: delay}
}

// MaybeDelay sleeps for a small random duration around delayAvgMs.
func (c *Controller) MaybeDelay() {
	if !c.enabled || c.delayAvgMs <= 0 {
		return
	}
	// +/- 50% jitter
	base := c.delayAvgMs
	jitter := rand.Int63n(base/2 + 1) //nolint:gosec // Non-cryptographic randomness is fine for chaos testing
	if rand.Intn(2) == 0 { //nolint:gosec // Non-cryptographic randomness is fine for chaos testing
		base -= jitter
	} else {
		base += jitter
	}
	t := time.NewTimer(time.Duration(base) * time.Millisecond)
	<-t.C
}

// ShouldFail randomly decides to fail based on configured percentage.
func (c *Controller) ShouldFail() bool {
	if !c.enabled || c.failPct <= 0 {
		return false
	}
	return rand.Float64()*100 < c.failPct //nolint:gosec // Non-cryptographic randomness is fine for chaos testing
}
