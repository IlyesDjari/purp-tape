package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/IlyesDjari/purp-tape/backend/internal/audit"
	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/helpers"
	"github.com/IlyesDjari/purp-tape/backend/internal/models"
	"github.com/IlyesDjari/purp-tape/backend/internal/validation"
	"github.com/google/uuid"
)

// ProjectHandlers contains all project-related HTTP handlers
type ProjectHandlers struct {
	db      *db.Database
	log     *slog.Logger
	auditor *audit.AuditLogger
}

// NewProjectHandlers creates a new project handler
func NewProjectHandlers(database *db.Database, log *slog.Logger) *ProjectHandlers {
	return &ProjectHandlers{
		db:      database,
		log:     log,
		auditor: audit.NewAuditLogger(database, log),
	}
}

// ListProjects handles GET /projects - lists all projects for the authenticated user
// ✅ PAGINATION: Supports limit and offset query parameters
func (h *ProjectHandlers) ListProjects(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}

	// Parse pagination parameters
	limit := 20 // Default limit
	offset := 0
	
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Retrieve paginated projects
	projects, total, err := h.db.GetUserProjectsPaginated(r.Context(), userID, limit, offset)
	if err != nil {
		h.log.Error("failed to get projects", "error", err)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	// Response with pagination metadata
	response := map[string]interface{}{
		"data": projects,
		"pagination": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"total":  total,
			"has_more": int64(offset+limit) < total,
		},
	}

	w.Header().Set("X-Total-Count", fmt.Sprintf("%d", total))
	helpers.WriteJSON(w, http.StatusOK, response)
}

// CreateProject handles POST /projects - creates a new project
func (h *ProjectHandlers) CreateProject(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteBadRequest(w, "invalid request body")
		return
	}

	// ✅ INPUT VALIDATION
	req.Name = validation.SanitizeString(req.Name)
	if err := validation.ValidateProjectName(req.Name); err != nil {
		helpers.WriteBadRequest(w, err.Error())
		return
	}

	req.Description = validation.SanitizeString(req.Description)
	if err := validation.ValidateDescription(req.Description); err != nil {
		helpers.WriteBadRequest(w, err.Error())
		return
	}

	project := &models.Project{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.db.CreateProject(r.Context(), project); err != nil {
		h.log.Error("failed to create project", "error", err)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	// ✅ AUDIT LOG: Log project creation
	h.auditor.LogProjectCreated(r.Context(), userID, project.ID, project.Name)

	helpers.WriteJSON(w, http.StatusCreated, project)
}

// GetProject handles GET /projects/{id} - gets a specific project
func (h *ProjectHandlers) GetProject(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}
	projectID := r.PathValue("id")

	project, err := h.db.GetProjectByID(r.Context(), projectID, userID)
	if err != nil || project == nil {
		h.log.Error("failed to get project", "error", err)
		helpers.WriteNotFound(w, "project not found")
		return
	}

	helpers.WriteJSON(w, http.StatusOK, project)
}
