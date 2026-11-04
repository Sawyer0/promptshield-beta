//go:build race

package scanner_test

// raceEnabled reports whether tests run under the race detector, which
// slows execution too much for throughput floors to be meaningful.
const raceEnabled = true
