package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/IlyesDjari/purp-tape/backend/internal/cache"
	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/storage"
)

// DownloadHandlers handles file download requests
type DownloadHandlers struct {
	db  *db.Database
	r2  *storage.R2Client
	log *slog.Logger
	urlCache *cache.PresignedURLCache
}

// NewDownloadHandlers creates new download handler
func NewDownloadHandlers(database *db.Database, r2Client *storage.R2Client, log *slog.Logger) *DownloadHandlers {
	return &DownloadHandlers{
		db:       database,
		r2:       r2Client,
		log:      log,
		urlCache: cache.NewPresignedURLCache(12 * time.Minute),
	}
}

// DownloadTrackVersion downloads a track version with memory-efficient streaming.
func (h *DownloadHandlers) DownloadTrackVersion(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	trackID := r.PathValue("track_id")
	versionID := r.PathValue("version_id")

	if trackID == "" || versionID == "" {
		http.Error(w, "missing parameters", http.StatusBadRequest)
		return
	}

	// Get track version (verify user has access)
	version, err := h.db.GetTrackVersionWithAccess(r.Context(), trackID, versionID, userID)
	if err != nil {
		h.log.Warn("track version not found or access denied",
			"track_id", trackID,
			"version_id", versionID,
			"user_id", userID)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if version == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	url, err := h.getOrCreateSignedURL(r, version.R2ObjectKey, 15*time.Minute)
	if err != nil {
		h.log.Error("failed to generate signed download URL",
			"error", err,
			"r2_key", version.R2ObjectKey)
		http.Error(w, "download failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.mp3\"", version.TrackID))
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)

	h.log.Info("file downloaded successfully",
		"track_id", trackID,
		"version_id", versionID,
		"user_id", userID,
		"r2_key", version.R2ObjectKey)
}

// DownloadOfflineFile downloads an offline file with memory-efficient streaming.
func (h *DownloadHandlers) DownloadOfflineFile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	downloadID := r.PathValue("download_id")
	if downloadID == "" {
		http.Error(w, "missing download_id", http.StatusBadRequest)
		return
	}

	// Get offline download (verify ownership)
	offlineDownload, err := h.db.GetOfflineDownloadByUserAndID(r.Context(), userID, downloadID)
	if err != nil {
		h.log.Warn("offline download not found", "download_id", downloadID, "user_id", userID)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if offlineDownload == nil || offlineDownload.Status != "completed" {
		http.Error(w, "download not available", http.StatusNotFound)
		return
	}

	url, err := h.getOrCreateSignedURL(r, offlineDownload.R2ObjectKey, 15*time.Minute)
	if err != nil {
		h.log.Error("failed to generate signed offline URL",
			"error", err,
			"r2_key", offlineDownload.R2ObjectKey)
		http.Error(w, "download failed", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("%s.mp3", offlineDownload.Title)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)

	h.log.Info("offline file downloaded",
		"download_id", downloadID,
		"user_id", userID,
		"r2_key", offlineDownload.R2ObjectKey)
}

func (h *DownloadHandlers) getOrCreateSignedURL(r *http.Request, objectKey string, expiry time.Duration) (string, error) {
	cacheKey := fmt.Sprintf("%s|%s", objectKey, expiry.String())
	if cachedURL, found := h.urlCache.Get(cacheKey); found {
		return cachedURL, nil
	}

	signedURL, err := h.r2.GenerateSignedURL(r.Context(), objectKey, expiry)
	if err != nil {
		return "", err
	}

	h.urlCache.Set(cacheKey, signedURL)
	return signedURL, nil
}
