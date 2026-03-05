package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SupabaseJWT represents the decoded JWT from Supabase
type SupabaseJWT struct {
	Sub          string                 `json:"sub"`
	Email        string                 `json:"email"`
	UserMetadata map[string]interface{} `json:"user_metadata"`
	Aud          string                 `json:"aud"`
	Iat          int64                  `json:"iat"`
	Exp          int64                  `json:"exp"`
	jwt.RegisteredClaims
}

// JWKSKey represents a key from Supabase's JWKS endpoint
type JWKSKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	// RSA fields
	N string `json:"n"`
	E string `json:"e"`
	// ECDSA fields (for P-256 curve)
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

type JWKS struct {
	Keys []JWKSKey `json:"keys"`
}

// Validator provides authentication validation with signature verification
type Validator struct {
	supabaseURL    string
	anonKey        string
	secretKey      string
	jwtSecret      string
	publicKeyCache map[string]interface{}
	keysMutex      sync.RWMutex
	keysExpiry     time.Time
}

// NewValidator creates a new auth validator with server-side verification
func NewValidator(supabaseURL, anonKey, secretKey, jwtSecret string) *Validator {
	return &Validator{
		supabaseURL:    supabaseURL,
		anonKey:        anonKey,
		secretKey:      secretKey,
		jwtSecret:      jwtSecret,
		publicKeyCache: make(map[string]interface{}),
	}
}

// ValidateToken validates a Bearer token with signature verification.
func (v *Validator) ValidateToken(authHeader string) (*SupabaseJWT, error) {
	// Extract the token from the Authorization header
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, fmt.Errorf("invalid authorization header format")
	}

	tokenString := parts[1]

	// Parse with signature verification (NOT just decode!)
	token, err := jwt.ParseWithClaims(tokenString, &SupabaseJWT{}, func(token *jwt.Token) (interface{}, error) {
		// Explicitly reject 'none' algorithm (security risk)
		if alg, ok := token.Header["alg"].(string); ok && alg == "none" {
			return nil, fmt.Errorf("token algorithm 'none' is not allowed")
		}

		if _, ok := token.Method.(*jwt.SigningMethodHMAC); ok {
			if v.jwtSecret != "" {
				return []byte(v.jwtSecret), nil
			}
			if v.secretKey != "" {
				return []byte(v.secretKey), nil
			}
			return nil, fmt.Errorf("missing JWT secret for HMAC token validation")
		}

		// Handle RSA (RS256) tokens
		if _, ok := token.Method.(*jwt.SigningMethodRSA); ok {
			kid, ok := token.Header["kid"].(string)
			if !ok || kid == "" {
				return nil, fmt.Errorf("missing key id for RSA token")
			}

			publicKey, err := v.getPublicKey(kid)
			if err != nil {
				return nil, fmt.Errorf("failed to get public key: %w", err)
			}
			return publicKey, nil
		}

		// Handle ECDSA (ES256) tokens - Supabase uses ES256
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); ok {
			kid, ok := token.Header["kid"].(string)
			if !ok || kid == "" {
				return nil, fmt.Errorf("missing key id for ECDSA token")
			}

			publicKey, err := v.getPublicKey(kid)
			if err != nil {
				return nil, fmt.Errorf("failed to get public key: %w", err)
			}
			return publicKey, nil
		}

		return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token signature or claims")
	}

	claims, ok := token.Claims.(*SupabaseJWT)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	// Verify expiration
	if claims.Exp < time.Now().Unix() {
		return nil, fmt.Errorf("token has expired")
	}

	return claims, nil
}

