-- Migration: Add search indexes for text search performance [HIGH: Search feature]

-- GiST index for full-text search on projects (name and description)
CREATE INDEX IF NOT EXISTS idx_projects_search ON projects USING gin(
  to_tsvector('english', name || ' ' || COALESCE(description, ''))
) WHERE deleted_at IS NULL AND is_private = false;

-- Simple LIKE index for projects name (faster than full-text for simple prefix search)
CREATE INDEX IF NOT EXISTS idx_projects_name_lower ON projects(LOWER(name)) 
WHERE deleted_at IS NULL AND is_private = false;

-- Index for tracks name search
CREATE INDEX IF NOT EXISTS idx_tracks_name_lower ON tracks(LOWER(name)) 
WHERE deleted_at IS NULL;

-- Index for users username search
CREATE INDEX IF NOT EXISTS idx_users_username_lower ON users(LOWER(username)) 
WHERE deleted_at IS NULL;

-- Composite index for search with project visibility
CREATE INDEX IF NOT EXISTS idx_projects_name_private ON projects(name, is_private) 
WHERE deleted_at IS NULL;
