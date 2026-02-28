-- Create track_versions table (for versioning)
CREATE TABLE IF NOT EXISTS track_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    track_id UUID NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    r2_object_key VARCHAR(512) NOT NULL, -- path in Cloudflare R2
    file_size BIGINT NOT NULL,
    checksum VARCHAR(64) NOT NULL, -- SHA256 hash for integrity check
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(track_id, version_number)
);

-- Create index on track_id for faster lookups
CREATE INDEX idx_track_versions_track_id ON track_versions(track_id);

-- Create index on created_at for sorting
CREATE INDEX idx_track_versions_created_at ON track_versions(created_at DESC);
