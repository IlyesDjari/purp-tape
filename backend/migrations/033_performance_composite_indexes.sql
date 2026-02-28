-- Migration: Add critical composite indexes for query performance [HIGH PRIORITY]
-- These indexes optimize the most frequent query patterns and reduce query cost
-- ============================================================================

-- ============================================================================
-- PROJECT FILTERING - HIGHEST PRIORITY
-- ============================================================================
-- Most common pattern: user wants their projects with metadata
CREATE INDEX IF NOT EXISTS idx_projects_user_deleted_meta ON projects(user_id, deleted_at) 
  INCLUDE (name, description, is_private, cover_image_id, created_at, updated_at);

-- For pagination queries with deleted_at check
CREATE INDEX IF NOT EXISTS idx_projects_deleted_user ON projects(deleted_at, user_id, updated_at DESC);

-- ============================================================================
-- TRACK QUERIES - HIGH PRIORITY  
-- ============================================================================
-- Get all tracks in project with soft-delete filtering
CREATE INDEX IF NOT EXISTS idx_tracks_project_deleted_created ON tracks(project_id, deleted_at, created_at DESC)
  INCLUDE (user_id, name, duration, updated_at);

-- For count queries
CREATE INDEX IF NOT EXISTS idx_tracks_project_deleted ON tracks(project_id, deleted_at);

-- ============================================================================
-- TRACK VERSIONS - HIGH PRIORITY
-- ============================================================================
-- Get latest version of a track efficiently
CREATE INDEX IF NOT EXISTS idx_track_versions_track_deleted_version ON track_versions(track_id, deleted_at, version_number DESC)
  INCLUDE (r2_object_key, file_size, checksum);

-- For batch operations
CREATE INDEX IF NOT EXISTS idx_track_versions_deleted_created ON track_versions(deleted_at, created_at DESC);

-- ============================================================================
-- PROJECT SHARES - CRITICAL FOR ACCESS CONTROL
-- ============================================================================
-- Check who has access to a project (active shares only)
CREATE INDEX IF NOT EXISTS idx_project_shares_shared_with_active ON project_shares(shared_with_id, revoked_at)
  INCLUDE (project_id, created_at)
  WHERE revoked_at IS NULL;

-- Verify project sharing for permission checks
CREATE INDEX IF NOT EXISTS idx_project_shares_project_user_active ON project_shares(project_id, shared_with_id)
  INCLUDE (revoked_at)
  WHERE revoked_at IS NULL;

-- ============================================================================
-- OFFLINE DOWNLOADS - CRITICAL FOR STORAGE QUOTA
-- ============================================================================
-- Check user's stored files by status (completed downloads)
CREATE INDEX IF NOT EXISTS idx_offline_downloads_user_status_size ON offline_downloads(user_id, status)
  INCLUDE (file_size_bytes, downloaded_at, created_at)
  WHERE status = 'completed';

-- For cleanup: find old downloads
CREATE INDEX IF NOT EXISTS idx_offline_downloads_status_created ON offline_downloads(status, created_at DESC)
  WHERE status IN ('pending', 'failed');

-- ============================================================================
-- COLLABORATORS - PERMISSION CHECKS  
-- ============================================================================
-- Check if user is collaborator on project
CREATE INDEX IF NOT EXISTS idx_collaborators_project_user_active ON collaborators(project_id, user_id)
  INCLUDE (role)
;

-- Get all collaborators for a project
CREATE INDEX IF NOT EXISTS idx_collaborators_project ON collaborators(project_id)
  INCLUDE (user_id, role, invited_at);

-- ============================================================================
-- NOTIFICATIONS - USER INBOX PERFORMANCE
-- ============================================================================
-- Unread notifications for user (most common query)
CREATE INDEX IF NOT EXISTS idx_notifications_user_read_created ON notifications(user_id, is_read, created_at DESC)
  INCLUDE (type, project_id, track_id, comment_id, actor_user_id);

-- Bulk operations: mark as read
CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON notifications(user_id, created_at DESC);

