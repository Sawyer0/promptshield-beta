package output

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"stylish", "stylish"},
		{"STYLISH", "stylish"},
		{"json", "json"},
		{"JSON", "json"},
		{"ndjson", "ndjson"},
		{"invalid", "invalid"}, // Returns unchanged if not valid
		{"", ""},
		{"json ", "json "}, // Spaces not trimmed
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := Normalize(tt.input); got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		format  string
		wantErr bool
	}{
		{"stylish", false},
		{"STYLISH", false},
		{"json", false},
		{"ndjson", false},
		{"", false}, // Empty is valid
		{"invalid", true},
		{"xml", true},
		{"  ", true},
		{"styl", true},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			err := Validate(tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate(%q) error = %v, wantErr = %v", tt.format, err, tt.wantErr)
			}
		})
	}
}
