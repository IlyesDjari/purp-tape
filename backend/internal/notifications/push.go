package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/models"
)

// PushNotificationService handles FCM push notifications
type PushNotificationService struct {
	db              *db.Database
	log             *slog.Logger
	fcmServerKey    string
	mu              sync.RWMutex
}

// NewPushNotificationService creates push notification service
func NewPushNotificationService(database *db.Database, fcmServerKey string, log *slog.Logger) *PushNotificationService {
	return &PushNotificationService{
		db:           database,
		log:          log,
		fcmServerKey: fcmServerKey,
	}
}

// DeviceToken represents a user's device push token
type DeviceToken struct {
	ID        string
	UserID    string
	Token     string
	Platform  string // "ios", "android", "web"
	IsActive  bool
	CreatedAt string
	LastUsedAt string
}

// NotificationPayload represents push notification payload
type NotificationPayload struct {
	Title       string            `json:"title"`
	Body        string            `json:"body"`
	Data        map[string]string `json:"data"`
	Badge       int               `json:"badge,omitempty"`
	Sound       string            `json:"sound,omitempty"`
	ClickAction string            `json:"click_action,omitempty"`
}

// RegisterDeviceToken stores a FCM device token
func (s *PushNotificationService) RegisterDeviceToken(ctx context.Context, userID, token, platform string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.db.UpsertDeviceToken(ctx, userID, token, platform); err != nil {
		s.log.Error("failed to register device token", "error", err, "user_id", userID)
		return fmt.Errorf("failed to register device token: %w", err)
	}

	s.log.Info("device token registered", "user_id", userID, "platform", platform)
	return nil
}

// SendPushNotification sends a push notification via FCM
func (s *PushNotificationService) SendPushNotification(ctx context.Context, userID string, payload *NotificationPayload) error {
	// Get active device tokens for user
	tokens, err := s.db.GetActiveDeviceTokens(ctx, userID)
	if err != nil {
		s.log.Error("failed to get device tokens", "error", err, "user_id", userID)
		return fmt.Errorf("failed to get device tokens: %w", err)
	}

	if len(tokens) == 0 {
		s.log.Debug("no active device tokens for user", "user_id", userID)
		return nil // User has no registered devices
	}

	// Send to all active devices (FCM handles duplicate suppression)
	for _, token := range tokens {
		go func(deviceToken string) {
			if err := s.sendToFCM(context.Background(), deviceToken, payload); err != nil {
				s.log.Error("failed to send FCM notification", "error", err, "user_id", userID)
				// Mark token as inactive if it fails
				_ = s.db.DeactivateDeviceToken(ctx, deviceToken)
			}
		}(token)
	}

	return nil
}

// sendToFCM sends push notification to Firebase Cloud Messaging
func (s *PushNotificationService) sendToFCM(ctx context.Context, token string, payload *NotificationPayload) error {
	// FCM HTTP v1 API call would go here
	// This is a placeholder for the actual FCM implementation
	
	if s.fcmServerKey == "" {
		s.log.Warn("FCM server key not configured, skipping push notification")
		return nil
	}

	// TODO: Implement actual FCM HTTP v1 API call
	// https://firebase.google.com/docs/cloud-messaging/send-message
	
	s.log.Debug("push notification sent via FCM", "token", token[:20]+"...", "title", payload.Title)
	return nil
}

// BroadcastNotification broadcasts to all user's devices
func (s *PushNotificationService) BroadcastNotification(ctx context.Context, userID string, title, body string, data map[string]string) error {
	payload := &NotificationPayload{
		Title: title,
		Body:  body,
		Data:  data,
		Sound: "default",
	}

	return s.SendPushNotification(ctx, userID, payload)
}

// InvalidateToken marks a device token as inactive
func (s *PushNotificationService) InvalidateToken(ctx context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.db.DeactivateDeviceToken(ctx, token); err != nil {
		s.log.Error("failed to deactivate device token", "error", err)
		return err
	}

	return nil
}
