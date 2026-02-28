-- Migration: Add soft delete support with deleted_at columns
-- Soft deletes allow data recovery and maintain audit trails

-- ============================================================================
-- USERS TABLE - Add soft delete
-- ============================================================================

ALTER TABLE users ADD COLUMN deleted_at TIMESTAMP;

CREATE INDEX idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NOT NULL;

-- Update RLS policy to hide deleted users
CREATE POLICY "users_hide_deleted" ON users
  FOR SELECT USING (deleted_at IS NULL);

-- ============================================================================
-- PROJECTS TABLE - Add soft delete
-- ============================================================================

ALTER TABLE projects ADD COLUMN deleted_at TIMESTAMP;

CREATE INDEX idx_projects_deleted_at ON projects(deleted_at) WHERE deleted_at IS NOT NULL;

-- Update RLS policies to hide deleted projects
CREATE POLICY "projects_hide_deleted_own" ON projects
  FOR SELECT USING (deleted_at IS NULL AND auth.uid() = user_id);

CREATE POLICY "projects_hide_deleted_shared" ON projects
  FOR SELECT USING (
    deleted_at IS NULL AND
    id IN (
      SELECT project_id FROM project_shares 
      WHERE shared_with_id = auth.uid() AND revoked_at IS NULL
    )
  );

-- ============================================================================
-- TRACKS TABLE - Add soft delete
-- ============================================================================

ALTER TABLE tracks ADD COLUMN deleted_at TIMESTAMP;

CREATE INDEX idx_tracks_deleted_at ON tracks(deleted_at) WHERE deleted_at IS NOT NULL;

-- Update RLS policies to hide deleted tracks
CREATE POLICY "tracks_hide_deleted_own" ON tracks
  FOR SELECT USING (
    deleted_at IS NULL AND
    project_id IN (SELECT id FROM projects WHERE user_id = auth.uid() AND deleted_at IS NULL)
  );

CREATE POLICY "tracks_hide_deleted_shared" ON tracks
  FOR SELECT USING (
    deleted_at IS NULL AND
    project_id IN (
      SELECT project_id FROM project_shares 
      WHERE shared_with_id = auth.uid() AND revoked_at IS NULL
    )
  );

-- ============================================================================
-- TRACK_VERSIONS TABLE - Add soft delete
-- ============================================================================

ALTER TABLE track_versions ADD COLUMN deleted_at TIMESTAMP;

CREATE INDEX idx_track_versions_deleted_at ON track_versions(deleted_at);

-- ============================================================================
-- Create automatic delete log for audit trail
-- ============================================================================

CREATE TABLE IF NOT EXISTS deleted_items_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_name VARCHAR(100) NOT NULL,
    item_id UUID NOT NULL,
    deleted_by UUID NOT NULL,
    deleted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reason VARCHAR(500),
    restored_at TIMESTAMP,
    restored_by UUID
);

CREATE INDEX idx_deleted_items_log_table ON deleted_items_log(table_name, item_id);
CREATE INDEX idx_deleted_items_log_deleted_at ON deleted_items_log(deleted_at DESC);

-- ============================================================================
-- Trigger: Log deletions when projects are soft-deleted
-- ============================================================================

CREATE OR REPLACE FUNCTION log_project_deletion()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL THEN
        INSERT INTO deleted_items_log (table_name, item_id, deleted_by)
        VALUES ('projects', NEW.id, auth.uid());
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_log_project_deletion
AFTER UPDATE ON projects
FOR EACH ROW
EXECUTE FUNCTION log_project_deletion();

-- ============================================================================
-- Function: Restore soft-deleted project (with audit trail)
-- ============================================================================

CREATE OR REPLACE FUNCTION restore_deleted_project(project_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    UPDATE projects
    SET deleted_at = NULL
    WHERE id = project_id AND deleted_at IS NOT NULL;

    UPDATE deleted_items_log
    SET restored_at = CURRENT_TIMESTAMP, restored_by = auth.uid()
    WHERE table_name = 'projects' AND item_id = project_id AND restored_at IS NULL;

    RETURN true;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Function: Permanently delete soft-deleted data (for GDPR compliance)
-- Only allows deletion of user's own data
-- ============================================================================

CREATE OR REPLACE FUNCTION permanently_delete_project_data(project_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    -- Only owner can permanently delete
    IF NOT EXISTS (
        SELECT 1 FROM projects WHERE id = project_id AND user_id = auth.uid()
    ) THEN
        RAISE EXCEPTION 'Not authorized to delete this project';
    END IF;

    -- Delete track versions
    DELETE FROM track_versions
    WHERE track_id IN (SELECT id FROM tracks WHERE project_id = project_id);

    -- Delete tracks
    DELETE FROM tracks WHERE project_id = project_id;

    -- Delete project
    DELETE FROM projects WHERE id = project_id;

    RETURN true;
END;
$$ LANGUAGE plpgsql;
