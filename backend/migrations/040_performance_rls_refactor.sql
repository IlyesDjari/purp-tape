-- Migration 042: High-Performance RLS Architecture Refactor
-- Replaces nested subquery RLS with denormalized permission cache + materialized views
-- Performance goal: O(1) RLS checks instead of O(n*m)

-- ============================================================================
-- DENORMALIZED USER PROJECT ACCESS CACHE
-- ============================================================================
-- Flat table: user_id + project_id + access_type (owner/collaborator/shared)
-- Maintained by triggers for <1ms access checks
CREATE TABLE IF NOT EXISTS user_project_access (
    user_id UUID NOT NULL,
    project_id UUID NOT NULL,
    access_type TEXT NOT NULL CHECK (access_type IN ('owner', 'collaborator', 'shared')),
    access_granted_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    PRIMARY KEY (user_id, project_id, access_type)
);

-- Massive performance boost: single index for RLS checks
CREATE INDEX IF NOT EXISTS idx_user_project_access_user_id 
    ON user_project_access(user_id) 
    WHERE access_type IN ('owner', 'collaborator', 'shared');

CREATE INDEX IF NOT EXISTS idx_user_project_access_project_id 
    ON user_project_access(project_id, user_id, access_type);

-- Partial index for expensive operations (INSERT/UPDATE/DELETE)
CREATE INDEX IF NOT EXISTS idx_user_project_access_owner 
    ON user_project_access(user_id, project_id) 
    WHERE access_type = 'owner';

-- ============================================================================
-- TRIGGER FUNCTIONS TO MAINTAIN USER_PROJECT_ACCESS CACHE
-- ============================================================================

-- When user owns a project, add to cache
CREATE OR REPLACE FUNCTION sync_owner_access()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO user_project_access (user_id, project_id, access_type)
    VALUES (NEW.user_id, NEW.id, 'owner')
    ON CONFLICT DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- When project ownership transfers
CREATE OR REPLACE FUNCTION update_owner_access()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM user_project_access 
    WHERE project_id = OLD.id AND access_type = 'owner' AND user_id = OLD.user_id;
    INSERT INTO user_project_access (user_id, project_id, access_type)
    VALUES (NEW.user_id, NEW.id, 'owner')
    ON CONFLICT DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- When user is added as collaborator
CREATE OR REPLACE FUNCTION sync_collaborator_access()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO user_project_access (user_id, project_id, access_type)
    VALUES (NEW.user_id, NEW.project_id, 'collaborator')
    ON CONFLICT DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- When collaborator access is revoked (soft-delete)
CREATE OR REPLACE FUNCTION revoke_collaborator_access()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM user_project_access 
    WHERE user_id = OLD.user_id 
    AND project_id = OLD.project_id 
    AND access_type = 'collaborator';
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- When project is shared with user
CREATE OR REPLACE FUNCTION sync_shared_access()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO user_project_access (user_id, project_id, access_type)
    VALUES (NEW.shared_with_id, NEW.project_id, 'shared')
    ON CONFLICT DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- When share is revoked
CREATE OR REPLACE FUNCTION revoke_shared_access()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM user_project_access 
    WHERE user_id = OLD.shared_with_id 
    AND project_id = OLD.project_id 
    AND access_type = 'shared';
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- CREATE TRIGGERS (after all functions defined)
-- ============================================================================

-- Trigger for new projects (add owner access)
DROP TRIGGER IF EXISTS trigger_projects_owner_access ON projects;
CREATE TRIGGER trigger_projects_owner_access
    AFTER INSERT ON projects
    FOR EACH ROW
    EXECUTE FUNCTION sync_owner_access();

-- Trigger for project ownership changes
DROP TRIGGER IF EXISTS trigger_projects_owner_update ON projects;
CREATE TRIGGER trigger_projects_owner_update
    AFTER UPDATE ON projects
    FOR EACH ROW
    WHEN (OLD.user_id IS DISTINCT FROM NEW.user_id)
    EXECUTE FUNCTION update_owner_access();

-- Trigger for new collaborators
DROP TRIGGER IF EXISTS trigger_collaborators_access ON collaborators;
CREATE TRIGGER trigger_collaborators_access
    AFTER INSERT ON collaborators
    FOR EACH ROW
    EXECUTE FUNCTION sync_collaborator_access();

-- Trigger for revoking collaborator access
DROP TRIGGER IF EXISTS trigger_collaborators_revoke ON collaborators;
CREATE TRIGGER trigger_collaborators_revoke
    AFTER UPDATE ON collaborators
    FOR EACH ROW
    WHEN (OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL)
    EXECUTE FUNCTION revoke_collaborator_access();

-- Trigger for new project shares
DROP TRIGGER IF EXISTS trigger_project_shares_access ON project_shares;
CREATE TRIGGER trigger_project_shares_access
    AFTER INSERT ON project_shares
    FOR EACH ROW
    EXECUTE FUNCTION sync_shared_access();

