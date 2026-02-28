package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/IlyesDjari/purp-tape/backend/internal/models"
)

type PaginationParams struct {
	Limit  int
	Offset int
}

func NewPaginationParams(limit, offset int) PaginationParams {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return PaginationParams{Limit: limit, Offset: offset}
}

type ProjectWithStats struct {
	Project      models.Project
	PlayCount    int64
	LikeCount    int64
	CommentCount int64
	TrackCount   int64
}

func (db *Database) GetLatestTrackVersion(ctx context.Context, trackID string) (*models.TrackVersion, error) {
	var v models.TrackVersion
	err := db.pool.QueryRow(ctx,
		`SELECT id, track_id, version_number, r2_object_key, file_size, checksum, created_at, deleted_at
		 FROM track_versions
		 WHERE track_id = $1 AND deleted_at IS NULL
		 ORDER BY version_number DESC
		 LIMIT 1`,
		trackID,
	).Scan(&v.ID, &v.TrackID, &v.VersionNumber, &v.R2ObjectKey, &v.FileSize, &v.Checksum, &v.CreatedAt, &v.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest track version: %w", err)
	}
	return &v, nil
}

func (db *Database) CountUserProjects(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := db.pool.QueryRow(ctx,
		`SELECT COUNT(DISTINCT p.id)
		 FROM projects p
		 LEFT JOIN project_shares ps ON p.id = ps.project_id
		 WHERE (p.user_id = $1 OR ps.shared_with_id = $1) AND p.deleted_at IS NULL`,
		userID,
	).Scan(&count)
	return count, err
}

func (db *Database) GetProjectTrackCount(ctx context.Context, projectID string) (int64, error) {
	var count int64
	err := db.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM tracks WHERE project_id = $1 AND deleted_at IS NULL`,
		projectID,
	).Scan(&count)
	return count, err
}

func (db *Database) GetProjectStats(ctx context.Context, projectID string) (map[string]interface{}, error) {
	var (
		totalPlays      sql.NullInt64
		uniqueListeners sql.NullInt64
		totalLikes      sql.NullInt64
		commentCount    sql.NullInt64
		avgPlayDuration sql.NullFloat64
		lastPlayAt      *time.Time
		computedAt      time.Time
	)

	err := db.pool.QueryRow(ctx,
		`SELECT
			COALESCE(total_plays, 0),
			COALESCE(unique_listeners, 0),
			COALESCE(total_likes, 0),
			COALESCE(comment_count, 0),
			COALESCE(avg_play_duration, 0),
			last_play_at,
			computed_at
		 FROM project_stats_materialized
		 WHERE project_id = $1`,
		projectID,
	).Scan(&totalPlays, &uniqueListeners, &totalLikes, &commentCount, &avgPlayDuration, &lastPlayAt, &computedAt)

	if err == sql.ErrNoRows {
		return map[string]interface{}{
			"total_plays":       int64(0),
			"unique_listeners":  int64(0),
			"total_likes":       int64(0),
			"comment_count":     int64(0),
			"avg_play_duration": float64(0),
			"last_play_at":      nil,
			"computed_at":       time.Now().Format(time.RFC3339),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project stats: %w", err)
	}

	return map[string]interface{}{
		"total_plays":       totalPlays.Int64,
		"unique_listeners":  uniqueListeners.Int64,
		"total_likes":       totalLikes.Int64,
		"comment_count":     commentCount.Int64,
		"avg_play_duration": avgPlayDuration.Float64,
		"last_play_at":      lastPlayAt,
		"computed_at":       computedAt.Format(time.RFC3339),
	}, nil
}

func (db *Database) RefreshProjectStatsView(ctx context.Context) error {
	_, err := db.pool.Exec(ctx, `REFRESH MATERIALIZED VIEW CONCURRENTLY project_stats_materialized`)
	if err != nil {
		return fmt.Errorf("failed to refresh project stats view: %w", err)
	}
	return nil
}

func (db *Database) GetUserStats(ctx context.Context, userID string) (map[string]interface{}, error) {
	var totalProjects int64
	var totalTracks int64
	var totalVersions int64
	var storageUsedMB int64
	var lastActiveAt *time.Time

	err := db.pool.QueryRow(ctx,
		`SELECT
			COALESCE(COUNT(DISTINCT p.id), 0),
			COALESCE(COUNT(DISTINCT t.id), 0),
			COALESCE(COUNT(DISTINCT v.id), 0),
			COALESCE(SUM(v.file_size) / 1024 / 1024, 0),
			MAX(GREATEST(p.updated_at, t.updated_at, v.created_at))
		 FROM projects p
		 LEFT JOIN tracks t ON p.id = t.project_id AND t.deleted_at IS NULL
		 LEFT JOIN track_versions v ON t.id = v.track_id AND v.deleted_at IS NULL
		 WHERE p.user_id = $1 AND p.deleted_at IS NULL`,
		userID,
	).Scan(&totalProjects, &totalTracks, &totalVersions, &storageUsedMB, &lastActiveAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	return map[string]interface{}{
		"total_projects":  totalProjects,
		"total_tracks":    totalTracks,
		"total_versions":  totalVersions,
		"storage_used_mb": storageUsedMB,
		"last_active_at":  lastActiveAt,
	}, nil
}

func (db *Database) GetDailyPlayStats(ctx context.Context, projectID string, days int) ([]map[string]interface{}, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT
		   DATE(ph.started_at) as date,
		   COUNT(*) as play_count,
		   COUNT(DISTINCT ph.listener_user_id) as unique_listeners,
		   AVG(ph.duration_seconds) as avg_duration
		 FROM play_history ph
		 WHERE ph.project_id = $1 AND ph.started_at > NOW() - $2::INTERVAL
		 GROUP BY DATE(ph.started_at)
		 ORDER BY DATE(ph.started_at) DESC`,
		projectID, fmt.Sprintf("%d days", days),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily play stats: %w", err)
	}
	defer rows.Close()

	var stats []map[string]interface{}
	for rows.Next() {
		var date interface{}
		var playCount int
		var uniqueListeners int
		var avgDuration *float64
		if err := rows.Scan(&date, &playCount, &uniqueListeners, &avgDuration); err != nil {
			return nil, fmt.Errorf("failed to scan daily play stats: %w", err)
		}
		stats = append(stats, map[string]interface{}{
			"date":             date,
			"play_count":       playCount,
			"unique_listeners": uniqueListeners,
			"avg_duration":     avgDuration,
		})
	}
	return stats, rows.Err()
}

