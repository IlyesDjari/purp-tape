package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/IlyesDjari/purp-tape/backend/internal/audit"
	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/helpers"
	"github.com/IlyesDjari/purp-tape/backend/internal/models"
)

// TrackRollbackHandlers handles version rollback operations
type TrackRollbackHandlers struct {
	db      *db.Database
	log     *slog.Logger
	auditor *audit.AuditLogger
}

// NewTrackRollbackHandlers creates a new rollback handler
func NewTrackRollbackHandlers(database *db.Database, log *slog.Logger) *TrackRollbackHandlers {
	return &TrackRollbackHandlers{
		db:      database,
		log:     log,
		auditor: audit.NewAuditLogger(database, log),
	}
}

// RollbackTrackVersion handles POST /tracks/{id}/versions/{version_number}/rollback
// ✅ FEATURE: Allows users to revert to a previous track version
func (h *TrackRollbackHandlers) RollbackTrackVersion(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}

	trackID := r.PathValue("track_id")
	versionNumberStr := r.PathValue("version_number")

	// Parse version number
	targetVersionNumber, err := strconv.Atoi(versionNumberStr)
	if err != nil || targetVersionNumber <= 0 {
		helpers.WriteBadRequest(w, "invalid version number")
		return
	}

	// ✅ Get target version
	targetVersion, err := h.db.GetTrackVersionByNumber(r.Context(), trackID, targetVersionNumber)
	if err != nil || targetVersion == nil {
		h.log.Error("track version not found", "error", err, "track_id", trackID, "version", targetVersionNumber)
		helpers.WriteNotFound(w, "track version not found")
		return
	}

	// ✅ Verify ownership (user owns the project that contains the track)
	track, err := h.db.GetTrackByID(r.Context(), trackID)
	if err != nil || track == nil {
		helpers.WriteNotFound(w, "track not found")
		return
	}

	project, err := h.db.GetProjectByID(r.Context(), track.ProjectID, userID)
	if err != nil || project == nil || project.UserID != userID {
		helpers.WriteForbidden(w, "you don't have permission to rollback this track")
		return
	}

	// ✅ Get current (latest) version
	currentVersion, err := h.db.GetLatestTrackVersion(r.Context(), trackID)
	if err != nil || currentVersion == nil {
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	// ✅ Check if already at target version
	if currentVersion.VersionNumber == targetVersionNumber {
		helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"message":   "already at this version",
			"version":   currentVersion,
		})
		return
	}

	// ✅ Update track to point to target version
	err = h.db.UpdateTrackActiveVersion(r.Context(), trackID, targetVersion.R2ObjectKey, targetVersion.FileSize, targetVersion.Checksum)
	if err != nil {
		h.log.Error("failed to rollback track version", "error", err)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	// ✅ AUDIT LOG: Log the rollback
	h.auditor.LogEvent(r.Context(), audit.EventTrackVersionDeleted, userID, trackID, map[string]interface{}{
		"action":              "rollback",
		"from_version":        currentVersion.VersionNumber,
		"to_version":          targetVersionNumber,
		"rolled_back_from":    currentVersion.ID,
		"rolled_back_to":      targetVersion.ID,
	})

	h.log.Info("track version rolled back", "track_id", trackID, "user_id", userID, "from_version", currentVersion.VersionNumber, "to_version", targetVersionNumber)

	response := map[string]interface{}{
		"message":      "version rolled back successfully",
		"track_id":     trackID,
		"version":      targetVersionNumber,
		"file_size":    targetVersion.FileSize,
		"checksum":     targetVersion.Checksum,
	}

	helpers.WriteJSON(w, http.StatusOK, response)
}

// GetTrackVersionHistory handles GET /tracks/{id}/versions - lists all versions with rollback capability
func (h *TrackRollbackHandlers) GetTrackVersionHistory(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}

	trackID := r.PathValue("track_id")

	// ✅ Verify access
	track, err := h.db.GetTrackByID(r.Context(), trackID)
	if err != nil || track == nil {
		helpers.WriteNotFound(w, "track not found")
		return
	}

	project, err := h.db.GetProjectByID(r.Context(), track.ProjectID, userID)
	if err != nil || project == nil {
		helpers.WriteForbidden(w, "you don't have access to this track")
		return
	}

	// ✅ Get all versions
	versions, err := h.db.GetTrackVersions(r.Context(), trackID)
	if err != nil {
		h.log.Error("failed to get track versions", "error", err)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	// ✅ Get current version
	currentVersion, _ := h.db.GetLatestTrackVersion(r.Context(), trackID)
	var currentVersionID *string
	if currentVersion != nil {
		currentVersionID = &currentVersion.ID
	}

	// Format response with rollback info
	type VersionInfo struct {
		*models.TrackVersion
		IsCurrent     bool   `json:"is_current"`
		CanRollback   bool   `json:"can_rollback"`
		RollbackURL   string `json:"rollback_url,omitempty"`
	}

	versionInfos := make([]VersionInfo, len(versions))
	for i, v := range versions {
		isCurrent := currentVersionID != nil && *currentVersionID == v.ID
		versionInfos[i] = VersionInfo{
			TrackVersion: &v,
			IsCurrent:    isCurrent,
			CanRollback:  !isCurrent, // Can rollback if not current version
			RollbackURL:  fmt.Sprintf("/tracks/%s/versions/%d/rollback", trackID, v.VersionNumber),
		}
	}

	response := map[string]interface{}{
		"track_id":     trackID,
		"versions":     versionInfos,
		"current":      currentVersionID,
		"total":        len(versions),
	}

	helpers.WriteJSON(w, http.StatusOK, response)
}
