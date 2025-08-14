package testutil

import (
	"testing"

	"github.com/promptshield/promptshield/internal/config"
)

// ConfigHelper provides test utilities for config validation
type ConfigHelper struct {
	t *testing.T
}

// NewConfigHelper creates a new config test helper
func NewConfigHelper(t *testing.T) *ConfigHelper {
	t.Helper()
	return &ConfigHelper{t: t}
}

// AssertValid checks that config validates without errors
func (h *ConfigHelper) AssertValid(cfg config.Config) {
	h.t.Helper()
	if errs := config.Validate(cfg); len(errs) > 0 {
		h.t.Errorf("unexpected validation errors: %v", errs)
	}
}

// AssertInvalid checks that config has validation error containing substr
func (h *ConfigHelper) AssertInvalid(cfg config.Config, wantErr string) {
	h.t.Helper()
	errs := config.Validate(cfg)
	if len(errs) == 0 {
		h.t.Error("expected validation error but got none")
		return
	}
	for _, err := range errs {
		if contains(err.Error(), wantErr) {
			return // Found expected error
		}
	}
	h.t.Errorf("no error contained %q, got: %v", wantErr, errs)
}

// ValidateTable runs table-driven validation tests
func (h *ConfigHelper) ValidateTable(tests []struct {
	name    string
	cfg     config.Config
	wantErr string
}) {
	h.t.Helper()
	for _, tt := range tests {
		h.t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr == "" {
				h.AssertValid(tt.cfg)
			} else {
				h.AssertInvalid(tt.cfg, tt.wantErr)
			}
		})
	}
}
