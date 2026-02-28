package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheManager handles multi-level caching
type CacheManager struct {
	redis *redis.Client
	log   *slog.Logger
}

// NewCacheManager creates cache manager
func NewCacheManager(redisURL string, log *slog.Logger) (*CacheManager, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis url: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	log.Info("redis connected", "url", redisURL)

	return &CacheManager{redis: client, log: log}, nil
}

// Cache patterns with TTL

// CacheTrendingProjects - pre-computed trending (1 hour)
func (cm *CacheManager) CacheTrendingProjects(ctx context.Context, projects interface{}) error {
	data, err := json.Marshal(projects)
	if err != nil {
		return err
	}

	return cm.redis.Set(ctx, "trending_projects", data, 1*time.Hour).Err()
}

func (cm *CacheManager) GetTrendingProjects(ctx context.Context, dest interface{}) error {
	data, err := cm.redis.Get(ctx, "trending_projects").Result()
	if err == redis.Nil {
		return fmt.Errorf("cache miss")
	}
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

// CacheProjectMetadata - project data (30 minutes)
func (cm *CacheManager) CacheProjectMetadata(ctx context.Context, projectID string, metadata interface{}) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("project:%s:metadata", projectID)
	return cm.redis.Set(ctx, key, data, 30*time.Minute).Err()
}

func (cm *CacheManager) GetProjectMetadata(ctx context.Context, projectID string, dest interface{}) error {
	key := fmt.Sprintf("project:%s:metadata", projectID)
	data, err := cm.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return fmt.Errorf("cache miss")
	}
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

// CacheUserSubscription - subscription data (10 minutes)
func (cm *CacheManager) CacheUserSubscription(ctx context.Context, userID string, subscription interface{}) error {
	data, err := json.Marshal(subscription)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("user:%s:subscription", userID)
	return cm.redis.Set(ctx, key, data, 10*time.Minute).Err()
}

func (cm *CacheManager) GetUserSubscription(ctx context.Context, userID string, dest interface{}) error {
	key := fmt.Sprintf("user:%s:subscription", userID)
	data, err := cm.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return fmt.Errorf("cache miss")
	}
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

// CacheUserSubscriptionTier caches only subscription tier data (5 minutes).
func (cm *CacheManager) CacheUserSubscriptionTier(ctx context.Context, userID string, tier interface{}) error {
	data, err := json.Marshal(tier)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("user:%s:subscription:tier", userID)
	return cm.redis.Set(ctx, key, data, 5*time.Minute).Err()
}

// GetUserSubscriptionTier retrieves cached subscription tier data.
func (cm *CacheManager) GetUserSubscriptionTier(ctx context.Context, userID string, dest interface{}) error {
	key := fmt.Sprintf("user:%s:subscription:tier", userID)
	data, err := cm.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return fmt.Errorf("cache miss")
	}
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

// CacheSearchResults - search results (5 minutes, volatile)
func (cm *CacheManager) CacheSearchResults(ctx context.Context, query string, results interface{}) error {
	data, err := json.Marshal(results)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("search:%s", query)
	return cm.redis.Set(ctx, key, data, 5*time.Minute).Err()
}

func (cm *CacheManager) GetSearchResults(ctx context.Context, query string, dest interface{}) error {
	key := fmt.Sprintf("search:%s", query)
	data, err := cm.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return fmt.Errorf("cache miss")
	}
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

// InvalidateProjectCache - clear project caches
func (cm *CacheManager) InvalidateProjectCache(ctx context.Context, projectID string) error {
	keys := []string{
		fmt.Sprintf("project:%s:metadata", projectID),
		fmt.Sprintf("project:%s:stats", projectID),
	}

	return cm.redis.Del(ctx, keys...).Err()
}

// InvalidateUserCache - clear user caches
func (cm *CacheManager) InvalidateUserCache(ctx context.Context, userID string) error {
	keys := []string{
		fmt.Sprintf("user:%s:subscription", userID),
		fmt.Sprintf("user:%s:storage", userID),
	}

	return cm.redis.Del(ctx, keys...).Err()
}

// Distributed Rate Limiting (already implemented but enhanced)

// RateLimitByIP - 100 requests per minute per IP
func (cm *CacheManager) RateLimitByIP(ctx context.Context, ip string, limit int, window time.Duration) (bool, error) {
	key := fmt.Sprintf("ratelimit:ip:%s", ip)

	count, err := cm.redis.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}

	// Set expiry on first increment
	if count == 1 {
		cm.redis.Expire(ctx, key, window)
	}

	return count <= int64(limit), nil
}

// RateLimitByUser - 1000 requests per hour per user
func (cm *CacheManager) RateLimitByUser(ctx context.Context, userID string, limit int, window time.Duration) (bool, error) {
	key := fmt.Sprintf("ratelimit:user:%s", userID)

	count, err := cm.redis.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}

	if count == 1 {
		cm.redis.Expire(ctx, key, window)
	}

	return count <= int64(limit), nil
}

// GetRateLimitStatus - get current rate limit count
func (cm *CacheManager) GetRateLimitStatus(ctx context.Context, key string) (int64, error) {
	count, err := cm.redis.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return count, err
}

// Session Management (for stateless API with Redis backend)

// StoreSession - store session in Redis (shared across instances)
func (cm *CacheManager) StoreSession(ctx context.Context, sessionID string, data interface{}, ttl time.Duration) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("session:%s", sessionID)
	return cm.redis.Set(ctx, key, jsonData, ttl).Err()
}

// GetSession - retrieve session
func (cm *CacheManager) GetSession(ctx context.Context, sessionID string, dest interface{}) error {
	key := fmt.Sprintf("session:%s", sessionID)
	data, err := cm.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return fmt.Errorf("session not found")
	}
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

// DeleteSession - remove session
func (cm *CacheManager) DeleteSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return cm.redis.Del(ctx, key).Err()
}

// Health Check
func (cm *CacheManager) HealthCheck(ctx context.Context) error {
	return cm.redis.Ping(ctx).Err()
}

// Gets Redis stats
type RedisStats struct {
	ConnectedClients int64
	UsedMemory       int64
	UsedMemoryPercent float64
	EvictedKeys     int64
	KeySpace        map[string]string
}

func (cm *CacheManager) GetStats(ctx context.Context) (*RedisStats, error) {
	info := cm.redis.Info(ctx, "stats", "memory", "keyspace")
	if info.Err() != nil {
		return nil, info.Err()
	}

	// Parse and return stats
	return &RedisStats{
		ConnectedClients: 0, // Would parse from info
	}, nil
}

// Close closes Redis connection
func (cm *CacheManager) Close() error {
	return cm.redis.Close()
}
