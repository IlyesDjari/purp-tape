package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/IlyesDjari/purp-tape/backend/internal/models"
)

// User Queries

// GetUserByID retrieves a user by ID.
func (db *Database) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	var user models.User
	err := db.pool.QueryRow(ctx,
		`SELECT id, email, username, avatar_url, created_at, updated_at, deleted_at FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Email, &user.Username, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// CreateUser creates a new user
func (db *Database) CreateUser(ctx context.Context, user *models.User) error {
	err := db.pool.QueryRow(ctx,
		`INSERT INTO users (id, email, username, avatar_url, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, email, username, avatar_url, created_at, updated_at`,
		user.ID, user.Email, user.Username, user.AvatarURL, user.CreatedAt, user.UpdatedAt,
	).Scan(&user.ID, &user.Email, &user.Username, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// EnsureAuthUser upserts a user row for authenticated requests.
// This prevents FK failures when domain actions reference users.id before any explicit signup sync.
func (db *Database) EnsureAuthUser(ctx context.Context, userID, email string) error {
	if strings.TrimSpace(userID) == "" {
		return errors.New("user id is required")
	}

	normalizedEmail := strings.TrimSpace(strings.ToLower(email))
	if normalizedEmail == "" {
		normalizedEmail = fmt.Sprintf("%s@auth.local", userID)
	}

	username := normalizedEmail
	if at := strings.Index(username, "@"); at > 0 {
		username = username[:at]
	}
	username = strings.TrimSpace(username)
	if username == "" {
		username = "user"
	}
	if len(username) > 32 {
		username = username[:32]
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO users (id, email, username, created_at, updated_at)
		 VALUES ($1, $2, $3, NOW(), NOW())
		 ON CONFLICT (id) DO UPDATE
		 SET email = EXCLUDED.email,
		     updated_at = NOW()`,
		userID, normalizedEmail, username,
	)
	if err != nil {
		return fmt.Errorf("failed to ensure auth user: %w", err)
	}

	return nil
}

// Project Queries

// GetProjectByID retrieves a project by ID with ownership and access control checks.
func (db *Database) GetProjectByID(ctx context.Context, projectID, userID string) (*models.Project, error) {
	var project models.Project
	err := db.pool.QueryRow(ctx,
		`SELECT id, user_id, name, description, created_at, updated_at FROM projects
		 WHERE id = $1 AND deleted_at IS NULL AND (user_id = $2 OR id IN (
			SELECT project_id FROM project_shares WHERE shared_with_id = $2 AND revoked_at IS NULL
		 ))`,
		projectID, userID,
	).Scan(&project.ID, &project.UserID, &project.Name, &project.Description, &project.CreatedAt, &project.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	return &project, nil
}

// GetUserProjects retrieves all projects for a user (owned and shared).
func (db *Database) GetUserProjects(ctx context.Context, userID string) ([]models.Project, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT DISTINCT p.id, p.user_id, p.name, p.description, p.created_at, p.updated_at
		 FROM projects p
		 LEFT JOIN project_shares ps ON p.id = ps.project_id
		 WHERE p.deleted_at IS NULL AND (p.user_id = $1 OR ps.shared_with_id = $1)
		 ORDER BY p.updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, p)
	}
	return projects, nil
}

