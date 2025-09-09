package audit

import (
	"io"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

// RotatingFileLogger writes audit events to a file with rotation delegated to lumberjack.
// We keep hash-chaining at the JSON record level; rollover is handled by lumberjack
// using size/age/backups. Default policy: max 100MB per file, keep 7 backups, max age 30 days.
type RotatingFileLogger struct {
	basePath string
	mu       sync.Mutex
	lj       *lumberjack.Logger
	inner    *FileLogger
}

func NewDailyRotatingLogger(basePath string) (*RotatingFileLogger, error) {
	// Use the provided basePath verbatim (no extension manipulation) to keep tests and docs stable.
	dir := filepath.Dir(basePath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	lj := &lumberjack.Logger{
		Filename:   basePath,
		MaxSize:    100, // megabytes
		MaxBackups: 7,
		MaxAge:     30, // days
		Compress:   false,
	}
	rl := &RotatingFileLogger{basePath: basePath, lj: lj}
	rl.inner = NewFileLogger(&syncWriter{w: lj})
	return rl, nil
}

func (l *RotatingFileLogger) Log(e Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	// Sanitize all event data before writing
	if e.Data != nil {
		e.Data = SanitizeMap(e.Data)
	}
	return l.inner.Log(e)
}

func (l *RotatingFileLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.lj != nil {
		return l.lj.Close()
	}
	return nil
}

// syncWriter adapts a WriteCloser to io.Writer while keeping Close
type syncWriter struct{ w io.WriteCloser }

func (s *syncWriter) Write(p []byte) (int, error) { return s.w.Write(p) }
