package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/helpers"
	"github.com/IlyesDjari/purp-tape/backend/internal/models"
)

// ShareHandlers handles project sharing and link generation
type ShareHandlers struct {
	db  *db.Database
	log *slog.Logger
}

// NewShareHandlers creates share handler
func NewShareHandlers(database *db.Database, log *slog.Logger) *ShareHandlers {
	return &ShareHandlers{db: database, log: log}
}

// GenerateShareLink handles POST /projects/{id}/share-link
func (h *ShareHandlers) GenerateShareLink(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}
	projectID := r.PathValue("project_id")

	var req struct {
		AccessLevel           string `json:"access_level"`             // 'viewer', 'commenter', 'collaborator'
		IsPublic              bool   `json:"is_public"`
		IsPasswordProtected   bool   `json:"is_password_protected"`
		Password              string `json:"password,omitempty"`        // if password protected
		ExpiresInDays         *int   `json:"expires_in_days,omitempty"` // optional expiry
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Verify ownership
	project, err := h.db.GetProjectByID(r.Context(), projectID, userID)
	if err != nil || project.UserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Validate access level
	validLevels := map[string]bool{
		"viewer":       true,
		"commenter":    true,
		"collaborator": true,
	}
	if !validLevels[req.AccessLevel] {
		req.AccessLevel = "viewer"
	}

	// Generate cryptographic hash (short unique ID for share link)
	hash := generateShareHash(projectID, userID)

	// Hash password if provided
	var passwordHash string
	if req.IsPasswordProtected && req.Password != "" {
		hashedPW, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			h.log.Error("failed to hash password", "error", err)
			http.Error(w, "failed to hash password", http.StatusInternalServerError)
			return
		}
		passwordHash = string(hashedPW)
	}

	// Calculate expiry
	var expiresAt *time.Time
	if req.ExpiresInDays != nil && *req.ExpiresInDays > 0 {
		expiry := time.Now().AddDate(0, 0, *req.ExpiresInDays)
		expiresAt = &expiry
	}

	// Create share link
	shareLink := &models.ShareLink{
		ID:                  uuid.New().String(),
		ProjectID:           projectID,
		CreatorID:           userID,
		Hash:                hash,
		AccessLevel:         req.AccessLevel,
		IsPublic:            req.IsPublic,
		IsPasswordProtected: req.IsPasswordProtected,
		PasswordHash:        passwordHash,
		ExpiresAt:           expiresAt,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	if err := h.db.CreateShareLink(r.Context(), shareLink); err != nil {
		h.log.Error("failed to create share link", "error", err)
		http.Error(w, "failed to create share link", http.StatusInternalServerError)
		return
	}

	h.log.Info("share link created", "project_id", projectID, "hash", hash, "access_level", req.AccessLevel)

	response := map[string]interface{}{
		"share_link": map[string]interface{}{
			"hash":     hash,
			"url":      fmt.Sprintf("https://purptape.app/p/%s", hash),
			"access":   req.AccessLevel,
			"expires":  expiresAt,
			"password": req.IsPasswordProtected,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetProjectByShareHash handles GET /share/{hash} - public share link access
func (h *ShareHandlers) GetProjectByShareHash(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("share_hash")

	// Get share link
	shareLink, err := h.db.GetShareLinkByHash(r.Context(), hash)
	if err != nil || shareLink == nil {
		h.log.Warn("share link not found", "hash", hash)
		helpers.WriteNotFound(w, "share link not found")
		return
	}

	// Check if revoked
	if shareLink.RevokedAt != nil {
		helpers.WriteForbidden(w, "share link revoked")
		return
	}

	// Check if expired
	if shareLink.ExpiresAt != nil && time.Now().After(*shareLink.ExpiresAt) {
		helpers.WriteForbidden(w, "share link expired")
		return
	}

	// Check password if protected - require separate POST to verify
	if shareLink.IsPasswordProtected {
		token := extractShareAccessToken(r)
		if token == "" || !verifyShareAccessToken(token, hash, getShareSigningKey()) {
			w.Header().Set("X-Password-Required", "true")
			helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "password required"})
			return
		}
	}

	// Get project (without needing user auth due to share link)
	project, err := h.db.GetProjectByID(r.Context(), shareLink.ProjectID, "")
	if err != nil || project == nil {
		helpers.WriteNotFound(w, "project not found")
		return
	}

	// Add share info to context for RBAC
	r.Header.Set("X-Share-Access", shareLink.AccessLevel)

	h.log.Info("project accessed via share link", "hash", hash, "project_id", shareLink.ProjectID)

	helpers.WriteJSON(w, http.StatusOK, project)
}

// VerifySharePassword handles POST /share/{hash}/verify - password-protected share link
// ✅ SECURE: Password in JSON body, not in URL query string
func (h *ShareHandlers) VerifySharePassword(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("share_hash")

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteBadRequest(w, "invalid request")
		return
	}

	if req.Password == "" {
		helpers.WriteBadRequest(w, "password required")
		return
	}

	// Get share link
	shareLink, err := h.db.GetShareLinkByHash(r.Context(), hash)
	if err != nil || shareLink == nil {
		helpers.WriteNotFound(w, "share link not found")
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(shareLink.PasswordHash), []byte(req.Password)); err != nil {
		h.log.Warn("invalid share password attempt for protected link")
		helpers.WriteForbidden(w, "invalid password")
		return
	}

	accessToken, expiresAt, err := generateShareAccessToken(hash, 15*time.Minute, getShareSigningKey())
	if err != nil {
		h.log.Error("failed to generate share access token", "error", err)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	h.log.Info("share link password verified", "hash", hash)
	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status":       "authenticated",
		"access_token": accessToken,
		"expires_at":   expiresAt,
	})
}

