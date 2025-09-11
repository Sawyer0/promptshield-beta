package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// SignHMAC256 computes an HMAC-SHA256 signature over a canonical JSON encoding of data.
// The secret is read from PS_DECISION_HMAC_KEY (base64). Returns base64(signature).
func SignHMAC256(data map[string]any) (string, error) {
	keyB64 := os.Getenv("PS_DECISION_HMAC_KEY")
	if keyB64 == "" {
		return "", fmt.Errorf("PS_DECISION_HMAC_KEY not set")
	}
	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return "", fmt.Errorf("invalid PS_DECISION_HMAC_KEY: %w", err)
	}
	// Canonicalize map by sorting keys before JSON encoding
	canonical := canonicalize(data)
	payload, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("marshal canonical json: %w", err)
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(payload)
	sig := mac.Sum(nil)
	return base64.StdEncoding.EncodeToString(sig), nil
}

// canonicalize produces a deterministically ordered representation suitable for signing.
func canonicalize(v any) any {
	switch x := v.(type) {
	case map[string]any:
		// Sort keys
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make(map[string]any, len(x))
		for _, k := range keys {
			out[k] = canonicalize(x[k])
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = canonicalize(x[i])
		}
		return out
	default:
		return x
	}
}