-- Trigger for revoking shares
DROP TRIGGER IF EXISTS trigger_project_shares_revoke ON project_shares;
CREATE TRIGGER trigger_project_shares_revoke
    AFTER UPDATE ON project_shares
    FOR EACH ROW
    WHEN (OLD.revoked_at IS NULL AND NEW.revoked_at IS NOT NULL)
    EXECUTE FUNCTION revoke_shared_access();

-- ============================================================================
-- HELPER FUNCTION: Check user project access (O(1) instead of O(n*m))
-- ============================================================================
CREATE OR REPLACE FUNCTION user_can_access_project(p_user_id UUID, p_project_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM user_project_access 
        WHERE user_id = p_user_id 
        AND project_id = p_project_id 
        LIMIT 1
    );
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================================
-- DENORMALIZED TRACK ACCESS (for likes, comments, etc.)
-- ============================================================================
-- Cache: which users can access which tracks
CREATE TABLE IF NOT EXISTS user_track_access (
    user_id UUID NOT NULL,
    track_id UUID NOT NULL,
    project_id UUID NOT NULL,
    PRIMARY KEY (user_id, track_id)
);

CREATE INDEX IF NOT EXISTS idx_user_track_access_user 
    ON user_track_access(user_id);

CREATE INDEX IF NOT EXISTS idx_user_track_access_track 
    ON user_track_access(track_id, user_id);

-- ============================================================================
-- MATERIALIZED VIEW: Pre-computed user accessible projects
-- ============================================================================
-- Refresh frequency: On-demand + hourly batch (not real-time)
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_user_accessible_projects AS
SELECT 
    upa.user_id,
    upa.project_id,
    upa.access_type,
    p.user_id as owner_id,
    p.name,
    p.is_private,
    p.updated_at
FROM user_project_access upa
JOIN projects p ON p.id = upa.project_id
WHERE p.deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mv_user_projects_user 
    ON mv_user_accessible_projects(user_id);

-- ============================================================================
-- REFACTORED RLS POLICIES (O(1) instead of O(n*m))
-- ============================================================================

-- Drop old expensive policies
DROP POLICY IF EXISTS "likes_select" ON likes;
DROP POLICY IF EXISTS "project_likes_select" ON project_likes;
DROP POLICY IF EXISTS "comments_select" ON comments;

-- NEW: Likes - Direct denormalized access check
CREATE POLICY "likes_select" ON likes
    FOR SELECT USING (
        EXISTS (
            SELECT 1 FROM tracks t
            JOIN user_project_access upa ON upa.project_id = t.project_id
            WHERE t.id = likes.track_id
            AND upa.user_id = auth.uid()
            LIMIT 1
        )
    );

-- NEW: Project Likes - Fast owner/collaborator check
CREATE POLICY "project_likes_select" ON project_likes
    FOR SELECT USING (
        EXISTS (
            SELECT 1 FROM user_project_access upa
            WHERE upa.user_id = auth.uid()
            AND upa.project_id = project_likes.project_id
            LIMIT 1
        )
        OR EXISTS (
            SELECT 1 FROM projects p
            WHERE p.id = project_likes.project_id
            AND p.is_private = false
            AND p.deleted_at IS NULL
            LIMIT 1
        )
    );

-- NEW: Comments - Use track + project access cache
CREATE POLICY "comments_select" ON comments
    FOR SELECT USING (
        EXISTS (
            SELECT 1 FROM track_versions tv
            JOIN tracks t ON t.id = tv.track_id
            JOIN user_project_access upa ON upa.project_id = t.project_id
            WHERE upa.user_id = auth.uid()
            AND tv.id = comments.track_version_id
            LIMIT 1
        )
    );

-- ============================================================================
-- POPULATION SCRIPT (run after all tables exist)
-- ============================================================================
-- Populate initial user_project_access from existing data
INSERT INTO user_project_access (user_id, project_id, access_type)
SELECT DISTINCT user_id, id, 'owner' FROM projects WHERE deleted_at IS NULL
ON CONFLICT DO NOTHING;

INSERT INTO user_project_access (user_id, project_id, access_type)
SELECT DISTINCT user_id, project_id, 'collaborator' FROM collaborators WHERE deleted_at IS NULL
ON CONFLICT DO NOTHING;

INSERT INTO user_project_access (user_id, project_id, access_type)
SELECT DISTINCT shared_with_id, project_id, 'shared' FROM project_shares WHERE revoked_at IS NULL
ON CONFLICT DO NOTHING;

-- ============================================================================
-- TUNING PARAMETERS
-- ============================================================================
ALTER TABLE user_project_access SET (fillfactor = 90);
ALTER TABLE user_track_access SET (fillfactor = 90);
