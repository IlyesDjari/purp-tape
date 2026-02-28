package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IlyesDjari/purp-tape/backend/internal/models"
)

type JobData struct {
	ID          string          `db:"id"`
	JobType     string          `db:"job_type"`
	Data        json.RawMessage `db:"data"`
	Status      string          `db:"status"`
	Attempts    int             `db:"attempts"`
	MaxAttempts int             `db:"max_attempts"`
}

type OfflineDownloadInfo struct {
	VersionID          string
	TrackID            string
	FileSize           int64
	R2ObjectKey        string
	Checksum           string
	TrackName          string
	Duration           int64
	ProjectID          string
	ProjectName        string
	CoverImageID       *string
	CoverR2Key         *string
	SubscriptionTier   string
	UsedStorage        int64
	ExistingDownloadID *string
	ExistingStatus     *string
}

type FinOpsSnapshot struct {
	TrackVersionBytes      int64
	ImageBytes             int64
	OfflineDownloadBytes   int64
	TotalActiveStorageBytes int64
	PendingCleanupJobs     int64
	FailedCleanupJobs      int64
	PendingCleanupBytes    int64
	EstimatedMonthlyCostUSD float64
	ActualMonthlyCostUSD    float64
	GoverningMonthlyCostUSD float64
}

type FinOpsCostEvent struct {
	ID         string
	Source     string
	Service    string
	Category   string
	AmountUSD  float64
	OccurredAt time.Time
	Metadata   map[string]interface{}
}

type FinOpsMonthlySummary struct {
	Days               int
	ActualCostUSD      float64
	StorageEstimatedUSD float64
	GoverningCostUSD   float64
	BudgetUSD          float64
	UtilizationRatio   float64
}

func (db *Database) CreateAuditLog(ctx context.Context, entry *models.AuditLog) error {
	err := db.pool.QueryRow(ctx,
		`INSERT INTO audit_logs (id, user_id, action, resource, details, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		entry.ID, entry.UserID, entry.Action, entry.Resource, entry.Details, entry.CreatedAt,
	).Scan(&entry.ID)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	return nil
}

func (db *Database) GetAuditLogsForUser(ctx context.Context, userID string, limit, offset int) ([]models.AuditLog, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, user_id, action, resource, details, created_at
		 FROM audit_logs
		 WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var entry models.AuditLog
		if err := rows.Scan(&entry.ID, &entry.UserID, &entry.Action, &entry.Resource, &entry.Details, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, entry)
	}
	return logs, rows.Err()
}

func (db *Database) GetAuditLogsForResource(ctx context.Context, resource string) ([]models.AuditLog, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, user_id, action, resource, details, created_at
		 FROM audit_logs
		 WHERE resource = $1
		 ORDER BY created_at DESC`,
		resource,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by resource: %w", err)
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var entry models.AuditLog
		if err := rows.Scan(&entry.ID, &entry.UserID, &entry.Action, &entry.Resource, &entry.Details, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, entry)
	}
	return logs, rows.Err()
}

func (db *Database) GetAllUserTracks(ctx context.Context, userID string) ([]models.Track, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, project_id, user_id, name, duration, created_at, updated_at
		 FROM tracks
		 WHERE user_id = $1
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tracks: %w", err)
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
	return tracks, rows.Err()
}

func (db *Database) GetUserPrivacySettings(ctx context.Context, userID string) (map[string]bool, error) {
	var settingsJSON []byte
	err := db.pool.QueryRow(ctx,
		`SELECT COALESCE(privacy_settings, '{"share_profile": true, "show_in_search": true, "allow_collaboration_requests": true}')
		 FROM users
		 WHERE id = $1`,
		userID,
	).Scan(&settingsJSON)
	if err == sql.ErrNoRows {
		return map[string]bool{"share_profile": true, "show_in_search": true, "allow_collaboration_requests": true}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get privacy settings: %w", err)
	}
	var settings map[string]bool
	if err := json.Unmarshal(settingsJSON, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse privacy settings: %w", err)
	}
	return settings, nil
}

func (db *Database) UpdateUserPrivacySettings(ctx context.Context, userID string, settings map[string]bool) error {
	settingsJSON, _ := json.Marshal(settings)
	_, err := db.pool.Exec(ctx,
		`UPDATE users
		 SET privacy_settings = $1, updated_at = NOW()
		 WHERE id = $2`,
		settingsJSON, userID,
	)
	return err
}

