package bootstrap

import (
	"io"
	"log/slog"

	loggingpkg "github.com/promptshield/promptshield/internal/logging"
)

// SetupLogging is a thin adapter used in tests to construct a logger with the desired options.
func SetupLogging(debug, quiet bool, format string, w io.Writer) *slog.Logger {
	return loggingpkg.New(loggingpkg.Options{
		Debug:  debug,
		Quiet:  quiet,
		Format: format,
		Writer: w,
	})
}
