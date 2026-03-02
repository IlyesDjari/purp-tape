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
	"github.com/IlyesDjari/purp-tape/backend/internal/storage"
	"github.com/IlyesDjari/purp-tape/backend/internal/validation"
	"github.com/google/uuid"
)

// ProjectHandlers contains all project-related HTTP handlers
type ProjectHandlers struct {
	db      *db.Database
	r2      *storage.R2Client
	log     *slog.Logger
	auditor *audit.AuditLogger
}

// NewProjectHandlers creates a new project handler
func NewProjectHandlers(database *db.Database, r2Client *storage.R2Client, log *slog.Logger) *ProjectHandlers {
	return &ProjectHandlers{
		db:      database,
		r2:      r2Client,
		log:     log,
		auditor: audit.NewAuditLogger(database, log),
	}
}

type projectListItem struct {
	ID            string     `json:"ID"`
	UserID        string     `json:"UserID"`
	Name          string     `json:"Name"`
	Description   string     `json:"Description"`
	CreatedAt     time.Time  `json:"CreatedAt"`
	UpdatedAt     time.Time  `json:"UpdatedAt"`
	IsPrivate     bool       `json:"IsPrivate"`
	CoverImageID  *string    `json:"CoverImageID"`
	DeletedAt     *time.Time `json:"DeletedAt"`
	CoverImageURL string     `json:"cover_image_url,omitempty"`
}

// ListProjects handles GET /projects - lists all projects for the authenticated user
// ✅ PAGINATION: Supports limit and offset query parameters
func (h *ProjectHandlers) ListProjects(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}
	userEmail, _ := r.Context().Value("user_email").(string)

	if err := h.db.EnsureAuthUser(r.Context(), userID, userEmail); err != nil {
		h.log.Error("failed to ensure auth user before listing projects", "error", err)
		helpers.WriteInternalError(w, h.log, err)
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

	items := make([]projectListItem, 0, len(projects))
	for _, project := range projects {
		item := projectListItem{
			ID:           project.ID,
			UserID:       project.UserID,
			Name:         project.Name,
			Description:  project.Description,
			CreatedAt:    project.CreatedAt,
			UpdatedAt:    project.UpdatedAt,
			IsPrivate:    project.IsPrivate,
			CoverImageID: project.CoverImageID,
			DeletedAt:    project.DeletedAt,
		}

		if h.r2 != nil && project.CoverImageID != nil && *project.CoverImageID != "" {
			if image, imageErr := h.db.GetImageByID(r.Context(), *project.CoverImageID); imageErr == nil && image != nil {
				if signedURL, urlErr := h.r2.GenerateSignedURL(r.Context(), image.R2ObjectKey, 24*time.Hour); urlErr == nil {
					item.CoverImageURL = signedURL
				}
			}
		}

		items = append(items, item)
	}

	// Response with pagination metadata
	response := map[string]interface{}{
		"data": items,
		"pagination": map[string]interface{}{
			"limit":    limit,
			"offset":   offset,
			"total":    total,
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
	userEmail, _ := r.Context().Value("user_email").(string)

	if err := h.db.EnsureAuthUser(r.Context(), userID, userEmail); err != nil {
		h.log.Error("failed to ensure auth user before creating project", "error", err)
		helpers.WriteInternalError(w, h.log, err)
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

	helpers.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"data": project,
	})
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

// DeleteProject handles DELETE /projects/{id} - soft deletes an owned project
func (h *ProjectHandlers) DeleteProject(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}

	projectID := r.PathValue("id")
	deleted, err := h.db.DeleteProject(r.Context(), projectID, userID)
	if err != nil {
		h.log.Error("failed to delete project", "error", err)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	if !deleted {
		helpers.WriteNotFound(w, "project not found")
		return
	}

	h.auditor.LogProjectDeleted(r.Context(), userID, projectID, "")
	w.WriteHeader(http.StatusNoContent)
}
