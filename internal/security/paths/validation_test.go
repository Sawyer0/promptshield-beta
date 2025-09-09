package paths

import (
	"strings"
	"testing"
)

func TestValidateCAFilePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid PEM file",
			path:    "/etc/ssl/certs/ca.pem",
			wantErr: false,
		},
		{
			name:    "valid CRT file",
			path:    "/etc/ssl/certs/ca.crt",
			wantErr: false,
		},
		{
			name:    "valid CERT file",
			path:    "/etc/ssl/certs/ca.cert",
			wantErr: false,
		},
		{
			name:    "valid CA-BUNDLE file",
			path:    "/etc/ssl/certs/ca.ca-bundle",
			wantErr: false,
		},
		{
			name:    "valid CER file",
			path:    "/etc/ssl/certs/ca.cer",
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "path traversal",
			path:    "/etc/ssl/../../../etc/passwd",
			wantErr: true,
			errMsg:  "path traversal",
		},
		{
			name:    "relative path with traversal",
			path:    "../etc/ssl/certs/ca.pem",
			wantErr: true,
			errMsg:  "path traversal",
		},
		{
			name:    "invalid extension",
			path:    "/etc/ssl/certs/ca.txt",
			wantErr: true,
			errMsg:  "invalid CA file extension",
		},
		{
			name:    "relative path",
			path:    "ca.pem",
			wantErr: true,
			errMsg:  "must be absolute",
		},
		{
			name:    "too long path",
			path:    "/" + strings.Repeat("a", 4100) + ".pem",
			wantErr: true,
			errMsg:  "too long",
		},
		{
			name:    "no extension",
			path:    "/etc/ssl/certs/ca",
			wantErr: true,
			errMsg:  "invalid CA file extension",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCAFilePath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateCAFilePath() expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateCAFilePath() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateCAFilePath() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateConfigFilePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid YAML file",
			path:    "/etc/promptshield/config.yaml",
			wantErr: false,
		},
		{
			name:    "valid YML file",
			path:    "/etc/promptshield/config.yml",
			wantErr: false,
		},
		{
			name:    "valid JSON file",
			path:    "/etc/promptshield/config.json",
			wantErr: false,
		},
		{
			name:    "valid TOML file",
			path:    "/etc/promptshield/config.toml",
			wantErr: false,
		},
		{
			name:    "invalid extension",
			path:    "/etc/promptshield/config.txt",
			wantErr: true,
			errMsg:  "invalid config file extension",
		},
		{
			name:    "path traversal",
			path:    "/etc/promptshield/../../../etc/passwd.yaml",
			wantErr: true,
			errMsg:  "path traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfigFilePath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateConfigFilePath() expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateConfigFilePath() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateConfigFilePath() unexpected error = %v", err)
				}
			}
		})
	}
}