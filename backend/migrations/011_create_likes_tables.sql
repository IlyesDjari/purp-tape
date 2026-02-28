-- Create likes table (who liked what)
CREATE TABLE IF NOT EXISTS likes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    track_id UUID NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, track_id) -- prevent duplicate likes
);

-- Create indexes
CREATE INDEX idx_likes_user_id ON likes(user_id);
CREATE INDEX idx_likes_track_id ON likes(track_id);

-- Similarly for projects
CREATE TABLE IF NOT EXISTS project_likes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, project_id) -- prevent duplicate likes
);

-- Create indexes
CREATE INDEX idx_project_likes_user_id ON project_likes(user_id);
CREATE INDEX idx_project_likes_project_id ON project_likes(project_id);