// GetUserProjectsPaginated retrieves user projects with pagination using denormalized access cache
// Performance: O(1) access check instead of O(n*m) subqueries
func (db *Database) GetUserProjectsPaginated(ctx context.Context, userID string, limit, offset int) ([]models.Project, int64, error) {
	if strings.TrimSpace(userID) == "" {
		return []models.Project{}, 0, nil
	}

	// Get total count using denormalized access table (1000x faster than nested JOINs)
	var total int64
	err := db.pool.QueryRow(ctx,
		`SELECT COUNT(DISTINCT upa.project_id) 
		 FROM user_project_access upa
		 JOIN projects p ON p.id = upa.project_id
		 WHERE upa.user_id = $1 AND p.deleted_at IS NULL`,
		userID,
	).Scan(&total)
	if err != nil {
		if isUndefinedTable(err, "user_project_access") {
			return db.getUserOwnedProjectsPaginated(ctx, userID, limit, offset)
		}
		return nil, 0, fmt.Errorf("failed to count projects: %w", err)
	}

	// Cache can be temporarily stale or not yet populated for freshly created projects.
	// Fallback to owner-based listing when cache reports zero rows.
	if total == 0 {
		return db.getUserOwnedProjectsPaginated(ctx, userID, limit, offset)
	}

	// Get paginated results using denormalized access
	rows, err := db.pool.Query(ctx,
		`SELECT DISTINCT p.id, p.user_id, p.name, p.description, p.created_at, p.updated_at, p.is_private, p.cover_image_id, p.deleted_at
		 FROM projects p
		 INNER JOIN user_project_access upa ON p.id = upa.project_id
		 WHERE upa.user_id = $1 AND p.deleted_at IS NULL
		 ORDER BY p.updated_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		if isUndefinedTable(err, "user_project_access") {
			return db.getUserOwnedProjectsPaginated(ctx, userID, limit, offset)
		}
		return nil, 0, fmt.Errorf("failed to get projects: %w", err)
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt, &p.IsPrivate, &p.CoverImageID, &p.DeletedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, p)
	}
	return projects, total, rows.Err()
}

func (db *Database) getUserOwnedProjectsPaginated(ctx context.Context, userID string, limit, offset int) ([]models.Project, int64, error) {
	var total int64
	err := db.pool.QueryRow(ctx,
		`SELECT COUNT(*)
		 FROM projects p
		 WHERE p.user_id = $1 AND p.deleted_at IS NULL`,
		userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count owned projects: %w", err)
	}

	rows, err := db.pool.Query(ctx,
		`SELECT p.id, p.user_id, p.name, p.description, p.created_at, p.updated_at, p.is_private, p.cover_image_id, p.deleted_at
		 FROM projects p
		 WHERE p.user_id = $1 AND p.deleted_at IS NULL
		 ORDER BY p.updated_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get owned projects: %w", err)
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var project models.Project
		if err := rows.Scan(&project.ID, &project.UserID, &project.Name, &project.Description, &project.CreatedAt, &project.UpdatedAt, &project.IsPrivate, &project.CoverImageID, &project.DeletedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan owned project: %w", err)
		}
		projects = append(projects, project)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate owned projects: %w", err)
	}

	return projects, total, nil
}

func isUndefinedTable(err error, tableName string) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
		if tableName == "" || strings.EqualFold(pgErr.TableName, tableName) {
			return true
		}
	}

	message := strings.ToLower(err.Error())
	if tableName == "" {
		return strings.Contains(message, "does not exist")
	}

	return strings.Contains(message, "relation \""+strings.ToLower(tableName)+"\" does not exist")
}

// CreateProject creates a new project
func (db *Database) CreateProject(ctx context.Context, project *models.Project) error {
	err := db.pool.QueryRow(ctx,
		`INSERT INTO projects (id, user_id, name, description, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, name, description, created_at, updated_at`,
		project.ID, project.UserID, project.Name, project.Description, project.CreatedAt, project.UpdatedAt,
	).Scan(&project.ID, &project.UserID, &project.Name, &project.Description, &project.CreatedAt, &project.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}
	return nil
}

