package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/helpers"
	"github.com/IlyesDjari/purp-tape/backend/internal/notifications"
)

// NotificationHandlers handles notification-related HTTP requests
type NotificationHandlers struct {
	db              *db.Database
	log             *slog.Logger
	notificationSvc *notifications.NotificationService
	pushSvc         *notifications.PushNotificationService
	prefsSvc        *notifications.PreferencesService
}

// NewNotificationHandlers creates notification handler
func NewNotificationHandlers(
	database *db.Database,
	notifSvc *notifications.NotificationService,
	pushSvc *notifications.PushNotificationService,
	prefsSvc *notifications.PreferencesService,
	log *slog.Logger,
) *NotificationHandlers {
	return &NotificationHandlers{
		db:              database,
		log:             log,
		notificationSvc: notifSvc,
		pushSvc:         pushSvc,
		prefsSvc:        prefsSvc,
	}
}

// GetNotifications retrieves paginated notifications for the user
func (h *NotificationHandlers) GetNotifications(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}

	limit := 20
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := helpers.ValidateAndParseInt(limitStr, 1, 100); err == nil {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := helpers.ValidateAndParseInt(offsetStr, 0, 999999); err == nil {
			offset = o
		}
	}

	notifs, total, err := h.notificationSvc.GetNotifications(r.Context(), userID, limit, offset)
	if err != nil {
		h.log.Error("failed to get notifications", "error", err)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	response := map[string]interface{}{
		"data": notifs,
		"pagination": map[string]interface{}{
			"limit":     limit,
			"offset":    offset,
			"total":     total,
			"has_more":  int64(offset+limit) < total,
		},
	}

	helpers.WriteJSON(w, http.StatusOK, response)
}

// RegisterDeviceToken registers a device for push notifications
func (h *NotificationHandlers) RegisterDeviceToken(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}

	var req struct {
		Token    string `json:"token"`
		Platform string `json:"platform"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteBadRequest(w, "invalid request body")
		return
	}

	if req.Token == "" || req.Platform == "" {
		helpers.WriteBadRequest(w, "token and platform are required")
		return
	}

	if err := h.pushSvc.RegisterDeviceToken(r.Context(), userID, req.Token, req.Platform); err != nil {
		h.log.Error("failed to register device token", "error", err)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	helpers.WriteJSON(w, http.StatusCreated, map[string]string{
		"status": "device_token_registered",
	})
}

// UpdatePreferences updates notification preferences
func (h *NotificationHandlers) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}

	var req struct {
		PushEnabled       *bool   `json:"push_enabled"`
		PushLikes         *bool   `json:"push_likes"`
		PushComments      *bool   `json:"push_comments"`
		PushFollows       *bool   `json:"push_follows"`
		PushShares        *bool   `json:"push_shares"`
		PushMentions      *bool   `json:"push_mentions"`
		QuietHoursEnabled *bool   `json:"quiet_hours_enabled"`
		QuietHoursStart   *string `json:"quiet_hours_start"`
		QuietHoursEnd     *string `json:"quiet_hours_end"`
		BundleByType      *bool   `json:"bundle_by_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteBadRequest(w, "invalid request body")
		return
	}

	prefs, err := h.prefsSvc.GetNotificationPreferences(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to get preferences", "error", err)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	// Update with provided values
	if req.PushEnabled != nil {
		prefs.PushEnabled = *req.PushEnabled
	}
	if req.PushLikes != nil {
		prefs.PushLikes = *req.PushLikes
	}
	if req.PushComments != nil {
		prefs.PushComments = *req.PushComments
	}
	if req.PushFollows != nil {
		prefs.PushFollows = *req.PushFollows
	}
	if req.PushShares != nil {
		prefs.PushShares = *req.PushShares
	}
	if req.PushMentions != nil {
		prefs.PushMentions = *req.PushMentions
	}
	if req.QuietHoursEnabled != nil {
		prefs.QuietHours = *req.QuietHoursEnabled
	}
	if req.QuietHoursStart != nil {
		prefs.QuietHoursStart = *req.QuietHoursStart
	}
	if req.QuietHoursEnd != nil {
		prefs.QuietHoursEnd = *req.QuietHoursEnd
	}
	if req.BundleByType != nil {
		prefs.BundleByType = *req.BundleByType
	}

	if err := h.prefsSvc.UpdateNotificationPreferences(r.Context(), userID, prefs); err != nil {
		h.log.Error("failed to update preferences", "error", err)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "preferences updated",
		"data":    prefs,
	})
}

// MarkAsRead marks a notification as read
func (h *NotificationHandlers) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}

	notificationID := r.PathValue("notification_id")
	if notificationID == "" {
		helpers.WriteBadRequest(w, "notification_id is required")
		return
	}

	if err := h.notificationSvc.MarkAsRead(r.Context(), notificationID, userID); err != nil {
		h.log.Error("failed to mark as read", "error", err)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]string{
		"status": "marked_as_read",
	})
}

// MarkAllAsRead marks all notifications as read
func (h *NotificationHandlers) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}

	if err := h.notificationSvc.MarkAllAsRead(r.Context(), userID); err != nil {
		h.log.Error("failed to mark all as read", "error", err)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]string{
		"status": "all_marked_as_read",
	})
}

// GetUnreadCount returns count of unread notifications
func (h *NotificationHandlers) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		helpers.WriteUnauthorized(w)
		return
	}

	count, err := h.notificationSvc.GetUnreadCount(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to get unread count", "error", err)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"unread_count": count,
	})
}
