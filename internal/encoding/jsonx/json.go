package jsonx

import (
	stdjson "encoding/json"
	"os"
	"strings"
	"sync"

	fastjson "github.com/goccy/go-json"
)

// Encoder is a drop-in compatible wrapper over json.Encoder that can route
// to either the standard library encoder or goccy/go-json depending on
// build tags or the PS_JSON_ENCODER environment variable.
//
// Env override (evaluated once on first use):
//
//	PS_JSON_ENCODER=fast | std   (default depends on build tag; std if none)
type Encoder struct {
	std     *stdjson.Encoder
	fast    *fastjson.Encoder
	useFast bool
}

// NewEncoder returns an Encoder that writes to w. The underlying implementation
// is selected by build tag default and can be overridden by PS_JSON_ENCODER.
func NewEncoder(w interface{ Write([]byte) (int, error) }) *Encoder {
	if shouldUseFast() {
		return &Encoder{fast: fastjson.NewEncoder(w), useFast: true}
	}
	return &Encoder{std: stdjson.NewEncoder(w)}
}

// Encode writes the JSON encoding of v followed by a newline.
func (e *Encoder) Encode(v any) error {
	if e.useFast {
		return e.fast.Encode(v)
	}
	return e.std.Encode(v)
}

// SetIndent sets indentation for the output.
func (e *Encoder) SetIndent(prefix, indent string) {
	if e.useFast {
		e.fast.SetIndent(prefix, indent)
		return
	}
	e.std.SetIndent(prefix, indent)
}

// Marshal encodes v to JSON bytes using the selected implementation.
func Marshal(v any) ([]byte, error) {
	if shouldUseFast() {
		return fastjson.Marshal(v)
	}
	return stdjson.Marshal(v)
}

// shouldUseFast returns true if the fast encoder should be used.
func shouldUseFast() bool {
	once.Do(initChoice)
	return choiceFast
}

var (
	once       sync.Once
	choiceFast bool
)

func initChoice() {
	// Default determined by build tag files. json_std.go sets defaultFast=false;
	// json_fast.go (with -tags jsonfast) sets defaultFast=true.
	choiceFast = defaultFast
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("PS_JSON_ENCODER"))); v != "" {
		switch v {
		case "fast", "go-json", "goccy":
			choiceFast = true
		case "std", "stdlib", "encoding":
			choiceFast = false
		}
	}
}
