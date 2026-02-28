package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/base64"
	"fmt"
	"time"
)

// ShareToken represents a cryptographically secure share link token
// MEDIUM FIX: Validates share link tokens to prevent enumeration attacks
type ShareToken struct {
	Token     string
	Hash      string
	ExpiresAt time.Time
}

// GenerateShareToken creates a secure random token for share links
// Uses cryptographically random 32-byte token, stored as base64
func GenerateShareToken(expiryDuration time.Duration) (*ShareToken, error) {
	// Generate 32 bytes of random data
	tokenBytes := make([]byte, 32)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random token: %w", err)
	}

	// Encode to base64 for transmission
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Hash token for secure storage
	hash := sha256.Sum256([]byte(token))
	hashHex := fmt.Sprintf("%x", hash)

	return &ShareToken{
		Token:     token,
		Hash:      hashHex,
		ExpiresAt: time.Now().Add(expiryDuration),
	}, nil
}

// VerifyShareToken checks if token matches hash and hasn't expired
func VerifyShareToken(token, storedHash string, expiresAt time.Time) bool {
	// Check expiry first (fail fast)
	if time.Now().After(expiresAt) {
		return false
	}

	// Compare token hash with stored hash
	hash := sha256.Sum256([]byte(token))
	hashHex := fmt.Sprintf("%x", hash)

	// Use constant-time comparison to prevent timing attacks
	return constantTimeCompare(hashHex, storedHash)
}

// constantTimeCompare prevents timing attacks by always taking same time
func constantTimeCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	result := 0
	for i := range a {
		result |= int(a[i]) ^ int(b[i])
	}
	return result == 0
}

// Value implements driver.Valuer for database/sql integration
func (st *ShareToken) Value() (driver.Value, error) {
	return st.Hash, nil
}

// Scan implements sql.Scanner for database/sql integration
func (st *ShareToken) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("cannot scan ShareToken from %T", value)
	}

	st.Hash = str
	return nil
}

// ShareLinkValidator provides methods to validate and manage share links
type ShareLinkValidator struct {
	tokenExpiry time.Duration
}

// NewShareLinkValidator creates a new validator
func NewShareLinkValidator(expiry time.Duration) *ShareLinkValidator {
	return &ShareLinkValidator{
		tokenExpiry: expiry,
	}
}

// CreateShareLink generates a new secure share link
func (slv *ShareLinkValidator) CreateShareLink() (*ShareToken, error) {
	return GenerateShareToken(slv.tokenExpiry)
}

// IsTokenValid checks if token is valid and hasn't expired
func (slv *ShareLinkValidator) IsTokenValid(token, storedHash string, expiresAt time.Time) bool {
	// Additional check: token must not be empty
	if token == "" || storedHash == "" {
		return false
	}

	return VerifyShareToken(token, storedHash, expiresAt)
}

// TokenAlmostExpired returns true if token expires within next day
func (slv *ShareLinkValidator) TokenAlmostExpired(expiresAt time.Time) bool {
	return time.Until(expiresAt) < 24*time.Hour && time.Until(expiresAt) > 0
}
