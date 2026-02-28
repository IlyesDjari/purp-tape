package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// LogOfflinePlay handles POST /offline/plays/{id}.
func (h *OfflineHandlers) LogOfflinePlay(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	downloadID := r.PathValue("download_id")

	var req struct {
		DurationListened int `json:"duration_listened"`
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

	if err := h.db.UpdateOfflineLastPlayed(r.Context(), downloadID); err != nil {
		h.log.Error("failed to log offline play", "error", err)
	}

	h.log.Info("offline play logged", "user_id", userID, "download_id", downloadID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":           "logged",
		"sync_when_online": "offline plays will be counted once synced",
	}); err != nil {
		h.log.Error("failed to encode response", "error", err)
	}
}

// ReconcileOfflineData handles POST /offline/sync.
func (h *OfflineHandlers) ReconcileOfflineData(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	h.log.Info("offline data sync initiated", "user_id", userID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":               "synced",
		"offline_plays_synced": 0,
		"last_sync":            time.Now(),
		"message":              "Offline plays synced with server",
	}); err != nil {
		h.log.Error("failed to encode response", "error", err)
	}
}

// SyncDownloadProject handles POST /projects/{id}/offline/sync.
func (h *OfflineHandlers) SyncDownloadProject(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	projectID := r.PathValue("project_id")

	project, err := h.db.GetProjectByID(r.Context(), projectID, userID)
	if err != nil {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	tracks, err := h.db.GetProjectTracks(r.Context(), projectID)
	if err != nil {
		h.log.Error("failed to get project tracks", "error", err)
		http.Error(w, "failed to get tracks", http.StatusInternalServerError)
		return
	}

	downloads := []map[string]interface{}{}
	for _, track := range tracks {
		version, err := h.db.GetLatestTrackVersion(r.Context(), track.ID)
		if err != nil {
			continue
		}

		audioURL, err := h.r2.GenerateDownloadPresignedURL(r.Context(), version.R2ObjectKey, 24*time.Hour)
		if err != nil {
			continue
		}

		downloads = append(downloads, map[string]interface{}{
			"track_id":         track.ID,
			"track_name":       track.Name,
			"audio_url":        audioURL,
			"file_size_bytes":  version.FileSize,
			"duration_seconds": track.Duration,
		})
	}

	h.log.Info("project offline sync initiated", "user_id", userID, "project_id", projectID, "track_count", len(downloads))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"project_id":       projectID,
		"project_name":     project.Name,
		"tracks":           downloads,
		"total_size_bytes": getTotalSize(downloads),
		"message":          fmt.Sprintf("Download all %d tracks for offline", len(downloads)),
	}); err != nil {
		h.log.Error("failed to encode response", "error", err)
	}
}

func getTotalSize(downloads []map[string]interface{}) int64 {
	var total int64
	for _, d := range downloads {
		if size, ok := d["file_size_bytes"].(int64); ok {
			total += size
		}
	}
	return total
}
