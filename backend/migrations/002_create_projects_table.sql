-- Create projects table (the "Vault")
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index on user_id for faster lookups
CREATE INDEX idx_projects_user_id ON projects(user_id);

-- Create index on created_at for sorting
CREATE INDEX idx_projects_created_at ON projects(created_at DESC);

-- Create index on updated_at for sorting
CREATE INDEX idx_projects_updated_at ON projects(updated_at DESC);
