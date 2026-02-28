package handlers

import (
	"encoding/json"
	"net/http"
)

// DeleteAllExpiredDownloads handles DELETE /offline/downloads/expired.
func (h *OfflineHandlers) DeleteAllExpiredDownloads(w http.ResponseWriter, r *http.Request) {
	deleted, err := h.db.CleanupExpiredOfflineDownloads(r.Context())
	if err != nil {
		h.log.Error("failed to cleanup expired offline downloads", "error", err)
		http.Error(w, "failed to cleanup expired downloads", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "ok",
		"deleted_count": deleted,
	})
}

// GetOfflineStorageInfo handles GET /offline/storage/info.
func (h *OfflineHandlers) GetOfflineStorageInfo(w http.ResponseWriter, r *http.Request) {
	h.GetOfflineStorageStatus(w, r)
}
