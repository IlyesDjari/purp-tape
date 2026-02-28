package notifications

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/models"
)

// PreferencesService manages notification preferences
type PreferencesService struct {
	database *db.Database
	log      *slog.Logger
}

// NewPreferencesService creates preferences service
func NewPreferencesService(database *db.Database, log *slog.Logger) *PreferencesService {
	return &PreferencesService{
		database: database,
		log:      log,
	}
}

// GetNotificationPreferences retrieves user's notification preferences
func (ps *PreferencesService) GetNotificationPreferences(ctx context.Context, userID string) (*models.NotificationPreferences, error) {
	prefs, err := ps.database.GetNotificationPreferences(ctx, userID)
	if err != nil {
		ps.log.Error("failed to get notification preferences", "error", err, "user_id", userID)
		return nil, fmt.Errorf("failed to get notification preferences: %w", err)
	}

	if prefs == nil {
		// Return default preferences if not found
		return &models.NotificationPreferences{
			UserID:          userID,
			PushEnabled:     true,
			PushLikes:       true,
			PushComments:    true,
			PushFollows:     true,
			PushShares:      true,
			PushMentions:    true,
			QuietHours:      false,
			QuietHoursStart: "22:00",
			QuietHoursEnd:   "09:00",
			BundleByType:    true,
		}, nil
	}

	return prefs, nil
}

// UpdateNotificationPreferences updates user's notification preferences
func (ps *PreferencesService) UpdateNotificationPreferences(ctx context.Context, userID string, prefs *models.NotificationPreferences) error {
	prefs.UserID = userID

	if err := ps.database.UpsertNotificationPreferences(ctx, prefs); err != nil {
		ps.log.Error("failed to update notification preferences", "error", err, "user_id", userID)
		return fmt.Errorf("failed to update notification preferences: %w", err)
	}

	ps.log.Info("notification preferences updated", "user_id", userID)
	return nil
}

// DisableAllPush disables all push notifications
func (ps *PreferencesService) DisableAllPush(ctx context.Context, userID string) error {
	prefs, err := ps.GetNotificationPreferences(ctx, userID)
	if err != nil {
		return err
	}

	prefs.PushEnabled = false
	prefs.PushLikes = false
	prefs.PushComments = false
	prefs.PushFollows = false
	prefs.PushShares = false
	prefs.PushMentions = false

	return ps.UpdateNotificationPreferences(ctx, userID, prefs)
}

// SetQuietHours sets quiet hours for push notifications
func (ps *PreferencesService) SetQuietHours(ctx context.Context, userID, startTime, endTime string) error {
	prefs, err := ps.GetNotificationPreferences(ctx, userID)
	if err != nil {
		return err
	}

	prefs.QuietHours = true
	prefs.QuietHoursStart = startTime
	prefs.QuietHoursEnd = endTime

	return ps.UpdateNotificationPreferences(ctx, userID, prefs)
}

// DisableQuietHours disables quiet hours
func (ps *PreferencesService) DisableQuietHours(ctx context.Context, userID string) error {
	prefs, err := ps.GetNotificationPreferences(ctx, userID)
	if err != nil {
		return err
	}

	prefs.QuietHours = false

	return ps.UpdateNotificationPreferences(ctx, userID, prefs)
}
