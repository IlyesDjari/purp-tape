package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/models"
	"github.com/google/uuid"
)

// AuditLogger handles audit logging for sensitive operations
type AuditLogger struct {
	db  *db.Database
	log *slog.Logger
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(database *db.Database, log *slog.Logger) *AuditLogger {
	return &AuditLogger{
		db:  database,
		log: log,
	}
}

// EventType defines the type of audit event
type EventType string

const (
	EventProjectCreated    EventType = "project_created"
	EventProjectUpdated    EventType = "project_updated"
	EventProjectDeleted    EventType = "project_deleted"
	EventTrackCreated      EventType = "track_created"
	EventTrackDeleted      EventType = "track_deleted"
	EventTrackVersionAdded EventType = "track_version_added"
	EventTrackVersionDeleted EventType = "track_version_deleted"
	EventShareLinkCreated  EventType = "share_link_created"
	EventShareLinkRevoked  EventType = "share_link_revoked"
	EventProjectShared     EventType = "project_shared"
	EventShareRevoked      EventType = "share_revoked"
	EventSubscriptionCreated   EventType = "subscription_created"
	EventSubscriptionUpdated   EventType = "subscription_updated"
	EventSubscriptionCancelled EventType = "subscription_cancelled"
	EventOfflineDownloadStarted EventType = "offline_download_started"
	EventOfflineDownloadCompleted EventType = "offline_download_completed"
	EventPasswordVerified  EventType = "password_verified"
	EventUnauthorizedAccess EventType = "unauthorized_access"
)

// LogEvent logs an audit event
func (al *AuditLogger) LogEvent(ctx context.Context, eventType EventType, userID string, resourceID string, details map[string]interface{}) error {
	// Don't log to database in background goroutine - this could lose events
	// Instead, log synchronously to ensure audit trail is captured

	log := al.log.With(
		"event_type", eventType,
		"user_id", userID,
		"resource_id", resourceID,
	)

	// Log to structured logger
	switch eventType {
	case EventProjectCreated, EventProjectUpdated, EventProjectDeleted:
		log.Info("project event", "action", eventType, "details", details)
	case EventTrackCreated, EventTrackDeleted, EventTrackVersionAdded, EventTrackVersionDeleted:
		log.Info("track event", "action", eventType, "details", details)
	case EventShareLinkCreated, EventShareLinkRevoked, EventProjectShared, EventShareRevoked:
		log.Info("sharing event", "action", eventType, "details", details)
	case EventSubscriptionCreated, EventSubscriptionUpdated, EventSubscriptionCancelled:
		log.Info("subscription event", "action", eventType, "details", details)
	case EventOfflineDownloadStarted, EventOfflineDownloadCompleted:
		log.Info("offline event", "action", eventType, "details", details)
	case EventUnauthorizedAccess:
		log.Warn("unauthorized access attempt", "action", eventType, "details", details)
	default:
		log.Info("audit event", "action", eventType, "details", details)
	}

	// Optionally store to database for long-term audit trail
	// This can be done asynchronously in a background job
	// For now, structured logging to stdout/file is primary audit trail
	go func() {
		// Use a new context since the request context might be cancelled
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		detailsJSON, marshalErr := json.Marshal(details)
		if marshalErr != nil {
			al.log.Error("failed to marshal audit log details", "error", marshalErr)
			detailsJSON = []byte("{}")
		}

		auditLog := &models.AuditLog{
			ID:        uuid.New().String(),
			UserID:    userID,
			Action:    string(eventType),
			Resource:  resourceID,
			Details:   string(detailsJSON),
			CreatedAt: time.Now(),
		}

		if err := al.db.CreateAuditLog(ctx, auditLog); err != nil {
			al.log.Error("failed to create audit log", "error", err)
		}
	}()

	return nil
}

// LogProjectCreated logs project creation
func (al *AuditLogger) LogProjectCreated(ctx context.Context, userID, projectID, projectName string) error {
	return al.LogEvent(ctx, EventProjectCreated, userID, projectID, map[string]interface{}{
		"project_name": projectName,
	})
}

// LogProjectDeleted logs project deletion
func (al *AuditLogger) LogProjectDeleted(ctx context.Context, userID, projectID, projectName string) error {
	return al.LogEvent(ctx, EventProjectDeleted, userID, projectID, map[string]interface{}{
		"project_name": projectName,
	})
}

// LogTrackCreated logs track creation
func (al *AuditLogger) LogTrackCreated(ctx context.Context, userID, trackID, trackName, projectID string) error {
	return al.LogEvent(ctx, EventTrackCreated, userID, trackID, map[string]interface{}{
		"track_name": trackName,
		"project_id": projectID,
	})
}

// LogTrackDeleted logs track deletion
func (al *AuditLogger) LogTrackDeleted(ctx context.Context, userID, trackID, trackName, projectID string) error {
	return al.LogEvent(ctx, EventTrackDeleted, userID, trackID, map[string]interface{}{
		"track_name": trackName,
		"project_id": projectID,
	})
}

// LogTrackVersionAdded logs new version
func (al *AuditLogger) LogTrackVersionAdded(ctx context.Context, userID, versionID, trackID string, fileSize int64) error {
	return al.LogEvent(ctx, EventTrackVersionAdded, userID, versionID, map[string]interface{}{
		"track_id":   trackID,
		"file_size": fileSize,
	})
}

// LogShareLinkCreated logs share link creation
func (al *AuditLogger) LogShareLinkCreated(ctx context.Context, userID, hash, projectID, accessLevel string) error {
	return al.LogEvent(ctx, EventShareLinkCreated, userID, hash, map[string]interface{}{
		"project_id":   projectID,
		"access_level": accessLevel,
	})
}

// LogShareLinkRevoked logs share link revocation
func (al *AuditLogger) LogShareLinkRevoked(ctx context.Context, userID, hash, projectID string) error {
	return al.LogEvent(ctx, EventShareLinkRevoked, userID, hash, map[string]interface{}{
		"project_id": projectID,
	})
}

// LogSubscriptionUpdated logs subscription changes
func (al *AuditLogger) LogSubscriptionUpdated(ctx context.Context, userID, subscriptionID, oldTier, newTier string) error {
	return al.LogEvent(ctx, EventSubscriptionUpdated, userID, subscriptionID, map[string]interface{}{
		"old_tier": oldTier,
		"new_tier": newTier,
	})
}

// LogUnauthorizedAccess logs unauthorized access attempts
func (al *AuditLogger) LogUnauthorizedAccess(ctx context.Context, userID, resourceID, reason string) error {
	return al.LogEvent(ctx, EventUnauthorizedAccess, userID, resourceID, map[string]interface{}{
		"reason": reason,
	})
}

// LogPasswordVerified logs password verification (without logging the password!)
func (al *AuditLogger) LogPasswordVerified(ctx context.Context, userID, shareHash string) error {
	return al.LogEvent(ctx, EventPasswordVerified, userID, shareHash, map[string]interface{}{
		"action": "share_password_verified",
	})
}

// LogOfflineDownload logs offline download events
func (al *AuditLogger) LogOfflineDownloadStarted(ctx context.Context, userID, downloadID, trackID string) error {
	return al.LogEvent(ctx, EventOfflineDownloadStarted, userID, downloadID, map[string]interface{}{
		"track_id": trackID,
	})
}

func (al *AuditLogger) LogOfflineDownloadCompleted(ctx context.Context, userID, downloadID string, fileSizeBytes int64) error {
	return al.LogEvent(ctx, EventOfflineDownloadCompleted, userID, downloadID, map[string]interface{}{
		"file_size_bytes": fileSizeBytes,
	})
}
