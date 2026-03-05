package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/finops"
	"github.com/IlyesDjari/purp-tape/backend/internal/helpers"
	"github.com/IlyesDjari/purp-tape/backend/internal/models"
	"github.com/IlyesDjari/purp-tape/backend/internal/storage"
)

// ImageHandlers handles image uploads (covers, artwork)
type ImageHandlers struct {
	db  *db.Database
	r2  *storage.R2Client
	log *slog.Logger
}

// NewImageHandlers creates image handler
func NewImageHandlers(database *db.Database, r2Client *storage.R2Client, log *slog.Logger) *ImageHandlers {
	return &ImageHandlers{db: database, r2: r2Client, log: log}
}

// UploadCover handles POST /projects/{id}/cover - uploads project cover art
func (h *ImageHandlers) UploadCover(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	projectID := r.PathValue("project_id")

	// Parse multipart form (10MB max for images)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		h.log.Warn("failed to parse image form", "error", err)
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	// Get file from form
	file, fileHeader, err := r.FormFile("cover")
	if err != nil {
		h.log.Warn("missing cover file", "error", err)
		http.Error(w, "missing cover file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get alt text
	altText := r.FormValue("alt_text")

	// Validate image file
	if err := storage.ValidateImageFile(fileHeader.Filename, fileHeader.Size); err != nil {
		h.log.Warn("invalid image file", "error", err, "filename", fileHeader.Filename)
		http.Error(w, fmt.Sprintf("invalid file: %v", err), http.StatusBadRequest)
		return
	}

	decision, guardErr := finops.EvaluateUploadGuard(r.Context(), h.db, fileHeader.Size)
	if guardErr != nil {
		h.log.Warn("failed to evaluate FinOps image upload guard", "error", guardErr)
	} else if decision.Block {
		h.log.Warn("blocked cover upload by FinOps budget guard",
			"user_id", userID,
			"project_id", projectID,
			"projected_monthly_usd", decision.ProjectedCostUSD,
			"budget_utilization_ratio", decision.UtilizationRatio,
			"reason", decision.Reason)
		http.Error(w, "cover upload temporarily blocked by budget guard", http.StatusServiceUnavailable)
		return
	}

	// Create R2 object key
	imageID := uuid.New().String()
	r2ObjectKey := fmt.Sprintf("covers/%s/%s/%s", userID, projectID, imageID)
	contentType := fileHeader.Header.Get("Content-Type")

	// Upload to R2
	uploadResult, err := h.r2.UploadFile(r.Context(), r2ObjectKey, file, contentType)
	if err != nil {
		h.log.Error("R2 cover upload failed", "error", err)
		http.Error(w, "failed to upload cover", http.StatusInternalServerError)
		return
	}

	// Create image record
	image := &models.Image{
		ID:          imageID,
		UserID:      userID,
		R2ObjectKey: uploadResult.Key,
		MimeType:    fileHeader.Header.Get("Content-Type"),
		FileSize:    uploadResult.FileSize,
		AltText:     altText,
		CreatedAt:   time.Now(),
	}

	if err := h.db.CreateImage(r.Context(), image); err != nil {
		h.log.Error("failed to create image record", "error", err)
		http.Error(w, "failed to save image", http.StatusInternalServerError)
		return
	}

	// Update project with cover image
	if err := h.db.UpdateProjectCover(r.Context(), projectID, imageID); err != nil {
		h.log.Error("failed to update project cover", "error", err)
		http.Error(w, "failed to set cover", http.StatusInternalServerError)
		return
	}

	h.log.Info("cover uploaded", "project_id", projectID, "image_id", imageID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(image)
}

// GetCoverSignedURL handles GET /images/{id}/url - gets signed URL for image
func (h *ImageHandlers) GetCoverSignedURL(w http.ResponseWriter, r *http.Request) {
	imageID := r.PathValue("image_id")

	image, err := h.db.GetImageByID(r.Context(), imageID)
	if err != nil {
		h.log.Warn("failed to get image", "error", err)
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}

	// Generate signed URL (24 hour expiry for covers)
	signedURL, err := h.r2.GenerateSignedURL(r.Context(), image.R2ObjectKey, 24*time.Hour)
	if err != nil {
		h.log.Error("failed to generate signed URL for image", "error", err)
		http.Error(w, "failed to generate URL", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"url":                signedURL,
		"expires_in_seconds": 86400, // 24 hours
		"mime_type":          image.MimeType,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
