-- Create offline downloads tracking table
CREATE TABLE IF NOT EXISTS offline_downloads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    track_version_id UUID NOT NULL REFERENCES track_versions(id) ON DELETE CASCADE,
    track_id UUID NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    
    -- File info
    file_size_bytes BIGINT NOT NULL,
    r2_object_key VARCHAR(512) NOT NULL,
    local_file_hash VARCHAR(64), -- SHA256 of downloaded file (device-side)
    
    -- Status tracking
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, downloading, completed, failed, removed
    download_progress_percent INT DEFAULT 0,
    
    -- Metadata for offline use
    title VARCHAR(255),
    artist_name VARCHAR(255),
    project_name VARCHAR(255),
    cover_image_url TEXT, -- signed URL at download time
    duration_seconds INT,
    
    -- Storage
    local_path_hash VARCHAR(64), -- hash of where file is stored locally (device)
    storage_used_bytes BIGINT, -- actual space on device
    
    -- Timestamps
    downloaded_at TIMESTAMP,
    last_played_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP -- optional: when to auto-delete (if storage full)
);

-- Indexes for fast queries
CREATE INDEX IF NOT EXISTS idx_offline_downloads_user_id ON offline_downloads(user_id);
CREATE INDEX IF NOT EXISTS idx_offline_downloads_status ON offline_downloads(status);
CREATE INDEX IF NOT EXISTS idx_offline_downloads_project_id ON offline_downloads(project_id);
CREATE INDEX IF NOT EXISTS idx_offline_downloads_downloaded_at ON offline_downloads(downloaded_at DESC);
CREATE INDEX IF NOT EXISTS idx_offline_downloads_created_at ON offline_downloads(created_at DESC);

-- Composite index for user's completed downloads
CREATE INDEX IF NOT EXISTS idx_offline_downloads_user_completed ON offline_downloads(user_id, status) WHERE status = 'completed';
