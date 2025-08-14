package bootstrap

import (
	"bytes"
	"strings"
	"testing"
)

func TestSetupLogging(t *testing.T) {
	tests := []struct {
		name     string
		debug    bool
		quiet    bool
		format   string
		testLog  string
		wantLog  bool
		wantJSON bool
	}{
		{"default", false, false, "", "info", true, false},
		{"debug", true, false, "", "debug", true, false},
		{"quiet", false, true, "", "info", false, false},
		{"json", false, false, "json", "info", true, true},
		{"quiet overrides", true, true, "", "debug", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := SetupLogging(tt.debug, tt.quiet, tt.format, &buf)

			// Log at appropriate level
			if tt.testLog == "debug" {
				logger.Debug("test message")
			} else {
				logger.Info("test message")
			}

			output := buf.String()
			hasOutput := output != ""

			if hasOutput != tt.wantLog {
				t.Errorf("output present = %v, want %v", hasOutput, tt.wantLog)
			}

			if tt.wantJSON && !strings.Contains(output, `"msg":"test message"`) {
				t.Error("expected JSON format")
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
	t.Run("debug=false hides debug", func(t *testing.T) {
		var buf bytes.Buffer
		logger := SetupLogging(false, false, "", &buf)
		logger.Debug("debug msg")
		if strings.Contains(buf.String(), "debug msg") {
			t.Error("debug message shown when debug=false")
		}
	})

	t.Run("debug=true shows debug", func(t *testing.T) {
		var buf bytes.Buffer
		logger := SetupLogging(true, false, "", &buf)
		logger.Debug("debug msg")
		if !strings.Contains(buf.String(), "debug msg") {
			t.Error("debug message not shown when debug=true")
		}
	})
}
