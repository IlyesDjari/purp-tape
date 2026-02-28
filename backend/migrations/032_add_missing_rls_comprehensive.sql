-- Migration: Add missing RLS policies for likes, follows, and play history [CRITICAL SECURITY]
-- ============================================================================

DROP POLICY IF EXISTS "likes_insert_own" ON likes;
DROP POLICY IF EXISTS "likes_delete_own" ON likes;
DROP POLICY IF EXISTS "likes_select" ON likes;
DROP POLICY IF EXISTS "likes_update_own" ON likes;

DROP POLICY IF EXISTS "project_likes_insert_own" ON project_likes;
DROP POLICY IF EXISTS "project_likes_delete_own" ON project_likes;
DROP POLICY IF EXISTS "project_likes_select" ON project_likes;
DROP POLICY IF EXISTS "project_likes_update_own" ON project_likes;

DROP POLICY IF EXISTS "user_follows_insert_own" ON user_follows;
DROP POLICY IF EXISTS "user_follows_delete_own" ON user_follows;
DROP POLICY IF EXISTS "user_follows_select_public" ON user_follows;
DROP POLICY IF EXISTS "user_follows_update_own" ON user_follows;

DROP POLICY IF EXISTS "play_history_insert_own" ON play_history;
DROP POLICY IF EXISTS "play_history_select_own" ON play_history;
DROP POLICY IF EXISTS "play_history_select_project_owner" ON play_history;
DROP POLICY IF EXISTS "play_history_update_own" ON play_history;
DROP POLICY IF EXISTS "play_history_delete_own" ON play_history;

-- ============================================================================
-- LIKES TABLE - RLS Policies [CRITICAL FIX]
-- ============================================================================
ALTER TABLE likes ENABLE ROW LEVEL SECURITY;

-- Users can only insert their own likes
CREATE POLICY "likes_insert_own" ON likes
  FOR INSERT WITH CHECK (auth.uid() = user_id);

-- Users can only delete their own likes
CREATE POLICY "likes_delete_own" ON likes
  FOR DELETE USING (auth.uid() = user_id);

-- Users can only see likes on tracks they have access to
CREATE POLICY "likes_select" ON likes
  FOR SELECT USING (
    track_id IN (
      SELECT id FROM tracks 
      WHERE project_id IN (
        SELECT id FROM projects WHERE user_id = auth.uid() OR deleted_at IS NULL
        UNION
        SELECT project_id FROM project_shares WHERE shared_with_id = auth.uid() AND revoked_at IS NULL
        UNION
        SELECT project_id FROM collaborators WHERE user_id = auth.uid() AND deleted_at IS NULL
      )
    )
  );

-- ============================================================================
-- PROJECT_LIKES TABLE - RLS Policies [CRITICAL FIX]
-- ============================================================================
ALTER TABLE project_likes ENABLE ROW LEVEL SECURITY;

-- Users can only insert their own project likes
CREATE POLICY "project_likes_insert_own" ON project_likes
  FOR INSERT WITH CHECK (auth.uid() = user_id);

-- Users can only delete their own project likes
CREATE POLICY "project_likes_delete_own" ON project_likes
  FOR DELETE USING (auth.uid() = user_id);

-- Users can see likes on public projects or those they own
CREATE POLICY "project_likes_select" ON project_likes
  FOR SELECT USING (
    project_id IN (
      SELECT id FROM projects 
      WHERE user_id = auth.uid() 
      OR (is_private = false AND deleted_at IS NULL)
    )
  );

-- ============================================================================
-- LIKES UPDATE POLICIES (if needed)
-- ============================================================================
CREATE POLICY "likes_update_own" ON likes
  FOR UPDATE USING (auth.uid() = user_id);

CREATE POLICY "project_likes_update_own" ON project_likes
  FOR UPDATE USING (auth.uid() = user_id);

-- ============================================================================
-- USER_FOLLOWS TABLE - RLS Policies [CRITICAL FIX]
-- ============================================================================
ALTER TABLE user_follows ENABLE ROW LEVEL SECURITY;

-- Users can only insert their own follows
CREATE POLICY "user_follows_insert_own" ON user_follows
  FOR INSERT WITH CHECK (auth.uid() = follower_id);

-- Users can only delete their own follows
CREATE POLICY "user_follows_delete_own" ON user_follows
  FOR DELETE USING (auth.uid() = follower_id);

-- Everyone can see follows (public feature)
CREATE POLICY "user_follows_select_public" ON user_follows
  FOR SELECT USING (true);

-- Users can update their own follow record
CREATE POLICY "user_follows_update_own" ON user_follows
  FOR UPDATE USING (auth.uid() = follower_id);

-- ============================================================================
-- PLAY_HISTORY TABLE - RLS Policies [CRITICAL FIX]
-- ============================================================================
ALTER TABLE play_history ENABLE ROW LEVEL SECURITY;

-- Users can only insert their own play history
CREATE POLICY "play_history_insert_own" ON play_history
  FOR INSERT WITH CHECK (auth.uid() = listener_user_id);

-- Users can see their own play history
CREATE POLICY "play_history_select_own" ON play_history
  FOR SELECT USING (auth.uid() = listener_user_id);

-- Project owners can see play analytics for their projects
CREATE POLICY "play_history_select_project_owner" ON play_history
  FOR SELECT USING (
    project_id IN (
      SELECT id FROM projects WHERE user_id = auth.uid()
    )
  );

-- Users can update their own play history (for resume functionality)
CREATE POLICY "play_history_update_own" ON play_history
  FOR UPDATE USING (auth.uid() = listener_user_id);

-- Users can delete their own play history
CREATE POLICY "play_history_delete_own" ON play_history
  FOR DELETE USING (auth.uid() = listener_user_id);

-- ============================================================================
-- INDEX OPTIMIZATION FOR RLS POLICIES
-- ============================================================================
-- These indexes make RLS policy checks faster
CREATE INDEX IF NOT EXISTS idx_likes_user_track ON likes(user_id, track_id);
CREATE INDEX IF NOT EXISTS idx_project_likes_user_project ON project_likes(user_id, project_id);
CREATE INDEX IF NOT EXISTS idx_user_follows_user_follow ON user_follows(follower_id, following_id);
CREATE INDEX IF NOT EXISTS idx_play_history_user_project ON play_history(listener_user_id, project_id);
CREATE INDEX IF NOT EXISTS idx_play_history_project_listener ON play_history(project_id, listener_user_id);
