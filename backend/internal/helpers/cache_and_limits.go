package helpers

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// RequestCacheKey defines a type for context cache keys
type RequestCacheKey string

const (
	userRoleCacheKey     RequestCacheKey = "cached_user_role"
	projectAccessCacheKey RequestCacheKey = "cached_project_access"
)

// CacheInContext stores a value in request context for reuse within same request
// NICE-TO-HAVE: Prevents redundant DB queries for same data within one request
func CacheInContext(ctx context.Context, key RequestCacheKey, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
}

// GetFromCache retrieves a cached value from context
func GetFromCache[T any](ctx context.Context, key RequestCacheKey) (T, bool) {
	val, ok := ctx.Value(key).(T)
	return val, ok
}

// CacheResult wraps the result of a query and caches it
func CacheResult[T any](ctx context.Context, key RequestCacheKey, result T) (T, context.Context) {
	return result, CacheInContext(ctx, key, result)
}

// ExtractPaginationParamsCached extracts limit and offset from query parameters
// Defaults: limit=20, max=100; offset=0
func ExtractPaginationParamsCached(r *http.Request) (int, int) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20 // default
	offset := 0

	if limitStr != "" {
		if l, err := parseIntParam(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := parseIntParam(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	return limit, offset
}

func parseIntParam(s string) (int, error) {
	var num int
	_, err := fmt.Sscanf(s, "%d", &num)
	return num, err
}

// CacheWithTTL stores data in context with TTL tracking
type CachedValue struct {
	Data      interface{}
	ExpiresAt time.Time
}

// IsCacheExpired checks if cached data has expired
func IsCacheExpired(cached *CachedValue) bool {
	return time.Now().After(cached.ExpiresAt)
}

// RateLimitKey for rate limiting by user or IP
type RateLimitKey string

// ExtractRateLimitKey gets user ID or IP for rate limiting
func ExtractRateLimitKey(r *http.Request) RateLimitKey {
	// Prefer user ID from context
	if userID, ok := r.Context().Value("user_id").(string); ok && userID != "" {
		return RateLimitKey("user:" + userID)
	}

	// Fall back to IP address
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			ip = host
		} else {
			ip = r.RemoteAddr
		}
	}
	return RateLimitKey("ip:" + ip)
}