func (db *Database) GetTopListeners(ctx context.Context, projectID string, limit int) ([]map[string]interface{}, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT COALESCE(listener_user_id::text, 'anonymous') AS listener_id, COUNT(*) AS play_count
		 FROM play_history
		 WHERE project_id = $1
		 GROUP BY listener_user_id
		 ORDER BY play_count DESC
		 LIMIT $2`,
		projectID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get top listeners: %w", err)
	}
	defer rows.Close()

	result := make([]map[string]interface{}, 0, limit)
	for rows.Next() {
		var listenerID string
		var playCount int64
		if err := rows.Scan(&listenerID, &playCount); err != nil {
			return nil, fmt.Errorf("failed to scan top listeners: %w", err)
		}
		result = append(result, map[string]interface{}{"listener_user_id": listenerID, "play_count": playCount})
	}
	return result, rows.Err()
}

func (db *Database) RecordPlayStart(ctx context.Context, trackVersionID, listenerUserID, device string) (string, error) {
	var playID string
	err := db.pool.QueryRow(ctx,
		`INSERT INTO play_history (id, track_version_id, listener_user_id, started_at, device)
		 VALUES (gen_random_uuid(), $1, NULLIF($2, '')::uuid, NOW(), $3)
		 RETURNING id`,
		trackVersionID, listenerUserID, device,
	).Scan(&playID)
	if err != nil {
		return "", fmt.Errorf("failed to record play start: %w", err)
	}
	return playID, nil
}

func (db *Database) RecordPlayEnd(ctx context.Context, playID string, durationListened int) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE play_history
		 SET ended_at = NOW(), duration_listened = $1
		 WHERE id = $2`,
		durationListened, playID,
	)
	if err != nil {
		return fmt.Errorf("failed to record play end: %w", err)
	}
	return nil
}

func (db *Database) GetTrackStats(ctx context.Context, trackID string) (map[string]interface{}, error) {
	var totalPlays int64
	var uniqueListeners int64
	var totalLikes int64
	err := db.pool.QueryRow(ctx,
		`SELECT
		 COALESCE(COUNT(ph.id), 0),
		 COALESCE(COUNT(DISTINCT ph.listener_user_id), 0),
		 COALESCE((SELECT COUNT(*) FROM likes WHERE track_id = $1), 0)
		 FROM play_history ph
		 WHERE ph.track_id = $1`,
		trackID,
	).Scan(&totalPlays, &uniqueListeners, &totalLikes)
	if err != nil {
		return nil, fmt.Errorf("failed to get track stats: %w", err)
	}
	return map[string]interface{}{"total_plays": totalPlays, "unique_listeners": uniqueListeners, "total_likes": totalLikes}, nil
}

