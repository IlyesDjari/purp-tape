-- Migration 036: Add critical composite indexes for query optimization
-- These indexes dramatically improve performance for common access patterns

-- ============================================================================
-- TRACKS - PROJECT OWNERSHIP & FILTERING
-- ============================================================================

-- Fast lookup when filtering tracks by project and optionally by user
CREATE INDEX IF NOT EXISTS idx_tracks_project_user ON tracks(project_id, user_id) 
WHERE deleted_at IS NULL;

-- For sorting tracks within a project by creation date
CREATE INDEX IF NOT EXISTS idx_tracks_project_created ON tracks(project_id, created_at DESC) 
WHERE deleted_at IS NULL;

-- ============================================================================
-- TRACK VERSIONS - VERSION HISTORY & CHECKSUMS
-- ============================================================================

-- Fast retrieval of all versions for a track with proper ordering
CREATE INDEX IF NOT EXISTS idx_track_versions_track_created ON track_versions(track_id, created_at DESC) 
WHERE deleted_at IS NULL;

-- For deduplication and integrity checks
CREATE INDEX IF NOT EXISTS idx_track_versions_checksum ON track_versions(checksum) 
WHERE deleted_at IS NULL;

-- ============================================================================
-- OFFLINE DOWNLOADS - USER STATUS & PROJECT TRACKING
-- ============================================================================

-- Critical for querying user's completed downloads
CREATE INDEX IF NOT EXISTS idx_offline_downloads_project_status ON offline_downloads(project_id, status) 
WHERE status = 'completed';

-- Fast lookup by user and status (storage quota checks, active downloads)
CREATE INDEX IF NOT EXISTS idx_offline_downloads_user_status ON offline_downloads(user_id, status);

-- ============================================================================
-- PROJECT SHARES - REVOCATION & EXPIRATION CHECKS
-- ============================================================================

-- For RLS policy evaluation: check active (non-revoked) shares
CREATE INDEX IF NOT EXISTS idx_project_shares_project_revoked ON project_shares(project_id, revoked_at) 
WHERE revoked_at IS NULL;

-- For checking if shared with user and not expired
CREATE INDEX IF NOT EXISTS idx_project_shares_shared_revoked ON project_shares(shared_with_id, revoked_at) 
WHERE revoked_at IS NULL;

-- Support expiration checks without non-immutable predicates in partial indexes
CREATE INDEX IF NOT EXISTS idx_project_shares_expires_at ON project_shares(expires_at)
WHERE revoked_at IS NULL;

-- ============================================================================
-- COLLABORATORS - ROLE LOOKUPS & SOFT DELETION
-- ============================================================================

-- For RLS and RBAC: find active collaborators on projects
CREATE INDEX IF NOT EXISTS idx_collaborators_project_role ON collaborators(project_id, role) 
;

-- Fast role lookup for specific user on specific project
CREATE INDEX IF NOT EXISTS idx_collaborators_project_user_role ON collaborators(project_id, user_id, role) 
;

-- ============================================================================
-- LIKES - DUPLICATE PREVENTION & ANALYTICS
-- ============================================================================

-- Prevent duplicate likes: fast check if user already liked this track
CREATE INDEX IF NOT EXISTS idx_likes_track_user_unique ON likes(track_id, user_id);

-- For analytics: track likes over time
CREATE INDEX IF NOT EXISTS idx_likes_created_at ON likes(created_at DESC);

-- ============================================================================
-- COMMENTS - THREAD ORDERING & MODERATION
-- ============================================================================

-- Fast retrieval of comments for a specific track version, ordered by time
CREATE INDEX IF NOT EXISTS idx_comments_track_version_created ON comments(track_version_id, created_at DESC);

-- For moderator tooling: find recent comments quickly
CREATE INDEX IF NOT EXISTS idx_comments_created_recent ON comments(created_at DESC);

-- ============================================================================
-- NOTIFICATIONS - DELIVERY & UNREAD STATUS
-- ============================================================================

-- Critical for inbox: get unread notifications for user
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread ON notifications(user_id, is_read) 
WHERE is_read = false;

-- Timeline ordering: user's notification history
CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON notifications(user_id, created_at DESC);

-- ============================================================================
-- PLAY HISTORY - ANALYTICS & DEDUPLICATION
-- ============================================================================

-- For analytics queries: plays per project
CREATE INDEX IF NOT EXISTS idx_play_history_project_date ON play_history(project_id, started_at DESC);

-- For preventing duplicate plays (check if user played same track recently)
CREATE INDEX IF NOT EXISTS idx_play_history_track_user ON play_history(track_id, listener_user_id, started_at DESC);

-- ============================================================================
-- BACKGROUND JOBS - STATUS & SCHEDULING
-- ============================================================================

-- Critical for job processor: find pending/failed jobs
CREATE INDEX IF NOT EXISTS idx_background_jobs_status_created ON background_jobs(status, created_at ASC) 
WHERE status IN ('pending', 'retry');

-- For retry logic: find jobs that failed and are ready for retry
CREATE INDEX IF NOT EXISTS idx_background_jobs_retry_created ON background_jobs(created_at ASC) 
WHERE status = 'retry' AND retry_count < max_retries;

-- ============================================================================
-- AUDIT LOGS - COMPLIANCE & INVESTIGATION
-- ============================================================================

-- Fast lookup by action type for compliance reports
CREATE INDEX IF NOT EXISTS idx_audit_logs_action_created ON audit_logs(action, created_at DESC);

-- For user action history
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_created ON audit_logs(user_id, created_at DESC);

-- ANALYZE to update query planner statistics
ANALYZE;
