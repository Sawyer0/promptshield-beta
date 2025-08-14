package config

import (
	"context"
	"strings"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.OutputFormat != "stylish" {
		t.Errorf("OutputFormat = %q, want stylish", cfg.OutputFormat)
	}
	if cfg.Workers != 0 {
		t.Errorf("Workers = %d, want 0", cfg.Workers)
	}
	if cfg.Debug != false {
		t.Error("Debug should be false")
	}
}

func TestValidate(t *testing.T) {
	assertValid := func(cfg Config) {
		if errs := Validate(cfg); len(errs) > 0 {
			t.Fatalf("unexpected validation errors: %v", errs)
		}
	}
	assertInvalid := func(cfg Config, want string) {
		errs := Validate(cfg)
		if len(errs) == 0 {
			t.Fatalf("expected validation error containing %q, got none", want)
		}
		for _, e := range errs {
			if strings.Contains(e.Error(), want) {
				return
			}
		}
		t.Fatalf("no error contained %q, got: %v", want, errs)
	}

	t.Run("output formats", func(t *testing.T) {
		assertValid(Config{OutputFormat: "stylish"})
		assertValid(Config{OutputFormat: "json"})
		assertValid(Config{OutputFormat: "github"})
		assertValid(Config{OutputFormat: "ndjson"})
		assertValid(Config{OutputFormat: ""})
		assertInvalid(Config{OutputFormat: "xml"}, "invalid output_format")
	})

	t.Run("workers", func(t *testing.T) {
		assertValid(Config{Workers: 0})
		assertValid(Config{Workers: 10})
		assertInvalid(Config{Workers: -1}, "workers must be >= 0")
	})

	t.Run("severity", func(t *testing.T) {
		assertValid(Config{FailOn: "WARNING"})
		assertValid(Config{FailOn: "HIGH"})
		assertValid(Config{FailOn: ""})
		assertInvalid(Config{FailOn: "UNKNOWN"}, "invalid fail_on")
	})

	t.Run("composition strategy", func(t *testing.T) {
		assertValid(Config{Composition: struct {
			Strategy string `yaml:"strategy" json:"strategy"`
		}{"first_match"}})
		assertValid(Config{Composition: struct {
			Strategy string `yaml:"strategy" json:"strategy"`
		}{"priority_order"}})
		assertInvalid(Config{Composition: struct {
			Strategy string `yaml:"strategy" json:"strategy"`
		}{"random"}}, "invalid composition.strategy")
	})
}

func TestCheckUnknownKeys(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{"valid", `output_format: json`, ""},
		{"empty", ``, ""},
		{"unknown key", `unknown: value`, "unknown config key: unknown"},
		{"nested unknown", `composition:
  invalid: true`, "unknown config key: composition.invalid"},
		{"invalid yaml", `[unclosed`, "parsing config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckUnknownKeys([]byte(tt.yaml))
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %v, want substring %q", err, tt.wantErr)
				}
			}
		})
	}
}

func TestReadEffective(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		cfg := ReadEffective(context.Background(), "", func(key string) any { return nil })
		if cfg.OutputFormat != "stylish" || cfg.Workers != 0 {
			t.Error("expected defaults")
		}
	})

	t.Run("overrides", func(t *testing.T) {
		cfg := ReadEffective(context.Background(), "", func(key string) any {
			switch key {
			case "output_format":
				return "json"
			case "workers":
				return 8
			}
			return nil
		})
		if cfg.OutputFormat != "json" || cfg.Workers != 8 {
			t.Errorf("got format=%q workers=%d", cfg.OutputFormat, cfg.Workers)
		}
	})

	t.Run("type conversions", func(t *testing.T) {
		cfg := ReadEffective(context.Background(), "", func(key string) any {
			switch key {
			case "workers":
				return int64(4)
			case "debug":
				return "true"
			}
			return nil
		})
		if cfg.Workers != 4 || !cfg.Debug {
			t.Error("type conversion failed")
		}
	})
}

func TestHelpers(t *testing.T) {
	if toString("test", "def") != "test" {
		t.Error("toString failed")
	}
	if toInt(42, 0) != 42 {
		t.Error("toInt failed")
	}
	if toBool("true", false) != true {
		t.Error("toBool failed")
	}
}

// boolPtr removed: no longer used
