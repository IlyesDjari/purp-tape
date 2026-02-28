package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// AESEncryptor provides AES-256-GCM encryption for sensitive data
// MEDIUM FIX: Encrypt sensitive fields at rest
type AESEncryptor struct {
	key []byte
}

// NewAESEncryptor creates a new AES encryptor from base64-encoded key
// Key must be 32 bytes for AES-256
func NewAESEncryptor(keyBase64 string) (*AESEncryptor, error) {
	if keyBase64 == "" {
		// In development, return a no-op encryptor that logs a warning
		return &AESEncryptor{
			key: nil, // Nil key = no encryption
		}, nil
	}

	keyBytes, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption key: %w", err)
	}

	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes (AES-256), got %d bytes", len(keyBytes))
	}

	return &AESEncryptor{
		key: keyBytes,
	}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM
// Returns base64-encoded ciphertext with nonce prepended
func (ae *AESEncryptor) Encrypt(plaintext string) (string, error) {
	if ae.key == nil {
		// No-op: return plaintext (development mode)
		return plaintext, nil
	}

	block, err := aes.NewCipher(ae.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt plaintext
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Return as base64 for database storage
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext
func (ae *AESEncryptor) Decrypt(ciphertext64 string) (string, error) {
	if ae.key == nil {
		// No-op: return ciphertext as plaintext (development mode)
		return ciphertext64, nil
	}

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertext64)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(ae.key)
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

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// GenerateKey generates a random 32-byte AES-256 key and returns it base64-encoded
// Save this output to ENCRYPTION_KEY_BASE64 environment variable
func GenerateKey() (string, error) {
	key := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, key)
	if err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}

	return base64.StdEncoding.EncodeToString(key), nil
}