// DeleteProject soft-deletes a project owned by the user.
func (db *Database) DeleteProject(ctx context.Context, projectID, userID string) (bool, error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to begin delete project transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `SELECT set_config('request.jwt.claim.sub', $1, true)`, userID)
	if err != nil {
		return false, fmt.Errorf("failed to set auth context for delete project: %w", err)
	}

	result, err := tx.Exec(ctx,
		`UPDATE projects
		 SET deleted_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		projectID, userID,
	)
	if err != nil {
		return false, fmt.Errorf("failed to delete project: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("failed to commit delete project transaction: %w", err)
	}

	return result.RowsAffected() > 0, nil
}

// Track Queries

// GetProjectTracks retrieves all tracks in a project.
func (db *Database) GetProjectTracks(ctx context.Context, projectID string) ([]models.Track, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, project_id, user_id, name, duration, created_at, updated_at
		 FROM tracks WHERE project_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracks: %w", err)
	}
	defer rows.Close()

	var tracks []models.Track
	for rows.Next() {
		var t models.Track
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.UserID, &t.Name, &t.Duration, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan track: %w", err)
		}
		tracks = append(tracks, t)
	}
	return tracks, nil
}

// GetProjectTracksPaginated retrieves tracks in a project with pagination.
func (db *Database) GetProjectTracksPaginated(ctx context.Context, projectID string, limit, offset int) ([]models.Track, int64, error) {
	var total int64

	// Get total count
	err := db.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM tracks WHERE project_id = $1 AND deleted_at IS NULL`,
		projectID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tracks: %w", err)
	}

	// Get paginated results
	rows, err := db.pool.Query(ctx,
		`SELECT id, project_id, user_id, name, duration, created_at, updated_at
		 FROM tracks
		 WHERE project_id = $1 AND deleted_at IS NULL
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		projectID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get tracks: %w", err)
	}
	defer rows.Close()

	var tracks []models.Track
	for rows.Next() {
		var t models.Track
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.UserID, &t.Name, &t.Duration, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan track: %w", err)
		}
		tracks = append(tracks, t)
	}

	return tracks, total, rows.Err()
}

// CreateTrack creates a new track
func (db *Database) CreateTrack(ctx context.Context, track *models.Track) error {
	err := db.pool.QueryRow(ctx,
		`INSERT INTO tracks (id, project_id, user_id, name, duration, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, project_id, user_id, name, duration, created_at, updated_at`,
		track.ID, track.ProjectID, track.UserID, track.Name, track.Duration, track.CreatedAt, track.UpdatedAt,
	).Scan(&track.ID, &track.ProjectID, &track.UserID, &track.Name, &track.Duration, &track.CreatedAt, &track.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create track: %w", err)
	}
	return nil
}

// DeleteTrack soft-deletes a track and all its versions.
func (db *Database) DeleteTrack(ctx context.Context, trackID string) (bool, error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to begin delete track transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx,
		`UPDATE tracks
		 SET deleted_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL`,
		trackID,
	)
	if err != nil {
		return false, fmt.Errorf("failed to soft-delete track: %w", err)
	}

	if result.RowsAffected() == 0 {
		return false, nil
	}

	if _, err := tx.Exec(ctx,
		`UPDATE track_versions
		 SET deleted_at = NOW()
		 WHERE track_id = $1 AND deleted_at IS NULL`,
		trackID,
	); err != nil {
		return false, fmt.Errorf("failed to soft-delete track versions: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("failed to commit delete track transaction: %w", err)
	}

	return true, nil
}

// TrackVersion Queries

// GetTrackVersions retrieves all versions of a track.
func (db *Database) GetTrackVersions(ctx context.Context, trackID string) ([]models.TrackVersion, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, track_id, version_number, r2_object_key, file_size, checksum, created_at
		 FROM track_versions WHERE track_id = $1 AND deleted_at IS NULL ORDER BY version_number DESC`,
		trackID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get track versions: %w", err)
	}
	defer rows.Close()

	var versions []models.TrackVersion
	for rows.Next() {
		var v models.TrackVersion
		if err := rows.Scan(&v.ID, &v.TrackID, &v.VersionNumber, &v.R2ObjectKey, &v.FileSize, &v.Checksum, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan track version: %w", err)
		}
		versions = append(versions, v)
	}
	return versions, nil
}

