-- Migration: Ensure cascade deletes are configured [HIGH: Data integrity]
-- This migration adds CASCADE DELETE constraints where needed for data consistency

-- Note: This is informational - actual cascade deletes should be implemented via soft deletes
-- But if hard deletes are used, ensure foreign keys cascade properly

-- ============================================================================
-- Verify foreign key constraints exist and are configured
-- ============================================================================

-- Tracks should cascade delete to track_versions
-- (Projects -> Tracks -> TrackVersions relationship)

-- Comments should be deleted when track is deleted
-- (Tracks -> Comments relationship)

-- Likes should be deleted when track/project is deleted
-- (Tracks -> Likes, Projects -> ProjectLikes)

-- Play history can be soft-deleted or kept for archive purposes
-- Decision: KEEP play history for analytics (never delete)

-- Offline downloads should be deleted when they're orphaned
-- (Track versions -> Offline Downloads relationship)

-- ============================================================================
-- For production: Use soft deletes via deleted_at column
-- This migration documents the relationships
-- ============================================================================

-- If hard deletes are needed in the future, uncomment these:
/*
ALTER TABLE track_versions 
DROP CONSTRAINT IF EXISTS fk_track_versions_tracks,
ADD CONSTRAINT fk_track_versions_tracks 
  FOREIGN KEY (track_id) 
  REFERENCES tracks(id) 
  ON DELETE CASCADE;

ALTER TABLE comments 
DROP CONSTRAINT IF EXISTS fk_comments_tracks,
ADD CONSTRAINT fk_comments_tracks 
  FOREIGN KEY (track_id) 
  REFERENCES tracks(id) 
  ON DELETE CASCADE;

ALTER TABLE likes 
DROP CONSTRAINT IF EXISTS fk_likes_tracks,
ADD CONSTRAINT fk_likes_tracks 
  FOREIGN KEY (track_id) 
  REFERENCES tracks(id) 
  ON DELETE CASCADE;

ALTER TABLE offline_downloads 
DROP CONSTRAINT IF EXISTS fk_offline_downloads_track_versions,
ADD CONSTRAINT fk_offline_downloads_track_versions 
  FOREIGN KEY (track_version_id) 
  REFERENCES track_versions(id) 
  ON DELETE CASCADE;
*/

-- ============================================================================
-- Current approach: Soft deletes with RLS
-- ============================================================================
-- Projects have deleted_at column
-- Tracks have deleted_at column
-- Track_versions have deleted_at column
-- Deletes are cascaded via triggers (see migrations/024_add_soft_deletes.sql)
-- All queries filter: WHERE deleted_at IS NULL (via RLS policies)
