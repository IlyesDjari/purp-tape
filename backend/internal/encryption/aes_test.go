package encryption

import (
	"encoding/base64"
	"strings"
	"testing"
)

func generateTestKey(t *testing.T) string {
	// Generate a valid AES-256 key (32 bytes)
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return base64.StdEncoding.EncodeToString(key)
}

func TestNewAESEncryptor_ValidKey(t *testing.T) {
	keyBase64 := generateTestKey(t)
	encryptor, err := NewAESEncryptor(keyBase64)

	if err != nil {
		t.Errorf("NewAESEncryptor() failed: %v", err)
	}

	if encryptor == nil {
		t.Errorf("NewAESEncryptor() returned nil")
	}

	if len(encryptor.key) != 32 {
		t.Errorf("expected key length 32, got %d", len(encryptor.key))
	}
}

func TestNewAESEncryptor_EmptyKey(t *testing.T) {
	encryptor, err := NewAESEncryptor("")

	if err != nil {
		t.Errorf("NewAESEncryptor with empty key failed: %v", err)
	}

	if encryptor == nil {
		t.Errorf("NewAESEncryptor returned nil")
	}

	if encryptor.key != nil {
		t.Errorf("expected nil key for empty input, got non-nil")
	}
}

func TestNewAESEncryptor_InvalidKey(t *testing.T) {
	tests := []string{
		"invalid-base64!!!",                           // Invalid base64
		base64.StdEncoding.EncodeToString([]byte("short")), // Too short
		base64.StdEncoding.EncodeToString(make([]byte, 64)), // Too long
	}

	for _, keyBase64 := range tests {
		_, err := NewAESEncryptor(keyBase64)
		if err == nil {
			t.Errorf("NewAESEncryptor(%q) expected error, got nil", keyBase64)
		}
	}
}

func TestEncrypt_Decrypt_RoundTrip(t *testing.T) {
	keyBase64 := generateTestKey(t)
	encryptor, _ := NewAESEncryptor(keyBase64)

	plaintext := "This is a secret message that should be encrypted"
	encrypted, err := encryptor.Encrypt(plaintext)

	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	if encrypted == "" {
		t.Errorf("Encrypt() returned empty string")
	}

	if encrypted == plaintext {
		t.Errorf("Encrypt() returned plaintext instead of ciphertext")
	}

	decrypted, err := encryptor.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt() failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypt() returned %q, expected %q", decrypted, plaintext)
	}
}

func TestEncrypt_DifferentCiphertexts(t *testing.T) {
	keyBase64 := generateTestKey(t)
	encryptor, _ := NewAESEncryptor(keyBase64)

	plaintext := "Same plaintext"
	encrypted1, _ := encryptor.Encrypt(plaintext)
	encrypted2, _ := encryptor.Encrypt(plaintext)

	// Same plaintext should produce different ciphertexts due to random nonce
	if encrypted1 == encrypted2 {
		t.Errorf("Two encryptions of same plaintext produced same ciphertext (nonce not random)")
	}
}

func TestEncrypt_NoOpMode(t *testing.T) {
	// Test no-op mode (development) with empty key
	encryptor, _ := NewAESEncryptor("")

	plaintext := "Let's check no-op behavior"
	encrypted, err := encryptor.Encrypt(plaintext)

	if err != nil {
		t.Errorf("Encrypt() in no-op mode failed: %v", err)
	}

	if encrypted != plaintext {
		t.Errorf("Encrypt() in no-op mode should return plaintext, got %q", encrypted)
	}

	decrypted, _ := encryptor.Decrypt(encrypted)
	if decrypted != plaintext {
		t.Errorf("Decrypt() in no-op mode failed")
	}
}

func TestDecrypt_InvalidCiphertext(t *testing.T) {
	keyBase64 := generateTestKey(t)
	encryptor, _ := NewAESEncryptor(keyBase64)

	tests := []string{
		"not-base64!!!",        // Invalid base64
		"dmFsaWRiYXNlNjQ=",     // Valid base64 but too short
		"",                     // Empty
	}

	for _, ciphertext := range tests {
		_, err := encryptor.Decrypt(ciphertext)
		if err == nil {
			t.Errorf("Decrypt(%q) expected error, got nil", ciphertext)
		}
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	keyBase64 := generateTestKey(t)
	encryptor1, _ := NewAESEncryptor(keyBase64)

	plaintext := "Secret"
	encrypted, _ := encryptor1.Encrypt(plaintext)

	// Create encryptor with different key
	wrongKey := make([]byte, 32)
	for i := range wrongKey {
		wrongKey[i] = byte(255 - i)
	}
	wrongKeyBase64 := base64.StdEncoding.EncodeToString(wrongKey)
	encryptor2, _ := NewAESEncryptor(wrongKeyBase64)

	_, err := encryptor2.Decrypt(encrypted)
	if err == nil {
		t.Errorf("Decrypt() with wrong key should fail")
	}
}

func TestEncrypt_LargeData(t *testing.T) {
	keyBase64 := generateTestKey(t)
	encryptor, _ := NewAESEncryptor(keyBase64)

	// Create large plaintext (1MB)
	plaintext := strings.Repeat("x", 1024*1024)
	encrypted, err := encryptor.Encrypt(plaintext)

	if err != nil {
		t.Fatalf("Encrypt() on large data failed: %v", err)
	}

	decrypted, err := encryptor.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt() on large data failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Large data round-trip failed")
	}
}
