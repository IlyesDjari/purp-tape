-- Create user subscription/premium status table
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    is_premium BOOLEAN DEFAULT FALSE,
    tier VARCHAR(50) DEFAULT 'free', -- 'free', 'pro', 'pro_plus', 'unlimited'
    storage_quota_mb BIGINT DEFAULT 10240, -- default 10GB
    storage_used_mb BIGINT DEFAULT 0,
    stripe_customer_id VARCHAR(255),
    stripe_subscription_id VARCHAR(255),
    purchase_date TIMESTAMP,
    renewal_date TIMESTAMP,
    canceled_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_stripe_customer_id ON subscriptions(stripe_customer_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_is_premium ON subscriptions(is_premium);