-- ============================================================================
-- SHARE LINKS - PUBLIC SHARES
-- ============================================================================
-- Quick hash lookup + revocation check
CREATE INDEX IF NOT EXISTS idx_share_links_hash_revoked ON share_links(hash, revoked_at)
  INCLUDE (project_id, expires_at, created_at);

-- List shares by project
CREATE INDEX IF NOT EXISTS idx_share_links_project_active ON share_links(project_id, revoked_at)
  WHERE revoked_at IS NULL;

-- ============================================================================
-- COMMENTS - SOCIAL FEATURES
-- ============================================================================
-- Get comments on tracks (paginated)
CREATE INDEX IF NOT EXISTS idx_comments_track_version_created ON comments(track_version_id, created_at DESC)
  INCLUDE (user_id, content);

-- ============================================================================
-- PLAY HISTORY - ANALYTICS
-- ============================================================================
-- Get user's recent plays (for statistics)
CREATE INDEX IF NOT EXISTS idx_play_history_user_started ON play_history(listener_user_id, started_at DESC)
  INCLUDE (project_id, track_id, duration_listened);

-- Get project stats by date range
CREATE INDEX IF NOT EXISTS idx_play_history_project_started ON play_history(project_id, started_at DESC)
  INCLUDE (listener_user_id, duration_listened);

-- Recent plays for trending
CREATE INDEX IF NOT EXISTS idx_play_history_recent ON play_history(project_id)
  INCLUDE (listener_user_id, started_at);

-- ============================================================================
-- AUDIT LOGS - COMPLIANCE
-- ============================================================================
-- Get audit logs for user
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_created ON audit_logs(user_id, created_at DESC)
  INCLUDE (action, resource, resource_id);

-- Get logs for specific resource
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_created ON audit_logs(resource_id, created_at DESC)
  INCLUDE (action, user_id);

-- ============================================================================
-- BACKGROUND JOBS - TASK PROCESSING
-- ============================================================================
-- Get pending jobs efficiently  
CREATE INDEX IF NOT EXISTS idx_background_jobs_status_created ON background_jobs(status, created_at ASC)
  INCLUDE (job_type)
  WHERE status IN ('pending', 'processing');

-- Cleanup completed jobs
CREATE INDEX IF NOT EXISTS idx_background_jobs_status_completed ON background_jobs(status, completed_at DESC)
  WHERE status = 'completed';

-- ============================================================================
-- SUBSCRIPTIONS - BILLING & QUOTAS
-- ============================================================================
-- Quick user subscription lookup
CREATE INDEX IF NOT EXISTS idx_subscriptions_user ON subscriptions(user_id)
  INCLUDE (is_premium, tier, storage_quota_mb, created_at);

-- For billing operations
CREATE INDEX IF NOT EXISTS idx_subscriptions_stripe ON subscriptions(stripe_customer_id)
  INCLUDE (user_id, is_premium);

-- ============================================================================
-- PERFORMANCE: STATISTICS & VIEWS
-- ============================================================================
-- Quick counts for dashboards
CREATE OR REPLACE VIEW user_project_counts AS
SELECT 
  u.id as user_id,
  COUNT(DISTINCT p.id) as owned_projects,
  COUNT(DISTINCT ps.project_id) as shared_projects,
  COUNT(DISTINCT c.id) as collaborated_projects
FROM users u
LEFT JOIN projects p ON u.id = p.user_id AND p.deleted_at IS NULL
LEFT JOIN project_shares ps ON u.id = ps.shared_with_id AND ps.revoked_at IS NULL
LEFT JOIN collaborators c ON u.id = c.user_id
GROUP BY u.id;

-- Quick storage stats
CREATE OR REPLACE VIEW user_storage_usage AS
SELECT 
  u.id as user_id,
  COALESCE(SUM(od.file_size_bytes), 0) as offline_storage_bytes,
  COALESCE(COUNT(DISTINCT od.id), 0) as offline_downloads,
  s.storage_quota_mb * 1024 * 1024 as total_quota_bytes
FROM users u
LEFT JOIN offline_downloads od ON u.id = od.user_id AND od.status = 'completed'
LEFT JOIN subscriptions s ON u.id = s.user_id
GROUP BY u.id, s.storage_quota_mb;
