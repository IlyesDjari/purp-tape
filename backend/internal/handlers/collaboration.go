package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/helpers"
	"github.com/IlyesDjari/purp-tape/backend/internal/models"
)

// CollaborationHandlers handles sharing, collaboration, and social features
type CollaborationHandlers struct {
	db  *db.Database
	log *slog.Logger
}

// NewCollaborationHandlers creates collaboration handler
func NewCollaborationHandlers(database *db.Database, log *slog.Logger) *CollaborationHandlers {
	return &CollaborationHandlers{db: database, log: log}
}

// UpdateProjectPrivacy handles PATCH /projects/{id}/privacy
func (h *CollaborationHandlers) UpdateProjectPrivacy(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	projectID := r.PathValue("project_id")

	var req struct {
		IsPrivate bool   `json:"is_private"`
		Genre     string `json:"genre"`
		ReleaseDate *time.Time `json:"release_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Verify ownership
	project, err := h.db.GetProjectByID(r.Context(), projectID, userID)
	if err != nil {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}
	if project.UserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Update privacy settings
	if err := h.db.UpdateProjectPrivacy(r.Context(), projectID, req.IsPrivate, req.Genre, req.ReleaseDate); err != nil {
		h.log.Error("failed to update privacy", "error", err)
		http.Error(w, "failed to update", http.StatusInternalServerError)
		return
	}

	h.log.Info("project privacy updated", "project_id", projectID, "is_private", req.IsPrivate)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"project_id": projectID,
		"is_private": req.IsPrivate,
	})
}

// AddCollaborator handles POST /projects/{id}/collaborators
func (h *CollaborationHandlers) AddCollaborator(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	projectID := r.PathValue("project_id")

	var req struct {
		CollaboratorEmail string `json:"collaborator_email"`
		Role             string `json:"role"` // "editor", "commenter", "viewer"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Verify ownership
	project, err := h.db.GetProjectByID(r.Context(), projectID, userID)
	if err != nil || project.UserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Get collaborator user by email
	collaborator, err := h.db.GetUserByEmail(r.Context(), req.CollaboratorEmail)
	if err != nil {
		h.log.Warn("collaborator not found", "email", req.CollaboratorEmail)
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	// Don't add self
	if collaborator.ID == userID {
		http.Error(w, "cannot add yourself as collaborator", http.StatusBadRequest)
		return
	}

	// Add collaborator
	collab := &models.Collaborator{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		UserID:    collaborator.ID,
		Role:      req.Role,
		InvitedAt: time.Now(),
	}

	if err := h.db.AddCollaborator(r.Context(), collab); err != nil {
		h.log.Error("failed to add collaborator", "error", err)
		http.Error(w, "failed to add collaborator", http.StatusInternalServerError)
		return
	}

	h.log.Info("collaborator added", "project_id", projectID, "user_id", collaborator.ID, "role", req.Role)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(collab)
}

// LikeTrack handles POST /tracks/{id}/like
func (h *CollaborationHandlers) LikeTrack(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	trackID := r.PathValue("track_id")

	like := &models.Like{
		ID:        uuid.New().String(),
		UserID:    userID,
		TrackID:   trackID,
		CreatedAt: time.Now(),
	}

	if err := h.db.CreateLike(r.Context(), like); err != nil {
		h.log.Error("failed to create like", "error", err)
		http.Error(w, "failed to like", http.StatusInternalServerError)
		return
	}

	h.log.Info("track liked", "user_id", userID, "track_id", trackID)

	w.WriteHeader(http.StatusCreated)
}

// UnlikeTrack handles DELETE /tracks/{id}/like
func (h *CollaborationHandlers) UnlikeTrack(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	trackID := r.PathValue("track_id")

	if err := h.db.DeleteLike(r.Context(), userID, trackID); err != nil {
		h.log.Error("failed to delete like", "error", err)
		http.Error(w, "failed to unlike", http.StatusInternalServerError)
		return
	}

	h.log.Info("track unliked", "user_id", userID, "track_id", trackID)

	w.WriteHeader(http.StatusNoContent)
}

// AddComment handles POST /tracks/{id}/comments
func (h *CollaborationHandlers) AddComment(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	trackVersionID := r.PathValue("track_version_id")

	var req struct {
		Content    string `json:"content"`
		TimestampMs *int `json:"timestamp_ms"` // optional: where in track
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "content cannot be empty", http.StatusBadRequest)
		return
	}

	comment := &models.Comment{
		ID:             uuid.New().String(),
		UserID:         userID,
		TrackVersionID: trackVersionID,
		Content:        req.Content,
		TimestampMs:    req.TimestampMs,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.db.CreateComment(r.Context(), comment); err != nil {
		h.log.Error("failed to create comment", "error", err)
		http.Error(w, "failed to add comment", http.StatusInternalServerError)
		return
	}

	h.log.Info("comment added", "user_id", userID, "track_version_id", trackVersionID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comment)
}

// GetComments handles GET /tracks/{id}/comments
func (h *CollaborationHandlers) GetComments(w http.ResponseWriter, r *http.Request) {
	trackVersionID := r.PathValue("track_version_id")

	comments, err := h.db.GetComments(r.Context(), trackVersionID)
	if err != nil {
		h.log.Error("failed to get comments", "error", err)
		http.Error(w, "failed to get comments", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

// FollowUser handles POST /users/{id}/follow
func (h *CollaborationHandlers) FollowUser(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	followingID := r.PathValue("user_id")

	if userID == followingID {
		http.Error(w, "cannot follow yourself", http.StatusBadRequest)
		return
	}

	follow := &models.UserFollow{
		ID:          uuid.New().String(),
		FollowerID:  userID,
		FollowingID: followingID,
		CreatedAt:   time.Now(),
	}

	if err := h.db.CreateFollow(r.Context(), follow); err != nil {
		h.log.Error("failed to create follow", "error", err)
		http.Error(w, "failed to follow", http.StatusInternalServerError)
		return
	}

	h.log.Info("user followed", "follower_id", userID, "following_id", followingID)

	w.WriteHeader(http.StatusCreated)
}

// UnfollowUser handles DELETE /users/{id}/follow
func (h *CollaborationHandlers) UnfollowUser(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	followingID := r.PathValue("user_id")

	if err := h.db.DeleteFollow(r.Context(), userID, followingID); err != nil {
		h.log.Error("failed to delete follow", "error", err)
		http.Error(w, "failed to unfollow", http.StatusInternalServerError)
		return
	}

	h.log.Info("user unfollowed", "follower_id", userID, "following_id", followingID)

	w.WriteHeader(http.StatusNoContent)
}

// GetFollowing handles GET /user/following
func (h *CollaborationHandlers) GetFollowing(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	following, err := h.db.GetFollowing(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to get following", "error", err)
		http.Error(w, "failed to get following", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(following)
}

// GetFollowers handles GET /user/followers
func (h *CollaborationHandlers) GetFollowers(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	followers, err := h.db.GetFollowers(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to get followers", "error", err)
		http.Error(w, "failed to get followers", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(followers)
}

// GetNotifications handles GET /user/notifications
func (h *CollaborationHandlers) GetNotifications(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	notifications, err := h.db.GetNotifications(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to get notifications", "error", err)
		http.Error(w, "failed to get notifications", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

// MarkNotificationAsRead handles PATCH /notifications/{id}/read
func (h *CollaborationHandlers) MarkNotificationAsRead(w http.ResponseWriter, r *http.Request) {
	notificationID := r.PathValue("notification_id")

	if err := h.db.MarkNotificationAsRead(r.Context(), notificationID); err != nil {
		h.log.Error("failed to mark notification", "error", err)
		http.Error(w, "failed to mark as read", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
