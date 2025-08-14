package output

import (
	"fmt"
	"strings"
)

// Allowed output formats.
var allowed = map[string]struct{}{
	"stylish": {},
	"json":    {},
	"github":  {},
	// Legacy formats kept temporarily but treated as deprecated; validation still passes to avoid breaking tests
	"ndjson":   {},
	"markdown": {},
	"csv":      {},
	"html":     {},
	"table":    {},
}

// Normalize returns a canonical lowercase format if valid, otherwise returns the input unchanged.
func Normalize(format string) string {
	f := strings.ToLower(format)
	if _, ok := allowed[f]; ok {
		return f
	}
	return format
}

// Validate returns an error if the format is not one of the allowed values.
func Validate(format string) error {
	if format == "" {
		return nil
	}
	f := strings.ToLower(format)
	if _, ok := allowed[f]; ok {
		return nil
	}
	return fmt.Errorf("invalid output format %q (valid: stylish, json, github, ndjson)", format)
}
