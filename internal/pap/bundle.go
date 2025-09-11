package pap

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"
	"strings"

	"github.com/google/uuid"
	pscrypto "github.com/promptshield/promptshield/internal/security/crypto"
)

// Bundle represents a signed, distributable policy package for a specific tenant/rulepack version.
// Signing uses HMAC-SHA256 with PS_RULEPACK_HMAC_KEY (base64) over a canonical JSON payload.
// Consumers should Verify before trusting.
//
// The payload is small (KB–MB) and meant for low-cost storage/CDN distribution.

type Bundle struct {
	TenantID   uuid.UUID       `json:"tenant_id"`
	RulepackID uuid.UUID       `json:"rulepack_id"`
	Version    int             `json:"version"`
	CreatedAt  time.Time       `json:"created_at"`
	DSL        json.RawMessage `json:"dsl"`
	Checksum   string          `json:"checksum_sha256"` // hex(SHA256(DSL))
	SigAlg     string          `json:"sig_alg"`         // hmac-sha256
	KeyID      string          `json:"key_id,omitempty"`
	Signature  string          `json:"signature_b64"`
}

// BuildBundle constructs a Bundle and signs it.
func BuildBundle(tenantID, rulepackID uuid.UUID, version int, dsl json.RawMessage) (Bundle, error) {
	b := Bundle{
		TenantID:   tenantID,
		RulepackID: rulepackID,
		Version:    version,
		CreatedAt:  time.Now().UTC(),
		DSL:        dsl,
		Checksum:   checksum(dsl),
		SigAlg:     "hmac-sha256",
		KeyID:      stringsTrim(os.Getenv("PS_RULEPACK_HMAC_KEY_ID")),
	}
	payload := map[string]any{
		"tenant_id":       b.TenantID.String(),
		"rulepack_id":     b.RulepackID.String(),
		"version":         b.Version,
		"created_at":      b.CreatedAt.Format(time.RFC3339),
		"checksum_sha256": b.Checksum,
		// avoid embedding DSL into the signed map directly to keep canonical payload small
	}
	sig, err := SignBundlePayload(payload)
	if err != nil {
		return Bundle{}, err
	}
	b.Signature = sig
	return b, nil
}

// VerifyBundle validates HMAC signature and checksum.
func VerifyBundle(b Bundle) error {
	if b.SigAlg != "hmac-sha256" {
		return fmt.Errorf("unsupported signature algorithm: %s", b.SigAlg)
	}
	payload := map[string]any{
		"tenant_id":       b.TenantID.String(),
		"rulepack_id":     b.RulepackID.String(),
		"version":         b.Version,
		"created_at":      b.CreatedAt.Format(time.RFC3339),
		"checksum_sha256": b.Checksum,
	}
	expected, err := SignBundlePayload(payload)
	if err != nil {
		return err
	}
	if subtleConstantTimeCompare(expected, b.Signature) == false {
		return fmt.Errorf("invalid bundle signature")
	}
	if checksum(b.DSL) != b.Checksum {
		return fmt.Errorf("checksum mismatch")
	}
	return nil
}

// SignBundlePayload signs the canonical payload using PS_RULEPACK_HMAC_KEY (base64).
func SignBundlePayload(payload map[string]any) (string, error) {
	// Reuse decision signer to ensure canonicalization + HMAC flow
	// It expects key in PS_DECISION_HMAC_KEY. Mirror the key if only bundle key is set.
	if os.Getenv("PS_DECISION_HMAC_KEY") == "" {
		if k := os.Getenv("PS_RULEPACK_HMAC_KEY"); k != "" {
			_ = os.Setenv("PS_DECISION_HMAC_KEY", k)
		}
	}
	return pscrypto.SignHMAC256(payload)
}

func checksum(dsl json.RawMessage) string {
	s := sha256.Sum256(dsl)
	return hex.EncodeToString(s[:])
}

func subtleConstantTimeCompare(a, b string) bool {
	ab, err1 := base64.StdEncoding.DecodeString(a)
	bb, err2 := base64.StdEncoding.DecodeString(b)
	if err1 != nil || err2 != nil {
		return false
	}
	if len(ab) != len(bb) {
		return false
	}
	var v byte
	for i := range ab {
		v |= ab[i] ^ bb[i]
	}
	return v == 0
}

func stringsTrim(s string) string {
	return strings.TrimSpace(s)
}

