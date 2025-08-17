package cred

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/99designs/keyring"
)

// Provider is a supported semantic provider name.
// Valid values: "openai", "anthropic".
type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
)

// SaveProviderAPIKey stores the API key for the given provider in the OS keyring.
func SaveProviderAPIKey(_ context.Context, provider Provider, apiKey string) error {
	p := strings.ToLower(string(provider))
	if p != string(ProviderOpenAI) && p != string(ProviderAnthropic) {
		return fmt.Errorf("unsupported provider: %s", provider)
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return errors.New("api key cannot be empty")
	}
	r, err := open()
	if err != nil {
		return err
	}
	item := keyring.Item{
		Key:   keyName(p),
		Label: fmt.Sprintf("PromptShield %s API key", p),
		Data:  []byte(apiKey),
	}
	return r.Set(item)
}

// GetProviderAPIKey retrieves the API key for the given provider from the OS keyring.
func GetProviderAPIKey(_ context.Context, provider Provider) (string, error) {
	p := strings.ToLower(string(provider))
	if p != string(ProviderOpenAI) && p != string(ProviderAnthropic) {
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
	r, err := open()
	if err != nil {
		return "", err
	}
	it, err := r.Get(keyName(p))
	if err != nil {
		return "", err
	}
	return string(it.Data), nil
}

func open() (keyring.Keyring, error) {
	// Let keyring choose the best backend for the OS.
	// ServiceName controls namespacing in most OS keychains.
	return keyring.Open(keyring.Config{
		ServiceName:   "promptshield",
		KeychainName:  "promptshield",
		WinCredPrefix: "promptshield",
	})
}

func keyName(provider string) string {
	switch provider {
	case string(ProviderOpenAI):
		return "openai_api_key"
	case string(ProviderAnthropic):
		return "anthropic_api_key"
	default:
		return provider + "_api_key"
	}
}
