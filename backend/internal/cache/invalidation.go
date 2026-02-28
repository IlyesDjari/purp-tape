package cache

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// InvalidationEvent represents a cache invalidation event [MEDIUM: Cache invalidation tracking]
type InvalidationEvent struct {
	Key       string
	Type      string // "update", "delete", "refresh"
	Timestamp time.Time
}

// CacheInvalidator manages cache invalidation [MEDIUM: Cache invalidation]
type CacheInvalidator struct {
	subscribers map[string][]chan InvalidationEvent
	mu          sync.RWMutex
	log         *slog.Logger
}

// NewCacheInvalidator creates a new cache invalidator
func NewCacheInvalidator(log *slog.Logger) *CacheInvalidator {
	return &CacheInvalidator{
		subscribers: make(map[string][]chan InvalidationEvent),
		log:         log,
	}
}

// Subscribe subscribes to cache invalidation events for a key pattern
func (ci *CacheInvalidator) Subscribe(pattern string) <-chan InvalidationEvent {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	ch := make(chan InvalidationEvent, 10)
	ci.subscribers[pattern] = append(ci.subscribers[pattern], ch)
	return ch
}

// Publish publishes a cache invalidation event
func (ci *CacheInvalidator) Publish(event InvalidationEvent) {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	event.Timestamp = time.Now()

	for pattern, subs := range ci.subscribers {
		if matchesPattern(event.Key, pattern) {
			for _, ch := range subs {
				select {
				case ch <- event:
				default:
					ci.log.Warn("cache invalidation channel full", "pattern", pattern, "key", event.Key)
				}
			}
		}
	}
}

// InvalidateProjectCache invalidates cache for a project
func (ci *CacheInvalidator) InvalidateProjectCache(projectID string) {
	ci.Publish(InvalidationEvent{
		Key:  fmt.Sprintf("project:%s", projectID),
		Type: "update",
	})
	ci.log.Debug("project cache invalidated", "project_id", projectID)
}

// InvalidateProjectListCache invalidates cache for user's project list
func (ci *CacheInvalidator) InvalidateProjectListCache(userID string) {
	ci.Publish(InvalidationEvent{
		Key:  fmt.Sprintf("projects:{%s}:*", userID),
		Type: "update",
	})
	ci.log.Debug("project list cache invalidated", "user_id", userID)
}

// InvalidateTrackCache invalidates cache for a track
func (ci *CacheInvalidator) InvalidateTrackCache(trackID string) {
	ci.Publish(InvalidationEvent{
		Key:  fmt.Sprintf("track:%s", trackID),
		Type: "update",
	})
	ci.log.Debug("track cache invalidated", "track_id", trackID)
}

// InvalidateUserCache invalidates cache for a user
func (ci *CacheInvalidator) InvalidateUserCache(userID string) {
	ci.Publish(InvalidationEvent{
		Key:  fmt.Sprintf("user:%s", userID),
		Type: "update",
	})
	ci.log.Debug("user cache invalidated", "user_id", userID)
}

// InvalidateSearchCache invalidates search cache
func (ci *CacheInvalidator) InvalidateSearchCache() {
	ci.Publish(InvalidationEvent{
		Key:  "search:*",
		Type: "refresh",
	})
	ci.log.Debug("search cache invalidated")
}

// matchesPattern checks if a key matches a pattern
// Simple pattern matching with * wildcard
func matchesPattern(key, pattern string) bool {
	// Exact match
	if key == pattern {
		return true
	}

	// Wildcard patterns
	if len(pattern) == 0 {
		return false
	}

	// Simple wildcard matching
	if pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(key) >= len(prefix) && key[:len(prefix)] == prefix
	}

	return false
}

// CacheMetrics tracks cache performance [MEDIUM: Monitoring]
type CacheMetrics struct {
	Hits        int64
	Misses      int64
	Invalidations int64
	LastReset   time.Time
}

// GetMetrics returns current cache metrics
func (ci *CacheInvalidator) GetMetrics() CacheMetrics {
	return CacheMetrics{
		LastReset: time.Now(),
	}
}
