-- ============================================================================
-- MIGRATION 018: Create background_jobs table
-- ============================================================================

CREATE TABLE IF NOT EXISTS background_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_type VARCHAR(100) NOT NULL, -- 'cleanup_orphaned_file', 'convert_video_to_audio', 'generate_waveform'
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'pending', 'processing', 'completed', 'failed'
    data JSONB NOT NULL, -- job-specific data (file paths, user_id, etc)
    result JSONB, -- job result/output
    error_message TEXT,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    attempts INT DEFAULT 0,
    max_attempts INT DEFAULT 3,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);

-- Create indexes for job processing
CREATE INDEX IF NOT EXISTS idx_background_jobs_status ON background_jobs(status);
CREATE INDEX IF NOT EXISTS idx_background_jobs_job_type ON background_jobs(job_type);
CREATE INDEX IF NOT EXISTS idx_background_jobs_created_at ON background_jobs(created_at DESC);

-- View: Pending jobs (for worker to process)
CREATE OR REPLACE VIEW pending_jobs AS
SELECT id, job_type, data, retry_count, max_retries
FROM background_jobs
WHERE status = 'pending' AND retry_count < max_retries
ORDER BY created_at ASC
LIMIT 100;

-- ============================================================================
-- MIGRATION 036: Fix background_jobs columns alignment
-- ============================================================================

ALTER TABLE background_jobs
  ADD COLUMN IF NOT EXISTS attempts INT DEFAULT 0,
  ADD COLUMN IF NOT EXISTS max_attempts INT DEFAULT 3,
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

UPDATE background_jobs
SET attempts = COALESCE(attempts, retry_count, 0),
    max_attempts = COALESCE(max_attempts, max_retries, 3),
    updated_at = COALESCE(updated_at, created_at, NOW());

-- ============================================================================
-- All migrations complete! ✅
-- ============================================================================
