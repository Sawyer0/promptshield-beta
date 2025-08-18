package otel

import (
	"go.opentelemetry.io/otel/attribute"
)

// ToAttributes converts a map to OpenTelemetry attributes, filtering by allowed keys
func ToAttributes(m map[string]any, allow []string) []attribute.KeyValue {
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