// GetTrackVersionsPaginated retrieves track versions with pagination.
func (db *Database) GetTrackVersionsPaginated(ctx context.Context, trackID string, limit, offset int) ([]models.TrackVersion, int64, error) {
	// Get total count
	var total int64
	err := db.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM track_versions WHERE track_id = $1 AND deleted_at IS NULL`,
		trackID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count track versions: %w", err)
	}

	// Get paginated results
	rows, err := db.pool.Query(ctx,
		`SELECT id, track_id, version_number, r2_object_key, file_size, checksum, created_at
		 FROM track_versions WHERE track_id = $1 AND deleted_at IS NULL
		 ORDER BY version_number DESC
		 LIMIT $2 OFFSET $3`,
		trackID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get track versions: %w", err)
	}
	defer rows.Close()

	var versions []models.TrackVersion
	for rows.Next() {
		var v models.TrackVersion
		if err := rows.Scan(&v.ID, &v.TrackID, &v.VersionNumber, &v.R2ObjectKey, &v.FileSize, &v.Checksum, &v.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan track version: %w", err)
		}
		versions = append(versions, v)
	}

	return versions, total, rows.Err()
}

// GetTrackByID retrieves a track by ID.
func (db *Database) GetTrackByID(ctx context.Context, trackID string) (*models.Track, error) {
	var track models.Track
	err := db.pool.QueryRow(ctx,
		`SELECT id, project_id, user_id, name, duration, created_at, updated_at
		 FROM tracks WHERE id = $1 AND deleted_at IS NULL`,
		trackID,
	).Scan(&track.ID, &track.ProjectID, &track.UserID, &track.Name, &track.Duration, &track.CreatedAt, &track.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get track: %w", err)
	}
	return &track, nil
}

// GetLatestTrackVersionNumber gets the highest version number for a track
func (db *Database) GetLatestTrackVersionNumber(ctx context.Context, trackID string) (int, error) {
	var versionNumber int
	err := db.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(version_number), 0) FROM track_versions WHERE track_id = $1`,
		trackID,
	).Scan(&versionNumber)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest version number: %w", err)
	}
	return versionNumber, nil
}

// CreateTrackVersion creates a new version of a track (without R2 data)
func (db *Database) CreateTrackVersion(ctx context.Context, version *models.TrackVersion) error {
	err := db.pool.QueryRow(ctx,
		`INSERT INTO track_versions (id, track_id, version_number, r2_object_key, file_size, checksum, created_at)
		 VALUES ($1, $2, $3, '', 0, '', $4)
		 RETURNING id, track_id, version_number, r2_object_key, file_size, checksum, created_at`,
		version.ID, version.TrackID, version.VersionNumber, version.CreatedAt,
	).Scan(&version.ID, &version.TrackID, &version.VersionNumber, &version.R2ObjectKey, &version.FileSize, &version.Checksum, &version.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create track version: %w", err)
	}
	return nil
}

// UpdateTrackVersionAfterUpload updates a track version after successful R2 upload
func (db *Database) UpdateTrackVersionAfterUpload(ctx context.Context, versionID, r2Key, checksum string, fileSize int64) error {
	err := db.pool.QueryRow(ctx,
		`UPDATE track_versions 
		 SET r2_object_key = $1, file_size = $2, checksum = $3 
		 WHERE id = $4
		 RETURNING id`,
		r2Key, fileSize, checksum, versionID,
	).Scan(&versionID)
	if err != nil {
		return fmt.Errorf("failed to update track version: %w", err)
	}
	return nil
}

// Share Queries

// CreateProjectShare creates a share link for a project
func (db *Database) CreateProjectShare(ctx context.Context, share *models.ProjectShare) error {
	err := db.pool.QueryRow(ctx,
		`INSERT INTO project_shares (id, project_id, shared_by_id, shared_with_id, share_token, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, project_id, shared_by_id, shared_with_id, share_token, expires_at, created_at`,
		share.ID, share.ProjectID, share.SharedByID, share.SharedWithID, share.ShareToken, share.ExpiresAt, share.CreatedAt,
	).Scan(&share.ID, &share.ProjectID, &share.SharedByID, &share.SharedWithID, &share.ShareToken, &share.ExpiresAt, &share.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create project share: %w", err)
	}
	return nil
}

