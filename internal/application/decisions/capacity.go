package decisions

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// CapacityEnv names the variable that sizes the in-memory decision log.
const CapacityEnv = "PS_DECISION_LOG_SIZE"

// Accepted range for CapacityEnv. The log lives in memory only; the upper
// bound keeps it to tens of megabytes even with many matches per record.
const (
	MinCapacity = 10
	MaxCapacity = 50_000
)

// ParseCapacity validates a CapacityEnv value. Blank means DefaultCapacity;
// anything that is not an integer in [MinCapacity, MaxCapacity] is an error
// and also returns DefaultCapacity.
func ParseCapacity(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DefaultCapacity, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return DefaultCapacity, fmt.Errorf("%s=%q is not an integer", CapacityEnv, raw)
	}
	if n < MinCapacity || n > MaxCapacity {
		return DefaultCapacity, fmt.Errorf("%s=%d is outside %d-%d", CapacityEnv, n, MinCapacity, MaxCapacity)
	}
	return n, nil
}

// NewLogFromEnv returns a Log sized by CapacityEnv. An invalid value is
// logged and the default capacity is used, so a typo never stops the gateway.
func NewLogFromEnv() *Log {
	capacity, err := ParseCapacity(os.Getenv(CapacityEnv))
	if err != nil {
		slog.Warn("invalid decision log size; using the default", "error", err, "default", DefaultCapacity)
	}
	return NewLog(capacity)
}
