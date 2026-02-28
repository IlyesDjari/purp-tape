package cache

import (
	"sync"
	"time"
)

// PresignedURLCache stores cached presigned URLs with expiration
// ✅ COST OPTIMIZATION: Reduces R2 API calls by caching presigned URLs
type PresignedURLCache struct {
	mu       sync.RWMutex
	cache    map[string]cacheEntry
	ttl      time.Duration
	stopChan chan struct{}
}

type cacheEntry struct {
	URL       string
	ExpiresAt time.Time
}

// NewPresignedURLCache creates a new cache with specified TTL
func NewPresignedURLCache(ttl time.Duration) *PresignedURLCache {
	cache := &PresignedURLCache{
		cache:    make(map[string]cacheEntry),
		ttl:      ttl,
		stopChan: make(chan struct{}),
	}

	// Start cleanup goroutine to remove expired entries every 5 minutes
	go cache.cleanupExpired()

	return cache
}

// Get retrieves a cached presigned URL if it exists and hasn't expired
func (c *PresignedURLCache) Get(objectKey string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[objectKey]
	if !exists {
		return "", false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		return "", false
	}

	return entry.URL, true
}

// Set stores a presigned URL in the cache with expiration
func (c *PresignedURLCache) Set(objectKey, url string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[objectKey] = cacheEntry{
		URL:       url,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// Clear removes all cached entries
func (c *PresignedURLCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]cacheEntry)
}

// Size returns the number of cached entries
func (c *PresignedURLCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.cache)
}

// cleanupExpired periodically removes expired entries
func (c *PresignedURLCache) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()

			now := time.Now()
			for key, entry := range c.cache {
				if now.After(entry.ExpiresAt) {
					delete(c.cache, key)
				}
			}

			c.mu.Unlock()

		case <-c.stopChan:
			return
		}
	}
}

// Stop gracefully shuts down the cleanup goroutine
func (c *PresignedURLCache) Stop() {
	close(c.stopChan)
}
