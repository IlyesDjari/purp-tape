-- Migration 043: Granular Cost Attribution for Per-User/Per-Project Chargeback
-- Enables detailed cost tracking by user and project for billing and optimization

-- ============================================================================
-- COST ATTRIBUTION TABLE
-- ============================================================================
-- Tracks storage costs attributable to specific users and projects
-- Populated daily by cleanup/harvest jobs from finops_cost_events
CREATE TABLE IF NOT EXISTS cost_attribution (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    project_id UUID NOT NULL,
    time_period DATE NOT NULL,  -- YYYY-MM-DD format, day granularity
    storage_gb_hours NUMERIC(14, 4) NOT NULL DEFAULT 0,
    storage_cost_usd NUMERIC(12, 4) NOT NULL DEFAULT 0,
    api_call_count INT NOT NULL DEFAULT 0,
    api_cost_usd NUMERIC(12, 4) NOT NULL DEFAULT 0,
    transfer_gb INT NOT NULL DEFAULT 0,
    transfer_cost_usd NUMERIC(12, 4) NOT NULL DEFAULT 0,
    total_cost_usd NUMERIC(12, 4) GENERATED ALWAYS AS (storage_cost_usd + api_cost_usd + transfer_cost_usd) STORED,
    created_at TIMESTAMP DEFAULT now(),
    CONSTRAINT fk_cost_attribution_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_cost_attribution_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    UNIQUE(user_id, project_id, time_period)
);

CREATE INDEX idx_cost_attribution_user_period ON cost_attribution(user_id, time_period DESC);
CREATE INDEX idx_cost_attribution_project_period ON cost_attribution(project_id, time_period DESC);
CREATE INDEX idx_cost_attribution_time_period ON cost_attribution(time_period DESC);

-- ============================================================================
-- MATERIALIZED VIEW: Monthly costs per user
-- ============================================================================
-- Refresh daily for accurate billing
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_user_monthly_costs AS
SELECT 
    user_id,
    DATE_TRUNC('month', time_period)::DATE as billing_month,
    SUM(storage_cost_usd) as total_storage_cost_usd,
    SUM(api_cost_usd) as total_api_cost_usd,
    SUM(transfer_cost_usd) as total_transfer_cost_usd,
    SUM(total_cost_usd) as total_cost_usd,
    COUNT(DISTINCT project_id) as projects_count,
    DATE_TRUNC('month', MAX(time_period))::DATE as last_activity_date
FROM cost_attribution
WHERE time_period >= CURRENT_DATE - INTERVAL '12 months'
GROUP BY user_id, DATE_TRUNC('month', time_period)
ORDER BY user_id, billing_month DESC;

CREATE INDEX idx_mv_user_monthly_costs_user ON mv_user_monthly_costs(user_id, billing_month DESC);
CREATE INDEX idx_mv_user_monthly_costs_month ON mv_user_monthly_costs(billing_month DESC);

-- ============================================================================
-- MATERIALIZED VIEW: Monthly costs per project
-- ============================================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_project_monthly_costs AS
SELECT 
    project_id,
    user_id,
    DATE_TRUNC('month', time_period)::DATE as billing_month,
    SUM(storage_cost_usd) as total_storage_cost_usd,
    SUM(api_cost_usd) as total_api_cost_usd,
    SUM(transfer_cost_usd) as total_transfer_cost_usd,
    SUM(total_cost_usd) as total_cost_usd,
    SUM(CAST(storage_gb_hours AS FLOAT) / 24) as avg_storage_gb_daily
FROM cost_attribution
WHERE time_period >= CURRENT_DATE - INTERVAL '12 months'
GROUP BY project_id, user_id, DATE_TRUNC('month', time_period)
ORDER BY project_id, billing_month DESC;

CREATE INDEX idx_mv_project_monthly_costs_project ON mv_project_monthly_costs(project_id, billing_month DESC);
CREATE INDEX idx_mv_project_monthly_costs_user ON mv_project_monthly_costs(user_id, billing_month DESC);

