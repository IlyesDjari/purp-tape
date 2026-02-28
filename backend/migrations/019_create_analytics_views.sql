-- Create analytics aggregation views for creator dashboards

-- View: Project statistics (plays, listeners, likes)
CREATE OR REPLACE VIEW project_statistics AS
SELECT
    p.id,
    p.name,
    p.user_id,
    COUNT(DISTINCT ph.id) as total_plays,
    COUNT(DISTINCT ph.listener_user_id) as unique_listeners,
    COUNT(DISTINCT pl.id) as total_likes,
    AVG(EXTRACT(EPOCH FROM (ph.ended_at - ph.started_at))) as avg_listen_duration_seconds,
    MAX(ph.started_at) as last_play_at,
    p.created_at,
    p.updated_at
FROM projects p
LEFT JOIN play_history ph ON p.id = ph.project_id
LEFT JOIN project_likes pl ON p.id = pl.project_id
GROUP BY p.id, p.name, p.user_id, p.created_at, p.updated_at;

-- View: Track statistics
CREATE OR REPLACE VIEW track_statistics AS
SELECT
    t.id,
    t.name,
    t.project_id,
    t.user_id,
    COUNT(DISTINCT ph.id) as total_plays,
    COUNT(DISTINCT ph.listener_user_id) as unique_listeners,
    COUNT(DISTINCT l.id) as total_likes,
    AVG(EXTRACT(EPOCH FROM (ph.ended_at - ph.started_at))) as avg_listen_duration_seconds,
    MAX(ph.started_at) as last_play_at
FROM tracks t
LEFT JOIN play_history ph ON t.id = ph.track_id
LEFT JOIN likes l ON t.id = l.track_id
GROUP BY t.id, t.name, t.project_id, t.user_id;

-- View: Daily engagement metrics
CREATE OR REPLACE VIEW daily_engagement AS
SELECT
    project_id,
    CAST(started_at AS DATE) as date,
    COUNT(*) as play_count,
    COUNT(DISTINCT listener_user_id) as unique_listeners,
    AVG(EXTRACT(EPOCH FROM (ended_at - started_at))) as avg_duration_seconds
FROM play_history
WHERE started_at > CURRENT_DATE - INTERVAL '90 days'
GROUP BY project_id, CAST(started_at AS DATE)
ORDER BY date DESC;

-- View: Top projects (trending)
CREATE OR REPLACE VIEW trending_projects AS
SELECT
    p.id,
    p.name,
    p.user_id,
    ps.total_plays,
    ps.unique_listeners,
    ps.total_likes,
    RANK() OVER (ORDER BY ps.total_plays DESC) as play_rank,
    RANK() OVER (ORDER BY ps.unique_listeners DESC) as listener_rank,
    RANK() OVER (ORDER BY ps.total_likes DESC) as likes_rank
FROM projects p
JOIN project_statistics ps ON p.id = ps.id
WHERE p.is_private = FALSE
ORDER BY ps.total_plays DESC;

-- View: Creator earnings/stats (for future payment integration)
CREATE OR REPLACE VIEW creator_stats AS
SELECT
    u.id,
    u.username,
    COUNT(DISTINCT p.id) as total_projects,
    COUNT(DISTINCT ph.id) as total_plays_received,
    COUNT(DISTINCT ph.listener_user_id) as total_unique_listeners,
    COUNT(DISTINCT uf.follower_id) as total_followers,
    s.tier as subscription_tier,
    s.is_premium
FROM users u
LEFT JOIN projects p ON u.id = p.user_id AND p.is_private = FALSE
LEFT JOIN play_history ph ON p.id = ph.project_id
LEFT JOIN user_follows uf ON u.id = uf.following_id
LEFT JOIN subscriptions s ON u.id = s.user_id
GROUP BY u.id, u.username, s.tier, s.is_premium;
