-- Create project_shares table (for access control and sharing)
CREATE TABLE IF NOT EXISTS project_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    shared_by_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    shared_with_id UUID REFERENCES users(id) ON DELETE CASCADE,
    share_token VARCHAR(255) UNIQUE NOT NULL, -- unique token for share links
    expires_at TIMESTAMP, -- NULL means never expires
    revoked_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index on project_id for faster lookups
CREATE INDEX idx_project_shares_project_id ON project_shares(project_id);

-- Create index on shared_with_id for finding shared projects
CREATE INDEX idx_project_shares_shared_with_id ON project_shares(shared_with_id);

-- Create index on share_token for looking up by token
CREATE INDEX idx_project_shares_share_token ON project_shares(share_token);

-- Create index for expired shares cleanup
CREATE INDEX idx_project_shares_expires_at ON project_shares(expires_at) WHERE expires_at IS NOT NULL;

-- Create index for active/revoked share filtering
CREATE INDEX idx_project_shares_revoked_at ON project_shares(revoked_at);