// RevokeShareLink handles DELETE /projects/{id}/share-link/{hash}
func (h *ShareHandlers) RevokeShareLink(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}
	projectID := r.PathValue("project_id")
	hash := r.PathValue("share_hash")

	// Verify ownership
	project, err := h.db.GetProjectByID(r.Context(), projectID, userID)
	if err != nil || project == nil || project.UserID != userID {
		helpers.WriteForbidden(w, "forbidden")
		return
	}

	// Verify this is their share link
	shareLink, err := h.db.GetShareLinkByHash(r.Context(), hash)
	if err != nil || shareLink == nil || shareLink.ProjectID != projectID || shareLink.CreatorID != userID {
		helpers.WriteNotFound(w, "share link not found")
		return
	}

	// Revoke link
	if err := h.db.RevokeShareLink(r.Context(), hash); err != nil {
		h.log.Error("failed to revoke share link", "error", err)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	h.log.Info("share link revoked", "hash", hash)
	helpers.WriteNoContent(w)
}

// RegenerateShareHash handles POST /projects/{id}/share-link/{hash}/regenerate
func (h *ShareHandlers) RegenerateShareHash(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}
	projectID := r.PathValue("project_id")
	hash := r.PathValue("share_hash")

	// Verify ownership
	project, err := h.db.GetProjectByID(r.Context(), projectID, userID)
	if err != nil || project == nil || project.UserID != userID {
		helpers.WriteForbidden(w, "forbidden")
		return
	}

	// Get old share link
	shareLink, err := h.db.GetShareLinkByHash(r.Context(), hash)
	if err != nil || shareLink == nil || shareLink.ProjectID != projectID {
		helpers.WriteNotFound(w, "share link not found")
		return
	}

	// Generate new hash
	newHash := generateShareHash(projectID, userID)

	// Create new share link with same settings
	newShareLink := &models.ShareLink{
		ID:                  uuid.New().String(),
		ProjectID:           projectID,
		CreatorID:           userID,
		Hash:                newHash,
		AccessLevel:         shareLink.AccessLevel,
		IsPublic:            shareLink.IsPublic,
		IsPasswordProtected: shareLink.IsPasswordProtected,
		PasswordHash:        shareLink.PasswordHash,
		ExpiresAt:           shareLink.ExpiresAt,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	if err := h.db.CreateShareLink(r.Context(), newShareLink); err != nil {
		h.log.Error("failed to create new share link", "error", err)
		http.Error(w, "failed to regenerate", http.StatusInternalServerError)
		return
	}

	// Revoke old link
	if err := h.db.RevokeShareLink(r.Context(), hash); err != nil {
		h.log.Warn("failed to revoke old link during regeneration", "error", err)
	}

	h.log.Info("share link regenerated", "old_hash", hash, "new_hash", newHash)

	response := map[string]interface{}{
		"new_hash": newHash,
		"new_url":  fmt.Sprintf("https://purptape.app/p/%s", newHash),
		"old_hash": hash, // mark as revoked in UI
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper: Generate cryptographic hash for share links
// ✅ SECURE: Uses cryptographically-random bytes and fits VARCHAR(32) hash column
func generateShareHash(projectID, userID string) string {
	// Generate random bytes and encode as URL-safe token
	randomBytes := make([]byte, 24)
	if _, err := rand.Read(randomBytes); err != nil {
		return uuid.New().String()[:32]
	}

	// Encode as URL-safe base64 and clamp to schema length
	hash := base64.URLEncoding.EncodeToString(randomBytes)
	if len(hash) > 32 {
		return hash[:32]
	}
	return hash
}

func getShareSigningKey() string {
	if key := strings.TrimSpace(os.Getenv("SHARE_LINK_SIGNING_KEY")); key != "" {
		return key
	}
	if key := strings.TrimSpace(os.Getenv("SUPABASE_SECRET_KEY")); key != "" {
		return key
	}
	return "purptape-share-link-fallback-key"
}

func extractShareAccessToken(r *http.Request) string {
	token := strings.TrimSpace(r.Header.Get("X-Share-Access-Token"))
	if token != "" {
		return token
	}

	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}

	return ""
}

func generateShareAccessToken(shareHash string, ttl time.Duration, signingKey string) (string, time.Time, error) {
	expiresAt := time.Now().UTC().Add(ttl)
	payload := map[string]interface{}{
		"hash": shareHash,
		"exp":  expiresAt.Unix(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to encode share payload: %w", err)
	}

	payloadEnc := base64.RawURLEncoding.EncodeToString(payloadBytes)

	h := hmac.New(sha256.New, []byte(signingKey))
	h.Write([]byte(payloadEnc))
	sig := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	return payloadEnc + "." + sig, expiresAt, nil
}

func verifyShareAccessToken(token, expectedShareHash, signingKey string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}

	payloadEnc := parts[0]
	providedSig := parts[1]

	h := hmac.New(sha256.New, []byte(signingKey))
	h.Write([]byte(payloadEnc))
	expectedSig := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	if !hmac.Equal([]byte(expectedSig), []byte(providedSig)) {
		return false
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadEnc)
	if err != nil {
		return false
	}

	var payload struct {
		Hash string `json:"hash"`
		Exp  int64  `json:"exp"`
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return false
	}

	if payload.Hash != expectedShareHash {
		return false
	}

	return time.Now().UTC().Unix() <= payload.Exp
}

