-- Add columns to projects for metadata and privacy
ALTER TABLE projects ADD COLUMN IF NOT EXISTS cover_image_id UUID REFERENCES images(id) ON DELETE SET NULL;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS is_private BOOLEAN DEFAULT TRUE;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS is_collaborative BOOLEAN DEFAULT FALSE;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS genre VARCHAR(100);
ALTER TABLE projects ADD COLUMN IF NOT EXISTS description_full TEXT;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS release_date DATE;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS play_count BIGINT DEFAULT 0;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS like_count INT DEFAULT 0;

-- Create index for filtering
CREATE INDEX IF NOT EXISTS idx_projects_is_private ON projects(is_private);
CREATE INDEX IF NOT EXISTS idx_projects_genre ON projects(genre);
CREATE INDEX IF NOT EXISTS idx_projects_play_count ON projects(play_count DESC);