// GetShareByToken retrieves a share by its token
func (db *Database) GetShareByToken(ctx context.Context, token string) (*models.ProjectShare, error) {
	var share models.ProjectShare
	err := db.pool.QueryRow(ctx,
		`SELECT id, project_id, shared_by_id, shared_with_id, share_token, expires_at, created_at
		 FROM project_shares WHERE share_token = $1 AND (expires_at IS NULL OR expires_at > NOW())`,
		token,
	).Scan(&share.ID, &share.ProjectID, &share.SharedByID, &share.SharedWithID, &share.ShareToken, &share.ExpiresAt, &share.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get share: %w", err)
	}
	return &share, nil
}

// OfflineDownload represents a track or project download for offline access.
type OfflineDownload struct {
	ID            string
	UserID        string
	TrackID       string
	R2ObjectKey   string
	FileSizeBytes int64
	Title         string
	Status        string
}

// GetTrackVersionWithAccess retrieves a track version if user has access.
func (db *Database) GetTrackVersionWithAccess(ctx context.Context, trackID, versionID, userID string) (*models.TrackVersion, error) {
	var version models.TrackVersion

	err := db.pool.QueryRow(ctx,
		`SELECT v.id, v.track_id, v.version_number, v.r2_object_key, v.file_size, v.checksum, v.created_at, v.deleted_at
		 FROM track_versions v
		 JOIN tracks t ON v.track_id = t.id
		 JOIN projects p ON t.project_id = p.id
		 WHERE v.id = $1 AND t.id = $2 AND v.deleted_at IS NULL
		 AND (p.user_id = $3 OR p.id IN (
		   SELECT project_id FROM project_shares WHERE shared_with_id = $3 AND revoked_at IS NULL
		 ) OR p.id IN (
		   SELECT project_id FROM collaborators WHERE user_id = $3 AND deleted_at IS NULL
		 ))`,
		versionID, trackID, userID,
	).Scan(&version.ID, &version.TrackID, &version.VersionNumber, &version.R2ObjectKey, &version.FileSize, &version.Checksum, &version.CreatedAt, &version.DeletedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get track version: %w", err)
	}

	return &version, nil
}

// GetOfflineDownloadByUserAndID retrieves an offline download if user owns it.
func (db *Database) GetOfflineDownloadByUserAndID(ctx context.Context, userID, downloadID string) (*OfflineDownload, error) {
	var download OfflineDownload

	err := db.pool.QueryRow(ctx,
		`SELECT id, user_id, track_id, r2_object_key, file_size_bytes, title, status
		 FROM offline_downloads
		 WHERE id = $1 AND user_id = $2`,
		downloadID, userID,
	).Scan(&download.ID, &download.UserID, &download.TrackID, &download.R2ObjectKey, &download.FileSizeBytes, &download.Title, &download.Status)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get offline download: %w", err)
	}

	return &download, nil
}

// GetTrackVersionByNumber retrieves a track version by its version number.
func (db *Database) GetTrackVersionByNumber(ctx context.Context, trackID string, versionNumber int) (*models.TrackVersion, error) {
	var version models.TrackVersion
	err := db.pool.QueryRow(ctx,
		`SELECT id, track_id, version_number, r2_object_key, file_size, checksum, created_at
		 FROM track_versions WHERE track_id = $1 AND version_number = $2`,
		trackID, versionNumber,
	).Scan(&version.ID, &version.TrackID, &version.VersionNumber, &version.R2ObjectKey, &version.FileSize, &version.Checksum, &version.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get track version by number: %w", err)
	}
	return &version, nil
}

// UpdateTrackActiveVersion updates the active version metadata marker by touching updated_at.
func (db *Database) UpdateTrackActiveVersion(ctx context.Context, trackID, r2ObjectKey string, fileSize int64, checksum string) error {
	err := db.pool.QueryRow(ctx,
		`UPDATE tracks SET updated_at = $1 WHERE id = $2 RETURNING id`,
		time.Now(), trackID,
	).Scan(&trackID)
	if err != nil {
		return fmt.Errorf("failed to update track active version: %w", err)
	}
	return nil
}

