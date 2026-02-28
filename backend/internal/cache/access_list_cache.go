package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// AccessListCache provides high-performance caching for user access lists
// Dramatically reduces database load for RLS policy evaluation
type AccessListCache struct {
	redis *redis.Client
	log   *slog.Logger
}

// NewAccessListCache creates a cache specifically for access list performance
func NewAccessListCache(redisClient *redis.Client, log *slog.Logger) *AccessListCache {
	return &AccessListCache{
		redis: redisClient,
		log:   log,
	}
}

// CacheUserProjectAccessList stores list of projects user can access (5 minute TTL)
// Key: user:{userID}:project_access
// Value: JSON array of project IDs
// Invalidation: On INSERT to collaborators, project_shares, or projects ownership change
func (alc *AccessListCache) CacheUserProjectAccessList(ctx context.Context, userID string, projectIDs []string) error {
	data, err := json.Marshal(projectIDs)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("user:%s:project_access", userID)
	// 5 minute TTL - balance between consistency and database load
	err = alc.redis.Set(ctx, key, data, 5*time.Minute).Err()
	if err != nil {
		alc.log.Warn("failed to cache project access list", "error", err, "user_id", userID)
		// Continue - cache miss is not fatal
	}
	return nil
}

// GetUserProjectAccessList retrieves cached project access list if available
// Returns ([]string, true) if cache hit, (nil, false) if cache miss
func (alc *AccessListCache) GetUserProjectAccessList(ctx context.Context, userID string) ([]string, bool) {
	key := fmt.Sprintf("user:%s:project_access", userID)
	data, err := alc.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, false // Cache miss
	}
	if err != nil {
		alc.log.Warn("cache read error", "error", err, "user_id", userID)
		return nil, false
	}

	var projectIDs []string
	if err := json.Unmarshal([]byte(data), &projectIDs); err != nil {
		alc.log.Warn("failed to deserialize project access list", "error", err)
		return nil, false
	}
	return projectIDs, true
}

// CacheUserTrackAccessList stores list of tracks user can access (3 minute TTL)
// Key: user:{userID}:track_access
// Value: JSON array of track IDs
// Invalidation: On changes to track ownership or project access
func (alc *AccessListCache) CacheUserTrackAccessList(ctx context.Context, userID string, trackIDs []string) error {
	data, err := json.Marshal(trackIDs)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("user:%s:track_access", userID)
	// 3 minute TTL - shorter since tracks change more frequently
	err = alc.redis.Set(ctx, key, data, 3*time.Minute).Err()
	if err != nil {
		alc.log.Warn("failed to cache track access list", "error", err, "user_id", userID)
	}
	return nil
}

// GetUserTrackAccessList retrieves cached track access list if available
func (alc *AccessListCache) GetUserTrackAccessList(ctx context.Context, userID string) ([]string, bool) {
	key := fmt.Sprintf("user:%s:track_access", userID)
	data, err := alc.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, false
	}
	if err != nil {
		alc.log.Warn("cache read error", "error", err, "user_id", userID)
		return nil, false
	}

	var trackIDs []string
	if err := json.Unmarshal([]byte(data), &trackIDs); err != nil {
		alc.log.Warn("failed to deserialize track access list", "error", err)
		return nil, false
	}
	return trackIDs, true
}

// InvalidateUserAccessCache invalidates all cached access lists for a user
// Call this when:
// - User gains/loses project access
// - Project shares are created/revoked
// - Collaborators are added/removed
func (alc *AccessListCache) InvalidateUserAccessCache(ctx context.Context, userID string) error {
	keys := []string{
		fmt.Sprintf("user:%s:project_access", userID),
		fmt.Sprintf("user:%s:track_access", userID),
	}

	if err := alc.redis.Del(ctx, keys...).Err(); err != nil {
		alc.log.Warn("failed to invalidate user access cache", "error", err, "user_id", userID)
		// Continue - stale cache is preferable to broken functionality
	}
	return nil
}

// InvalidateProjectAccessCache invalidates access cache for all users with project access
// Used when project ownership changes or is deleted
// For large audiences, consider eventual consistency (let TTL expire)
func (alc *AccessListCache) InvalidateProjectAccessCache(ctx context.Context, affectedUserIDs []string) error {
	if len(affectedUserIDs) == 0 {
		return nil
	}

	// Build cache key list
	keys := make([]string, 0, len(affectedUserIDs)*2)
	for _, userID := range affectedUserIDs {
		keys = append(keys, fmt.Sprintf("user:%s:project_access", userID))
		keys = append(keys, fmt.Sprintf("user:%s:track_access", userID))
	}

	if err := alc.redis.Del(ctx, keys...).Err(); err != nil {
		alc.log.Warn("failed to invalidate project access cache",
			"error", err, "affected_users", len(affectedUserIDs))
	}
	return nil
}

// CacheProjectAccessibility stores whether a specific user can access a specific project
// Key: user:{userID}:access:project:{projectID}
// Value: "1" (true) or "0" (false)
// Very short TTL (1 minute) for fine-grained access changes
func (alc *AccessListCache) CacheProjectAccessibility(ctx context.Context, userID, projectID string, hasAccess bool) error {
	key := fmt.Sprintf("user:%s:access:project:%s", userID, projectID)
	value := "0"
	if hasAccess {
		value = "1"
	}

	return alc.redis.Set(ctx, key, value, 1*time.Minute).Err()
}

// GetProjectAccessibility retrieves cached accessibility for user-project pair
func (alc *AccessListCache) GetProjectAccessibility(ctx context.Context, userID, projectID string) (bool, bool) {
	key := fmt.Sprintf("user:%s:access:project:%s", userID, projectID)
	data, err := alc.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, false // Cache miss
	}
	if err != nil {
		return false, false
	}
	return data == "1", true // Return (hasAccess, cacheHit)
}

// PurgeLargeKeyPattern removes all keys matching a pattern (use sparingly - O(n) operation)
// Example: PurgeLargeKeyPattern(ctx, "user:*:project_access") to invalidate all project access caches
// NOTE: This is slow for large datasets. Prefer specific user invalidation where possible.
func (alc *AccessListCache) PurgeLargeKeyPattern(ctx context.Context, pattern string) error {
	iter := alc.redis.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := alc.redis.Del(ctx, iter.Val()).Err(); err != nil {
			alc.log.Warn("failed to delete key during purge", "error", err, "key", iter.Val())
		}
	}
	return iter.Err()
}

// CacheStats returns cache statistics for monitoring
func (alc *AccessListCache) CacheStats(ctx context.Context) (map[string]interface{}, error) {
	info, err := alc.redis.Info(ctx, "stats").Result()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"info": info,
	}, nil
}
