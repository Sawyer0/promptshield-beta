package audit

import (
	"crypto/sha256"
	"encoding/hex"
	stdjson "encoding/json"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/promptshield/promptshield/internal/encoding/jsonx"
	"github.com/promptshield/promptshield/internal/shared/redact"
)

type Logger interface {
	Log(event Event) error
}

type Event struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Hash      string                 `json:"hash"`
	PrevHash  string                 `json:"prev_hash"`
}

type FileLogger struct {
	enc     *jsonx.Encoder
	mu      sync.Mutex
	prevSum string
}

func NewFileLogger(w io.Writer) *FileLogger { return &FileLogger{enc: jsonx.NewEncoder(w)} }

func (l *FileLogger) Log(e Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()
    // Always sanitize data before hashing/writing to ensure redaction
    if e.Data != nil {
        e.Data = SanitizeMap(e.Data)
    }
	e.Timestamp = time.Now().UTC()
	e.PrevHash = l.prevSum
	e.Hash = hashEvent(e)
	if err := l.enc.Encode(e); err != nil {
		return err
	}
	l.prevSum = e.Hash
	return nil
}

// SanitizeMap returns a redacted deep copy of the provided map, applying
// token redaction to all string values. Nested maps and slices are handled.
func SanitizeMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = sanitizeValue(v)
	}
	return out
}

func sanitizeValue(v any) any {
	switch x := v.(type) {
	case string:
		return redact.Redact(x)
	case map[string]any:
		return SanitizeMap(x)
	case []any:
		y := make([]any, len(x))
		for i := range x {
			y[i] = sanitizeValue(x[i])
		}
		return y
	case []string:
		y := make([]string, len(x))
		for i := range x {
			y[i] = redact.Redact(x[i])
		}
		return y
	default:
		return v
	}
}

func hashEvent(e Event) string {
	// Tamper-evident hash chain using SHA-256 over a canonical JSON payload.
	// Canonicalization sorts map keys recursively to ensure stable hashes.
	type canonKV struct {
		K string      `json:"k"`
		V interface{} `json:"v"`
	}
	type canonEvent struct {
		T int64       `json:"t"`
		Y string      `json:"y"`
		D interface{} `json:"d"`
		P string      `json:"p"`
	}

	var canonicalize func(interface{}) interface{}
	canonicalize = func(v interface{}) interface{} {
		switch x := v.(type) {
		case map[string]interface{}:
			// Sort keys for deterministic order
			keys := make([]string, 0, len(x))
			for k := range x {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			out := make([]canonKV, 0, len(keys))
			for _, k := range keys {
				out = append(out, canonKV{K: k, V: canonicalize(x[k])})
			}
			return out
		case []interface{}:
			out := make([]interface{}, len(x))
			for i := range x {
				out[i] = canonicalize(x[i])
			}
			return out
		default:
			return x
		}
	}

	payload := canonEvent{
		T: e.Timestamp.UnixNano(),
		Y: e.Type,
		D: canonicalize(e.Data),
		P: e.PrevHash,
	}
	b, _ := stdjson.Marshal(payload)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
