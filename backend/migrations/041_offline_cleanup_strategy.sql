-- ============================================================================
-- OFFLINE DOWNLOADS - Add expiration and cleanup
-- ============================================================================

-- Add expires_at column to offline_downloads if not already present
DO $$
BEGIN
	IF to_regclass('public.offline_downloads') IS NOT NULL THEN
		EXECUTE 'ALTER TABLE offline_downloads ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP WITH TIME ZONE';
		EXECUTE 'CREATE INDEX IF NOT EXISTS offline_downloads_expires_at_idx ON offline_downloads(expires_at) WHERE status = ''completed'' AND expires_at IS NOT NULL';
	END IF;
END $$;

-- ============================================================================
-- OFFLINE DOWNLOADS - Cleanup Strategy
-- ============================================================================
-- 
-- Offline downloads are automatically expired based on subscription tier:
-- - Free tier: 7 days (manual deletion required)
-- - Pro tier: 30 days (auto-cleanup after expiration)
-- - Pro+/Unlimited: 90 days (auto-cleanup after expiration)
--
-- Run cleanup job daily:
--   SELECT COUNT(*) FROM cleanup_expired_offline_downloads();
--
-- Or run manually:
--   DELETE FROM offline_downloads 
--   WHERE status = 'completed' AND expires_at < NOW();
--

-- Drop old cleanup function if it exists
DROP FUNCTION IF EXISTS cleanup_expired_offline_downloads();

-- Create cleanup function
DO $$
BEGIN
	IF to_regclass('public.offline_downloads') IS NOT NULL THEN
		EXECUTE $exec$
		CREATE OR REPLACE FUNCTION cleanup_expired_offline_downloads()
		RETURNS TABLE(deleted_count BIGINT)
		LANGUAGE plpgsql
		AS $func$
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
		$func$;
		$exec$;
	END IF;
END $$;

-- ============================================================================
-- Trigger to set expires_at based on subscription tier (optional)
-- ============================================================================
--
-- CREATE OR REPLACE FUNCTION set_offline_download_expiration()
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
