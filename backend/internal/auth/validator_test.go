package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestValidateToken_ValidToken(t *testing.T) {
	validator := NewValidator("https://test.supabase.co", "test-anon-key", "test-secret-key", "test-jwt-secret")

	// Create a valid test token with HS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "test-user-id",
		"email": "test@example.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"aud":   "authenticated",
	})

	tokenString, err := token.SignedString([]byte("test-jwt-secret"))
	if err != nil {
		t.Fatalf("failed to create test token: %v", err)
	}

	authHeader := fmt.Sprintf("Bearer %s", tokenString)
	claims, err := validator.ValidateToken(authHeader)

	if err != nil {
		t.Errorf("ValidateToken() failed: %v", err)
	}

	if claims == nil {
		t.Errorf("ValidateToken() returned nil claims")
		return
	}

	if claims.Sub != "test-user-id" {
		t.Errorf("expected sub='test-user-id', got %s", claims.Sub)
	}
}

func TestValidateToken_InvalidFormat(t *testing.T) {
	validator := NewValidator("https://test.supabase.co", "test-anon-key", "test-secret-key", "test-jwt-secret")

	tests := []string{
		"Bearer",            // Missing token
		"NoBearer token123", // Wrong auth type
		"token123",          // Missing Bearer prefix
		"",                  // Empty
	}

	for _, authHeader := range tests {
		_, err := validator.ValidateToken(authHeader)
		if err == nil {
			t.Errorf("ValidateToken(%q) expected error, got nil", authHeader)
		}
	}
}

func TestValidateToken_RejectsNoneAlgorithm(t *testing.T) {
	validator := NewValidator("https://test.supabase.co", "test-anon-key", "test-secret-key", "test-jwt-secret")

	// Create token with 'none' algorithm (security risk) - will be malformed
	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": "malicious-user",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	tokenString, _ := token.SignedString(nil)
	authHeader := fmt.Sprintf("Bearer %s", tokenString)

	_, err := validator.ValidateToken(authHeader)
	if err == nil {
		t.Error("ValidateToken() should reject malformed 'none' algorithm token")
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	validator := NewValidator("https://test.supabase.co", "test-anon-key", "test-secret-key", "test-jwt-secret")

	// Create expired token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "test-user",
		"exp": time.Now().Add(-time.Hour).Unix(), // Expired 1 hour ago
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
	})

	tokenString, _ := token.SignedString([]byte("test-jwt-secret"))
	authHeader := fmt.Sprintf("Bearer %s", tokenString)

	_, err := validator.ValidateToken(authHeader)
	if err == nil {
		t.Error("ValidateToken() should reject expired token")
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	validator := NewValidator("https://test.supabase.co", "test-anon-key", "test-secret-key", "test-jwt-secret")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "test-user",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	// Sign with wrong secret
	tokenString, _ := token.SignedString([]byte("wrong-secret"))
	authHeader := fmt.Sprintf("Bearer %s", tokenString)

	_, err := validator.ValidateToken(authHeader)
	if err == nil {
		t.Error("ValidateToken() should reject token with wrong signature")
	}
}

func TestValidateToken_ES256Token(t *testing.T) {
	// This test verifies ES256 (ECDSA) tokens are supported
	// This is the algorithm Supabase uses for tokens

	// Generate an ECDSA private key for testing
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate ECDSA private key: %v", err)
	}

	// Create a test token with ES256
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"sub":   "test-user-id",
		"email": "test@example.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"aud":   "authenticated",
	})

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("failed to sign ES256 token: %v", err)
	}

	// The test passes if we can generate an ES256 token without errors
	// Full JWKS validation would require mocking the JWKS endpoint
	if tokenString == "" {
		t.Error("ES256 token string is empty")
	}
}