func (db *Database) GetPendingJobs(ctx context.Context, limit int) ([]*JobData, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, job_type, data, status, attempts, max_attempts
		 FROM background_jobs
		 WHERE status = 'pending' AND attempts < max_attempts
		 ORDER BY created_at ASC
		 LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*JobData
	for rows.Next() {
		var job JobData
		if err := rows.Scan(&job.ID, &job.JobType, &job.Data, &job.Status, &job.Attempts, &job.MaxAttempts); err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		jobs = append(jobs, &job)
	}
	return jobs, rows.Err()
}

// ClaimPendingJobs atomically claims pending jobs using FOR UPDATE SKIP LOCKED.
// This enables safe horizontal scaling across multiple API instances.
func (db *Database) ClaimPendingJobs(ctx context.Context, limit int) ([]*JobData, error) {
	if limit <= 0 {
		limit = 10
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin claim jobs transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		WITH claimable AS (
			SELECT id
			FROM background_jobs
			WHERE status = 'pending' AND attempts < max_attempts
			ORDER BY created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		UPDATE background_jobs bj
		SET status = 'processing', started_at = NOW(), updated_at = NOW()
		FROM claimable c
		WHERE bj.id = c.id
		RETURNING bj.id, bj.job_type, bj.data, bj.status, bj.attempts, bj.max_attempts
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to claim pending jobs: %w", err)
	}
	defer rows.Close()

	jobs := make([]*JobData, 0, limit)
	for rows.Next() {
		var job JobData
		if err := rows.Scan(&job.ID, &job.JobType, &job.Data, &job.Status, &job.Attempts, &job.MaxAttempts); err != nil {
			return nil, fmt.Errorf("failed to scan claimed job: %w", err)
		}
		jobs = append(jobs, &job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading claimed jobs: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit claimed jobs transaction: %w", err)
	}

	return jobs, nil
}

func (db *Database) CreateBackgroundJob(ctx context.Context, jobType string, data map[string]interface{}, maxAttempts int) (string, error) {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal background job data: %w", err)
	}

	var jobID string
	err = db.pool.QueryRow(ctx,
		`INSERT INTO background_jobs (job_type, data, status, attempts, max_attempts, created_at)
		 VALUES ($1, $2, 'pending', 0, $3, NOW())
		 RETURNING id`,
		jobType, dataJSON, maxAttempts,
	).Scan(&jobID)
	if err != nil {
		return "", fmt.Errorf("failed to create background job: %w", err)
	}

	return jobID, nil
}

func (db *Database) MarkJobCompleted(ctx context.Context, jobID string, result json.RawMessage) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE background_jobs
		 SET status = 'completed', result = $1, completed_at = NOW()
		 WHERE id = $2`,
		result, jobID,
	)
	if err != nil {
		return fmt.Errorf("failed to mark job completed: %w", err)
	}
	return nil
}

func (db *Database) MarkJobFailed(ctx context.Context, jobID string, errorMsg string) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE background_jobs
		 SET status = CASE WHEN (attempts + 1) >= max_attempts THEN 'failed' ELSE 'pending' END,
		     attempts = attempts + 1,
		     error_message = $1,
		     started_at = NULL
		 WHERE id = $2`,
		errorMsg, jobID,
	)
	if err != nil {
		return fmt.Errorf("failed to mark job failed: %w", err)
	}
	return nil
}

