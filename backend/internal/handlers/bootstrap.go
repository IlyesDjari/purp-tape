package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/helpers"
)

// BootstrapFounder handles initial founder setup
// POST /bootstrap/founder
// Body: { "setup_token": "PURPTAPE_FOUNDER_SETUP_TOKEN" }
// Only works if no founder exists yet
type BootstrapHandler struct {
	db  *db.Database
	log *slog.Logger
}

type BootstrapRequest struct {
	SetupToken string `json:"setup_token"`
}

type BootstrapResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Email   string `json:"email,omitempty"`
}

// NewBootstrapHandler creates bootstrap handler
func NewBootstrapHandler(database *db.Database, log *slog.Logger) *BootstrapHandler {
	return &BootstrapHandler{db: database, log: log}
}

// SetupFounder allows the current user to be set as founder using a setup token
// This is a one-time setup during application initialization
func (bh *BootstrapHandler) SetupFounder(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}

	// Get the setup token from request
	var req BootstrapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteBadRequest(w, "invalid request body")
		return
	}

	// Verify setup token from environment
	expectedToken := os.Getenv("PURPTAPE_FOUNDER_SETUP_TOKEN")
	if expectedToken == "" || expectedToken != req.SetupToken {
		bh.log.Warn("invalid founder setup token attempt", "user_id", userID)
		helpers.WriteForbidden(w, "invalid setup token")
		return
	}

	ctx := r.Context()

	// Check if founder already exists (prevent second setup)
	user, err := bh.db.GetUserByID(ctx, userID)
	if err != nil {
		bh.log.Error("failed to get user", "error", err)
		helpers.WriteInternalError(w, bh.log, err)
		return
	}

	if user.Role == "founder" || user.Role == "admin" {
		helpers.WriteJSON(w, http.StatusOK, BootstrapResponse{
			Success: true,
			Message: "User already has admin/founder role",
			Email:   user.Email,
		})
		return
	}

	// Promote current user to founder
	if err := bh.db.SetUserRole(ctx, userID, "founder"); err != nil {
		bh.log.Error("failed to set founder role", "error", err, "user_id", userID)
		helpers.WriteInternalError(w, bh.log, err)
		return
	}

	bh.log.Info("founder role assigned", "user_id", userID, "email", user.Email)

	helpers.WriteJSON(w, http.StatusOK, BootstrapResponse{
		Success: true,
		Message: "Founder role successfully assigned",
		Email:   user.Email,
	})
}

// CheckFounderStatus returns whether a founder exists
func (bh *BootstrapHandler) CheckFounderStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Query if any founder exists
	var founderExists bool
	err := bh.db.Pool().QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE role = 'founder' LIMIT 1)`).Scan(&founderExists)

	if err != nil {
		bh.log.Error("failed to check founder status", "error", err)
		founderExists = false
	}

	response := map[string]interface{}{
		"has_founder": founderExists,
		"setup_token_required": !founderExists && os.Getenv("PURPTAPE_FOUNDER_SETUP_TOKEN") != "",
	}

	helpers.WriteJSON(w, http.StatusOK, response)
}