// getPublicKey retrieves the public key from Supabase's JWKS endpoint with caching
func (v *Validator) getPublicKey(keyID string) (interface{}, error) {
	// Check cache first
	v.keysMutex.RLock()
	if key, exists := v.publicKeyCache[keyID]; exists && time.Now().Before(v.keysExpiry) {
		v.keysMutex.RUnlock()
		return key, nil
	}
	v.keysMutex.RUnlock()

	// Fetch from JWKS endpoint (Supabase serves JWKS under /auth/v1/)
	jwksURL := fmt.Sprintf("%s/auth/v1/.well-known/jwks.json", v.supabaseURL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWKS request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read JWKS response: %w", err)
	}

	var jwks JWKS
	if err := json.Unmarshal(body, &jwks); err != nil {
		return nil, fmt.Errorf("failed to parse JWKS: %w", err)
	}

	// Update cache with new keys (valid for 24 hours)
	v.keysMutex.Lock()
	defer v.keysMutex.Unlock()
	v.publicKeyCache = make(map[string]interface{})
	for _, key := range jwks.Keys {
		var pubKey interface{}
		var err error

		// Try to parse as RSA key
		if key.Kty == "RSA" {
			pubKey, err = parseRSAPublicKeyFromJWKS(key)
		} else if key.Kty == "EC" {
			// Try to parse as ECDSA key (Supabase uses EC keys with ES256)
			pubKey, err = parseECDSAPublicKeyFromJWKS(key)
		} else {
			continue
		}

		if err == nil && pubKey != nil {
			v.publicKeyCache[key.Kid] = pubKey
		}
	}
	v.keysExpiry = time.Now().Add(24 * time.Hour)

	// Return the requested key
	if key, exists := v.publicKeyCache[keyID]; exists {
		return key, nil
	}

	return nil, fmt.Errorf("public key not found in JWKS: %s", keyID)
}

func parseRSAPublicKeyFromJWKS(key JWKSKey) (*rsa.PublicKey, error) {
	if key.Kty != "RSA" {
		return nil, fmt.Errorf("unsupported key type: %s", key.Kty)
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, fmt.Errorf("invalid modulus encoding: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, fmt.Errorf("invalid exponent encoding: %w", err)
	}

	e := 0
	for _, b := range eBytes {
		e = (e << 8) | int(b)
	}
	if e == 0 {
		return nil, fmt.Errorf("invalid exponent value")
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: e,
	}, nil
}

// parseECDSAPublicKeyFromJWKS parses an ECDSA public key from JWKS format
// This is used for ES256 tokens from Supabase
func parseECDSAPublicKeyFromJWKS(key JWKSKey) (*ecdsa.PublicKey, error) {
	if key.Kty != "EC" {
		return nil, fmt.Errorf("unsupported key type for ECDSA: %s", key.Kty)
	}

	if key.Crv != "P-256" {
		return nil, fmt.Errorf("unsupported curve: %s, only P-256 is supported", key.Crv)
	}

	// Decode X and Y coordinates
	xBytes, err := base64.RawURLEncoding.DecodeString(key.X)
	if err != nil {
		return nil, fmt.Errorf("invalid X coordinate encoding: %w", err)
	}

	yBytes, err := base64.RawURLEncoding.DecodeString(key.Y)
	if err != nil {
		return nil, fmt.Errorf("invalid Y coordinate encoding: %w", err)
	}

	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}, nil
}

// GetUserIDFromToken extracts the user ID from a valid token
func (v *Validator) GetUserIDFromToken(authHeader string) (string, error) {
	claims, err := v.ValidateToken(authHeader)
	if err != nil {
		return "", err
	}
	return claims.Sub, nil
}

// GetCurrentUser fetches the current user's session from Supabase
// Uses server-side Supabase API to validate token (more secure than client-provided token)
func (v *Validator) GetCurrentUser(userID, authHeader string) (map[string]interface{}, error) {
	// Validate token first
	claims, err := v.ValidateToken(authHeader)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Ensure the token's user ID matches the requested user ID
	if claims.Sub != userID {
		return nil, fmt.Errorf("token user ID does not match requested user ID")
	}

	// Call Supabase Auth API to get current user
	req, err := http.NewRequest("GET", v.supabaseURL+"/auth/v1/user", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", authHeader)
	req.Header.Set("apikey", v.anonKey)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get current user: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var user map[string]interface{}
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, err
	}

	return user, nil
}
