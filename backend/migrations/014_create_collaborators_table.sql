-- Create collaborators table for project access control
CREATE TABLE IF NOT EXISTS collaborators (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'viewer', -- "owner", "editor", "commenter", "viewer"
    invited_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    joined_at TIMESTAMP,
    UNIQUE(project_id, user_id) -- can't add same user twice
);

-- Create indexes
CREATE INDEX idx_collaborators_project_id ON collaborators(project_id);
CREATE INDEX idx_collaborators_user_id ON collaborators(user_id);
CREATE INDEX idx_collaborators_role ON collaborators(role);
