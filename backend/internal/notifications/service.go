package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/models"
)

// NotificationService coordinates multi-channel notification delivery
type NotificationService struct {
	database   *db.Database
	log        *slog.Logger
	push       *PushNotificationService
	preferences *PreferencesService
}

// NewNotificationService creates a notification coordination service
func NewNotificationService(
	database *db.Database,
	push *PushNotificationService,
	preferences *PreferencesService,
	log *slog.Logger,
) *NotificationService {
	return &NotificationService{
		database:    database,
		log:         log,
		push:        push,
		preferences: preferences,
	}
}

// NotificationRequest represents a notification to be sent
type NotificationRequest struct {
	UserID    string
	Type      string // "like", "comment", "follow", "share", "mention"
	ActorID   *string
	TrackID   *string
	ProjectID *string
	CommentID *string
	Content   string
	Data      map[string]string
}

// SendNotification sends a notification through all enabled channels
func (s *NotificationService) SendNotification(ctx context.Context, req *NotificationRequest) error {
	// Create in-app notification (always stored)
	inAppNotif := &models.Notification{
		UserID:    req.UserID,
		ActorUserID: req.ActorID,
		Type:      req.Type,
		TrackID:   req.TrackID,
		ProjectID: req.ProjectID,
		CommentID: req.CommentID,
		Content:   req.Content,
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	// Store in database
	if err := s.database.CreateNotification(ctx, inAppNotif); err != nil {
		s.log.Error("failed to create notification", "error", err, "user_id", req.UserID)
		return fmt.Errorf("failed to create notification: %w", err)
	}

	// Get user preferences
	prefs, err := s.preferences.GetNotificationPreferences(ctx, req.UserID)
	if err != nil {
		s.log.Warn("failed to get notification preferences", "error", err, "user_id", req.UserID)
		// Continue anyway - default to send
		prefs = &NotificationPreferences{
			UserID:                   req.UserID,
			PushEnabled:              true,
			PushLikes:                true,
			PushComments:             true,
			PushFollows:              true,
			PushShares:               true,
		}
	}

	// Send push notification if enabled
	if prefs.PushEnabled && s.shouldSendPush(req.Type, prefs) {
		title, body := s.getNotificationText(ctx, req)
		if err := s.push.BroadcastNotification(ctx, req.UserID, title, body, req.Data); err != nil {
			s.log.Error("failed to send push notification", "error", err, "user_id", req.UserID)
			// Don't fail the entire operation if push fails
		}
	}

	s.log.Info("notification sent",
		"user_id", req.UserID,
		"type", req.Type,
		"has_push", prefs.PushEnabled)

	return nil
}

// shouldSendPush checks if push should be sent based on preferences
func (s *NotificationService) shouldSendPush(notifType string, prefs *NotificationPreferences) bool {
	switch notifType {
	case "like":
		return prefs.PushLikes
	case "comment":
		return prefs.PushComments
	case "follow":
		return prefs.PushFollows
	case "share":
		return prefs.PushShares
	default:
		return true
	}
}

// getNotificationText returns title and body for a notification
func (s *NotificationService) getNotificationText(ctx context.Context, req *NotificationRequest) (string, string) {
	actor := "Someone"
	if req.ActorID != nil {
		user, err := s.database.GetUserByID(ctx, *req.ActorID)
		if err == nil && user != nil {
			actor = user.Username
		}
	}

	switch req.Type {
	case "like":
		return "New Like 🎵", fmt.Sprintf("%s liked your track", actor)
	case "comment":
		return "New Comment 💬", fmt.Sprintf("%s commented on your track", actor)
	case "follow":
		return "New Follower ⭐", fmt.Sprintf("%s started following you", actor)
	case "share":
		return "Project Shared 🔗", fmt.Sprintf("%s shared a project with you", actor)
	case "mention":
		return "You're Mentioned 👤", fmt.Sprintf("%s mentioned you: %s", actor, req.Content)
	default:
		return "PurpTape ✨", req.Content
	}
}

// BulkNotify sends notification to multiple users (for announcements, etc)
func (s *NotificationService) BulkNotify(ctx context.Context, userIDs []string, req *NotificationRequest) error {
	for _, userID := range userIDs {
		// Create copy with specific user
		userReq := *req
		userReq.UserID = userID

		// Send async
		go func() {
			if err := s.SendNotification(context.Background(), &userReq); err != nil {
				s.log.Error("failed to send bulk notification", "error", err, "user_id", userID)
			}
		}()
	}

	return nil
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(ctx context.Context, notificationID, userID string) error {
	if err := s.database.MarkNotificationAsRead(ctx, notificationID, userID); err != nil {
		s.log.Error("failed to mark notification as read", "error", err, "notification_id", notificationID)
		return err
	}

	return nil
}

// MarkAllAsRead marks all notifications as read for a user
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID string) error {
	if err := s.database.MarkAllNotificationsAsRead(ctx, userID); err != nil {
		s.log.Error("failed to mark all notifications as read", "error", err, "user_id", userID)
		return err
	}

	return nil
}

// GetNotifications retrieves paginated notifications for a user
func (s *NotificationService) GetNotifications(ctx context.Context, userID string, limit, offset int) ([]models.Notification, int64, error) {
	return s.database.GetNotificationsPaginated(ctx, userID, limit, offset)
}

// GetUnreadCount gets count of unread notifications
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID string) (int, error) {
	return s.database.GetUnreadNotificationCount(ctx, userID)
}