func (db *Database) UpdateJobStatus(ctx context.Context, jobID, status string, result json.RawMessage, errorMsg string) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE background_jobs
		 SET status = $1,
		     result = CASE WHEN $2::jsonb IS NULL THEN result ELSE $2 END,
		     error_message = CASE WHEN $3 = '' THEN error_message ELSE $3 END,
		     updated_at = NOW()
		 WHERE id = $4`,
		status, result, errorMsg, jobID,
	)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}
	return nil
}

func (db *Database) GetUserSubscription(ctx context.Context, userID string) (map[string]interface{}, error) {
	var tier string
	var quotaMB int64
	var usedMB int64

	err := db.pool.QueryRow(ctx,
		`SELECT COALESCE(tier, 'free'), COALESCE(storage_quota_mb, 1024), COALESCE(storage_used_mb, 0)
		 FROM subscriptions
		 WHERE user_id = $1`,
		userID,
	).Scan(&tier, &quotaMB, &usedMB)
	if err == sql.ErrNoRows {
		return map[string]interface{}{"tier": "free", "storage_quota_mb": int64(1024), "storage_used_mb": int64(0)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user subscription: %w", err)
	}

	return map[string]interface{}{"tier": tier, "storage_quota_mb": quotaMB, "storage_used_mb": usedMB}, nil
}

func (db *Database) GetUserStorageUsed(ctx context.Context, userID string) (int64, error) {
	var usedMB int64
	err := db.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(tv.file_size), 0) / (1024 * 1024)
		 FROM track_versions tv
		 JOIN tracks t ON tv.track_id = t.id
		 JOIN projects p ON t.project_id = p.id
		 WHERE p.user_id = $1 AND tv.deleted_at IS NULL`,
		userID,
	).Scan(&usedMB)
	if err != nil {
		return 0, fmt.Errorf("failed to get user storage used: %w", err)
	}
	return usedMB, nil
}

func (db *Database) GetOfflineDownloadInfo(ctx context.Context, userID, trackID string) (*OfflineDownloadInfo, error) {
	query := `
		SELECT
			tv.id as version_id,
			tv.track_id,
			tv.file_size,
			tv.r2_object_key,
			tv.checksum,
			t.name as track_name,
			t.duration,
			p.id as project_id,
			p.name as project_name,
			p.cover_image_id,
			i.r2_object_key as cover_r2_key,
			COALESCE(s.tier, 'free') as subscription_tier,
			COALESCE(SUM(od.file_size_bytes), 0) as used_storage,
			od.id as existing_download_id,
			od.status as existing_status
		FROM track_versions tv
		JOIN tracks t ON tv.track_id = t.id
		JOIN projects p ON t.project_id = p.id
		LEFT JOIN subscriptions s ON s.user_id = $1
		LEFT JOIN images i ON p.cover_image_id = i.id
		LEFT JOIN offline_downloads od ON od.user_id = $1 AND od.track_version_id = tv.id AND od.status = 'completed'
		WHERE t.id = $2
		  AND tv.id = (
			SELECT id FROM track_versions
			WHERE track_id = $2 AND deleted_at IS NULL
			ORDER BY version_number DESC
			LIMIT 1
		  )
		  AND (p.user_id = $1 OR p.id IN (
			SELECT project_id FROM project_shares WHERE shared_with_id = $1 AND revoked_at IS NULL
		  ))
		GROUP BY tv.id, t.id, p.id, s.tier, i.r2_object_key, od.id`

	var info OfflineDownloadInfo
	err := db.pool.QueryRow(ctx, query, userID, trackID).Scan(
		&info.VersionID,
		&info.TrackID,
		&info.FileSize,
		&info.R2ObjectKey,
		&info.Checksum,
		&info.TrackName,
		&info.Duration,
		&info.ProjectID,
		&info.ProjectName,
		&info.CoverImageID,
		&info.CoverR2Key,
		&info.SubscriptionTier,
		&info.UsedStorage,
		&info.ExistingDownloadID,
		&info.ExistingStatus,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get offline download info: %w", err)
	}
	return &info, nil
}

