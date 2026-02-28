-- ============================================
-- ENTERPRISE OPTIMIZATION MIGRATIONS
-- ============================================

-- 1. ADD COVERING INDEXES FOR HOT QUERIES
CREATE INDEX IF NOT EXISTS idx_projects_user_created ON projects(user_id, created_at DESC) 
INCLUDE (name, description, is_private, cover_image_id, play_count, like_count);

-- For offline downloads queries
CREATE INDEX IF NOT EXISTS idx_offline_downloads_user_completed ON offline_downloads(user_id, downloaded_at DESC)
WHERE status = 'completed';

-- For analytics queries
CREATE INDEX IF NOT EXISTS idx_play_history_project_date ON play_history(project_id, started_at DESC)
INCLUDE (listener_user_id, duration_listened, device);

-- For trending queries (only public projects)
CREATE INDEX IF NOT EXISTS idx_projects_trending ON projects(play_count DESC)
INCLUDE (user_id, name, cover_image_id)
WHERE is_private = FALSE;

-- For search queries
CREATE INDEX IF NOT EXISTS idx_projects_name_search ON projects USING gin(to_tsvector('english', name || ' ' || COALESCE(description, '')))
WHERE is_private = FALSE;

-- 2. PARTIAL INDEXES FOR SPECIFIC CONDITIONS
-- For deleted records
CREATE INDEX IF NOT EXISTS idx_share_links_active ON share_links(project_id)
WHERE revoked_at IS NULL;

-- For pending jobs (much smaller than all jobs)
CREATE INDEX IF NOT EXISTS idx_background_jobs_pending ON background_jobs(created_at ASC)
WHERE status IN ('pending', 'processing');

-- For active comments
CREATE INDEX IF NOT EXISTS idx_comments_track ON comments(track_version_id);

-- 3. COMPOSITE INDEXES FOR COMMON JOINS
CREATE INDEX IF NOT EXISTS idx_track_versions_latest ON track_versions(track_id, version_number DESC)
INCLUDE (r2_object_key, file_size, checksum);

CREATE INDEX IF NOT EXISTS idx_collaborators_project_user ON collaborators(project_id, user_id)
INCLUDE (role);

-- 4. STATISTICS FOR QUERY PLANNER
-- Tell PostgreSQL about column value distributions
ANALYZE projects;
ANALYZE play_history;
ANALYZE offline_downloads;
ANALYZE tracks;

-- 5. ADD MATERIALIZED VIEW FOR TRENDING (pre-computed)
CREATE MATERIALIZED VIEW IF NOT EXISTS trending_projects_computed AS
SELECT 
  p.id,
  p.name,
  p.user_id,
  COUNT(DISTINCT ph.id) as total_plays,
  COUNT(DISTINCT ph.listener_user_id) as unique_listeners,
  COUNT(DISTINCT pl.id) as total_likes,
  RANK() OVER (ORDER BY COUNT(DISTINCT ph.id) DESC) as play_rank,
  MAX(ph.started_at) as last_play_at
FROM projects p
LEFT JOIN play_history ph ON p.id = ph.project_id 
  AND ph.started_at > NOW() - INTERVAL '7 days'
LEFT JOIN project_likes pl ON p.id = pl.project_id
WHERE p.is_private = FALSE
GROUP BY p.id, p.name, p.user_id
ORDER BY total_plays DESC;

-- Refresh daily at 2 AM UTC
CREATE INDEX IF NOT EXISTS idx_trending_projects_rank ON trending_projects_computed(play_rank);

-- 6. PARTITIONING FOR LARGE TABLES
-- Intentionally omitted in baseline migration because play_history is not created as a partitioned table.

-- 7. VACUUM & ANALYZE
VACUUM ANALYZE projects;
VACUUM ANALYZE play_history;
VACUUM ANALYZE offline_downloads;

-- 8. TABLE STATISTICS (TOAST COMPRESSION)
ALTER TABLE comments SET (fillfactor = 70);
ALTER TABLE projects SET (fillfactor = 80);

-- 9. REINDEX (remove bloat, optimize)
-- Run during maintenance window:
-- REINDEX INDEX CONCURRENTLY idx_projects_user_created;
-- REINDEX INDEX CONCURRENTLY idx_play_history_project_date;

-- 10. DATA RETENTION POLICIES
CREATE TABLE IF NOT EXISTS archived_plays (
  LIKE play_history INCLUDING ALL
);

-- Function to archive old plays
CREATE OR REPLACE FUNCTION archive_old_plays() RETURNS void AS $$
BEGIN
  INSERT INTO archived_plays
  SELECT * FROM play_history 
  WHERE started_at < NOW() - INTERVAL '180 days';
  
  DELETE FROM play_history
  WHERE started_at < NOW() - INTERVAL '180 days';
END;
$$ LANGUAGE plpgsql;

-- Run monthly: SELECT archive_old_plays();

-- 11. CONNECTION POOLING SETTINGS
-- Set in PostgreSQL config (postgresql.conf):
-- shared_buffers = 25% of RAM
-- effective_cache_size = 75% of RAM
-- work_mem = (RAM - shared_buffers) / (max_connections * 2)
-- maintenance_work_mem = RAM / 4

-- 12. MONITORING QUERIES

-- Check index usage
CREATE OR REPLACE VIEW index_usage AS
SELECT 
  schemaname,
  relname AS tablename,
  indexrelname AS indexname,
  idx_scan as scans,
  idx_tup_read as tuples_read,
  idx_tup_fetch as tuples_fetched,
  round(100.0 * pg_relation_size(indexrelid) / pg_relation_size(relid)) as index_size_percent
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;

-- Check table bloat
CREATE OR REPLACE VIEW table_bloat AS
SELECT 
  schemaname,
  tablename,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as table_size,
  round(100.0 * (pg_relation_size(schemaname||'.'||tablename)) / 
    pg_total_relation_size(schemaname||'.'||tablename)) as table_bloat_percent
FROM pg_tables
WHERE schemaname != 'pg_catalog'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Check slow queries (if pg_stat_statements installed)
-- CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
-- SELECT query, mean_exec_time, calls FROM pg_stat_statements ORDER BY mean_exec_time DESC;
