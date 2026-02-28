-- ============================================================================
-- OFFLINE DOWNLOADS - Add expiration and cleanup
-- ============================================================================

-- Add expires_at column to offline_downloads (idempotent with IF NOT EXISTS)
ALTER TABLE offline_downloads 
ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP WITH TIME ZONE;

-- Index for efficient cleanup queries
CREATE INDEX IF NOT EXISTS idx_offline_downloads_expires_at 
ON offline_downloads(expires_at) 
WHERE status = 'completed' AND expires_at IS NOT NULL;

-- ============================================================================
-- OFFLINE DOWNLOADS - Cleanup Function
-- ============================================================================
-- Cleanup strategy: auto-expire based on subscription tier
-- - Free tier: 7 days (manual deletion required)
-- - Pro tier: 30 days (auto-cleanup after expiration)
-- - Pro+/Unlimited: 90 days (auto-cleanup after expiration)
--
-- Run cleanup job daily: SELECT cleanup_expired_offline_downloads();

DROP FUNCTION IF EXISTS cleanup_expired_offline_downloads() CASCADE;

CREATE OR REPLACE FUNCTION cleanup_expired_offline_downloads()
RETURNS TABLE(deleted_count BIGINT)
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    count BIGINT;
BEGIN
    DELETE FROM offline_downloads
    WHERE status = 'completed'
      AND expires_at IS NOT NULL
      AND expires_at < NOW();
    
    GET DIAGNOSTICS count = ROW_COUNT;
    RETURN QUERY SELECT count;
END;
$$;

-- ============================================================================
-- OPTIONAL: Trigger to set expires_at based on subscription tier
-- ============================================================================
-- Uncomment if implementing subscription-based expiration
-- RETURNS TRIGGER
-- LANGUAGE plpgsql
-- AS $$
-- DECLARE
--   subscription_tier VARCHAR(64);
--   expiration_days INT;
-- BEGIN
--   -- Get user's subscription tier
--   SELECT COALESCE(s.tier, 'free') INTO subscription_tier
--   FROM subscriptions s WHERE s.user_id = NEW.user_id
--   ORDER BY s.created_at DESC LIMIT 1;
--
--   -- Set expiration based on tier
--   CASE subscription_tier
--     WHEN 'pro' THEN expiration_days := 30;
--     WHEN 'pro_plus' THEN expiration_days := 90;
--     WHEN 'unlimited' THEN expiration_days := 365;
--     ELSE expiration_days := 7;
--   END CASE;
--
--   NEW.expires_at := NOW() + INTERVAL '1 day' * expiration_days;
--   RETURN NEW;
-- END;
-- $$;
--
-- CREATE TRIGGER set_offline_expiration
-- BEFORE INSERT ON offline_downloads
-- FOR EACH ROW
-- EXECUTE FUNCTION set_offline_download_expiration();
--
