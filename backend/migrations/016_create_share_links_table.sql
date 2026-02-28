-- Create shares table with cryptographic links
CREATE TABLE IF NOT EXISTS share_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    creator_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    hash VARCHAR(32) NOT NULL UNIQUE, -- short cryptographic hash
    access_level VARCHAR(50) NOT NULL DEFAULT 'viewer', -- 'viewer', 'commenter', 'collaborator'
    is_public BOOLEAN DEFAULT FALSE,
    is_password_protected BOOLEAN DEFAULT FALSE,
    password_hash VARCHAR(255), -- bcrypt hash
    expires_at TIMESTAMP,
    revoked_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index for fast hash lookups
CREATE INDEX IF NOT EXISTS idx_share_links_hash ON share_links(hash);
CREATE INDEX IF NOT EXISTS idx_share_links_project_id ON share_links(project_id);
CREATE INDEX IF NOT EXISTS idx_share_links_revoked_at ON share_links(revoked_at);
