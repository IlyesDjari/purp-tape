package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/helpers"
	"github.com/IlyesDjari/purp-tape/backend/internal/models"
)

// SearchHandlers handles search functionality
type SearchHandlers struct {
	db  *db.Database
	log *slog.Logger
}

// NewSearchHandlers creates search handler
func NewSearchHandlers(database *db.Database, log *slog.Logger) *SearchHandlers {
	return &SearchHandlers{db: database, log: log}
}

// SearchAll handles GET /search?q=query
func (h *SearchHandlers) SearchAll(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "missing search query", http.StatusBadRequest)
		return
	}

	// Get limit from query (default 20, max 100)
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		limit = parseIntOr(l, 20)
	}
	if limit > 100 {
		limit = 100
	}

	// Search projects and tracks
	projects, _, err := h.db.SearchProjects(r.Context(), query, limit)
	if err != nil {
		h.log.Error("failed to search projects", "error", err)
		projects = []models.Project{}
	}

	tracks, _, err := h.db.SearchTracks(r.Context(), query, limit)
	if err != nil {
		h.log.Error("failed to search tracks", "error", err)
		tracks = []models.Track{}
	}

	users, _, err := h.db.SearchUsers(r.Context(), query, limit)
	if err != nil {
		h.log.Error("failed to search users", "error", err)
		users = []models.User{}
	}

	response := map[string]interface{}{
		"query":    query,
		"projects": projects,
		"tracks":   tracks,
		"users":    users,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SearchProjects handles GET /search/projects?q=query
func (h *SearchHandlers) SearchProjects(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "missing search query", http.StatusBadRequest)
		return
	}

	// Only search user's own projects and shared projects
	projects, err := h.db.SearchUserProjects(r.Context(), userID, query, 20)
	if err != nil {
		h.log.Error("failed to search projects", "error", err)
		http.Error(w, "search failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

// GetTrending handles GET /discover/trending
func (h *SearchHandlers) GetTrending(w http.ResponseWriter, r *http.Request) {
	timeframe := r.URL.Query().Get("timeframe")
	if timeframe == "" {
		timeframe = "week" // default
	}

	trending, err := h.db.GetTrendingProjects(r.Context(), timeframe, 50)
	if err != nil {
		h.log.Error("failed to get trending", "error", err)
		http.Error(w, "failed to get trending", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trending)
}

// GetPublicProjects handles GET /discover/public
func (h *SearchHandlers) GetPublicProjects(w http.ResponseWriter, r *http.Request) {
	genre := r.URL.Query().Get("genre")

	public, err := h.db.GetPublicProjects(r.Context(), genre, 50)
	if err != nil {
		h.log.Error("failed to get public projects", "error", err)
		http.Error(w, "failed to get projects", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(public)
}

// Helper function
func parseIntOr(s string, defaultValue int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return n
}

func scanInt(s string, n *int) (int, error) {
	parsed, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	*n = parsed
	return parsed, nil
}
