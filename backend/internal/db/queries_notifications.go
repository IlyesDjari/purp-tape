package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/IlyesDjari/purp-tape/backend/internal/models"
)

// UpsertDeviceToken inserts or updates a device token
func (db *Database) UpsertDeviceToken(ctx context.Context, userID, token, platform string) error {
	query := `
		INSERT INTO device_tokens (user_id, token, platform, is_active, last_used_at)
		VALUES ($1, $2, $3, TRUE, CURRENT_TIMESTAMP)
		ON CONFLICT (token) DO UPDATE
		SET is_active = TRUE, last_used_at = CURRENT_TIMESTAMP
		WHERE device_tokens.user_id = $1
	`

	if err := db.pool.QueryRow(ctx, query, userID, token, platform).Err(); err != nil {
		return fmt.Errorf("failed to upsert device token: %w", err)
	}

	return nil
}

// GetActiveDeviceTokens retrieves all active device tokens for a user
func (db *Database) GetActiveDeviceTokens(ctx context.Context, userID string) ([]string, error) {
	query := `
		SELECT token FROM device_tokens
		WHERE user_id = $1 AND is_active = TRUE
		ORDER BY last_used_at DESC
	`

	rows, err := db.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device tokens: %w", err)
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, fmt.Errorf("failed to scan device token: %w", err)
		}
		tokens = append(tokens, token)
	}

	return tokens, rows.Err()
}

// DeactivateDeviceToken marks a device token as inactive
func (db *Database) DeactivateDeviceToken(ctx context.Context, token string) error {
	query := `
		UPDATE device_tokens
		SET is_active = FALSE
		WHERE token = $1
	`

	if err := db.pool.QueryRow(ctx, query, token).Err(); err != nil {
		return fmt.Errorf("failed to deactivate device token: %w", err)
	}

	return nil
}

// GetNotificationPreferences retrieves user's notification preferences
func (db *Database) GetNotificationPreferences(ctx context.Context, userID string) (*models.NotificationPreferences, error) {
	query := `
		SELECT 
			id, user_id, push_enabled, push_likes, push_comments, push_follows, push_shares, push_mentions,
			quiet_hours_enabled, quiet_hours_start, quiet_hours_end, bundle_by_type,
			created_at, updated_at
		FROM notification_preferences
		WHERE user_id = $1
	`

	var prefs models.NotificationPreferences
	err := db.pool.QueryRow(ctx, query, userID).Scan(
		&prefs.ID, &prefs.UserID, &prefs.PushEnabled, &prefs.PushLikes, &prefs.PushComments,
		&prefs.PushFollows, &prefs.PushShares, &prefs.PushMentions,
		&prefs.QuietHours, &prefs.QuietHoursStart, &prefs.QuietHoursEnd, &prefs.BundleByType,
		&prefs.CreatedAt, &prefs.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get notification preferences: %w", err)
	}

	return &prefs, nil
}

// UpsertNotificationPreferences inserts or updates notification preferences
func (db *Database) UpsertNotificationPreferences(ctx context.Context, prefs *models.NotificationPreferences) error {
	query := `
		INSERT INTO notification_preferences 
		(user_id, push_enabled, push_likes, push_comments, push_follows, push_shares, push_mentions,
		 quiet_hours_enabled, quiet_hours_start, quiet_hours_end, bundle_by_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id) DO UPDATE
		SET 
			push_enabled = $2, push_likes = $3, push_comments = $4, push_follows = $5, push_shares = $6, push_mentions = $7,
			quiet_hours_enabled = $8, quiet_hours_start = $9, quiet_hours_end = $10, bundle_by_type = $11,
			updated_at = CURRENT_TIMESTAMP
	`

	if err := db.pool.QueryRow(ctx, query,
		prefs.UserID, prefs.PushEnabled, prefs.PushLikes, prefs.PushComments,
		prefs.PushFollows, prefs.PushShares, prefs.PushMentions,
		prefs.QuietHours, prefs.QuietHoursStart, prefs.QuietHoursEnd, prefs.BundleByType,
	).Err(); err != nil {
		return fmt.Errorf("failed to upsert notification preferences: %w", err)
	}

	return nil
}

// CreateNotification inserts a notification
func (db *Database) CreateNotification(ctx context.Context, notification interface{}) error {
	notif := notification.(*Notification)
	query := `
		INSERT INTO notifications 
		(user_id, actor_user_id, type, track_id, project_id, comment_id, content, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	err := db.pool.QueryRow(ctx, query,
		notif.UserID, notif.ActorUserID, notif.Type, notif.TrackID, notif.ProjectID,
		notif.CommentID, notif.Content, false, notif.CreatedAt,
	).Scan(&notif.ID)

	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return nil
}

// MarkNotificationAsRead marks a notification as read
func (db *Database) MarkNotificationAsRead(ctx context.Context, notificationID, userID string) error {
	query := `
		UPDATE notifications
		SET is_read = TRUE
		WHERE id = $1 AND user_id = $2
	`

	if err := db.pool.QueryRow(ctx, query, notificationID, userID).Err(); err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	return nil
}

// MarkAllNotificationsAsRead marks all notifications as read for a user
func (db *Database) MarkAllNotificationsAsRead(ctx context.Context, userID string) error {
	query := `
		UPDATE notifications
		SET is_read = TRUE
		WHERE user_id = $1 AND is_read = FALSE
	`

	if err := db.pool.QueryRow(ctx, query, userID).Err(); err != nil {
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}

	return nil
}

// GetNotificationsPaginated retrieves paginated notifications for a user
func (db *Database) GetNotificationsPaginated(ctx context.Context, userID string, limit, offset int) ([]interface{}, int64, error) {
	query := `
		SELECT id, user_id, actor_user_id, type, track_id, project_id, comment_id, content, is_read, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get notifications: %w", err)
	}
	defer rows.Close()

	notifications := make([]interface{}, 0)
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.ActorUserID, &n.Type, &n.TrackID, &n.ProjectID,
			&n.CommentID, &n.Content, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, n)
	}

	// Get total count
	var total int64
	countQuery := `SELECT COUNT(*) FROM notifications WHERE user_id = $1`
	if err := db.pool.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	return notifications, total, nil
}

// GetUnreadNotificationCount returns count of unread notifications
func (db *Database) GetUnreadNotificationCount(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE`

	var count int
	if err := db.pool.QueryRow(ctx, query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}

	return count, nil
}

// Internal type for scanning
type Notification struct {
	ID          string
	UserID      string
	ActorUserID *string
	Type        string
	TrackID     *string
	ProjectID   *string
	CommentID   *string
	Content     string
	IsRead      bool
	CreatedAt   string
}
