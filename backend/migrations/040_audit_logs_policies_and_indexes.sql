-- Create audit_logs table for tracking sensitive operations
CREATE TABLE IF NOT EXISTS audit_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  action VARCHAR(255) NOT NULL,
  resource VARCHAR(255) NOT NULL,
  details JSONB DEFAULT '{}',
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  
  -- Indexes for fast querying
  CONSTRAINT audit_logs_check CHECK (action != '')
);

-- Index for user-based audit log queries
CREATE INDEX IF NOT EXISTS audit_logs_user_id_idx ON audit_logs(user_id, created_at DESC);

-- Index for resource-based queries
CREATE INDEX IF NOT EXISTS audit_logs_resource_idx ON audit_logs(resource, created_at DESC);

-- Index for event type queries  
CREATE INDEX IF NOT EXISTS audit_logs_action_idx ON audit_logs(action, created_at DESC);

-- Set retention policy (90 days default for logs)
-- DELETE from audit_logs WHERE created_at < NOW() - INTERVAL '90 days';

-- ============================================================================
-- AUDIT LOGS - RLS Policy (if needed)
-- ============================================================================
-- Users can only view their own audit logs
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "audit_logs_select_own" ON audit_logs;
DROP POLICY IF EXISTS "audit_logs_insert_own" ON audit_logs;

CREATE POLICY "audit_logs_select_own" ON audit_logs
  FOR SELECT USING (auth.uid() = user_id);

-- Authenticated users can insert their own audit logs
CREATE POLICY "audit_logs_insert_own" ON audit_logs
  FOR INSERT WITH CHECK (auth.uid() = user_id);
