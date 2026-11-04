package telemetry

import (
	"net/url"
	"strings"

	"go.opentelemetry.io/otel/attribute"
)

// toAttributes converts a map to OpenTelemetry attributes, filtering by allowed keys
func toAttributes(m map[string]any, allow []string) []attribute.KeyValue {
	allowed := make(map[string]struct{}, len(allow))
	for _, k := range allow {
		allowed[k] = struct{}{}
	}

	var kvs []attribute.KeyValue
	for k, v := range m {
		if _, ok := allowed[k]; !ok {
			continue
		}

		switch x := v.(type) {
		case string:
			kvs = append(kvs, attribute.String(k, x))
		case bool:
			kvs = append(kvs, attribute.Bool(k, x))
		case int:
			kvs = append(kvs, attribute.Int(k, x))
		case int64:
			kvs = append(kvs, attribute.Int64(k, x))
		case float64:
			kvs = append(kvs, attribute.Float64(k, x))
		}
	}
	return kvs
}

// sanitizeEndpoint sanitizes an endpoint URL to extract just the host
func sanitizeEndpoint(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}

	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		// If parsing fails or no host, try to clean the string
		return strings.TrimPrefix(strings.TrimPrefix(s, "http://"), "https://")
	}
	return u.Host
}

// orDefault returns def when s is empty or whitespace.
func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}
