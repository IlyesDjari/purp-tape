package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/finops"
	"github.com/IlyesDjari/purp-tape/backend/internal/helpers"
	"github.com/IlyesDjari/purp-tape/backend/internal/models"
	"github.com/IlyesDjari/purp-tape/backend/internal/storage"
	"github.com/IlyesDjari/purp-tape/backend/internal/validation"
	"github.com/google/uuid"
)

// TrackHandlers contains all track-related HTTP handlers
type TrackHandlers struct {
	db  *db.Database
	r2  *storage.R2Client
	log *slog.Logger
}

// NewTrackHandlers creates a new track handler
func NewTrackHandlers(database *db.Database, r2Client *storage.R2Client, log *slog.Logger) *TrackHandlers {
	return &TrackHandlers{db: database, r2: r2Client, log: log}
}

// ListTracks lists all tracks in a project with pagination.
func (h *TrackHandlers) ListTracks(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}
	projectID := r.PathValue("project_id")

	// Extract pagination parameters
	limit, offset := helpers.ExtractPaginationParams(r)

	// Verify access first (fail fast)
	project, err := h.db.GetProjectByID(r.Context(), projectID, userID)
	if err != nil || project == nil {
		helpers.WriteForbidden(w, "access denied")
		return
	}

	// Query with pagination
	tracks, total, err := h.db.GetProjectTracksPaginated(r.Context(), projectID, limit, offset)
	if err != nil {
		h.log.Error("failed to get tracks", "error", err)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	// Return with pagination metadata
	response := map[string]interface{}{
		"data": tracks,
		"pagination": map[string]interface{}{
			"limit":    limit,
			"offset":   offset,
			"total":    total,
			"has_more": int64(offset+limit) < total,
		},
	}

	helpers.WriteJSON(w, http.StatusOK, response)
}

// CreateTrack handles POST /projects/{id}/tracks - creates a new track
func (h *TrackHandlers) CreateTrack(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}
	projectID := r.PathValue("project_id")

	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteBadRequest(w, "invalid request body")
		return
	}

	// ✅ INPUT VALIDATION
	if err := validation.ValidateTrackName(req.Name); err != nil {
		helpers.WriteBadRequest(w, err.Error())
		return
	}

	track := &models.Track{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		UserID:    userID,
		Name:      req.Name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.db.CreateTrack(r.Context(), track); err != nil {
		h.log.Error("failed to create track", "error", err)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	helpers.WriteJSON(w, http.StatusCreated, track)
}

// DeleteTrack handles DELETE /tracks/{track_id} - soft-deletes a track
func (h *TrackHandlers) DeleteTrack(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}

	trackID := r.PathValue("track_id")
	if err := validation.ValidateTrackID(trackID); err != nil {
		helpers.WriteBadRequest(w, "invalid track id")
		return
	}

	track, err := h.db.GetTrackByID(r.Context(), trackID)
	if err != nil || track == nil {
		helpers.WriteNotFound(w, "track not found")
		return
	}

	project, err := h.db.GetProjectByID(r.Context(), track.ProjectID, userID)
	if err != nil || project == nil {
		helpers.WriteForbidden(w, "access denied")
		return
	}

	deleted, err := h.db.DeleteTrack(r.Context(), trackID)
	if err != nil {
		h.log.Error("failed to delete track", "error", err, "track_id", trackID)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	if !deleted {
		helpers.WriteNotFound(w, "track not found")
		return
	}

	h.log.Info("track deleted", "track_id", trackID, "user_id", userID)
	helpers.WriteJSON(w, http.StatusOK, map[string]any{"deleted": true, "track_id": trackID})
}

// ListTrackVersions lists all versions of a track.
func (h *TrackHandlers) ListTrackVersions(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}
	trackID := r.PathValue("track_id")
	if err := validation.ValidateTrackID(trackID); err != nil {
		helpers.WriteBadRequest(w, "invalid track id")
		return
	}

	// Verify user has access
	track, err := h.db.GetTrackByID(r.Context(), trackID)
	if err != nil || track == nil {
		helpers.WriteNotFound(w, "track not found")
		return
	}

	project, err := h.db.GetProjectByID(r.Context(), track.ProjectID, userID)
	if err != nil || project == nil {
		helpers.WriteForbidden(w, "access denied")
		return
	}

	// Get versions with soft-delete filtering applied
	versions, err := h.db.GetTrackVersions(r.Context(), trackID)
	if err != nil {
		h.log.Error("failed to get track versions", "error", err)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	engagement, err := h.db.GetTrackVersionEngagementBatch(r.Context(), trackID)
	if err != nil {
		h.log.Error("failed to get track version engagement", "error", err)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"versions": versions,
		"engagement": map[string]interface{}{
			"track_likes":    engagement.TrackLikes,
			"comment_counts": engagement.CommentCounts,
		},
	})
}

