-- Create images table for covers, artwork, etc
CREATE TABLE IF NOT EXISTS images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    r2_object_key VARCHAR(512) NOT NULL,
    mime_type VARCHAR(50) NOT NULL,
    file_size BIGINT NOT NULL,
    alt_text TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index on user_id
CREATE INDEX idx_images_user_id ON images(user_id);
