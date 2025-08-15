package scanner

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"runtime"
	"strings"
	"time"

	sharederrors "github.com/promptshield/promptshield/internal/shared/errors"
	"github.com/promptshield/promptshield/pkg/types"
)

// ScanFile opens the specified file and scans it in a streaming manner.
func (s *Scanner) ScanFile(ctx context.Context, path string) (types.ScanResult, error) {
	if s.logger != nil {
		s.logger.Debug("scan file begin", "path", path, "pid", os.Getpid(), "go", runtime.Version())
	}
	if s.fileTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.fileTimeout)
		defer cancel()
	}
	// If caller didn't provide a deadline and a global total budget is configured, apply it for this file as a soft bound
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && s.totalScanBudget > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.totalScanBudget)
		defer cancel()
	}
	select {
	case <-ctx.Done():
		return types.ScanResult{}, ctx.Err()
	default:
	}

	// Pre-open validation: ensure file info is sane and not excessively large when configured
	if fi, statErr := os.Stat(path); statErr == nil {
		// Reject directories early
		if fi.IsDir() {
			return types.ScanResult{}, fmt.Errorf("opening %s: %w", path, fs.ErrInvalid)
		}
		if s.maxFileBytes > 0 && fi.Size() > s.maxFileBytes {
			return types.ScanResult{}, fmt.Errorf("%s exceeds max file size limit (%d bytes)", path, s.maxFileBytes)
		}
		// Apply a per-file budget if configured and no explicit fileTimeout is set
		if s.fileTimeout <= 0 && s.budgets.PerFile > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, s.budgets.PerFile)
			defer cancel()
		}
		// Optional size guard: if bufferSizeBytes is huge, still allow; otherwise cap extremely large files if a global total timeout is set elsewhere
		// Keep streaming-first; do not read into memory.
		// Future: enforce explicit max size from config; here we only reject negative/overflow conditions implicitly.
		if fi.Size() < 0 {
			return types.ScanResult{}, fmt.Errorf("invalid file size for %s", path)
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return types.ScanResult{}, fmt.Errorf("opening %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	// First try fast scanner path
	res, err := s.ScanReader(ctx, f, path)
	if err == nil {
		if s.logger != nil {
			s.logger.Debug("scan file end", "path", path, "violations", len(res.Violations), "bytes", res.Metrics.BytesRead, "lines", res.Metrics.LinesRead)
		}
		return res, nil
	}
	// Fallback for very long lines: restart with chunked reader
	if !errors.Is(err, bufio.ErrTooLong) && !strings.Contains(err.Error(), "token too long") {
		return types.ScanResult{}, err
	}
	if _, seekErr := f.Seek(0, io.SeekStart); seekErr != nil {
		return types.ScanResult{}, err
	}
	r, e := s.scanChunked(ctx, f, path)
	if s.logger != nil && e == nil {
		s.logger.Debug("scan file end (chunked)", "path", path, "violations", len(r.Violations), "bytes", r.Metrics.BytesRead, "lines", r.Metrics.LinesRead)
	}
	return r, e
}

// ScanReader scans content from the provided reader. Caller controls the reader lifetime.
func (s *Scanner) ScanReader(ctx context.Context, r io.Reader, inputName string) (types.ScanResult, error) {
	result := types.ScanResult{Input: inputName}
	if s.logger != nil {
		s.logger.Debug("scan reader begin", "input", inputName)
	}

	// Use a scanner with a large token buffer to handle long lines/JSONL entries.
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, s.bufferSizeBytes)

	var lineNum int64
	start := time.Now()
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			if s.quarantineOnTimeout {
				// Return a synthetic violation indicating timeout
				result.Violations = append(result.Violations, types.Violation{RuleID: "timeout", Message: "scan timed out", Severity: "HIGH"})
				result.DurationMs = time.Since(start).Milliseconds()
				return result, nil
			}
			return types.ScanResult{}, ctx.Err()
		default:
		}

		// Enforce memory ceiling best-effort using heap metrics
		if s.maxResidentMemoryBytes > 0 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			// Use Sys (obtained from the OS) as an upper-bound approximation
			if uint64(m.Sys) > s.maxResidentMemoryBytes {
				if s.quarantineOnError {
					result.Violations = append(result.Violations, types.Violation{RuleID: "memory_limit", Message: "memory ceiling exceeded", Severity: "HIGH"})
					result.DurationMs = time.Since(start).Milliseconds()
					return result, nil
				}
				return types.ScanResult{}, fmt.Errorf("memory ceiling exceeded: sys=%d bytes", m.Sys)
			}
		}

		lineNum++
		line := scanner.Text()
		result.Metrics.LinesRead++
		result.Metrics.BytesRead += int64(len(line))
		if s.maxStreamBytes > 0 && result.Metrics.BytesRead > s.maxStreamBytes {
			if s.quarantineOnError {
				result.Violations = append(result.Violations, types.Violation{RuleID: "stream_limit", Message: "stream byte limit exceeded", Severity: "HIGH"})
				result.DurationMs = time.Since(start).Milliseconds()
				return result, nil
			}
			return types.ScanResult{}, sharederrors.ErrStreamLimitExceeded
		}

		compiledRuleSeen := make(map[string]struct{})
		if s.evaluateLine(&result, line, lineNum, compiledRuleSeen) && s.firstMatch {
			// If first_match strategy, we could short-circuit the entire file on first line match.
			// However, strategy typically applies per input line; keep scanning lines.
		}
	}

	if err := scanner.Err(); err != nil {
		if s.quarantineOnError {
			result.Violations = append(result.Violations, types.Violation{RuleID: "scan_error", Message: err.Error(), Severity: "HIGH"})
			result.DurationMs = time.Since(start).Milliseconds()
			return result, nil
		}
		return types.ScanResult{}, fmt.Errorf("scanning %s: %w", inputName, err)
	}
	if s.logger != nil {
		s.logger.Debug("scan reader end", "input", inputName, "violations", len(result.Violations), "bytes", result.Metrics.BytesRead, "lines", result.Metrics.LinesRead)
	}
	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}

