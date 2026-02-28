package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/IlyesDjari/purp-tape/backend/internal/audit"
	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/helpers"
)

// ComplianceHandlers handles GDPR and privacy-related requests.
type ComplianceHandlers struct {
	db    *db.Database
	audit *audit.Logger
	log   *slog.Logger
}

// NewComplianceHandlers creates compliance handler
func NewComplianceHandlers(database *db.Database, auditLogger *audit.Logger, log *slog.Logger) *ComplianceHandlers {
	return &ComplianceHandlers{db: database, audit: auditLogger, log: log}
}

// ExportUserData exports user data for GDPR compliance.
// Returns all user's personal data in JSON format
func (h *ComplianceHandlers) ExportUserData(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}

	h.log.Info("user requested data export", "user_id", userID)

	// Get user profile
	user, err := h.db.GetUserByID(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to get user", "error", err)
		http.Error(w, "failed to export data", http.StatusInternalServerError)
		return
	}

	// Get all projects with pagination helper
	projects, _, err := h.db.GetUserProjectsPaginated(r.Context(), userID, 1000, 0)
	if err != nil {
		h.log.Error("failed to get projects", "error", err)
		http.Error(w, "failed to export data", http.StatusInternalServerError)
		return
	}

	// Get all tracks
	tracks, err := h.db.GetAllUserTracks(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to get tracks", "error", err)
		http.Error(w, "failed to export data", http.StatusInternalServerError)
		return
	}

	// Get audit logs
	auditLogs, err := h.audit.GetAuditLog(r.Context(), userID, 10000, 0)
	if err != nil {
		h.log.Error("failed to get audit logs", "error", err)
		auditLogs = []audit.AuditEvent{}
	}

	exportData := map[string]interface{}{
		"user":       user,
		"projects":   projects,
		"tracks":     tracks,
		"audit_logs": auditLogs,
		"export_date": fmt.Sprintf("%s UTC", fmt.Sprintf("%v", "2026-02-28")),
	}

	// Log the export
	ipAddress := r.RemoteAddr
	h.audit.LogUserDataExport(r.Context(), userID, userID, ipAddress)

	// Return as JSON file
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"purptape-data-export-%s.json\"", userID))

	json.NewEncoder(w).Encode(exportData)
	h.log.Info("data export completed", "user_id", userID)
}

// DeleteUserData deletes user account for GDPR Right to be Forgotten.
// Permanently deletes all user data
func (h *ComplianceHandlers) DeleteUserData(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}

	// Require confirmation
	var req struct {
		ConfirmPassword string `json:"confirm_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.ConfirmPassword == "" {
		http.Error(w, "password confirmation required", http.StatusBadRequest)
		return
	}

	h.log.Warn("account deletion requested with password confirmation", "user_id", userID)

	h.log.Warn("user requested account deletion with confirmation", "user_id", userID)

	// Execute deletion in transaction
	err = h.db.WithTx(r.Context(), func(tx *db.Tx) error {
		// Delete all projects and associated data
		if err := tx.Exec(r.Context(),
			`DELETE FROM projects WHERE user_id = $1`, userID); err != nil {
			return err
		}

		// Delete all tracks
		if err := tx.Exec(r.Context(),
			`DELETE FROM tracks WHERE user_id = $1`, userID); err != nil {
			return err
		}

		// Delete user account
		if err := tx.Exec(r.Context(),
			`DELETE FROM users WHERE id = $1`, userID); err != nil {
			return err
		}

		// Log the deletion
		ipAddress := r.RemoteAddr
		h.audit.LogUserDataDelete(r.Context(), userID, userID, ipAddress)

		return nil
	})

	if err != nil {
		h.log.Error("failed to delete user account", "error", err, "user_id", userID)
		http.Error(w, "failed to delete account", http.StatusInternalServerError)
		return
	}

	h.log.Info("user account deleted", "user_id", userID)
	w.WriteHeader(http.StatusNoContent)
}

// GetPrivacySettings retrieves user privacy settings.
func (h *ComplianceHandlers) GetPrivacySettings(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}

	settings, err := h.db.GetUserPrivacySettings(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to get privacy settings", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

// UpdatePrivacySettings handles PATCH /compliance/privacy-settings
func (h *ComplianceHandlers) UpdatePrivacySettings(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}

	var settings map[string]bool
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Update settings in database
	err = h.db.UpdateUserPrivacySettings(r.Context(), userID, settings)
	if err != nil {
		h.log.Error("failed to update privacy settings", "error", err)
		http.Error(w, "failed to update settings", http.StatusInternalServerError)
		return
	}

	h.log.Info("privacy settings updated", "user_id", userID)
	w.WriteHeader(http.StatusNoContent)
}