func (db *Database) GetOfflineStorageUsed(ctx context.Context, userID string) (int64, error) {
	var total int64
	err := db.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(file_size_bytes), 0)
		 FROM offline_downloads
		 WHERE user_id = $1 AND status = 'completed'`,
		userID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get offline storage used: %w", err)
	}
	return total, nil
}

func (db *Database) GetFinOpsSnapshot(ctx context.Context, storageCostPerGBMonth float64) (*FinOpsSnapshot, error) {
	if storageCostPerGBMonth <= 0 {
		storageCostPerGBMonth = 0.015
	}

	var snapshot FinOpsSnapshot

	err := db.pool.QueryRow(ctx, `
		WITH storage AS (
			SELECT
				(SELECT COALESCE(SUM(file_size), 0) FROM track_versions WHERE deleted_at IS NULL) AS track_version_bytes,
				(SELECT COALESCE(SUM(file_size), 0) FROM images) AS image_bytes,
				(SELECT COALESCE(SUM(file_size_bytes), 0) FROM offline_downloads WHERE status = 'completed' AND deleted_at IS NULL) AS offline_download_bytes
		), jobs AS (
			SELECT
				COALESCE(COUNT(*) FILTER (WHERE job_type = 'cleanup_r2_file' AND status = 'pending'), 0) AS pending_cleanup_jobs,
				COALESCE(COUNT(*) FILTER (WHERE job_type = 'cleanup_r2_file' AND status = 'failed'), 0) AS failed_cleanup_jobs,
				COALESCE(SUM(CASE WHEN job_type = 'cleanup_r2_file' AND status = 'pending' THEN (data->>'file_size')::bigint ELSE 0 END), 0) AS pending_cleanup_bytes
			FROM background_jobs
		), spend AS (
			SELECT COALESCE(SUM(usd_amount), 0)::double precision AS actual_monthly_cost_usd
			FROM finops_cost_events
			WHERE occurred_at >= NOW() - INTERVAL '30 days'
		)
		SELECT
			s.track_version_bytes,
			s.image_bytes,
			s.offline_download_bytes,
			s.track_version_bytes + s.image_bytes + s.offline_download_bytes AS total_active_storage_bytes,
			j.pending_cleanup_jobs,
			j.failed_cleanup_jobs,
			j.pending_cleanup_bytes,
			sp.actual_monthly_cost_usd
		FROM storage s
		CROSS JOIN jobs j
		CROSS JOIN spend sp
	`).Scan(
		&snapshot.TrackVersionBytes,
		&snapshot.ImageBytes,
		&snapshot.OfflineDownloadBytes,
		&snapshot.TotalActiveStorageBytes,
		&snapshot.PendingCleanupJobs,
		&snapshot.FailedCleanupJobs,
		&snapshot.PendingCleanupBytes,
		&snapshot.ActualMonthlyCostUSD,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get FinOps snapshot: %w", err)
	}

	snapshot.EstimatedMonthlyCostUSD = (float64(snapshot.TotalActiveStorageBytes) / (1024 * 1024 * 1024)) * storageCostPerGBMonth
	if snapshot.ActualMonthlyCostUSD > snapshot.EstimatedMonthlyCostUSD {
		snapshot.GoverningMonthlyCostUSD = snapshot.ActualMonthlyCostUSD
	} else {
		snapshot.GoverningMonthlyCostUSD = snapshot.EstimatedMonthlyCostUSD
	}

	return &snapshot, nil
}

func (db *Database) CreateFinOpsCostEvent(ctx context.Context, event *FinOpsCostEvent) error {
	if event == nil {
		return fmt.Errorf("cost event is required")
	}

	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}

	if event.Source == "" {
		event.Source = "manual"
	}

	if event.Service == "" {
		event.Service = "r2"
	}

	if event.Category == "" {
		event.Category = "storage"
	}

	metadata := event.Metadata
	if metadata == nil {
		metadata = map[string]interface{}{}
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal FinOps metadata: %w", err)
	}

	err = db.pool.QueryRow(ctx,
		`INSERT INTO finops_cost_events (source, service, category, usd_amount, occurred_at, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		event.Source,
		event.Service,
		event.Category,
		event.AmountUSD,
		event.OccurredAt,
		metadataJSON,
	).Scan(&event.ID)
	if err != nil {
		return fmt.Errorf("failed to create FinOps cost event: %w", err)
	}

	return nil
}

