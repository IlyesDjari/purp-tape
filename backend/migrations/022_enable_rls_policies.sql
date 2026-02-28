-- ============================================================================
-- SUPABASE ROW-LEVEL SECURITY (RLS) POLICIES
-- ============================================================================
-- Enable RLS on all public tables to ensure users can only access their data
-- Run these commands in Supabase SQL Editor after creating tables
-- ============================================================================

-- Local/dev compatibility: provide auth.uid() when running outside Supabase.
CREATE SCHEMA IF NOT EXISTS auth;

CREATE OR REPLACE FUNCTION auth.uid()
RETURNS UUID
LANGUAGE sql
STABLE
AS $$
  SELECT NULLIF(current_setting('request.jwt.claim.sub', true), '')::uuid;
$$;

-- ============================================================================
-- USERS TABLE - RLS Policies
-- ============================================================================

-- Enable RLS on users table
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Users can view their own profile
CREATE POLICY "users_select_own" ON users
  FOR SELECT USING (auth.uid() = id);

-- Users can update their own profile
CREATE POLICY "users_update_own" ON users
  FOR UPDATE USING (auth.uid() = id);

-- Users cannot delete profiles (soft delete only)
CREATE POLICY "users_no_delete" ON users
  FOR DELETE USING (false);

-- ============================================================================
-- PROJECTS TABLE - RLS Policies
-- ============================================================================

-- Enable RLS on projects table
ALTER TABLE projects ENABLE ROW LEVEL SECURITY;

-- Users can read projects they own
CREATE POLICY "projects_select_own" ON projects
  FOR SELECT USING (auth.uid() = user_id);

-- Users can read projects shared with them
CREATE POLICY "projects_select_shared" ON projects
  FOR SELECT USING (id IN (
    SELECT project_id FROM project_shares 
    WHERE shared_with_id = auth.uid() AND (revoked_at IS NULL)
  ));

-- Users can update their own projects
CREATE POLICY "projects_update_own" ON projects
  FOR UPDATE USING (auth.uid() = user_id);

-- Users can delete their own projects
CREATE POLICY "projects_delete_own" ON projects
  FOR DELETE USING (auth.uid() = user_id);

-- Users can insert projects for themselves
CREATE POLICY "projects_insert_own" ON projects
  FOR INSERT WITH CHECK (auth.uid() = user_id);

-- ============================================================================
-- TRACKS TABLE - RLS Policies
-- ============================================================================

-- Enable RLS on tracks table
ALTER TABLE tracks ENABLE ROW LEVEL SECURITY;

-- Users can read tracks from their own projects
CREATE POLICY "tracks_select_own_project" ON tracks
  FOR SELECT USING (project_id IN (
    SELECT id FROM projects WHERE user_id = auth.uid()
  ));

-- Users can read tracks from projects shared with them
CREATE POLICY "tracks_select_shared_project" ON tracks
  FOR SELECT USING (project_id IN (
    SELECT project_id FROM project_shares 
    WHERE shared_with_id = auth.uid() AND revoked_at IS NULL
  ));

-- Users can update/delete tracks in their own projects
CREATE POLICY "tracks_update_own_project" ON tracks
  FOR UPDATE USING (project_id IN (
    SELECT id FROM projects WHERE user_id = auth.uid()
  ));

CREATE POLICY "tracks_delete_own_project" ON tracks
  FOR DELETE USING (project_id IN (
    SELECT id FROM projects WHERE user_id = auth.uid()
  ));

-- Users can insert tracks to their own projects
CREATE POLICY "tracks_insert_own_project" ON tracks
  FOR INSERT WITH CHECK (project_id IN (
    SELECT id FROM projects WHERE user_id = auth.uid()
  ));

-- ============================================================================
-- TRACK_VERSIONS TABLE - RLS Policies
-- ============================================================================

-- Enable RLS on track_versions table
ALTER TABLE track_versions ENABLE ROW LEVEL SECURITY;

