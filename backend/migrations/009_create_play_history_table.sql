-- Create play_history table for analytics
CREATE TABLE IF NOT EXISTS play_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    track_version_id UUID NOT NULL REFERENCES track_versions(id) ON DELETE CASCADE,
    listener_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    track_id UUID NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    duration_listened INT, -- in seconds
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    device VARCHAR(100), -- "iOS", "web", etc
    country VARCHAR(2) -- ISO country code
);

-- Create indexes for analytics queries
CREATE INDEX idx_play_history_track_version_id ON play_history(track_version_id);
CREATE INDEX idx_play_history_listener_user_id ON play_history(listener_user_id);
CREATE INDEX idx_play_history_project_id ON play_history(project_id);
CREATE INDEX idx_play_history_started_at ON play_history(started_at DESC);

-- Create composite index for daily analytics
CREATE INDEX idx_play_history_project_date ON play_history(project_id, CAST(started_at AS DATE));
