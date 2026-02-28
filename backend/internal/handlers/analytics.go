package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/helpers"
)

// AnalyticsHandlers handles analytics and statistics
type AnalyticsHandlers struct {
	db  *db.Database
	log *slog.Logger
}

// NewAnalyticsHandlers creates analytics handler
func NewAnalyticsHandlers(database *db.Database, log *slog.Logger) *AnalyticsHandlers {
	return &AnalyticsHandlers{db: database, log: log}
}

// GetProjectAnalytics handles GET /projects/{id}/analytics
func (h *AnalyticsHandlers) GetProjectAnalytics(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	projectID := r.PathValue("project_id")

	// Verify user owns project
	project, err := h.db.GetProjectByID(r.Context(), projectID, userID)
	if err != nil {
		h.log.Warn("project not found", "project_id", projectID)
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	// Get play history stats
	stats, err := h.db.GetProjectStats(r.Context(), projectID)
	if err != nil {
		h.log.Error("failed to get project stats", "error", err)
		http.Error(w, "failed to get analytics", http.StatusInternalServerError)
		return
	}

	// Get plays per day (last 30 days)
	dailyPlays, err := h.db.GetDailyPlayStats(r.Context(), projectID, 30)
	if err != nil {
		h.log.Error("failed to get daily stats", "error", err)
		dailyPlays = []map[string]interface{}{}
	}

	// Get top listeners
	topListeners, err := h.db.GetTopListeners(r.Context(), projectID, 10)
	if err != nil {
		h.log.Error("failed to get top listeners", "error", err)
		topListeners = []map[string]interface{}{}
	}

	response := map[string]interface{}{
		"project_id":      project.ID,
		"total_plays":     stats["total_plays"],
		"unique_listeners": stats["unique_listeners"],
		"average_duration": stats["average_duration"],
		"daily_plays":     dailyPlays,
		"top_listeners":   topListeners,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RecordPlay handles POST /tracks/{id}/play-start - records play event
func (h *AnalyticsHandlers) RecordPlay(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	trackVersionID := r.PathValue("track_id")

	var req struct {
		Device string `json:"device"` // "iOS", "web", etc
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Record play start
	playID, err := h.db.RecordPlayStart(r.Context(), trackVersionID, userID, req.Device)
	if err != nil {
		h.log.Error("failed to record play", "error", err)
		http.Error(w, "failed to record play", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"play_id": playID})
}

// CompletePlay handles POST /plays/{id}/complete - records play completion
func (h *AnalyticsHandlers) CompletePlay(w http.ResponseWriter, r *http.Request) {
	playID := r.PathValue("play_id")

	var req struct {
		DurationListened int `json:"duration_listened"` // seconds
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Record play end
	if err := h.db.RecordPlayEnd(r.Context(), playID, req.DurationListened); err != nil {
		h.log.Error("failed to complete play", "error", err)
		http.Error(w, "failed to record completion", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetTrackStats handles GET /tracks/{id}/stats
func (h *AnalyticsHandlers) GetTrackStats(w http.ResponseWriter, r *http.Request) {
	trackID := r.PathValue("track_id")

	stats, err := h.db.GetTrackStats(r.Context(), trackID)
	if err != nil {
		h.log.Error("failed to get track stats", "error", err)
		http.Error(w, "failed to get stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetUserStats handles GET /user/stats - user's overall stats
func (h *AnalyticsHandlers) GetUserStats(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	stats, err := h.db.GetUserStats(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to get user stats", "error", err)
		http.Error(w, "failed to get stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