-- Users can read versions from their own tracks
CREATE POLICY "track_versions_select" ON track_versions
  FOR SELECT USING (track_id IN (
    SELECT id FROM tracks 
    WHERE project_id IN (
      SELECT id FROM projects WHERE user_id = auth.uid()
      UNION
      SELECT project_id FROM project_shares 
      WHERE shared_with_id = auth.uid() AND revoked_at IS NULL
    )
  ));

-- ============================================================================
-- SHARE_LINKS TABLE - RLS Policies  
-- ============================================================================

DO $$
BEGIN
  IF to_regclass('public.share_links') IS NOT NULL THEN
    EXECUTE 'ALTER TABLE share_links ENABLE ROW LEVEL SECURITY';
    EXECUTE ''
      || 'CREATE POLICY "share_links_select_public" ON share_links '
      || 'FOR SELECT USING (revoked_at IS NULL AND (expires_at IS NULL OR expires_at > NOW()))';
    EXECUTE ''
      || 'CREATE POLICY "share_links_update_own" ON share_links '
      || 'FOR UPDATE USING (auth.uid() = creator_id)';
    EXECUTE ''
      || 'CREATE POLICY "share_links_delete_own" ON share_links '
      || 'FOR DELETE USING (auth.uid() = creator_id)';
    EXECUTE ''
      || 'CREATE POLICY "share_links_insert_own_project" ON share_links '
      || 'FOR INSERT WITH CHECK (project_id IN (SELECT id FROM projects WHERE user_id = auth.uid()))';
  END IF;
END $$;

-- ============================================================================
-- OFFLINE_DOWNLOADS TABLE - RLS Policies
-- ============================================================================

DO $$
BEGIN
  IF to_regclass('public.offline_downloads') IS NOT NULL THEN
    EXECUTE 'ALTER TABLE offline_downloads ENABLE ROW LEVEL SECURITY';
    EXECUTE 'CREATE POLICY "offline_downloads_select_own" ON offline_downloads FOR SELECT USING (auth.uid() = user_id)';
    EXECUTE 'CREATE POLICY "offline_downloads_insert_own" ON offline_downloads FOR INSERT WITH CHECK (auth.uid() = user_id)';
    EXECUTE 'CREATE POLICY "offline_downloads_update_own" ON offline_downloads FOR UPDATE USING (auth.uid() = user_id)';
    EXECUTE 'CREATE POLICY "offline_downloads_delete_own" ON offline_downloads FOR DELETE USING (auth.uid() = user_id)';
  END IF;
END $$;

-- ============================================================================
-- AUDIT_LOGS TABLE - RLS Policies
-- ============================================================================

DO $$
BEGIN
  IF to_regclass('public.audit_logs') IS NOT NULL THEN
    EXECUTE 'ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY';
    EXECUTE 'CREATE POLICY "audit_logs_select_own" ON audit_logs FOR SELECT USING (auth.uid() = user_id)';
    EXECUTE 'CREATE POLICY "audit_logs_insert_own" ON audit_logs FOR INSERT WITH CHECK (auth.uid() = user_id)';
  END IF;
END $$;

-- ============================================================================
-- SUBSCRIPTIONS TABLE - RLS Policies
-- ============================================================================

DO $$
BEGIN
  IF to_regclass('public.subscriptions') IS NOT NULL THEN
    EXECUTE 'ALTER TABLE subscriptions ENABLE ROW LEVEL SECURITY';
    EXECUTE 'CREATE POLICY "subscriptions_select_own" ON subscriptions FOR SELECT USING (auth.uid() = user_id)';
    EXECUTE 'CREATE POLICY "subscriptions_update_own" ON subscriptions FOR UPDATE USING (auth.uid() = user_id)';
  END IF;
END $$;

-- ============================================================================
-- VERIFICATION
-- ============================================================================
-- Run this query to verify RLS is enabled on all tables:
-- SELECT tablename, rowsecurity 
-- FROM pg_tables 
-- WHERE schemaname = 'public' AND rowsecurity = true
-- ORDER BY tablename;
-- ============================================================================
