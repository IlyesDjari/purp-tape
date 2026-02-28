-- Create tracks table
CREATE TABLE IF NOT EXISTS tracks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    duration INTEGER NOT NULL DEFAULT 0, -- duration in seconds
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index on project_id for faster lookups
CREATE INDEX idx_tracks_project_id ON tracks(project_id);

-- Create index on user_id for faster lookups
CREATE INDEX idx_tracks_user_id ON tracks(user_id);

-- Create index on created_at for sorting
CREATE INDEX idx_tracks_created_at ON tracks(created_at DESC);