// scanChunked handles very long lines without loading them fully in memory.
// It incrementally builds lines up to bufferSizeBytes, and when a single line
// exceeds that, it evaluates in segments with a small overlap to reduce the
// chance of missing cross-boundary matches.
func (s *Scanner) scanChunked(ctx context.Context, r io.Reader, inputName string) (types.ScanResult, error) {
	result := types.ScanResult{Input: inputName}
	reader := bufio.NewReader(r)
	var (
		lineBuf   []byte
		lineNum   int64
		chunk     = make([]byte, 64*1024)
		overlapSz = 8 * 1024
	)
	if s.chunkOverlapBytes > 0 {
		overlapSz = s.chunkOverlapBytes
	}
	for {
		n, readErr := reader.Read(chunk)
		if n > 0 {
			data := chunk[:n]
			for len(data) > 0 {
				// Find newline
				idx := bytesIndexByte(data, '\n')
				if idx >= 0 {
					// append part up to newline
					lineBuf = append(lineBuf, data[:idx]...)
					// evaluate complete line
					lineNum++
					result.Metrics.LinesRead++
					result.Metrics.BytesRead += int64(len(lineBuf) + 1)
					if s.maxStreamBytes > 0 && result.Metrics.BytesRead > s.maxStreamBytes {
						if s.quarantineOnError {
							result.Violations = append(result.Violations, types.Violation{RuleID: "stream_limit", Message: "stream byte limit exceeded", Severity: "HIGH"})
							return result, nil
						}
						return types.ScanResult{}, sharederrors.ErrStreamLimitExceeded
					}
					s.evaluateLongLine(&result, lineBuf, lineNum, overlapSz)
					lineBuf = lineBuf[:0]
					data = data[idx+1:]
					continue
				}
				// no newline in data; append and continue
				lineBuf = append(lineBuf, data...)
				result.Metrics.BytesRead += int64(len(data))
				if s.maxStreamBytes > 0 && result.Metrics.BytesRead > s.maxStreamBytes {
					if s.quarantineOnError {
						result.Violations = append(result.Violations, types.Violation{RuleID: "stream_limit", Message: "stream byte limit exceeded", Severity: "HIGH"})
						return result, nil
					}
					return types.ScanResult{}, sharederrors.ErrStreamLimitExceeded
				}
				break
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				if len(lineBuf) > 0 {
					lineNum++
					result.Metrics.LinesRead++
					s.evaluateLongLine(&result, lineBuf, lineNum, overlapSz)
				}
				return result, nil
			}
			return types.ScanResult{}, fmt.Errorf("chunked read %s: %w", inputName, readErr)
		}
		select {
		case <-ctx.Done():
			return types.ScanResult{}, ctx.Err()
		default:
		}
	}
}
