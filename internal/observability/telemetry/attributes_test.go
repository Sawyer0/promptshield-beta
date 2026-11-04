package telemetry

import "testing"

func TestToAttributesFiltersByAllowList(t *testing.T) {
	attrs := toAttributes(map[string]any{
		"version": "1.0", "workers": 4, "ci": true, "secret": "nope", "ratio": 0.5,
	}, []string{"version", "workers", "ci"})
	if len(attrs) != 3 {
		t.Fatalf("got %d attributes, want 3: %v", len(attrs), attrs)
	}
	for _, kv := range attrs {
		if kv.Key == "secret" {
			t.Fatal("non-allowed key leaked into attributes")
		}
	}
}

func TestSanitizeEndpoint(t *testing.T) {
	tests := map[string]string{
		"https://otel.example.com:4317/v1/traces": "otel.example.com:4317",
		"http://collector:4318":                   "collector:4318",
		"collector:4317":                          "collector:4317",
		"   ":                                     "",
	}
	for in, want := range tests {
		if got := sanitizeEndpoint(in); got != want {
			t.Errorf("sanitizeEndpoint(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestOrDefault(t *testing.T) {
	if got := orDefault(" ", "dev"); got != "dev" {
		t.Errorf("blank: got %q", got)
	}
	if got := orDefault("1.2.3", "dev"); got != "1.2.3" {
		t.Errorf("set: got %q", got)
	}
}
