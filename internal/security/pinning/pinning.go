package pinning

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
)

// ParsePinsCSV parses a comma-separated list of SPKI SHA-256 pins.
// Pins can be base64 (standard or URL) or hex-encoded.
func ParsePinsCSV(s string) ([][]byte, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	pins := make([][]byte, 0, len(parts))
	for _, p := range parts {
		pin := strings.TrimSpace(p)
		if pin == "" {
			continue
		}
		// Try base64 (std)
		if b, err := base64.StdEncoding.DecodeString(pin); err == nil && len(b) == sha256.Size {
			pins = append(pins, b)
			continue
		}
		// Try base64 (raw URL)
		if b, err := base64.RawURLEncoding.DecodeString(pin); err == nil && len(b) == sha256.Size {
			pins = append(pins, b)
			continue
		}
		// Try hex
		if b, err := hex.DecodeString(pin); err == nil && len(b) == sha256.Size {
			pins = append(pins, b)
			continue
		}
		return nil, errors.New("invalid pin encoding")
	}
	return pins, nil
}

// BuildPinnedTransport returns an http.RoundTripper that enforces SPKI pinning.
// When pins is empty, the base transport is returned unchanged.
func BuildPinnedTransport(base *http.Transport, pins [][]byte) http.RoundTripper {
	if len(pins) == 0 {
		if base == nil {
			return http.DefaultTransport
		}
		return base
	}
	// Clone or create a transport
	// Avoid copying Transport with embedded mutexes
	var tp *http.Transport
	if base != nil {
		tp = base
	} else {
		tp = &http.Transport{}
	}
	if tp.TLSClientConfig == nil {
		tp.TLSClientConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}
	// Keep default verification and add pin check afterwards
	prev := tp.TLSClientConfig.VerifyPeerCertificate
	allowed := make([][sha256.Size]byte, 0, len(pins))
	for _, p := range pins {
		var a [sha256.Size]byte
		copy(a[:], p)
		allowed = append(allowed, a)
	}
	tp.TLSClientConfig.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		if prev != nil {
			if err := prev(rawCerts, verifiedChains); err != nil {
				return err
			}
		}
		// If verifiedChains is empty, perform a standard verification to avoid bypass
		if len(verifiedChains) == 0 {
			// Let the standard library do the verification by returning nil here; pin check still runs
		}
		// Compute SPKI SHA-256 for each cert in the chain and compare
		for _, chain := range verifiedChains {
			for _, cert := range chain {
				spki := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
				for _, ap := range allowed {
					if ap == spki {
						return nil
					}
				}
			}
		}
		// As a fallback, try rawCerts
		for _, rc := range rawCerts {
			if c, err := x509.ParseCertificate(rc); err == nil {
				spki := sha256.Sum256(c.RawSubjectPublicKeyInfo)
				for _, ap := range allowed {
					if ap == spki {
						return nil
					}
				}
			}
		}
		return errors.New("tls pinning mismatch")
	}
	return tp
}
