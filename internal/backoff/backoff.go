package backoff

import (
	"math/rand"
	"time"
)

// FullJitter returns an exponential backoff with 'full jitter' (AWS recipe).
//
//	sleep = rand(0, min(cap, base*2^attempt))
//
// base must be >0; attempt >=0.
func FullJitter(base, cap time.Duration, attempt int) time.Duration {
	if base <= 0 {
		base = 50 * time.Millisecond
	}
	if cap <= 0 {
		cap = 30 * time.Second
	}
	max := base << attempt // base*2^attempt
	if max > cap {
		max = cap
	}
	if max <= 0 {
		max = cap
	}
	return time.Duration(rand.Int63n(int64(max)))
}

// EqualJitter is a variant that adds half of the calculated backoff plus jitter on the other half.
//
//	sleep = max/2 + rand(0, max/2)
func EqualJitter(base, cap time.Duration, attempt int) time.Duration {
	max := base << attempt
	if max > cap {
		max = cap
	}
	half := max / 2
	return half + time.Duration(rand.Int63n(int64(half)))
}

func init() { rand.Seed(time.Now().UnixNano()) }

