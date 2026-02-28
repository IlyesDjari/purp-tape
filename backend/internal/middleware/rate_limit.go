package middleware

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimitEntry tracks request count and window start time.
type RateLimitEntry struct {
	count     uint32    // Request count in current window
	windowStart time.Time // When the current window started
}

// RateLimiter tracks requests per IP/user with memory-efficient design
type RateLimiter struct {
	requests map[string]*RateLimitEntry
	mu       sync.RWMutex
	log      *slog.Logger
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(log *slog.Logger) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*RateLimitEntry),
		log:      log,
	}

	// Cleanup old entries every minute
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

// isAllowed checks if request is within rate limit (100 requests per minute per IP).
func (rl *RateLimiter) isAllowed(identifier string) bool {
	return rl.isAllowedWithLimit(identifier, 100)
}

func (rl *RateLimiter) isAllowedWithLimit(identifier string, limitPerMinute uint32) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	const windowDuration = time.Minute

	// Get or create entry
	entry, exists := rl.requests[identifier]
	if !exists {
		rl.requests[identifier] = &RateLimitEntry{
			count:       1,
			windowStart: now,
		}
		return true
	}

	// Check if window has expired
	if now.Sub(entry.windowStart) > windowDuration {
		// Reset window
		entry.count = 1
		entry.windowStart = now
		return true
	}

	// Check limit
	if entry.count >= limitPerMinute {
		return false
	}

	// Increment counter
	entry.count++
	return true
}

// cleanup removes old entries to prevent memory leaks.
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	// Remove entries older than 2 minutes (more aggressive)
	cutoff := now.Add(-2 * time.Minute)

	for identifier, entry := range rl.requests {
		if entry.windowStart.Before(cutoff) {
			delete(rl.requests, identifier)
		}
	}
}

// RateLimitMiddleware enforces rate limiting (100 requests per minute per IP).
func RateLimitMiddleware(rl *RateLimiter, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client IP
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}

			// Check X-Forwarded-For header (for proxied requests)
			if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
				ip = forwardedFor
			}

			// Check rate limit
			if !rl.isAllowed(ip) {
				log.Warn("rate limit exceeded", "ip", ip)
				w.Header().Set("Retry-After", "60")
				http.Error(w, "Too many requests. Maximum 100 requests per minute.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// StrictRateLimitMiddleware for sensitive endpoints (10 requests per minute per IP)
func StrictRateLimitMiddleware(rl *RateLimiter, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}

			if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
				ip = forwardedFor
			}

			// Custom limit for strict endpoints
			const strictLimit = 10
			allowed := rl.isAllowedWithLimit(ip, strictLimit)

			if !allowed {
				log.Warn("strict rate limit exceeded", "ip", ip)
				w.Header().Set("Retry-After", "60")
				http.Error(w, fmt.Sprintf("Too many requests. Maximum %d requests per minute.", strictLimit), http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
