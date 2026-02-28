-- ============================================
-- CRITICAL PERFORMANCE INDEXES - TIER 1
-- ============================================
-- These indexes dramatically improve query performance for soft-delete filtering
-- and common access patterns identified in the code audit.

-- HIGH IMPACT: Fixes soft-delete filtering on projects
CREATE INDEX IF NOT EXISTS idx_projects_user_deleted_updated 
ON projects(user_id, deleted_at, updated_at DESC)
INCLUDE (name, description, is_private, cover_image_id);

-- HIGH IMPACT: Fixes soft-delete filtering on tracks
CREATE INDEX IF NOT EXISTS idx_tracks_project_deleted_created 
ON tracks(project_id, deleted_at, created_at DESC)
INCLUDE (name, user_id, duration);

-- HIGH IMPACT: Fixes soft-delete filtering on track_versions
CREATE INDEX IF NOT EXISTS idx_track_versions_track_deleted 
ON track_versions(track_id, deleted_at, version_number DESC)
INCLUDE (r2_object_key, file_size, checksum);

-- HIGH IMPACT: Fixes sharing queries for access control
CREATE INDEX IF NOT EXISTS idx_project_shares_recipient_active 
ON project_shares(shared_with_id, revoked_at, expires_at DESC)
WHERE revoked_at IS NULL;

-- HIGH IMPACT: Fixes offline download tracking queries
CREATE INDEX IF NOT EXISTS idx_offline_downloads_user_status_date 
ON offline_downloads(user_id, status, downloaded_at DESC)
WHERE status = 'completed';

-- HIGH IMPACT: Fixes play history analytics queries
CREATE INDEX IF NOT EXISTS idx_play_history_project_date 
ON play_history(project_id, started_at DESC)
INCLUDE (listener_user_id, duration_listened, device);

-- MEDIUM IMPACT: Fixes collaborator role lookups
CREATE INDEX IF NOT EXISTS idx_collaborators_project_user_active 
ON collaborators(project_id, user_id)
;

-- MEDIUM IMPACT: Fixes comment filtering
CREATE INDEX IF NOT EXISTS idx_comments_project_deleted 
ON comments(track_version_id, deleted_at)
WHERE deleted_at IS NULL;

-- MEDIUM IMPACT: Fixes likes queries
CREATE INDEX IF NOT EXISTS idx_likes_track_deleted 
ON likes(track_id, deleted_at)
WHERE deleted_at IS NULL;

-- ============================================
-- QUERY ANALYSIS HELPER
-- ============================================
-- To verify index effectiveness, run:
--   SELECT 
--     schemaname, tablename, indexname, idx_scan as scans, 
--     idx_tup_read as tuples_read, idx_tup_fetch as tuples_fetched
--   FROM pg_stat_user_indexes
--   ORDER BY idx_scan DESC;
--
-- For slow queries, check explain plans:
--   EXPLAIN ANALYZE SELECT ... ;
-- ============================================
