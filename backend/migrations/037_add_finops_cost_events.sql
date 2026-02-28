-- Migration 039: Add FinOps actual-cost ledger for budget governance

CREATE TABLE IF NOT EXISTS finops_cost_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source VARCHAR(64) NOT NULL,
    service VARCHAR(64) NOT NULL,
    category VARCHAR(64) NOT NULL,
    usd_amount NUMERIC(12, 4) NOT NULL CHECK (usd_amount >= 0),
    occurred_at TIMESTAMP NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_finops_cost_events_occurred_at
    ON finops_cost_events (occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_finops_cost_events_service_occurred
    ON finops_cost_events (service, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_finops_cost_events_source_occurred
    ON finops_cost_events (source, occurred_at DESC);