func (db *Database) GetProjectsWithStats(ctx context.Context, userID string, limit, offset int) ([]ProjectWithStats, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT
		   p.id, p.user_id, p.name, p.description, p.created_at, p.updated_at, p.is_private, p.cover_image_id, p.deleted_at,
		   COALESCE(COUNT(DISTINCT ph.id), 0) as play_count,
		   COALESCE(COUNT(DISTINCT pl.id), 0) as like_count,
		   COALESCE(COUNT(DISTINCT c.id), 0) as comment_count,
		   COALESCE(COUNT(DISTINCT t.id), 0) as track_count
		 FROM projects p
		 LEFT JOIN play_history ph ON p.id = ph.project_id
		 LEFT JOIN project_likes pl ON p.id = pl.project_id
		 LEFT JOIN comments c ON p.id = c.project_id
		 LEFT JOIN tracks t ON p.id = t.project_id AND t.deleted_at IS NULL
		 WHERE p.user_id = $1 AND p.deleted_at IS NULL
		 GROUP BY p.id
		 ORDER BY p.updated_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects with stats: %w", err)
	}
	defer rows.Close()

	var results []ProjectWithStats
	for rows.Next() {
		var p models.Project
		var stats ProjectWithStats
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt,
			&p.IsPrivate, &p.CoverImageID, &p.DeletedAt,
			&stats.PlayCount, &stats.LikeCount, &stats.CommentCount, &stats.TrackCount); err != nil {
			return nil, fmt.Errorf("failed to scan project stats: %w", err)
		}
		stats.Project = p
		results = append(results, stats)
	}
	return results, rows.Err()
}

func (db *Database) GetTracksWithLikeCount(ctx context.Context, projectID string) ([]map[string]interface{}, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT
		   t.id, t.name, t.duration, t.created_at, t.updated_at,
		   COALESCE(COUNT(l.id), 0) as like_count
		 FROM tracks t
		 LEFT JOIN likes l ON t.id = l.track_id
		 WHERE t.project_id = $1 AND t.deleted_at IS NULL
		 GROUP BY t.id
		 ORDER BY t.created_at DESC`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracks with like count: %w", err)
	}
	defer rows.Close()

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id string
		var name string
		var duration int
		var createdAt interface{}
		var updatedAt interface{}
		var likeCount int64
		if err := rows.Scan(&id, &name, &duration, &createdAt, &updatedAt, &likeCount); err != nil {
			return nil, fmt.Errorf("failed to scan track like count: %w", err)
		}
		results = append(results, map[string]interface{}{
			"id":         id,
			"name":       name,
			"duration":   duration,
			"created_at": createdAt,
			"updated_at": updatedAt,
			"like_count": likeCount,
		})
	}
	return results, rows.Err()
}

func (db *Database) SearchProjects(ctx context.Context, query string, limit int, offset ...int) ([]models.Project, int64, error) {
	o := 0
	if len(offset) > 0 {
		o = offset[0]
	}

	var total int64
	err := db.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM projects
		 WHERE to_tsvector('english', name || ' ' || COALESCE(description, '')) @@ plainto_tsquery('english', $1)
		   AND deleted_at IS NULL
		   AND is_private = false`,
		query,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count searched projects: %w", err)
	}

	rows, err := db.pool.Query(ctx,
		`SELECT id, user_id, name, description, created_at, updated_at, is_private, cover_image_id, deleted_at
		 FROM projects
		 WHERE to_tsvector('english', name || ' ' || COALESCE(description, '')) @@ plainto_tsquery('english', $1)
		   AND deleted_at IS NULL
		   AND is_private = false
		 ORDER BY ts_rank(to_tsvector('english', name || ' ' || COALESCE(description, '')), plainto_tsquery('english', $1)) DESC,
		          updated_at DESC
		 LIMIT $2 OFFSET $3`,
		query, limit, o,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search projects: %w", err)
	}
	defer rows.Close()

	items := make([]models.Project, 0)
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt,
			&p.IsPrivate, &p.CoverImageID, &p.DeletedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan searched project: %w", err)
		}
		items = append(items, p)
	}
	return items, total, rows.Err()
}