// UploadTrackVersion handles POST /tracks/{track_id}/versions - uploads a new version of a track
func (h *TrackHandlers) UploadTrackVersion(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}
	trackID := r.PathValue("track_id")
	if err := validation.ValidateTrackID(trackID); err != nil {
		helpers.WriteBadRequest(w, "invalid track id")
		return
	}

	// Parse multipart form (100MB max)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		h.log.Warn("failed to parse multipart form or file too large", "error", err)
		http.Error(w, "file too large (max 100MB)", http.StatusRequestEntityTooLarge)
		return
	}

	// Get file from form
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		h.log.Warn("missing file in upload", "error", err)
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate audio file
	if err := storage.ValidateAudioFile(fileHeader.Filename, fileHeader.Size); err != nil {
		h.log.Warn("invalid audio file", "error", err, "filename", fileHeader.Filename)
		http.Error(w, fmt.Sprintf("invalid file: %v", err), http.StatusBadRequest)
		return
	}

	// Check subscription quota before creating version.
	quotaMB := int64(1024)
	usedMB := int64(0)

	subscription, subErr := h.db.GetUserSubscription(r.Context(), userID)
	if subErr != nil {
		h.log.Warn("subscription lookup failed, using default quota",
			"error", subErr,
			"user_id", userID,
			"default_quota_mb", quotaMB)

		if calculatedUsedMB, usedErr := h.db.GetUserStorageUsed(r.Context(), userID); usedErr == nil {
			usedMB = calculatedUsedMB
		} else {
			h.log.Warn("storage usage lookup failed, defaulting usage to 0", "error", usedErr, "user_id", userID)
		}
	} else {
		if value, ok := subscription["storage_quota_mb"].(int64); ok {
			quotaMB = value
		}
		if value, ok := subscription["storage_used_mb"].(int64); ok {
			usedMB = value
		}
	}

	availableStorageMB := quotaMB - usedMB
	fileSizeMB := fileHeader.Size / (1024 * 1024)
	if fileSizeMB > availableStorageMB {
		h.log.Warn("storage quota exceeded",
			"user_id", userID,
			"available_mb", availableStorageMB,
			"requested_mb", fileSizeMB)
		http.Error(w, "insufficient storage quota", http.StatusPaymentRequired)
		return
	}

	decision, guardErr := finops.EvaluateUploadGuard(r.Context(), h.db, fileHeader.Size)
	if guardErr != nil {
		h.log.Warn("failed to evaluate FinOps upload guard", "error", guardErr)
	} else if decision.Block {
		h.log.Warn("blocked track upload by FinOps budget guard",
			"user_id", userID,
			"track_id", trackID,
			"projected_monthly_usd", decision.ProjectedCostUSD,
			"budget_utilization_ratio", decision.UtilizationRatio,
			"reason", decision.Reason)
		http.Error(w, "upload temporarily blocked by budget guard", http.StatusServiceUnavailable)
		return
	}

	// Get latest version number to increment
	latestVersion, err := h.db.GetLatestTrackVersionNumber(r.Context(), trackID)
	if err != nil {
		h.log.Error("failed to get latest version", "error", err, "track_id", trackID)
		http.Error(w, "failed to get version number", http.StatusInternalServerError)
		return
	}

	nextVersion := latestVersion + 1
	versionID := uuid.New().String()

	// Create R2 object key (path in bucket) - enforces user_id prefix
	r2ObjectKey := fmt.Sprintf("tracks/%s/%s/v%d-%s", userID, trackID, nextVersion, uuid.New().String())
	contentType := fileHeader.Header.Get("Content-Type")

	// Step 1: Upload to Cloudflare R2 FIRST (before DB so we can retry)
	uploadResult, err := h.r2.UploadFile(r.Context(), r2ObjectKey, file, contentType)
	if err != nil {
		h.log.Error("R2 upload failed", "error", err, "track_id", trackID, "version", nextVersion)
		http.Error(w, "failed to upload file", http.StatusInternalServerError)
		return
	}

	// Begin transaction for atomic DB operations
	tx, err := h.db.Pool().Begin(r.Context())
	if err != nil {
		h.log.Error("failed to begin transaction", "error", err)
		_ = h.r2.DeleteFile(r.Context(), uploadResult.Key)
		http.Error(w, "failed to finalize upload", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	// Step 3: Create track version record with R2 metadata WITHIN transaction
	version := &models.TrackVersion{
		ID:            versionID,
		TrackID:       trackID,
		VersionNumber: nextVersion,
		R2ObjectKey:   uploadResult.Key,
		FileSize:      uploadResult.FileSize,
		Checksum:      uploadResult.Checksum,
		CreatedAt:     time.Now(),
	}

	if _, err := tx.Exec(r.Context(),
		`INSERT INTO track_versions (id, track_id, version_number, r2_object_key, file_size, checksum, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
		version.ID, version.TrackID, version.VersionNumber, version.R2ObjectKey, version.FileSize, version.Checksum,
	); err != nil {
		h.log.Error("failed to create track version in transaction", "error", err)
		_ = h.r2.DeleteFile(r.Context(), uploadResult.Key)
		http.Error(w, "failed to finalize upload", http.StatusInternalServerError)
		return
	}

	// Step 4: Commit transaction - either both DB ops succeed or both fail
	if err := tx.Commit(r.Context()); err != nil {
		h.log.Error("failed to commit transaction", "error", err)
		_ = h.r2.DeleteFile(r.Context(), uploadResult.Key)
		http.Error(w, "failed to finalize upload", http.StatusInternalServerError)
		return
	}

	h.log.Info("track version uploaded successfully",
		"track_id", trackID,
		"version", nextVersion,
		"file_size", uploadResult.FileSize,
		"checksum", uploadResult.Checksum,
	)

	// Return version info
	version.R2ObjectKey = uploadResult.Key
	version.FileSize = uploadResult.FileSize
	version.Checksum = uploadResult.Checksum

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(version)
}

// GetSignedPlayURL handles GET /tracks/{track_id}/play - generates a signed URL for playback
func (h *TrackHandlers) GetSignedPlayURL(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}
	trackID := r.PathValue("track_id")
	if err := validation.ValidateTrackID(trackID); err != nil {
		helpers.WriteBadRequest(w, "invalid track id")
		return
	}

	// Verify user has access to this track's project before generating signed URL
	track, err := h.db.GetTrackByID(r.Context(), trackID)
	if err != nil || track == nil {
		h.log.Warn("track not found", "track_id", trackID, "error", err)
		helpers.WriteNotFound(w, "track not found")
		return
	}

	// Check user has access to the project that contains this track
	project, err := h.db.GetProjectByID(r.Context(), track.ProjectID, userID)
	if err != nil || project == nil {
		h.log.Warn("unauthorized access attempt to track",
			"user_id", userID,
			"track_id", trackID,
			"project_id", track.ProjectID)
		helpers.WriteForbidden(w, "access denied")
		return
	}

	// Now safe to get and return signed URL (single-row query)
	latestVersion, err := h.db.GetLatestTrackVersion(r.Context(), trackID)
	if err != nil || latestVersion == nil {
		h.log.Error("failed to get latest track version", "error", err, "track_id", trackID)
		helpers.WriteNotFound(w, "no versions available")
		return
	}

	// Generate signed URL (valid for 1 minute - shorter for security)
	signedURL, err := h.r2.GenerateSignedURL(r.Context(), latestVersion.R2ObjectKey, 60*time.Second)
	if err != nil {
		h.log.Error("failed to generate signed URL", "error", err, "track_id", trackID)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	h.log.Info("signed URL generated",
		"user_id", userID,
		"track_id", trackID,
		"version", latestVersion.VersionNumber,
	)

	// Return signed URL response
	response := map[string]interface{}{
		"url":                signedURL,
		"expires_in_seconds": 60, // 1 minute
		"version":            latestVersion.VersionNumber,
		"file_size":          latestVersion.FileSize,
		"checksum":           latestVersion.Checksum,
	}

	helpers.WriteJSON(w, http.StatusOK, response)
}
