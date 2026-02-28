-- Migration: Schema updates for MEDIUM priority features [MEDIUM: Code health, missing functionality]

-- ============================================================================
-- Add missing indexes for logging and compliance
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_audit_logs_action_resource ON audit_logs(action, resource);
CREATE INDEX IF NOT EXISTS idx_gdpr_requests_user_status ON gdpr_data_requests(user_id, status);
CREATE INDEX IF NOT EXISTS idx_user_consents_user_type ON user_consents(user_id, consent_type);

-- ============================================================================
-- Ensure soft delete columns exist everywhere needed
-- ============================================================================

ALTER TABLE comments ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE likes ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE project_likes ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE offline_downloads ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE share_links ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- ============================================================================
-- Add constraints for data integrity [MEDIUM: Cascade deletes]
-- ============================================================================

-- Ensure NOT NULL for critical fields
ALTER TABLE projects ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE tracks ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE track_versions ALTER COLUMN track_id SET NOT NULL;
ALTER TABLE play_history ALTER COLUMN project_id SET NOT NULL;

-- ============================================================================
-- Performance optimization for analytics queries
-- ============================================================================

-- Partial index for completed offline downloads
CREATE INDEX IF NOT EXISTS idx_offline_downloads_completed ON offline_downloads(user_id, created_at DESC)
WHERE status = 'completed' AND deleted_at IS NULL;

-- Partial index for active play history
CREATE INDEX IF NOT EXISTS idx_play_history_active ON play_history(project_id, listener_user_id)
;

-- Composite index for common filter patterns
CREATE INDEX IF NOT EXISTS idx_projects_user_private ON projects(user_id, is_private)
WHERE deleted_at IS NULL;

-- ============================================================================
-- Comments visibility for sensitive data
-- ============================================================================

-- Ensure comments have proper ownership tracking
ALTER TABLE comments ADD COLUMN IF NOT EXISTS user_id UUID;
DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1
		FROM pg_constraint
		WHERE conname = 'fk_comments_user'
			AND conrelid = 'public.comments'::regclass
	) THEN
		ALTER TABLE comments ADD CONSTRAINT fk_comments_user FOREIGN KEY (user_id) REFERENCES users(id);
	END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_comments_user_id ON comments(user_id);

-- ============================================================================
-- Ensure all critical tables have updated_at
-- ============================================================================

ALTER TABLE comments ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE likes ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- ============================================================================
-- Create indexes for search optimization [MEDIUM: Search functionality]
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_projects_name_trgm ON projects USING gin(name gin_trgm_ops)
WHERE deleted_at IS NULL AND is_private = false;

CREATE INDEX IF NOT EXISTS idx_tracks_name_trgm ON tracks USING gin(name gin_trgm_ops)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_username_trgm ON users USING gin(username gin_trgm_ops)
WHERE deleted_at IS NULL;

-- Note: Requires `CREATE EXTENSION IF NOT EXISTS pg_trgm;` to be run separately