-- ============================================================================
-- VIEW: Current month costs (real-time from cost_attribution)
-- ============================================================================
CREATE OR REPLACE VIEW v_current_month_user_costs AS
SELECT 
    user_id,
    SUM(storage_cost_usd) as storage_cost_usd,
    SUM(api_cost_usd) as api_cost_usd,
    SUM(transfer_cost_usd) as transfer_cost_usd,
    SUM(total_cost_usd) as total_cost_usd,
    COUNT(DISTINCT project_id) as projects_count,
    MAX(time_period) as last_updated
FROM cost_attribution
WHERE time_period >= DATE_TRUNC('month', CURRENT_DATE)::DATE
GROUP BY user_id;

CREATE OR REPLACE VIEW v_current_month_project_costs AS
SELECT 
    project_id,
    user_id,
    SUM(storage_cost_usd) as storage_cost_usd,
    SUM(api_cost_usd) as api_cost_usd,
    SUM(transfer_cost_usd) as transfer_cost_usd,
    SUM(total_cost_usd) as total_cost_usd,
    MAX(time_period) as last_updated
FROM cost_attribution
WHERE time_period >= DATE_TRUNC('month', CURRENT_DATE)::DATE
GROUP BY project_id, user_id;

-- ============================================================================
-- FUNCTION: Calculate user's current month invoiceable amount
-- ============================================================================
CREATE OR REPLACE FUNCTION user_current_month_cost(p_user_id UUID)
RETURNS NUMERIC AS $$
    SELECT COALESCE(SUM(total_cost_usd), 0::NUMERIC)
    FROM cost_attribution
    WHERE user_id = p_user_id 
    AND time_period >= DATE_TRUNC('month', CURRENT_DATE)::DATE;
$$ LANGUAGE sql STABLE;

-- ============================================================================
-- FUNCTION: Calculate project's current month cost
-- ============================================================================
CREATE OR REPLACE FUNCTION project_current_month_cost(p_project_id UUID)
RETURNS NUMERIC AS $$
    SELECT COALESCE(SUM(total_cost_usd), 0::NUMERIC)
    FROM cost_attribution
    WHERE project_id = p_project_id
    AND time_period >= DATE_TRUNC('month', CURRENT_DATE)::DATE;
$$ LANGUAGE sql STABLE;

-- ============================================================================
-- FUNCTION: Get top cost drivers for user
-- ============================================================================
CREATE OR REPLACE FUNCTION user_cost_breakdown(p_user_id UUID, p_month DATE DEFAULT CURRENT_DATE)
RETURNS TABLE (
    project_id UUID,
    project_name VARCHAR,
    storage_cost_usd NUMERIC,
    api_cost_usd NUMERIC,
    transfer_cost_usd NUMERIC,
    total_cost_usd NUMERIC,
    cost_percentage NUMERIC
) AS $$
    WITH user_costs AS (
        SELECT 
            ca.project_id,
            p.name,
            SUM(ca.storage_cost_usd) as storage_cost,
            SUM(ca.api_cost_usd) as api_cost,
            SUM(ca.transfer_cost_usd) as transfer_cost,
            SUM(ca.total_cost_usd) as total_cost
        FROM cost_attribution ca
        JOIN projects p ON p.id = ca.project_id
        WHERE ca.user_id = p_user_id
        AND DATE_TRUNC('month', ca.time_period)::DATE = DATE_TRUNC('month', p_month)::DATE
        GROUP BY ca.project_id, p.name
    ),
    user_total AS (
        SELECT SUM(total_cost) as total FROM user_costs
    )
    SELECT 
        uc.project_id,
        uc.name,
        uc.storage_cost,
        uc.api_cost,
        uc.transfer_cost,
        uc.total_cost,
        ROUND(CASE 
            WHEN ut.total > 0 THEN (uc.total_cost / ut.total * 100)
            ELSE 0
        END, 2)
    FROM user_costs uc, user_total ut
    ORDER BY uc.total_cost DESC;
$$ LANGUAGE sql STABLE;

-- ============================================================================
-- FUNCTION: Get top cost drivers for project
-- ============================================================================
CREATE OR REPLACE FUNCTION project_cost_details(p_project_id UUID, p_month DATE DEFAULT CURRENT_DATE)
RETURNS TABLE (
    time_period DATE,
    storage_cost_usd NUMERIC,
    api_cost_usd NUMERIC,
    transfer_cost_usd NUMERIC,
    total_cost_usd NUMERIC,
    storage_gb_hours NUMERIC
) AS $$
    SELECT 
        time_period,
        storage_cost_usd,
        api_cost_usd,
        transfer_cost_usd,
        total_cost_usd,
        storage_gb_hours
    FROM cost_attribution
    WHERE project_id = p_project_id
    AND DATE_TRUNC('month', time_period)::DATE = DATE_TRUNC('month', p_month)::DATE
    ORDER BY time_period DESC;
