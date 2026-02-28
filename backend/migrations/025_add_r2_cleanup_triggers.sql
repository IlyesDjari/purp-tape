-- Migration: Add R2 file cleanup on delete
-- Automatically queues cleanup jobs when files are deleted
-- This prevents orphaned files from accumulating storage costs

-- ============================================================================
-- Ensure background_jobs table exists with required schema
-- ============================================================================

CREATE TABLE IF NOT EXISTS background_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_type VARCHAR(100) NOT NULL,
    data JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    result JSONB,
    error_message TEXT,
    attempts INT DEFAULT 0,
    max_attempts INT DEFAULT 3,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_background_jobs_status ON background_jobs(status) WHERE status IN ('pending', 'processing');
CREATE INDEX IF NOT EXISTS idx_background_jobs_created_at ON background_jobs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_background_jobs_job_type ON background_jobs(job_type);

-- ============================================================================
-- Trigger: Queue R2 cleanup when track versions are deleted
-- ============================================================================

CREATE OR REPLACE FUNCTION queue_r2_cleanup_on_track_version_delete()
RETURNS TRIGGER AS $$
BEGIN
    -- Only queue if file actually exists
    IF OLD.r2_object_key IS NOT NULL THEN
        INSERT INTO background_jobs (id, job_type, data, status, created_at)
        VALUES (
            gen_random_uuid(),
            'cleanup_r2_file',
            jsonb_build_object(
                'r2_object_key', OLD.r2_object_key,
                'file_size', OLD.file_size,
                'track_id', OLD.track_id
            ),
            'pending',
            CURRENT_TIMESTAMP
        );
    END IF;

    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_queue_r2_cleanup_on_track_version_delete ON track_versions;
CREATE TRIGGER trigger_queue_r2_cleanup_on_track_version_delete
AFTER DELETE ON track_versions
FOR EACH ROW
EXECUTE FUNCTION queue_r2_cleanup_on_track_version_delete();

-- ============================================================================
-- Trigger: Queue R2 cleanup when images are deleted
-- ============================================================================

CREATE OR REPLACE FUNCTION queue_r2_cleanup_on_image_delete()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.r2_object_key IS NOT NULL THEN
        INSERT INTO background_jobs (id, job_type, data, status, created_at)
        VALUES (
            gen_random_uuid(),
            'cleanup_r2_file',
            jsonb_build_object(
                'r2_object_key', OLD.r2_object_key,
                'file_size', OLD.file_size,
                'image_id', OLD.id,
                'user_id', OLD.user_id
            ),
            'pending',
            CURRENT_TIMESTAMP
        );
    END IF;

    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_queue_r2_cleanup_on_image_delete ON images;
CREATE TRIGGER trigger_queue_r2_cleanup_on_image_delete
AFTER DELETE ON images
FOR EACH ROW
EXECUTE FUNCTION queue_r2_cleanup_on_image_delete();

-- ============================================================================
-- Trigger: Queue bulk cleanup when projects are soft-deleted
-- ============================================================================

CREATE OR REPLACE FUNCTION queue_r2_cleanup_on_project_soft_delete()
RETURNS TRIGGER AS $$
BEGIN
    -- Only when transitioning to deleted state
    IF NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL THEN
        -- Queue cleanup of all associated track version files
        INSERT INTO background_jobs (id, job_type, data, status, created_at)
        SELECT
            gen_random_uuid(),
            'cleanup_r2_file',
            jsonb_build_object(
                'r2_object_key', tv.r2_object_key,
                'file_size', tv.file_size,
                'track_id', tv.track_id
            ),
            'pending',
            CURRENT_TIMESTAMP
        FROM track_versions tv
        JOIN tracks t ON tv.track_id = t.id
        WHERE t.project_id = NEW.id
        AND tv.r2_object_key IS NOT NULL;

        -- Queue cleanup of project cover image if exists
        IF NEW.cover_image_id IS NOT NULL THEN
            INSERT INTO background_jobs (id, job_type, data, status, created_at)
            SELECT
                gen_random_uuid(),
                'cleanup_r2_file',
                jsonb_build_object(
                    'r2_object_key', i.r2_object_key,
                    'file_size', i.file_size,
                    'image_id', i.id
                ),
                'pending',
                CURRENT_TIMESTAMP
            FROM images i
            WHERE i.id = NEW.cover_image_id;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_queue_r2_cleanup_on_project_soft_delete ON projects;
CREATE TRIGGER trigger_queue_r2_cleanup_on_project_soft_delete
AFTER UPDATE ON projects
FOR EACH ROW
EXECUTE FUNCTION queue_r2_cleanup_on_project_soft_delete();

-- ============================================================================
-- Helper function: Get pending R2 cleanup jobs
-- ============================================================================

CREATE OR REPLACE FUNCTION get_pending_r2_cleanup_jobs(limit_count INT DEFAULT 10)
RETURNS TABLE (
    job_id UUID,
    r2_object_key VARCHAR(512),
    file_size BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        bj.id,
        (bj.data->>'r2_object_key')::VARCHAR(512),
        (bj.data->>'file_size')::BIGINT
    FROM background_jobs bj
    WHERE bj.job_type = 'cleanup_r2_file'
    AND bj.status = 'pending'
    AND bj.attempts < bj.max_attempts
    ORDER BY bj.created_at ASC
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Helper function: Mark job as completed
-- ============================================================================

CREATE OR REPLACE FUNCTION mark_job_completed(job_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    UPDATE background_jobs
    SET status = 'completed', completed_at = CURRENT_TIMESTAMP
    WHERE id = job_id;

    RETURN true;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Helper function: Mark job as failed and increment attempt count
-- ============================================================================

CREATE OR REPLACE FUNCTION mark_job_failed(job_id UUID, error_msg TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    UPDATE background_jobs
    SET
        status = CASE
            WHEN (attempts + 1) >= max_attempts THEN 'failed'
            ELSE 'pending'
        END,
        attempts = attempts + 1,
        error_message = error_msg,
        started_at = NULL
    WHERE id = job_id;

    RETURN true;
END;
$$ LANGUAGE plpgsql;
