package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("config validation failed for %s=%v: %s", e.Field, e.Value, e.Message)
}

// ValidateConfig performs comprehensive validation of configuration values
// This complements Validate in config.go and aligns with the current Config struct shape.
func ValidateConfig(cfg Config) []ValidationError {
	var errors []ValidationError

	// Validate worker count
	if cfg.Workers < 0 {
		errors = append(errors, ValidationError{
			Field:   "workers",
			Value:   cfg.Workers,
			Message: "workers must be non-negative (0 = auto)",
		})
	}
	if cfg.Workers > 1000 {
		errors = append(errors, ValidationError{
			Field:   "workers",
			Value:   cfg.Workers,
			Message: "workers must not exceed 1000",
		})
	}

	// Validate timeout strings when provided
	if d := cfg.Performance.Timeout; d != "" {
		if dur, err := time.ParseDuration(d); err != nil || dur < 1*time.Millisecond || dur > 30*time.Second {
			errors = append(errors, ValidationError{Field: "performance.timeout", Value: d, Message: "must be a duration between 1ms and 30s"})
		}
	}
	if d := cfg.Performance.PerRuleTimeout; d != "" {
		if dur, err := time.ParseDuration(d); err != nil || dur < 1*time.Millisecond || dur > 5*time.Second {
			errors = append(errors, ValidationError{Field: "performance.per_rule_timeout", Value: d, Message: "must be a duration between 1ms and 5s"})
		}
	}
	if d := cfg.Performance.TotalScanTimeout; d != "" {
		if dur, err := time.ParseDuration(d); err != nil || dur < 1*time.Millisecond || dur > 5*time.Minute {
			errors = append(errors, ValidationError{Field: "performance.total_scan_timeout", Value: d, Message: "must be a duration between 1ms and 5m"})
		}
	}

	// Validate buffer size
	if cfg.Performance.BufferBytes > 0 && cfg.Performance.BufferBytes < 1024 {
		errors = append(errors, ValidationError{
			Field:   "performance.buffer_bytes",
			Value:   cfg.Performance.BufferBytes,
			Message: "buffer size must be at least 1KB",
		})
	}
	if cfg.Performance.BufferBytes > 100*1024*1024 {
		errors = append(errors, ValidationError{
			Field:   "performance.buffer_bytes",
			Value:   cfg.Performance.BufferBytes,
			Message: "buffer size must not exceed 100MB",
		})
	}

	// Validate fail threshold (keep consistent with severity package and config.Validate)
	if cfg.FailOn != "" {
		validThresholds := []string{"INFO", "WARNING", "HIGH", "ERROR", "CRITICAL"}
		valid := false
		for _, t := range validThresholds {
			if strings.EqualFold(cfg.FailOn, t) {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, ValidationError{Field: "fail_on", Value: cfg.FailOn, Message: "must be one of: INFO, WARNING, HIGH, ERROR, CRITICAL"})
		}
	}

	return errors
}

// ValidateAPIKey validates the format of API keys for different providers
func ValidateAPIKey(provider, key string) error {
	if key == "" {
		return errors.New("API key cannot be empty")
	}

	switch provider {
	case "openai":
		// OpenAI keys start with sk- and have specific format
		if !regexp.MustCompile(`^sk-[a-zA-Z0-9]{48}$`).MatchString(key) {
			return errors.New("OpenAI API key must start with 'sk-' and be 51 characters total")
		}
	case "anthropic":
		// Anthropic keys start with sk-ant- and have specific format
		if !regexp.MustCompile(`^sk-ant-[a-zA-Z0-9\-_]{95}$`).MatchString(key) {
			return errors.New("Anthropic API key must start with 'sk-ant-' and be 108 characters total")
		}
	default:
		// Generic validation for unknown providers
		if len(key) < 16 {
			return errors.New("API key must be at least 16 characters")
		}
		if len(key) > 512 {
			return errors.New("API key must not exceed 512 characters")
		}
	}

	return nil
}

// ValidateEnforcementMode validates enforcement mode values
func ValidateEnforcementMode(mode string) error {
	if mode == "" {
		return nil // empty is allowed, defaults to "observe"
	}
	validModes := []string{"observe", "redact", "quarantine", "enforce"}
	for _, m := range validModes {
		if mode == m {
			return nil
		}
	}
	return fmt.Errorf("enforcement mode must be one of: %v", validModes)
}

// ValidateResourceLimits validates resource limit configurations
func ValidateResourceLimits(maxFileBytes, maxMemoryBytes int64, maxPatternLength int) []ValidationError {
	var errors []ValidationError

	if maxFileBytes > 0 && maxFileBytes < 1024 {
		errors = append(errors, ValidationError{Field: "max_file_bytes", Value: maxFileBytes, Message: "max file size must be at least 1KB if specified"})
	}

	if maxMemoryBytes > 0 && maxMemoryBytes < 1024*1024 {
		errors = append(errors, ValidationError{Field: "max_memory_bytes", Value: maxMemoryBytes, Message: "max memory must be at least 1MB if specified"})
	}

	if maxPatternLength < 1 {
		errors = append(errors, ValidationError{Field: "max_pattern_length", Value: maxPatternLength, Message: "max pattern length must be at least 1"})
	}
	if maxPatternLength > 10000 {
		errors = append(errors, ValidationError{Field: "max_pattern_length", Value: maxPatternLength, Message: "max pattern length must not exceed 10000 to avoid catastrophic backtracking"})
	}

	return errors
}
