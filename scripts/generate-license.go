package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

type license struct {
	Organization string    `json:"org"`
	ExpiresAt    time.Time `json:"expires_at"`
	Tier         string    `json:"tier"`
}

func main() {
	org := flag.String("org", "", "organization name")
	tier := flag.String("tier", "startup", "tier: startup|business|enterprise")
	days := flag.Int("days", 30, "validity in days")
	privKeyB64 := flag.String("privkey", "", "ed25519 private key (base64, 64 bytes)")
	privKeyFile := flag.String("privkey-file", "", "path to file containing ed25519 private key (base64)")
	flag.Parse()

	if *org == "" {
		fatal("-org is required")
	}
	switch strings.ToLower(*tier) {
	case "startup", "business", "enterprise":
	default:
		fatal("-tier must be startup|business|enterprise")
	}

	var priv ed25519.PrivateKey
	var err error
	if *privKeyFile != "" {
		b, e := os.ReadFile(*privKeyFile)
		if e != nil {
			fatal("reading private key file: %v", e)
		}
		s := strings.TrimSpace(string(b))
		priv, err = decodePrivateKey(s)
		if err != nil {
			fatal("invalid private key: %v", err)
		}
	} else if *privKeyB64 != "" {
		priv, err = decodePrivateKey(*privKeyB64)
		if err != nil {
			fatal("invalid private key: %v", err)
		}
	} else {
		pub, p, e := ed25519.GenerateKey(nil)
		if e != nil {
			fatal("generating key: %v", e)
		}
		fmt.Println("Generated a new keypair for convenience:")
		fmt.Printf("PUBLIC KEY  (export PROMPTSHIELD_LICENSE_PUBLIC_KEY): %s\n", base64.RawURLEncoding.EncodeToString(pub))
		fmt.Printf("PRIVATE KEY (use with -privkey): %s\n\n", base64.RawURLEncoding.EncodeToString(p))
		priv = p
	}

	lic := license{
		Organization: *org,
		Tier:         strings.ToLower(*tier),
		ExpiresAt:    time.Now().AddDate(0, 0, *days),
	}
	payload, err := json.Marshal(lic)
	if err != nil {
		fatal("marshal payload: %v", err)
	}
	sig := ed25519.Sign(priv, payload)
	token := base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(sig)
	fmt.Println(token)
}

func fatal(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func decodePrivateKey(b64 string) (ed25519.PrivateKey, error) {
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return nil, err
	}
	if l := len(b); l != ed25519.PrivateKeySize {
		return nil, errors.New("invalid private key length")
	}
	return ed25519.PrivateKey(b), nil
}
