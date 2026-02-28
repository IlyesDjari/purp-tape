-- Migration: Add missing RLS policies to collaborators, images, comments, and play_history tables
-- This is CRITICAL for data security - prevents unauthorized access to sensitive data

-- ============================================================================
-- COLLABORATORS TABLE - Add comprehensive RLS policies [CRITICAL FIX]
-- ============================================================================

ALTER TABLE collaborators ENABLE ROW LEVEL SECURITY;

-- Only project owner can view/manage collaborators on their project
CREATE POLICY "collaborators_select_own_project" ON collaborators
  FOR SELECT USING (
    project_id IN (SELECT id FROM projects WHERE user_id = auth.uid())
  );

-- Collaborators can view their own role
CREATE POLICY "collaborators_select_self" ON collaborators
  FOR SELECT USING (auth.uid() = user_id);

-- Only project owner can add/update collaborators
CREATE POLICY "collaborators_insert_own_project" ON collaborators
  FOR INSERT WITH CHECK (
    project_id IN (SELECT id FROM projects WHERE user_id = auth.uid())
  );

CREATE POLICY "collaborators_update_own_project" ON collaborators
  FOR UPDATE USING (
    project_id IN (SELECT id FROM projects WHERE user_id = auth.uid())
  );

CREATE POLICY "collaborators_delete_own_project" ON collaborators
  FOR DELETE USING (
    project_id IN (SELECT id FROM projects WHERE user_id = auth.uid())
  );

-- ============================================================================
-- IMAGES TABLE - Add RLS policies [CRITICAL FIX]
-- ============================================================================

ALTER TABLE images ENABLE ROW LEVEL SECURITY;

-- Users can only view their own images
CREATE POLICY "images_select_own" ON images
  FOR SELECT USING (auth.uid() = user_id);

-- Users can only insert their own images
CREATE POLICY "images_insert_own" ON images
  FOR INSERT WITH CHECK (auth.uid() = user_id);

-- Users can only update their own images
CREATE POLICY "images_update_own" ON images
  FOR UPDATE USING (auth.uid() = user_id);

-- Users can only delete their own images
CREATE POLICY "images_delete_own" ON images
  FOR DELETE USING (auth.uid() = user_id);

-- ============================================================================
-- COMMENTS TABLE - Add RLS policies [CRITICAL FIX]
-- ============================================================================

ALTER TABLE comments ENABLE ROW LEVEL SECURITY;

-- Users can read comments on projects they have access to
CREATE POLICY "comments_select" ON comments
  FOR SELECT USING (
    track_version_id IN (
      SELECT tv.id
      FROM track_versions tv
      JOIN tracks t ON t.id = tv.track_id
      WHERE t.project_id IN (
        SELECT id FROM projects WHERE user_id = auth.uid()
        UNION
        SELECT project_id FROM project_shares WHERE shared_with_id = auth.uid() AND revoked_at IS NULL
        UNION
        SELECT project_id FROM collaborators WHERE user_id = auth.uid()
      )
    )
  );

-- Users can insert comments on projects they have access to
CREATE POLICY "comments_insert" ON comments
  FOR INSERT WITH CHECK (
    auth.uid() = user_id AND
    track_version_id IN (
      SELECT tv.id
      FROM track_versions tv
      JOIN tracks t ON t.id = tv.track_id
      WHERE t.project_id IN (
        SELECT id FROM projects WHERE user_id = auth.uid()
        UNION
        SELECT project_id FROM project_shares WHERE shared_with_id = auth.uid() AND revoked_at IS NULL
        UNION
        SELECT project_id FROM collaborators WHERE user_id = auth.uid()
      )
    )
  );

-- Users can only delete their own comments
CREATE POLICY "comments_delete_own" ON comments
  FOR DELETE USING (auth.uid() = user_id);

-- Users can update their own comments
CREATE POLICY "comments_update_own" ON comments
  FOR UPDATE USING (auth.uid() = user_id);

-- ============================================================================
-- PLAY_HISTORY TABLE - Add RLS policies [CRITICAL FIX]
-- ============================================================================

ALTER TABLE play_history ENABLE ROW LEVEL SECURITY;

-- Only backend/system can INSERT play history (via service role or trigger)
-- Users should NOT insert directly - this is logged by the system
-- Prevent users from creating fake play history
CREATE POLICY "play_history_deny_user_insert" ON play_history
  FOR INSERT WITH CHECK (false);  -- Deny all user inserts

-- Users can only view plays on their own projects
CREATE POLICY "play_history_select_own_project" ON play_history
  FOR SELECT USING (
    project_id IN (SELECT id FROM projects WHERE user_id = auth.uid())
  );

-- ============================================================================
-- LIKES TABLE - Ensure RLS is enabled and configured
-- ============================================================================

ALTER TABLE likes ENABLE ROW LEVEL SECURITY;

-- Users can view all likes (public data)
CREATE POLICY "likes_select_all" ON likes
  FOR SELECT USING (true);

-- Users can only create their own likes
CREATE POLICY "likes_insert_own" ON likes
  FOR INSERT WITH CHECK (auth.uid() = user_id);

-- Users can only delete their own likes
CREATE POLICY "likes_delete_own" ON likes
  FOR DELETE USING (auth.uid() = user_id);

-- ============================================================================
-- PROJECT_LIKES TABLE - Ensure RLS is enabled
-- ============================================================================

ALTER TABLE project_likes ENABLE ROW LEVEL SECURITY;

-- Users can view all project likes (public data)
CREATE POLICY "project_likes_select_all" ON project_likes
  FOR SELECT USING (true);

-- Users can only create their own likes
CREATE POLICY "project_likes_insert_own" ON project_likes
  FOR INSERT WITH CHECK (auth.uid() = user_id);

-- Users can only delete their own likes
CREATE POLICY "project_likes_delete_own" ON project_likes
  FOR DELETE USING (auth.uid() = user_id);

-- ============================================================================
-- NOTIFICATIONS TABLE - Ensure RLS is enabled
-- ============================================================================

ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;

-- Users can only view their own notifications
CREATE POLICY "notifications_select_own" ON notifications
  FOR SELECT USING (auth.uid() = user_id);

-- Only system can insert notifications (via triggers)
CREATE POLICY "notifications_insert_system" ON notifications
  FOR INSERT WITH CHECK (false);  -- Prevent direct user inserts

-- Users can only update their own notifications (mark as read)
CREATE POLICY "notifications_update_own" ON notifications
  FOR UPDATE USING (auth.uid() = user_id);

-- Users can delete their own notifications
CREATE POLICY "notifications_delete_own" ON notifications
  FOR DELETE USING (auth.uid() = user_id);