// CleanupOldOfflineDownloads removes completed and expired offline downloads older than N days.
func (db *Database) CleanupOldOfflineDownloads(ctx context.Context, olderThanDays int) (int64, error) {
	result, err := db.pool.Exec(ctx,
		`DELETE FROM offline_downloads
		 WHERE status = 'completed' AND created_at < NOW() - INTERVAL '1 day' * $1
		 AND expires_at IS NOT NULL AND expires_at < NOW()`,
		olderThanDays,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup offline downloads: %w", err)
	}
	return result.RowsAffected(), nil
}

// CleanupExpiredOfflineDownloads removes completed downloads that are already expired.
func (db *Database) CleanupExpiredOfflineDownloads(ctx context.Context) (int64, error) {
	result, err := db.pool.Exec(ctx,
		`DELETE FROM offline_downloads
		 WHERE status = 'completed' AND expires_at < NOW()`,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired downloads: %w", err)
	}
	return result.RowsAffected(), nil
}

// GetUserOfflineDownloads retrieves all offline downloads for a user.
func (db *Database) GetUserOfflineDownloads(ctx context.Context, userID string) ([]models.OfflineDownload, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, user_id, track_version_id, track_id, project_id, file_size_bytes, r2_object_key,
		        status, title, artist_name, project_name, duration_seconds, created_at, updated_at
		 FROM offline_downloads WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get offline downloads: %w", err)
	}
	defer rows.Close()

	var downloads []models.OfflineDownload
	for rows.Next() {
		var offline models.OfflineDownload
		if err := rows.Scan(
			&offline.ID,
			&offline.UserID,
			&offline.TrackVersionID,
			&offline.TrackID,
			&offline.ProjectID,
			&offline.FileSizeBytes,
			&offline.R2ObjectKey,
			&offline.Status,
			&offline.Title,
			&offline.ArtistName,
			&offline.ProjectName,
			&offline.DurationSeconds,
			&offline.CreatedAt,
			&offline.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan offline download: %w", err)
		}
		downloads = append(downloads, offline)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed iterating offline downloads: %w", err)
	}

	return downloads, nil
}

// DeleteOfflineDownload removes a specific offline download.
func (db *Database) DeleteOfflineDownload(ctx context.Context, downloadID string) error {
	err := db.pool.QueryRow(ctx,
		`DELETE FROM offline_downloads WHERE id = $1 RETURNING id`,
		downloadID,
	).Scan(&downloadID)
	if err != nil {
		return fmt.Errorf("failed to delete offline download: %w", err)
	}
	return nil
}

// TrackVersionEngagement holds aggregated engagement metrics for a version list response.
type TrackVersionEngagement struct {
	TrackLikes    int64
	CommentCounts map[string]int64
}

