-- Migration: Add missing database indexes for performance [HIGH PRIORITY]
-- These indexes significantly improve query performance for common operations

-- ============================================================================
-- PLAY_HISTORY INDEXES
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_play_history_track_id ON play_history(track_id);
CREATE INDEX IF NOT EXISTS idx_play_history_project_id ON play_history(project_id);
CREATE INDEX IF NOT EXISTS idx_play_history_started_at ON play_history(started_at DESC);

-- Composite index for analytics queries
CREATE INDEX IF NOT EXISTS idx_play_history_project_date ON play_history(project_id, started_at DESC);

-- Partial index for recent plays only (optimization)
CREATE INDEX IF NOT EXISTS idx_play_history_recent ON play_history(project_id, listener_user_id);

-- ============================================================================
-- LIKES INDEXES
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_project_likes_project_id ON project_likes(project_id);
CREATE INDEX IF NOT EXISTS idx_project_likes_user_id ON project_likes(user_id);
CREATE INDEX IF NOT EXISTS idx_track_likes_track_id ON likes(track_id);
CREATE INDEX IF NOT EXISTS idx_track_likes_user_id ON likes(user_id);

-- Composite for checking user's like on specific item
CREATE INDEX IF NOT EXISTS idx_track_likes_user_track ON likes(user_id, track_id);
CREATE INDEX IF NOT EXISTS idx_project_likes_user_project ON project_likes(user_id, project_id);

-- ============================================================================
-- COMMENTS INDEXES
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_comments_track_version_id ON comments(track_version_id);
CREATE INDEX IF NOT EXISTS idx_comments_user_id ON comments(user_id);
CREATE INDEX IF NOT EXISTS idx_comments_created_at ON comments(created_at DESC);

-- Composite for fast comment filtering
CREATE INDEX IF NOT EXISTS idx_comments_track_version_created ON comments(track_version_id, created_at DESC);

-- ============================================================================
-- NOTIFICATIONS INDEXES
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_is_read ON notifications(is_read);

-- Composite for unread notifications query
CREATE INDEX IF NOT EXISTS idx_notifications_user_read ON notifications(user_id, is_read);
CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON notifications(user_id, created_at DESC);

-- ============================================================================
-- COLLABORATORS INDEXES
-- ============================================================================

-- Partial index for active collaborators (not deleted)
CREATE INDEX IF NOT EXISTS idx_collaborators_project_role ON collaborators(project_id, role) 
;

-- For checking specific collaborator access
CREATE INDEX IF NOT EXISTS idx_collaborators_project_user ON collaborators(project_id, user_id);

-- ============================================================================
-- OFFLINE DOWNLOADS INDEXES
-- ============================================================================

-- Composite for user's completed downloads
CREATE INDEX IF NOT EXISTS idx_offline_downloads_user_status ON offline_downloads(user_id, status) 
WHERE status = 'completed';

-- For efficient deletion cleanup
CREATE INDEX IF NOT EXISTS idx_offline_downloads_track_version ON offline_downloads(track_version_id);

-- ============================================================================
-- PROJECT SHARES INDEXES
-- ============================================================================

-- For checking shared access
CREATE INDEX IF NOT EXISTS idx_project_shares_shared_with ON project_shares(shared_with_id, revoked_at);

-- For active (non-revoked) shares
CREATE INDEX IF NOT EXISTS idx_project_shares_active ON project_shares(project_id, shared_with_id) 
WHERE revoked_at IS NULL;

-- ============================================================================
-- FOLLOW INDEXES
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_user_follows_follower_id ON user_follows(follower_id);
CREATE INDEX IF NOT EXISTS idx_user_follows_following_id ON user_follows(following_id);

-- ============================================================================
-- BACKGROUND JOBS INDEXES
-- ============================================================================

-- Critical for job processor
CREATE INDEX IF NOT EXISTS idx_background_jobs_status ON background_jobs(status) 
WHERE status IN ('pending', 'processing');

CREATE INDEX IF NOT EXISTS idx_background_jobs_job_type ON background_jobs(job_type);
CREATE INDEX IF NOT EXISTS idx_background_jobs_created_at ON background_jobs(created_at DESC);

-- ============================================================================
-- SOFT DELETE INDEXES
-- ============================================================================

-- For efficient soft-delete filtering
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_projects_deleted_at ON projects(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tracks_deleted_at ON tracks(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_track_versions_deleted_at ON track_versions(deleted_at) WHERE deleted_at IS NOT NULL;

-- ============================================================================
-- PERFORMANCE VERIFICATION
-- ============================================================================

-- After migration, run these to verify indexes are used:
-- EXPLAIN ANALYZE SELECT * FROM play_history WHERE project_id = 'xyz' AND started_at > NOW() - INTERVAL '30 days';
-- EXPLAIN ANALYZE SELECT * FROM comments WHERE track_id = 'xyz' ORDER BY created_at DESC LIMIT 20;
-- EXPLAIN ANALYZE SELECT * FROM notifications WHERE user_id = 'xyz' AND is_read = false;
-- EXPLAIN ANALYZE SELECT * FROM offline_downloads WHERE user_id = 'xyz' AND status = 'completed';
