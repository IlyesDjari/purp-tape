-- Migration: Create notification_preferences table
-- Stores user preferences for notification delivery channels and types

CREATE TABLE IF NOT EXISTS notification_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    
    -- Push notification settings
    push_enabled BOOLEAN DEFAULT TRUE,
    push_likes BOOLEAN DEFAULT TRUE,
    push_comments BOOLEAN DEFAULT TRUE,
    push_follows BOOLEAN DEFAULT TRUE,
    push_shares BOOLEAN DEFAULT TRUE,
    push_mentions BOOLEAN DEFAULT TRUE,
    
    -- Quiet hours settings
    quiet_hours_enabled BOOLEAN DEFAULT FALSE,
    quiet_hours_start VARCHAR(5) DEFAULT '22:00', -- HH:MM format
    quiet_hours_end VARCHAR(5) DEFAULT '09:00', -- HH:MM format
    
    -- Aggregation settings
    bundle_by_type BOOLEAN DEFAULT TRUE, -- Aggregate similar notifications
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_notification_preferences_user_id ON notification_preferences(user_id);
CREATE INDEX IF NOT EXISTS idx_notification_preferences_push_enabled ON notification_preferences(push_enabled) WHERE push_enabled = TRUE;

-- RLS Policies
ALTER TABLE notification_preferences ENABLE ROW LEVEL SECURITY;

-- Users can only view their own preferences
CREATE POLICY "notification_preferences_select_own" ON notification_preferences
  FOR SELECT USING (auth.uid() = user_id);

-- Users can update their own preferences
CREATE POLICY "notification_preferences_update_own" ON notification_preferences
  FOR UPDATE USING (auth.uid() = user_id);

-- Users can insert their own preferences
CREATE POLICY "notification_preferences_insert_own" ON notification_preferences
  FOR INSERT WITH CHECK (auth.uid() = user_id);

-- Prevent deletion to maintain audit trail
CREATE POLICY "notification_preferences_no_delete" ON notification_preferences
  FOR DELETE USING (false);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_notification_preferences_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_notification_preferences_updated_at ON notification_preferences;

CREATE TRIGGER trigger_notification_preferences_updated_at
BEFORE UPDATE ON notification_preferences
FOR EACH ROW
EXECUTE FUNCTION update_notification_preferences_timestamp();
