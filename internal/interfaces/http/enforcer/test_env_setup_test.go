package enforcerhttp

import (
	"os"
)

// init test env to ensure DB is used during HTTP tests
func init() {
	// Map test DSN to runtime DSN if needed
	if os.Getenv("PS_PG_DSN") == "" {
		if v := os.Getenv("PS_TEST_PG_DSN"); v != "" {
			_ = os.Setenv("PS_PG_DSN", v)
		}
	}
	// Disable dev bypass in tests
	_ = os.Setenv("PS_DEV_BYPASS_AUTH", "false")
}