func (db *Database) GetFinOpsMonthlySummary(ctx context.Context, days int, storageCostPerGBMonth, budgetUSD float64) (*FinOpsMonthlySummary, error) {
	if days <= 0 {
		days = 30
	}

	if storageCostPerGBMonth <= 0 {
		storageCostPerGBMonth = 0.015
	}

	snapshot, err := db.GetFinOpsSnapshot(ctx, storageCostPerGBMonth)
	if err != nil {
		return nil, err
	}

	var actualCost float64
	err = db.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(usd_amount), 0)::double precision
		 FROM finops_cost_events
		 WHERE occurred_at >= NOW() - make_interval(days => $1)`,
		days,
	).Scan(&actualCost)
	if err != nil {
		return nil, fmt.Errorf("failed to compute FinOps monthly summary: %w", err)
	}

	governing := snapshot.GoverningMonthlyCostUSD
	if actualCost > governing {
		governing = actualCost
	}

	utilization := 0.0
	if budgetUSD > 0 {
		utilization = governing / budgetUSD
	}

	return &FinOpsMonthlySummary{
		Days:                days,
		ActualCostUSD:       actualCost,
		StorageEstimatedUSD: snapshot.EstimatedMonthlyCostUSD,
		GoverningCostUSD:    governing,
		BudgetUSD:           budgetUSD,
		UtilizationRatio:    utilization,
	}, nil
}

func (db *Database) GetOfflineDownload(ctx context.Context, userID, versionID string) (*models.OfflineDownload, error) {
	var od models.OfflineDownload
	err := db.pool.QueryRow(ctx,
		`SELECT id, user_id, track_version_id, track_id, project_id, file_size_bytes, r2_object_key,
		        status, title, artist_name, project_name, duration_seconds, created_at, updated_at
		 FROM offline_downloads
		 WHERE user_id = $1 AND track_version_id = $2
		 ORDER BY created_at DESC
		 LIMIT 1`,
		userID, versionID,
	).Scan(&od.ID, &od.UserID, &od.TrackVersionID, &od.TrackID, &od.ProjectID, &od.FileSizeBytes, &od.R2ObjectKey,
		&od.Status, &od.Title, &od.ArtistName, &od.ProjectName, &od.DurationSeconds, &od.CreatedAt, &od.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get offline download: %w", err)
	}
	return &od, nil
}

func (db *Database) GetOfflineDownloadByID(ctx context.Context, downloadID string) (*models.OfflineDownload, error) {
	var od models.OfflineDownload
	err := db.pool.QueryRow(ctx,
		`SELECT id, user_id, track_version_id, track_id, project_id, file_size_bytes, r2_object_key,
		        status, title, artist_name, project_name, duration_seconds, created_at, updated_at
		 FROM offline_downloads
		 WHERE id = $1`,
		downloadID,
	).Scan(&od.ID, &od.UserID, &od.TrackVersionID, &od.TrackID, &od.ProjectID, &od.FileSizeBytes, &od.R2ObjectKey,
		&od.Status, &od.Title, &od.ArtistName, &od.ProjectName, &od.DurationSeconds, &od.CreatedAt, &od.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get offline download by id: %w", err)
	}
	return &od, nil
}

func (db *Database) CreateOfflineDownload(ctx context.Context, od *models.OfflineDownload) error {
	err := db.pool.QueryRow(ctx,
		`INSERT INTO offline_downloads (id, user_id, track_version_id, track_id, project_id, file_size_bytes, r2_object_key,
		                             status, title, artist_name, project_name, duration_seconds, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		 RETURNING id`,
		od.ID, od.UserID, od.TrackVersionID, od.TrackID, od.ProjectID, od.FileSizeBytes, od.R2ObjectKey,
		od.Status, od.Title, od.ArtistName, od.ProjectName, od.DurationSeconds, od.CreatedAt, od.UpdatedAt,
	).Scan(&od.ID)
	if err != nil {
		return fmt.Errorf("failed to create offline download: %w", err)
	}
	return nil
}

func (db *Database) UpdateOfflineDownloadStatus(ctx context.Context, downloadID, status string, metadata map[string]interface{}) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE offline_downloads
		 SET status = $1, updated_at = NOW()
		 WHERE id = $2`,
		status, downloadID,
	)
	if err != nil {
		return fmt.Errorf("failed to update offline download status: %w", err)
	}
	return nil
}

func (db *Database) GetImageByID(ctx context.Context, imageID string) (*models.Image, error) {
	var image models.Image
	err := db.pool.QueryRow(ctx,
		`SELECT id, user_id, r2_object_key, mime_type, file_size, alt_text, created_at
		 FROM images
		 WHERE id = $1`,
		imageID,
	).Scan(&image.ID, &image.UserID, &image.R2ObjectKey, &image.MimeType, &image.FileSize, &image.AltText, &image.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	}
	return &image, nil
}

func (db *Database) GetOfflineDownloadCount(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := db.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM offline_downloads WHERE user_id = $1 AND status = 'completed'`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count offline downloads: %w", err)
	}
	return count, nil
}

func (db *Database) UpdateOfflineLastPlayed(ctx context.Context, downloadID string) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE offline_downloads SET last_played_at = NOW(), updated_at = NOW() WHERE id = $1`,
		downloadID,
	)
	if err != nil {
		return fmt.Errorf("failed to update offline last played: %w", err)
	}
	return nil
}