$$ LANGUAGE sql STABLE;

-- ============================================================================
-- TRIGGER: Automatically update user's cost cache when attribution changes
-- ============================================================================
CREATE OR REPLACE FUNCTION refresh_user_cost_cache()
RETURNS TRIGGER AS $$
BEGIN
    -- In production, you might update a materialized view or push to analytics
    -- For now, this is a placeholder for cache invalidation logic
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_cost_attribution_change
    AFTER INSERT OR UPDATE OR DELETE ON cost_attribution
    FOR EACH ROW
    EXECUTE FUNCTION refresh_user_cost_cache();

-- ============================================================================
-- POPULATION FUNCTION: Calculate and insert daily cost attribution
-- ============================================================================
-- Run this daily via background job to populate cost_attribution from raw events
CREATE OR REPLACE FUNCTION populate_daily_cost_attribution(p_date DATE)
RETURNS INT AS $$
DECLARE
    v_inserted INT := 0;
BEGIN
    -- This function would:
    -- 1. Query active projects and users from p_date
    -- 2. Calculate storage costs from R2 metrics
    -- 3. Allocate bandwidth/API costs to projects
    -- 4. Insert rows into cost_attribution
    
    -- Placeholder implementation - real version needs integration with finops_cost_events
    -- and project storage metrics
    
    RETURN v_inserted;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- STORED PROCEDURE: Generate invoice for user for given month
-- ============================================================================
CREATE OR REPLACE FUNCTION generate_user_invoice(
    p_user_id UUID,
    p_month DATE
)
RETURNS TABLE (
    invoice_month VARCHAR,
    total_amount_usd NUMERIC,
    storage_amount_usd NUMERIC,
    api_amount_usd NUMERIC,
    transfer_amount_usd NUMERIC,
    breakdown TEXT
) AS $$
    SELECT 
        TO_CHAR(DATE_TRUNC('month', ca.time_period)::DATE, 'YYYY-MM'),
        SUM(ca.total_cost_usd),
        SUM(ca.storage_cost_usd),
        SUM(ca.api_cost_usd),
        SUM(ca.transfer_cost_usd),
        JSON_AGG(
            JSON_BUILD_OBJECT(
                'project_id', ca.project_id,
                'storage_cost', ca.storage_cost_usd,
                'api_cost', ca.api_cost_usd,
                'transfer_cost', ca.transfer_cost_usd,
                'total', ca.total_cost_usd
            )
            ORDER BY ca.total_cost_usd DESC
        )::TEXT
    FROM cost_attribution ca
    WHERE ca.user_id = p_user_id
    AND DATE_TRUNC('month', ca.time_period)::DATE = DATE_TRUNC('month', p_month)::DATE
    GROUP BY DATE_TRUNC('month', ca.time_period);
$$ LANGUAGE sql STABLE;

-- ============================================================================
-- MATERIALIZED VIEW REFRESH FUNCTIONS
-- ============================================================================
-- Run these daily from job processor:
-- SELECT refresh_user_monthly_costs();
-- SELECT refresh_project_monthly_costs();

CREATE OR REPLACE FUNCTION refresh_user_monthly_costs()
RETURNS TEXT AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_user_monthly_costs;
    RETURN 'User monthly costs refreshed';
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION refresh_project_monthly_costs()
RETURNS TEXT AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_project_monthly_costs;
    RETURN 'Project monthly costs refreshed';
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- INITIAL DATA: Populate cost_attribution with sample data for testing
-- ============================================================================
-- In production, this is populated by populate_daily_cost_attribution() job
-- Commented out for safety:
-- INSERT INTO cost_attribution (user_id, project_id, time_period, storage_gb_hours, storage_cost_usd)
-- SELECT u.id, p.id, CURRENT_DATE, 100, 1.50
-- FROM users u, projects p
-- WHERE p.user_id = u.id
-- LIMIT 10;