func (db *Database) SearchTracks(ctx context.Context, query string, limit int, offset ...int) ([]models.Track, int64, error) {
	o := 0
	if len(offset) > 0 {
		o = offset[0]
	}

	var total int64
	err := db.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM tracks
		 WHERE to_tsvector('english', name) @@ plainto_tsquery('english', $1)
		   AND deleted_at IS NULL
		   AND project_id IN (SELECT id FROM projects WHERE deleted_at IS NULL AND is_private = false)`,
		query,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count searched tracks: %w", err)
	}

	rows, err := db.pool.Query(ctx,
		`SELECT id, project_id, user_id, name, duration, created_at, updated_at, deleted_at
		 FROM tracks
		 WHERE to_tsvector('english', name) @@ plainto_tsquery('english', $1)
		   AND deleted_at IS NULL
		   AND project_id IN (SELECT id FROM projects WHERE deleted_at IS NULL AND is_private = false)
		 ORDER BY ts_rank(to_tsvector('english', name), plainto_tsquery('english', $1)) DESC,
		          updated_at DESC
		 LIMIT $2 OFFSET $3`,
		query, limit, o,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search tracks: %w", err)
	}
	defer rows.Close()

	items := make([]models.Track, 0)
	for rows.Next() {
		var t models.Track
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.UserID, &t.Name, &t.Duration, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan searched track: %w", err)
		}
		items = append(items, t)
	}
	return items, total, rows.Err()
}

func (db *Database) SearchUsers(ctx context.Context, query string, limit int, offset ...int) ([]models.User, int64, error) {
	o := 0
	if len(offset) > 0 {
		o = offset[0]
	}

	var total int64
	err := db.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM users
		 WHERE to_tsvector('english', username) @@ plainto_tsquery('english', $1)
		   AND deleted_at IS NULL`,
		query,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count searched users: %w", err)
	}

	rows, err := db.pool.Query(ctx,
		`SELECT id, email, username, avatar_url, created_at, updated_at, deleted_at
		 FROM users
		 WHERE to_tsvector('english', username) @@ plainto_tsquery('english', $1)
		   AND deleted_at IS NULL
		 ORDER BY ts_rank(to_tsvector('english', username), plainto_tsquery('english', $1)) DESC,
		          created_at DESC
		 LIMIT $2 OFFSET $3`,
		query, limit, o,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search users: %w", err)
	}
	defer rows.Close()

	items := make([]models.User, 0)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan searched user: %w", err)
		}
		items = append(items, u)
	}
	return items, total, rows.Err()
}

func (db *Database) SearchUserProjects(ctx context.Context, userID, query string, limit int) ([]models.Project, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT DISTINCT p.id, p.user_id, p.name, p.description, p.created_at, p.updated_at, p.is_private, p.cover_image_id, p.deleted_at
		 FROM projects p
		 LEFT JOIN project_shares ps ON p.id = ps.project_id
		 WHERE (p.user_id = $1 OR ps.shared_with_id = $1)
		   AND p.deleted_at IS NULL
		   AND to_tsvector('english', p.name || ' ' || COALESCE(p.description, '')) @@ plainto_tsquery('english', $2)
		 ORDER BY p.updated_at DESC
		 LIMIT $3`,
		userID, query, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search user projects: %w", err)
	}
	defer rows.Close()

	projects := make([]models.Project, 0)
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt, &p.IsPrivate, &p.CoverImageID, &p.DeletedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user project: %w", err)
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (db *Database) GetTrendingProjects(ctx context.Context, timeframe string, limit int) ([]map[string]interface{}, error) {
	interval := "7 days"
	if timeframe == "day" {
		interval = "1 day"
	} else if timeframe == "month" {
		interval = "30 days"
	}

	rows, err := db.pool.Query(ctx,
		`SELECT p.id, p.name, p.user_id, COALESCE(COUNT(ph.id), 0) AS play_count
		 FROM projects p
		 LEFT JOIN play_history ph ON ph.project_id = p.id AND ph.started_at > NOW() - $1::INTERVAL
		 WHERE p.deleted_at IS NULL AND p.is_private = false
		 GROUP BY p.id
		 ORDER BY play_count DESC, p.updated_at DESC
		 LIMIT $2`,
		interval, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get trending projects: %w", err)
	}
	defer rows.Close()

	out := make([]map[string]interface{}, 0, limit)
	for rows.Next() {
		var id, name, userID string
		var playCount int64
		if err := rows.Scan(&id, &name, &userID, &playCount); err != nil {
			return nil, fmt.Errorf("failed to scan trending project: %w", err)
		}
		out = append(out, map[string]interface{}{"id": id, "name": name, "user_id": userID, "play_count": playCount})
	}
	return out, rows.Err()
}

func (db *Database) GetPublicProjects(ctx context.Context, genre string, limit int) ([]models.Project, error) {
	query := `SELECT id, user_id, name, description, created_at, updated_at, is_private, cover_image_id, deleted_at
		 FROM projects
		 WHERE deleted_at IS NULL AND is_private = false`
	args := []interface{}{}
	if genre != "" {
		query += ` AND genre = $1`
		args = append(args, genre)
	}
	query += ` ORDER BY updated_at DESC LIMIT $2`
	args = append(args, limit)

	rows, err := db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get public projects: %w", err)
	}
	defer rows.Close()

	projects := make([]models.Project, 0, limit)
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt, &p.IsPrivate, &p.CoverImageID, &p.DeletedAt); err != nil {
			return nil, fmt.Errorf("failed to scan public project: %w", err)
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}