// GetTrackVersionEngagementBatch returns like count and per-version comment counts in batch.
func (db *Database) GetTrackVersionEngagementBatch(ctx context.Context, trackID string) (*TrackVersionEngagement, error) {
	engagement := &TrackVersionEngagement{CommentCounts: make(map[string]int64)}

	err := db.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM likes WHERE track_id = $1`,
		trackID,
	).Scan(&engagement.TrackLikes)
	if err != nil {
		return nil, fmt.Errorf("failed to get track likes: %w", err)
	}

	rows, err := db.pool.Query(ctx,
		`SELECT tv.id, COALESCE(COUNT(c.id), 0)
		 FROM track_versions tv
		 LEFT JOIN comments c ON c.track_version_id = tv.id
		 WHERE tv.track_id = $1 AND tv.deleted_at IS NULL
		 GROUP BY tv.id`,
		trackID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get track version comment counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var versionID string
		var count int64
		if err := rows.Scan(&versionID, &count); err != nil {
			return nil, fmt.Errorf("failed to scan version engagement: %w", err)
		}
		engagement.CommentCounts[versionID] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed iterating version engagement: %w", err)
	}

	return engagement, nil
}

// GetUserRoleOptimized resolves project access role in a single query.
// Returns one of: owner, editor, commenter, viewer, denied.
func (db *Database) GetUserRoleOptimized(ctx context.Context, projectID, userID string) (string, error) {
	var role string

	err := db.pool.QueryRow(ctx, `
		SELECT COALESCE(
			CASE WHEN p.user_id = $2 THEN 'owner' END,
			c.role,
			CASE WHEN ps.id IS NOT NULL THEN 'viewer' END,
			'denied'
		) AS role
		FROM projects p
		LEFT JOIN collaborators c
			ON p.id = c.project_id AND c.user_id = $2 AND c.deleted_at IS NULL
		LEFT JOIN project_shares ps
			ON p.id = ps.project_id AND ps.shared_with_id = $2 AND ps.revoked_at IS NULL
		WHERE p.id = $1
		LIMIT 1
	`, projectID, userID).Scan(&role)

	if err == sql.ErrNoRows {
		return "denied", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get user role: %w", err)
	}

	return role, nil
}

// ============================================================================
// PERFORMANCE-OPTIMIZED ACCESS CHECKING (O(1) instead of O(n*m))
// ============================================================================

// GetUserProjectAccessList returns all project IDs accessible by a user (for batching)
// Uses denormalized user_project_access table for O(log n) lookup
func (db *Database) GetUserProjectAccessList(ctx context.Context, userID string) ([]string, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT DISTINCT project_id FROM user_project_access 
		 WHERE user_id = $1 AND access_type IN ('owner', 'collaborator', 'shared')`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user project access: %w", err)
	}
	defer rows.Close()

	var projectIDs []string
	for rows.Next() {
		var projectID string
		if err := rows.Scan(&projectID); err != nil {
			return nil, fmt.Errorf("failed to scan project ID: %w", err)
		}
		projectIDs = append(projectIDs, projectID)
	}
	return projectIDs, rows.Err()
}

// GetUserTrackAccessList returns all track IDs accessible by a user (for batching)
// Uses denormalized user_track_access table for O(log n) lookup
func (db *Database) GetUserTrackAccessList(ctx context.Context, userID string) ([]string, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT DISTINCT track_id FROM user_track_access WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user track access: %w", err)
	}
	defer rows.Close()

	var trackIDs []string
	for rows.Next() {
		var trackID string
		if err := rows.Scan(&trackID); err != nil {
			return nil, fmt.Errorf("failed to scan track ID: %w", err)
		}
		trackIDs = append(trackIDs, trackID)
	}
	return trackIDs, rows.Err()
}

// CanUserAccessProject checks in O(1) if user can access a project
// Uses direct denormalized table instead of nested subqueries
func (db *Database) CanUserAccessProject(ctx context.Context, userID, projectID string) (bool, error) {
	var exists bool
	err := db.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM user_project_access 
			WHERE user_id = $1 AND project_id = $2
			LIMIT 1
		)`,
		userID, projectID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check project access: %w", err)
	}
	return exists, nil
}

// CanUserAccessTrack checks in O(1) if user can access a track
// Uses direct denormalized table instead of nested subqueries
func (db *Database) CanUserAccessTrack(ctx context.Context, userID, trackID string) (bool, error) {
	var exists bool
	err := db.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM user_track_access 
			WHERE user_id = $1 AND track_id = $2
			LIMIT 1
		)`,
		userID, trackID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check track access: %w", err)
	}
	return exists, nil
}

// GetAccessibleTracksCount returns count of accessible tracks for user (O(1) with index)
func (db *Database) GetAccessibleTracksCount(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := db.pool.QueryRow(ctx,
		`SELECT COUNT(DISTINCT track_id) FROM user_track_access WHERE user_id = $1`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count accessible tracks: %w", err)
	}
	return count, nil
}

// RefreshMaterializedView can be called by job processor to refresh mv_user_accessible_projects
// Run this hourly or on-demand to keep materialized view fresh
func (db *Database) RefreshAccessibilityMaterializedView(ctx context.Context) error {
	_, err := db.pool.Exec(ctx,
		`REFRESH MATERIALIZED VIEW CONCURRENTLY mv_user_accessible_projects`,
	)
	if err != nil {
		return fmt.Errorf("failed to refresh materialized view: %w", err)
	}
	return nil
}
