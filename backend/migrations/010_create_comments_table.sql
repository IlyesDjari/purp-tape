-- Create comments table for collaboration & feedback
CREATE TABLE IF NOT EXISTS comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    track_version_id UUID NOT NULL REFERENCES track_versions(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    timestamp_ms INT, -- timestamp in the track where comment applies
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_comments_user_id ON comments(user_id);
CREATE INDEX idx_comments_track_version_id ON comments(track_version_id);
CREATE INDEX idx_comments_created_at ON comments(created_at DESC);

-- Create composite index for fetching comments on a version
CREATE INDEX idx_comments_track_timestamp ON comments(track_version_id, timestamp_ms);
