package severity

import "testing"

func TestMeetsThreshold(t *testing.T) {
	tests := []struct {
		name string
		sev  string
		thr  string
		want bool
	}{
		// Exact matches
		{"INFO=INFO", "INFO", "INFO", true},
		{"WARNING=WARNING", "WARNING", "WARNING", true},
		{"HIGH=HIGH", "HIGH", "HIGH", true},
		{"ERROR=ERROR", "ERROR", "ERROR", true},
		{"CRITICAL=CRITICAL", "CRITICAL", "CRITICAL", true},

		// Higher meets lower
		{"WARNING>INFO", "WARNING", "INFO", true},
		{"HIGH>INFO", "HIGH", "INFO", true},
		{"HIGH>WARNING", "HIGH", "WARNING", true},
		{"ERROR>HIGH", "ERROR", "HIGH", true},
		{"CRITICAL>ERROR", "CRITICAL", "ERROR", true},

		// Lower doesn't meet higher
		{"INFO<WARNING", "INFO", "WARNING", false},
		{"WARNING<HIGH", "WARNING", "HIGH", false},
		{"HIGH<ERROR", "HIGH", "ERROR", false},
		{"ERROR<CRITICAL", "ERROR", "CRITICAL", false},

		// Case insensitive
		{"lowercase", "info", "INFO", true},
		{"uppercase", "INFO", "info", true},

		// Unknown handling
		{"unknown sev", "UNKNOWN", "INFO", false},
		{"unknown thr", "INFO", "UNKNOWN", false},
		{"both unknown", "UNKNOWN", "INVALID", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MeetsThreshold(tt.sev, tt.thr); got != tt.want {
				t.Errorf("MeetsThreshold(%q, %q) = %v, want %v", tt.sev, tt.thr, got, tt.want)
			}
		})
	}
}

func TestMeetsThresholdOrdering(t *testing.T) {
	severities := []string{"INFO", "WARNING", "HIGH", "ERROR", "CRITICAL"}

	for i, lower := range severities {
		for j, higher := range severities {
			shouldMeet := i <= j
			got := MeetsThreshold(higher, lower)
			if got != shouldMeet {
				t.Errorf("MeetsThreshold(%q, %q) = %v, expected %v", higher, lower, got, shouldMeet)
			}
		}
	}
}
