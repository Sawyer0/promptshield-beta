package report

import (
	"io"
	"sync"

	"github.com/promptshield/promptshield/internal/encoding/jsonx"
	"github.com/promptshield/promptshield/pkg/types"
)

// NDJSONEventWriter writes newline-delimited JSON events for violations and a final summary.
// Each Write* call emits one compact JSON object followed by a newline.
type NDJSONEventWriter struct {
	enc *jsonx.Encoder
	mu  sync.Mutex
}

func NewNDJSONEventWriter(w io.Writer) *NDJSONEventWriter {
	return &NDJSONEventWriter{enc: jsonx.NewEncoder(w)}
}

// WriteViolation emits a single violation event line with file context.
func (n *NDJSONEventWriter) WriteViolation(file string, v types.Violation) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.enc.Encode(map[string]any{
		"type":     "violation",
		"file":     file,
		"rule_id":  v.RuleID,
		"message":  v.Message,
		"severity": v.Severity,
		"line":     v.Line,
		"column":   v.Column,
	})
}

// WriteSummary emits a final summary line after all violations are written.
func (n *NDJSONEventWriter) WriteSummary(filesScanned int, violationCount int) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.enc.Encode(map[string]any{
		"type":            "summary",
		"files_scanned":   filesScanned,
		"violation_count": violationCount,
	})
}
