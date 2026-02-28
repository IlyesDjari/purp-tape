-- Migration: Add compliance and privacy features [MEDIUM: Compliance]

-- ============================================================================
-- Privacy Settings
-- ============================================================================

ALTER TABLE users ADD COLUMN IF NOT EXISTS privacy_settings JSONB DEFAULT (
  jsonb_build_object(
    'share_profile', true,
    'show_in_search', true,
    'allow_collaboration_requests', true
  )
);

CREATE INDEX IF NOT EXISTS idx_users_privacy_settings ON users USING gin(privacy_settings);

-- ============================================================================
-- Audit Logs Table (for compliance and security auditing)
-- ============================================================================

CREATE TABLE IF NOT EXISTS audit_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  action VARCHAR(50) NOT NULL,
  resource VARCHAR(50) NOT NULL,
  resource_id VARCHAR(255),
  changes JSONB,
  ip_address INET,
  user_agent TEXT,
  status VARCHAR(20) NOT NULL DEFAULT 'success',
  error TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS resource_id VARCHAR(255);
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS changes JSONB;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS ip_address INET;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS user_agent TEXT;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'success';
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS error TEXT;

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource, resource_id);

-- Retention policy: Keep audit logs for 7 years (GDPR, SEC compliance)
-- Delete old records after 7 years:
-- DELETE FROM audit_logs WHERE created_at < NOW() - INTERVAL '7 years';

-- ============================================================================
-- RLS Policy for Audit Logs (users can only see their own)
-- ============================================================================

ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_policies
    WHERE schemaname = 'public' AND tablename = 'audit_logs' AND policyname = 'audit_logs_select_own'
  ) THEN
    EXECUTE 'CREATE POLICY "audit_logs_select_own" ON audit_logs FOR SELECT USING (auth.uid() = user_id)';
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_policies
    WHERE schemaname = 'public' AND tablename = 'audit_logs' AND policyname = 'audit_logs_insert_own'
  ) THEN
    EXECUTE 'CREATE POLICY "audit_logs_insert_own" ON audit_logs FOR INSERT WITH CHECK (auth.uid() = user_id)';
  END IF;
END $$;

-- ============================================================================
-- GDPR Compliance Table - Data Requests
-- ============================================================================

CREATE TABLE IF NOT EXISTS gdpr_data_requests (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  request_type VARCHAR(50) NOT NULL, -- 'export', 'delete', 'rectification'
  status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'processing', 'completed', 'failed'
  requested_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  completed_at TIMESTAMP,
  data_file_path TEXT, -- S3 path for exported data
  reason TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_gdpr_requests_user ON gdpr_data_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_gdpr_requests_status ON gdpr_data_requests(status);
CREATE INDEX IF NOT EXISTS idx_gdpr_requests_requested_at ON gdpr_data_requests(requested_at DESC);

-- 30-day response requirement for GDPR
-- Check: SELECT user_id, request_type, requested_at FROM gdpr_data_requests 
--        WHERE status = 'pending' AND requested_at < NOW() - INTERVAL '30 days';

-- ============================================================================
-- Consent Tracking (for GDPR/CCPA compliance)
-- ============================================================================

CREATE TABLE IF NOT EXISTS user_consents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  consent_type VARCHAR(50) NOT NULL, -- 'marketing', 'analytics', 'data_sharing'
  granted BOOLEAN NOT NULL,
  consent_date TIMESTAMP NOT NULL,
  ip_address INET,
  user_agent TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(user_id, consent_type)
);

CREATE INDEX IF NOT EXISTS idx_consents_user ON user_consents(user_id);
CREATE INDEX IF NOT EXISTS idx_consents_type ON user_consents(consent_type);
CREATE INDEX IF NOT EXISTS idx_consents_granted ON user_consents(granted);

-- ============================================================================
-- Deleted Users Log (for accounting and compliance)
-- ============================================================================

CREATE TABLE IF NOT EXISTS deleted_users_log (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  email VARCHAR(255),
  username VARCHAR(255),
  reason VARCHAR(255),
  requested_by UUID, -- Admin or user themselves
  deleted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_deleted_users_email ON deleted_users_log(email);
CREATE INDEX IF NOT EXISTS idx_deleted_users_deleted_at ON deleted_users_log(deleted_at DESC);
