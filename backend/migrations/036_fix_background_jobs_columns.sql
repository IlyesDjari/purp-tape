-- Migration 038: Align background_jobs schema with runtime job processor expectations

ALTER TABLE background_jobs
  ADD COLUMN IF NOT EXISTS attempts INT DEFAULT 0,
  ADD COLUMN IF NOT EXISTS max_attempts INT DEFAULT 3,
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

UPDATE background_jobs
SET attempts = COALESCE(attempts, retry_count, 0),
    max_attempts = COALESCE(max_attempts, max_retries, 3),
    updated_at = COALESCE(updated_at, created_at, NOW());
