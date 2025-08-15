package paths

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateCAFilePath validates that a file path is safe for loading CA certificates.
// It checks for:
// - Valid file extensions (.pem, .crt, .cert, .ca-bundle)
// - No path traversal attempts
// - Reasonable path length limits
func ValidateCAFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("CA file path cannot be empty")
	}

	// Check path length to prevent potential buffer overflows
	if len(path) > 4096 {
		return fmt.Errorf("CA file path too long (max 4096 characters)")
	}

	// Check for traversal attempts before cleaning
	if strings.Contains(path, "..") {
		return fmt.Errorf("CA file path contains path traversal components")
	}

	// Clean the path 
	cleanPath := filepath.Clean(path)

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(cleanPath))
	validExtensions := map[string]bool{
		".pem":       true,
		".crt":       true,
		".cert":      true,
		".ca-bundle": true,
		".cer":       true,
	}

	if !validExtensions[ext] {
		return fmt.Errorf("invalid CA file extension %q (allowed: .pem, .crt, .cert, .ca-bundle, .cer)", ext)
	}

	// Ensure it's an absolute path for security
	if !filepath.IsAbs(cleanPath) {
		return fmt.Errorf("CA file path must be absolute")
	}

	return nil
}

// ValidateConfigFilePath validates configuration file paths with similar security checks
func ValidateConfigFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("config file path cannot be empty")
	}

	if len(path) > 4096 {
		return fmt.Errorf("config file path too long (max 4096 characters)")
	}

	// Check for traversal attempts before cleaning
	if strings.Contains(path, "..") {
		return fmt.Errorf("config file path contains path traversal components")
	}

	cleanPath := filepath.Clean(path)

	ext := strings.ToLower(filepath.Ext(cleanPath))
	validExtensions := map[string]bool{
		".yaml": true,
		".yml":  true,
		".json": true,
		".toml": true,
	}

	if !validExtensions[ext] {
		return fmt.Errorf("invalid config file extension %q (allowed: .yaml, .yml, .json, .toml)", ext)
	}

	return nil
}