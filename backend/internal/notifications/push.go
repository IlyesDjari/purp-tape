package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
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

// sendToFCM sends push notification to Firebase Cloud Messaging via HTTP v1 API
func (s *PushNotificationService) sendToFCM(ctx context.Context, token string, payload *NotificationPayload) error {
	if s.fcmServerKey == "" {
		s.log.Warn("FCM server key not configured, skipping push notification")
		return nil
	}

	// Construct FCM HTTP v1 API request body
	fcmRequest := map[string]interface{}{
		"message": map[string]interface{}{
			"token": token,
			"notification": map[string]interface{}{
				"title": payload.Title,
				"body":  payload.Body,
			},
		},
	}

	// Add optional data fields if provided
	if len(payload.Data) > 0 {
		fcmRequest["message"].(map[string]interface{})["data"] = payload.Data
	}

	// Add optional android/apns configuration
	if payload.Sound != "" {
		androidConfig := map[string]interface{}{
			"notification": map[string]interface{}{
				"sound": payload.Sound,
			},
		}
		if payload.Badge > 0 {
			androidConfig["notification"].(map[string]interface{})["notification_count"] = payload.Badge
		}
		fcmRequest["message"].(map[string]interface{})["android"] = androidConfig

		apnsConfig := map[string]interface{}{
			"payload": map[string]interface{}{
				"aps": map[string]interface{}{
					"sound": payload.Sound,
					"badge": payload.Badge,
				},
			},
		}
		fcmRequest["message"].(map[string]interface{})["apns"] = apnsConfig
	}

	requestBody, err := json.Marshal(fcmRequest)
	if err != nil {
		s.log.Error("failed to marshal FCM request", "error", err)
		return fmt.Errorf("failed to marshal FCM request: %w", err)
	}

	// Create HTTP request to FCM API
	// Using service account key for authentication (in production, use OAuth2)
	requestURL := "https://fcm.googleapis.com/v1/projects/purptape-app/messages:send"
	// Note: In production, extract project ID from service account JSON

	execCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(execCtx, http.MethodPost, requestURL, bytes.NewReader(requestBody))
	if err != nil {
		s.log.Error("failed to create FCM request", "error", err)
		return fmt.Errorf("failed to create FCM request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.fcmServerKey)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.log.Error("failed to send FCM request", "error", err)
		return fmt.Errorf("failed to send FCM request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		s.log.Error("FCM returned error", "status", resp.StatusCode, "response", string(body))
		return fmt.Errorf("FCM error: status %d", resp.StatusCode)
	}

	var fcmResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&fcmResponse); err != nil {
		s.log.Error("failed to decode FCM response", "error", err)
		return fmt.Errorf("failed to decode FCM response: %w", err)
	}

	s.log.Info("push notification sent via FCM", "token", token[:20]+"...", "title", payload.Title)
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
