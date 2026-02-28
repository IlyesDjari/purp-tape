-- Migration: Create device_tokens table for push notifications
-- Stores Firebase Cloud Messaging tokens for multi-device notification delivery

CREATE TABLE IF NOT EXISTS device_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    platform VARCHAR(20) NOT NULL, -- "ios", "android", "web"
    is_active BOOLEAN DEFAULT TRUE,
    last_used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_device_tokens_user_id ON device_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_device_tokens_user_active ON device_tokens(user_id, is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_device_tokens_platform ON device_tokens(platform);
CREATE INDEX IF NOT EXISTS idx_device_tokens_token ON device_tokens(token);
CREATE INDEX IF NOT EXISTS idx_device_tokens_created_at ON device_tokens(created_at DESC);

-- RLS Policies
ALTER TABLE device_tokens ENABLE ROW LEVEL SECURITY;

-- Users can only view their own device tokens
CREATE POLICY "device_tokens_select_own" ON device_tokens
  FOR SELECT USING (auth.uid() = user_id);

-- Users cannot insert (only app can)
CREATE POLICY "device_tokens_no_direct_insert" ON device_tokens
  FOR INSERT WITH CHECK (false);

-- Users can update their own tokens (mark inactive)
CREATE POLICY "device_tokens_update_own" ON device_tokens
  FOR UPDATE USING (auth.uid() = user_id);

-- Users can delete their own tokens
CREATE POLICY "device_tokens_delete_own" ON device_tokens
  FOR DELETE USING (auth.uid() = user_id);
