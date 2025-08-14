package termui

import (
	"fmt"
)

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiFaint  = "\x1b[2m"
	ansiRed    = "\x1b[31m"
	ansiYellow = "\x1b[33m"
	ansiBlue   = "\x1b[34m"
	ansiCyan   = "\x1b[36m"
	ansiGreen  = "\x1b[32m"
)

func Bold(enabled bool, s string) string {
	if !enabled {
		return s
	}
	return ansiBold + s + ansiReset
}
func Dim(enabled bool, s string) string {
	if !enabled {
		return s
	}
	return ansiFaint + s + ansiReset
}
func Red(enabled bool, s string) string {
	if !enabled {
		return s
	}
	return ansiRed + s + ansiReset
}
func Yellow(enabled bool, s string) string {
	if !enabled {
		return s
	}
	return ansiYellow + s + ansiReset
}
func Blue(enabled bool, s string) string {
	if !enabled {
		return s
	}
	return ansiBlue + s + ansiReset
}
func Cyan(enabled bool, s string) string {
	if !enabled {
		return s
	}
	return ansiCyan + s + ansiReset
}
func Green(enabled bool, s string) string {
	if !enabled {
		return s
	}
	return ansiGreen + s + ansiReset
}

// Heading formats a section heading consistently.
func Heading(color bool, s string) string {
	return Bold(color, s)
}

// Bullet formats a two-column bullet line: left command and right hint.
// Width of left column is fixed for simple alignment in monospaced terminals.
func Bullet(color bool, left string, right string) string {
	// 30 char left column padding
	const leftWidth = 30
	if len(left) >= leftWidth {
		return fmt.Sprintf("%s  %s", left, Dim(color, right))
	}
	pad := make([]byte, leftWidth-len(left))
	for i := range pad {
		pad[i] = ' '
	}
	return fmt.Sprintf("%s%s  %s", left, string(pad), Dim(color, right))
}
