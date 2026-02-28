package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/IlyesDjari/purp-tape/backend/internal/models"
)

// InitiateDownload handles POST /tracks/{id}/offline/download.
func (h *OfflineHandlers) InitiateDownload(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	trackID := r.PathValue("track_id")

	version, err := h.db.GetLatestTrackVersion(r.Context(), trackID)
	if err != nil {
		h.log.Warn("track version not found", "track_id", trackID)
		http.Error(w, "track not found", http.StatusNotFound)
		return
	}

	track, err := h.db.GetTrackByID(r.Context(), trackID)
	if err != nil {
		http.Error(w, "track not found", http.StatusNotFound)
		return
	}

	project, err := h.db.GetProjectByID(r.Context(), track.ProjectID, userID)
	if err != nil {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	offlineQuota := int64(1024 * 1024 * 1024)
	if subscription, _ := h.db.GetUserSubscription(r.Context(), userID); subscription != nil {
		tier, _ := subscription["tier"].(string)
		if tier == "pro" {
			offlineQuota = 5 * 1024 * 1024 * 1024
		} else if tier == "pro_plus" || tier == "unlimited" {
			offlineQuota = 20 * 1024 * 1024 * 1024
		}
	}

	usedStorage, err := h.db.GetOfflineStorageUsed(r.Context(), userID)
	if err != nil {
		usedStorage = 0
	}

	existing, err := h.db.GetOfflineDownload(r.Context(), userID, version.ID)
	if err == nil && existing != nil && existing.Status == "completed" {
		h.log.Info("track already in offline mode", "user_id", userID, "track_id", trackID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(existing)
		return
	}

	if usedStorage+version.FileSize > offlineQuota {
		h.log.Warn("offline storage quota exceeded", "user_id", userID, "available", offlineQuota-usedStorage, "needed", version.FileSize)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPaymentRequired)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":           "storage quota exceeded",
			"quota_bytes":     offlineQuota,
			"used_bytes":      usedStorage,
			"available_bytes": offlineQuota - usedStorage,
			"required_bytes":  version.FileSize,
			"action":          "delete_other_downloads_or_upgrade",
		})
		return
	}

	var coverURL string
	if project.CoverImageID != nil {
		if cover, err := h.db.GetImageByID(r.Context(), *project.CoverImageID); err == nil {
			if url, err := h.r2.GenerateDownloadPresignedURL(r.Context(), cover.R2ObjectKey, 24*time.Hour); err == nil {
				coverURL = url
			}
		}
	}

	audioURL, err := h.r2.GenerateDownloadPresignedURL(r.Context(), version.R2ObjectKey, 24*time.Hour)
	if err != nil {
		h.log.Error("failed to generate presigned URL", "error", err)
		http.Error(w, "failed to generate download URL", http.StatusInternalServerError)
		return
	}

	offlineDownload := &models.OfflineDownload{
		ID:                uuid.New().String(),
		UserID:            userID,
		TrackVersionID:    version.ID,
		TrackID:           trackID,
		ProjectID:         track.ProjectID,
		FileSizeBytes:     version.FileSize,
		R2ObjectKey:       version.R2ObjectKey,
		Status:            "pending",
		Title:             track.Name,
		ArtistName:        "",
		ProjectName:       project.Name,
		DurationSeconds:   track.Duration,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := h.db.CreateOfflineDownload(r.Context(), offlineDownload); err != nil {
		h.log.Error("failed to create offline download record", "error", err)
		http.Error(w, "failed to initiate download", http.StatusInternalServerError)
		return
	}

	h.log.Info("offline download initiated", "user_id", userID, "track_id", trackID, "file_size", version.FileSize)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"download_id":        offlineDownload.ID,
		"audio_url":          audioURL,
		"cover_image_url":    coverURL,
		"title":              track.Name,
		"artist":             project.Name,
		"file_size_bytes":    version.FileSize,
		"duration_seconds":   track.Duration,
		"expires_in_seconds": 86400,
		"checksum":           version.Checksum,
	})
}

// ConfirmDownload handles POST /offline/downloads/{id}/confirm.
func (h *OfflineHandlers) ConfirmDownload(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	downloadID := r.PathValue("download_id")

	var req struct {
		LocalFileHash    string `json:"local_file_hash"`
		StorageUsedBytes int64  `json:"storage_used_bytes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	download, err := h.db.GetOfflineDownloadByID(r.Context(), downloadID)
	if err != nil || download.UserID != userID {
		http.Error(w, "download not found", http.StatusNotFound)
		return
	}

	if err := h.db.UpdateOfflineDownloadStatus(r.Context(), downloadID, "completed", map[string]interface{}{
		"downloaded_at":      time.Now(),
		"local_file_hash":    req.LocalFileHash,
		"storage_used_bytes": req.StorageUsedBytes,
	}); err != nil {
		h.log.Error("failed to update download status", "error", err)
		http.Error(w, "failed to confirm download", http.StatusInternalServerError)
		return
	}

	h.log.Info("offline download confirmed", "user_id", userID, "download_id", downloadID)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "completed", "download_id": downloadID}); err != nil {
		h.log.Error("failed to encode response", "error", err)
	}
}

// GetOfflineDownloads handles GET /offline/downloads.
func (h *OfflineHandlers) GetOfflineDownloads(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	downloads, err := h.db.GetUserOfflineDownloads(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to get offline downloads", "error", err)
		http.Error(w, "failed to get downloads", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"downloads": downloads}); err != nil {
		h.log.Error("failed to encode response", "error", err)
	}
}

// DeleteOfflineDownload handles DELETE /offline/downloads/{id}.
func (h *OfflineHandlers) DeleteOfflineDownload(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	downloadID := r.PathValue("download_id")

	download, err := h.db.GetOfflineDownloadByID(r.Context(), downloadID)
	if err != nil || download.UserID != userID {
		http.Error(w, "download not found", http.StatusNotFound)
		return
	}

	if err := h.db.UpdateOfflineDownloadStatus(r.Context(), downloadID, "removed", map[string]interface{}{"updated_at": time.Now()}); err != nil {
		h.log.Error("failed to delete offline download", "error", err)
		http.Error(w, "failed to delete", http.StatusInternalServerError)
		return
	}

	h.log.Info("offline download deleted", "user_id", userID, "download_id", downloadID)
	w.WriteHeader(http.StatusNoContent)
}

// GetOfflineStorageStatus handles GET /offline/storage.
func (h *OfflineHandlers) GetOfflineStorageStatus(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	offlineQuota := int64(1024 * 1024 * 1024)
	if subscription, _ := h.db.GetUserSubscription(r.Context(), userID); subscription != nil {
		tier, _ := subscription["tier"].(string)
		if tier == "pro" {
			offlineQuota = 5 * 1024 * 1024 * 1024
		} else if tier == "pro_plus" || tier == "unlimited" {
			offlineQuota = 20 * 1024 * 1024 * 1024
		}
	}

	usedStorage, err := h.db.GetOfflineStorageUsed(r.Context(), userID)
	if err != nil {
		usedStorage = 0
	}

	downloadCount, err := h.db.GetOfflineDownloadCount(r.Context(), userID)
	if err != nil {
		downloadCount = 0
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"quota_bytes":     offlineQuota,
		"used_bytes":      usedStorage,
		"available_bytes": offlineQuota - usedStorage,
		"usage_percent":   float64(usedStorage) / float64(offlineQuota) * 100,
		"track_count":     downloadCount,
		"tier":            "free",
	})
}
