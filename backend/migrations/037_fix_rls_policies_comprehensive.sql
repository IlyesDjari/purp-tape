-- Migration 037: Fix critical RLS policy gaps in likes, follows, and play history
-- Addresses security issue where likes_select policy was showing all non-deleted projects

-- ============================================================================
-- LIKES TABLE - CORRECTED RLS POLICIES [CRITICAL SECURITY FIX]
-- ============================================================================

-- Drop the buggy policy (allows reading all non-deleted projects)
DROP POLICY IF EXISTS "likes_select" ON likes;
DROP POLICY IF EXISTS "likes_select_all" ON likes;

-- Correct policy: Users can only see likes on tracks they have access to
CREATE POLICY "likes_select" ON likes
  FOR SELECT USING (
    track_id IN (
      SELECT id FROM tracks 
      WHERE project_id IN (
        -- User owns the project
        (SELECT id FROM projects WHERE user_id = auth.uid() AND deleted_at IS NULL)
        UNION
        -- User is a collaborator
        (SELECT project_id FROM collaborators WHERE user_id = auth.uid())
        UNION
        -- Project is shared with user (and not revoked)
        (SELECT project_id FROM project_shares WHERE shared_with_id = auth.uid() AND revoked_at IS NULL)
      )
    )
  );

-- ============================================================================
-- PROJECT_LIKES TABLE - CORRECTED RLS POLICIES [CRITICAL SECURITY FIX]
-- ============================================================================

-- Drop the buggy policy
DROP POLICY IF EXISTS "project_likes_select" ON project_likes;
DROP POLICY IF EXISTS "project_likes_select_all" ON project_likes;

-- Correct policy: Users can see likes on projects they have access to
CREATE POLICY "project_likes_select" ON project_likes
  FOR SELECT USING (
    project_id IN (
      -- User owns the project
      (SELECT id FROM projects WHERE user_id = auth.uid())
      UNION
      -- User is a collaborator
      (SELECT project_id FROM collaborators WHERE user_id = auth.uid())
      UNION
      -- Project is shared with user (and not revoked)
      (SELECT project_id FROM project_shares WHERE shared_with_id = auth.uid() AND revoked_at IS NULL)
      UNION
      -- Public projects (is_private = false)
      (SELECT id FROM projects WHERE is_private = false AND deleted_at IS NULL)
    )
  );

-- ============================================================================
-- USER_FOLLOWS TABLE - STRENGTHENED RLS POLICIES
-- ============================================================================

-- Drop existing policies to replace them
DROP POLICY IF EXISTS "user_follows_insert_own" ON user_follows;
DROP POLICY IF EXISTS "user_follows_delete_own" ON user_follows;
DROP POLICY IF EXISTS "user_follows_select_public" ON user_follows;
DROP POLICY IF EXISTS "user_follows_update_own" ON user_follows;

-- Users can only insert their own follows
CREATE POLICY "user_follows_insert_own" ON user_follows
  FOR INSERT WITH CHECK (auth.uid() = follower_id);

-- Users can only delete their own follows
CREATE POLICY "user_follows_delete_own" ON user_follows
  FOR DELETE USING (auth.uid() = follower_id);

-- Anyone can view follows (public social graph)
CREATE POLICY "user_follows_select_public" ON user_follows
  FOR SELECT USING (true);

-- Users can update their own follow records
CREATE POLICY "user_follows_update_own" ON user_follows
  FOR UPDATE USING (auth.uid() = follower_id);

-- ============================================================================
-- PLAY_HISTORY TABLE - STRENGTHENED RLS POLICIES
-- ============================================================================

-- Drop existing policies
DROP POLICY IF EXISTS "play_history_insert_own" ON play_history;
DROP POLICY IF EXISTS "play_history_select_own" ON play_history;
DROP POLICY IF EXISTS "play_history_select_project_owner" ON play_history;

-- Users can only insert their own play history
CREATE POLICY "play_history_insert_own" ON play_history
  FOR INSERT WITH CHECK (auth.uid() = listener_user_id);

-- Users can view their own play history
CREATE POLICY "play_history_select_own" ON play_history
  FOR SELECT USING (auth.uid() = listener_user_id);

-- Project owners can see play analytics for their projects (not all users)
CREATE POLICY "play_history_select_project_owner" ON play_history
  FOR SELECT USING (
    EXISTS (
      SELECT 1 FROM projects 
      WHERE id = play_history.project_id AND user_id = auth.uid()
    )
  );

-- ============================================================================
-- COMMENTS TABLE - ADD MISSING RLS POLICIES [MEDIUM PRIORITY]
-- ============================================================================

DROP POLICY IF EXISTS "comments_select" ON comments;
DROP POLICY IF EXISTS "comments_insert_own" ON comments;

-- Users can select comments on tracks they have access to
CREATE POLICY "comments_select" ON comments
  FOR SELECT USING (
    track_version_id IN (
      SELECT tv.id
      FROM track_versions tv
      JOIN tracks t ON t.id = tv.track_id
      WHERE project_id IN (
        (SELECT id FROM projects WHERE user_id = auth.uid() AND deleted_at IS NULL)
        UNION
        (SELECT project_id FROM collaborators WHERE user_id = auth.uid())
        UNION
        (SELECT project_id FROM project_shares WHERE shared_with_id = auth.uid() AND revoked_at IS NULL)
      )
    )
  );

-- Users can only insert their own comments
CREATE POLICY "comments_insert_own" ON comments
  FOR INSERT WITH CHECK (auth.uid() = user_id);

-- ============================================================================
-- TAGS TABLE - ADD MISSING RLS POLICIES [MEDIUM PRIORITY]
-- ============================================================================

ALTER TABLE tags ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS "tags_select" ON tags;

-- Tags are global taxonomy metadata and safe to expose read-only.
CREATE POLICY "tags_select" ON tags
  FOR SELECT USING (true);
