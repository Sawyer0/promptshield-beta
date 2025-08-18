package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
)

// getEncryptionKey returns the encryption key from environment or generates one
func getEncryptionKey() ([]byte, error) {
	// Try to get key from environment first
	if keyStr := os.Getenv("PS_ENCRYPTION_KEY"); keyStr != "" {
		key, err := base64.StdEncoding.DecodeString(keyStr)
		if err != nil {
			return nil, fmt.Errorf("invalid PS_ENCRYPTION_KEY format: %w", err)
		}
		if len(key) != 32 {
			return nil, fmt.Errorf("PS_ENCRYPTION_KEY must be 32 bytes (256 bits)")
		}
		return key, nil
	}
	
	// Fallback: derive from tenant-specific environment variable
	if seedStr := os.Getenv("PS_TENANT_SECRET"); seedStr != "" {
		// Use a deterministic key derivation for consistent encryption
		key := make([]byte, 32)
		copy(key, []byte(seedStr))
		return key, nil
	}
	
	return nil, fmt.Errorf("encryption key not configured - set PS_ENCRYPTION_KEY or PS_TENANT_SECRET")
}

// EncryptString encrypts a string using AES-256-GCM
func EncryptString(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	
	key, err := getEncryptionKey()
	if err != nil {
		return "", fmt.Errorf("failed to get encryption key: %w", err)
	}
	
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	
	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	
	// Encrypt the plaintext
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	
	// Return base64 encoded result
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString decrypts a string using AES-256-GCM
func DecryptString(encryptedData string) (string, error) {
	if encryptedData == "" {
		return "", nil
	}
	
	key, err := getEncryptionKey()
	if err != nil {
		return "", fmt.Errorf("failed to get encryption key: %w", err)
	}
	
	// Decode base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}
	
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	
	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	
	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}
	
	return string(plaintext), nil
}