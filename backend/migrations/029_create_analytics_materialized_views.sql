-- Migration: Create materialized views for analytics performance [HIGH: Analytics performance]
-- These views pre-aggregate data to avoid expensive JOINs on every query

-- ============================================================================
-- Project statistics materialized view
-- ============================================================================

CREATE MATERIALIZED VIEW IF NOT EXISTS project_stats_view AS
SELECT 
  p.id as project_id,
  p.user_id,
  p.name,
  COALESCE(COUNT(DISTINCT ph.id), 0) as total_plays,
  COALESCE(COUNT(DISTINCT ph.listener_user_id), 0) as unique_listeners,
  COALESCE(COUNT(DISTINCT pl.id), 0) as total_likes,
  COALESCE(COUNT(DISTINCT c.id), 0) as total_comments,
  COALESCE(COUNT(DISTINCT t.id), 0) as track_count,
  MAX(ph.started_at) as last_play_at,
  AVG(ph.duration_listened) as avg_play_duration
FROM projects p
LEFT JOIN play_history ph ON p.id = ph.project_id
LEFT JOIN project_likes pl ON p.id = pl.project_id
LEFT JOIN track_versions c_tv ON c_tv.track_id IN (SELECT id FROM tracks WHERE project_id = p.id)
LEFT JOIN comments c ON c.track_version_id = c_tv.id
LEFT JOIN tracks t ON p.id = t.project_id AND t.deleted_at IS NULL
WHERE p.deleted_at IS NULL
GROUP BY p.id, p.user_id, p.name;

-- Index for fast lookups
CREATE INDEX IF NOT EXISTS idx_project_stats_id ON project_stats_view(project_id);
CREATE INDEX IF NOT EXISTS idx_project_stats_user ON project_stats_view(user_id);

-- ============================================================================
-- Trending projects materialized view
-- ============================================================================

CREATE MATERIALIZED VIEW IF NOT EXISTS trending_projects_view AS
SELECT 
  p.id,
  p.name,
  p.user_id,
  COALESCE(COUNT(DISTINCT ph.id), 0) as play_count_30d,
  COALESCE(COUNT(DISTINCT ph.listener_user_id), 0) as unique_listeners_30d,
  COALESCE(COUNT(DISTINCT pl.id), 0) as like_count,
  ROW_NUMBER() OVER (ORDER BY COUNT(DISTINCT ph.id) DESC, COUNT(DISTINCT ph.listener_user_id) DESC) as rank
FROM projects p
LEFT JOIN play_history ph ON p.id = ph.project_id AND ph.started_at > NOW() - INTERVAL '30 days'
LEFT JOIN project_likes pl ON p.id = pl.project_id
WHERE p.deleted_at IS NULL AND p.is_private = false
GROUP BY p.id, p.name, p.user_id
HAVING COUNT(DISTINCT ph.id) > 0
ORDER BY play_count_30d DESC, unique_listeners_30d DESC
LIMIT 1000;

CREATE INDEX IF NOT EXISTS idx_trending_projects_rank ON trending_projects_view(rank);

-- ============================================================================
-- Daily engagement materialized view
-- ============================================================================

CREATE MATERIALIZED VIEW IF NOT EXISTS daily_engagement_view AS
SELECT 
  DATE(ph.started_at) as engagement_date,
  p.id as project_id,
  COUNT(*) as play_count,
  COUNT(DISTINCT ph.listener_user_id) as unique_listeners,
  AVG(ph.duration_listened) as avg_duration_seconds
FROM play_history ph
JOIN projects p ON p.id = ph.project_id
WHERE p.deleted_at IS NULL AND ph.started_at > NOW() - INTERVAL '90 days'
GROUP BY DATE(ph.started_at), p.id;

CREATE INDEX IF NOT EXISTS idx_engagement_date_project ON daily_engagement_view(engagement_date, project_id);

-- ============================================================================
-- User activity materialized view
-- ============================================================================

CREATE MATERIALIZED VIEW IF NOT EXISTS user_activity_view AS
SELECT 
  u.id as user_id,
  u.username,
  COUNT(DISTINCT p.id) as project_count,
  COUNT(DISTINCT t.id) as track_count,
  COALESCE(SUM(CASE WHEN ph.listener_user_id = p.user_id THEN 1 ELSE 0 END), 0) as plays_on_own_projects,
  COUNT(DISTINCT ph.listener_user_id) as times_listened_by_others,
  MAX(p.updated_at) as last_project_update
FROM users u
LEFT JOIN projects p ON u.id = p.user_id AND p.deleted_at IS NULL
LEFT JOIN tracks t ON p.id = t.project_id AND t.deleted_at IS NULL
LEFT JOIN play_history ph ON p.id = ph.project_id
WHERE u.deleted_at IS NULL
GROUP BY u.id, u.username;

CREATE INDEX IF NOT EXISTS idx_user_activity_id ON user_activity_view(user_id);

-- ============================================================================
-- Refresh schedule (manual for now, but can be automated with pg_cron)
-- ============================================================================

-- To refresh materialized views, run:
-- REFRESH MATERIALIZED VIEW CONCURRENTLY project_stats_view;
-- REFRESH MATERIALIZED VIEW CONCURRENTLY trending_projects_view;
-- REFRESH MATERIALIZED VIEW CONCURRENTLY daily_engagement_view;
-- REFRESH MATERIALIZED VIEW CONCURRENTLY user_activity_view;

-- Suggested refresh schedule:
-- - project_stats_view: Every 5 minutes (for dashboard accuracy)
-- - trending_projects_view: Every hour (less critical)
-- - daily_engagement_view: Every hour (less critical)
-- - user_activity_view: Every 24 hours (daily summaries)

-- Example with pg_cron extension (requires installation):
-- SELECT cron.schedule('refresh_project_stats', '*/5 * * * *', 'REFRESH MATERIALIZED VIEW CONCURRENTLY project_stats_view');
