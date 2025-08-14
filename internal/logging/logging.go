package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Options configures the logger factory.
type Options struct {
	Debug  bool
	Quiet  bool
	Format string    // "text" (default) or "json"
	Writer io.Writer // defaults to os.Stderr
}

// New constructs a structured logger to stderr by default.
// - Debug sets level to Debug; otherwise Info.
// - Quiet forces Error level regardless of Debug.
// - Format selects text or json handler.
func New(opts Options) *slog.Logger {
	w := opts.Writer
	if w == nil {
		w = os.Stderr
	}
	level := slog.LevelInfo
	if opts.Debug || strings.EqualFold(os.Getenv("PS_DEBUG"), "true") {
		level = slog.LevelDebug
	}
	if opts.Quiet {
		level = slog.LevelError
	}
	handlerOpts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if opts.Format == "json" {
		handler = slog.NewJSONHandler(w, handlerOpts)
	} else {
		handler = slog.NewTextHandler(w, handlerOpts)
	}
	return slog.New(handler)
}
