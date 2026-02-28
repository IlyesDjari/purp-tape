package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
)

// AuditEvent represents an auditable action [MEDIUM: Audit logging for compliance]
type AuditEvent struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Action    string                 `json:"action"` // "create", "update", "delete", etc.
	Resource  string                 `json:"resource"` // "project", "track", "user", etc.
	ResourceID string                `json:"resource_id"`
	Changes   map[string]interface{} `json:"changes"` // Old value -> new value
	IPAddress string                 `json:"ip_address"`
	UserAgent string                 `json:"user_agent"`
	Status    string                 `json:"status"` // "success", "failure"
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// Logger logs audit events
type Logger struct {
	db  *db.Database
	log *slog.Logger
}

// NewLogger creates a new audit logger
func NewLogger(database *db.Database, log *slog.Logger) *Logger {
	return &Logger{db: database, log: log}
}

// LogAction logs an audit event
func (al *Logger) LogAction(ctx context.Context, event AuditEvent) error {
	event.Timestamp = time.Now()

	// Log to slog
	al.log.Info("audit event",
		"user_id", event.UserID,
		"action", event.Action,
		"resource", event.Resource,
		"resource_id", event.ResourceID,
		"status", event.Status,
		"ip_address", event.IPAddress,
	)

	// Store in database for audit trail
	changesJSON, _ := json.Marshal(event.Changes)
	err := al.db.Pool().QueryRow(ctx,
		`INSERT INTO audit_logs (user_id, action, resource, resource_id, changes, ip_address, user_agent, status, error, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id`,
		event.UserID, event.Action, event.Resource, event.ResourceID, changesJSON, event.IPAddress, event.UserAgent, event.Status, event.Error, event.Timestamp,
	).Scan(&event.ID)

	return err
}

// LogProjectCreated logs project creation
func (al *Logger) LogProjectCreated(ctx context.Context, userID, projectID, name string, ipAddress string) error {
	return al.LogAction(ctx, AuditEvent{
		UserID:     userID,
		Action:     "create",
		Resource:   "project",
		ResourceID: projectID,
		Changes: map[string]interface{}{
			"name": name,
		},
		IPAddress: ipAddress,
		Status:    "success",
	})
}

// LogProjectDeleted logs project soft delete
func (al *Logger) LogProjectDeleted(ctx context.Context, userID, projectID string, ipAddress string) error {
	return al.LogAction(ctx, AuditEvent{
		UserID:     userID,
		Action:     "delete",
		Resource:   "project",
		ResourceID: projectID,
		IPAddress:  ipAddress,
		Status:     "success",
	})
}

// LogTrackUploaded logs track upload
func (al *Logger) LogTrackUploaded(ctx context.Context, userID, trackID, trackName string, fileSizeBytes int64, ipAddress string) error {
	return al.LogAction(ctx, AuditEvent{
		UserID:     userID,
		Action:     "create",
		Resource:   "track",
		ResourceID: trackID,
		Changes: map[string]interface{}{
			"name":      trackName,
			"file_size": fileSizeBytes,
		},
		IPAddress: ipAddress,
		Status:    "success",
	})
}

// LogShareCreated logs when a project is shared
func (al *Logger) LogShareCreated(ctx context.Context, userID, projectID, sharedWithUserID string, ipAddress string) error {
	return al.LogAction(ctx, AuditEvent{
		UserID:     userID,
		Action:     "share",
		Resource:   "project",
		ResourceID: projectID,
		Changes: map[string]interface{}{
			"shared_with": sharedWithUserID,
		},
		IPAddress: ipAddress,
		Status:    "success",
	})
}

// LogShareRevoked logs when a share is revoked
func (al *Logger) LogShareRevoked(ctx context.Context, userID, projectID, sharedWithUserID string, ipAddress string) error {
	return al.LogAction(ctx, AuditEvent{
		UserID:     userID,
		Action:     "revoke",
		Resource:   "project_share",
		ResourceID: projectID,
		Changes: map[string]interface{}{
			"shared_with": sharedWithUserID,
		},
		IPAddress: ipAddress,
		Status:    "success",
	})
}

// LogUserDataExport logs GDPR data export requests [MEDIUM: Compliance - GDPR]
func (al *Logger) LogUserDataExport(ctx context.Context, userID, requestedByUserID string, ipAddress string) error {
	return al.LogAction(ctx, AuditEvent{
		UserID:     requestedByUserID,
		Action:     "export",
		Resource:   "user_data",
		ResourceID: userID,
		IPAddress:  ipAddress,
		Status:     "success",
	})
}

// LogUserDataDelete logs GDPR data deletion requests [MEDIUM: Compliance - GDPR]
func (al *Logger) LogUserDataDelete(ctx context.Context, userID, requestedByUserID string, ipAddress string) error {
	return al.LogAction(ctx, AuditEvent{
		UserID:     requestedByUserID,
		Action:     "delete",
		Resource:   "user_account",
		ResourceID: userID,
		IPAddress:  ipAddress,
		Status:     "success",
	})
}

// LogUnauthorizedAccess logs failed access attempts
func (al *Logger) LogUnauthorizedAccess(ctx context.Context, userID, resource, resourceID, reason string, ipAddress string) error {
	return al.LogAction(ctx, AuditEvent{
		UserID:     userID,
		Action:     "access_denied",
		Resource:   resource,
		ResourceID: resourceID,
		IPAddress:  ipAddress,
		Status:     "failure",
		Error:      reason,
	})
}

// GetAuditLog retrieves audit logs for a user
func (al *Logger) GetAuditLog(ctx context.Context, userID string, limit, offset int) ([]AuditEvent, error) {
	rows, err := al.db.Pool().Query(ctx,
		`SELECT id, user_id, action, resource, resource_id, changes, ip_address, user_agent, status, error, created_at
		 FROM audit_logs
		 WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []AuditEvent
	for rows.Next() {
		var event AuditEvent
		var changes []byte

		if err := rows.Scan(&event.ID, &event.UserID, &event.Action, &event.Resource, &event.ResourceID,
			&changes, &event.IPAddress, &event.UserAgent, &event.Status, &event.Error, &event.Timestamp); err != nil {
			return nil, err
		}

		if len(changes) > 0 {
			json.Unmarshal(changes, &event.Changes)
		}

		events = append(events, event)
	}

	return events, rows.Err()
}
