package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/IlyesDjari/purp-tape/backend/internal/config"
	"github.com/IlyesDjari/purp-tape/backend/internal/helpers"
)

// RateLimiter tracks request counts per key
// NICE-TO-HAVE: Add rate limiting to prevent abuse
type ConfigurableRateLimiter struct {
	cfg    *config.Config
	log    *slog.Logger
	mu     sync.RWMutex
	counts map[string]*configurableRequestTracker
}

type configurableRequestTracker struct {
	count     int
	resetAt   time.Time
	exceeds   int
}

// NewRateLimiter creates a new rate limiter
func NewConfigurableRateLimiter(cfg *config.Config, log *slog.Logger) *ConfigurableRateLimiter {
	rl := &ConfigurableRateLimiter{
		cfg:    cfg,
		log:    log,
		counts: make(map[string]*configurableRequestTracker),
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

// RateLimitMiddleware enforces request rate limiting
func (rl *ConfigurableRateLimiter) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := string(helpers.ExtractRateLimitKey(r))
		now := time.Now()

		rl.mu.Lock()
		tracker, exists := rl.counts[key]
		if !exists || now.After(tracker.resetAt) {
			tracker = &configurableRequestTracker{
				count:   1,
				resetAt: now.Add(rl.cfg.RateLimitWindow),
			}
			rl.counts[key] = tracker
			rl.mu.Unlock()
			next.ServeHTTP(w, r)
			return
		}

		tracker.count++
		if tracker.count > rl.cfg.RateLimitRequests {
			tracker.exceeds++
			rl.log.Warn("rate limit exceeded",
				"key", key,
				"count", tracker.count,
				"limit", rl.cfg.RateLimitRequests,
				"excess_count", tracker.exceeds)

			rl.mu.Unlock()

			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", rl.cfg.RateLimitWindow.Seconds()))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		rl.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

// cleanup removes expired entries
func (rl *ConfigurableRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, tracker := range rl.counts {
		if now.After(tracker.resetAt.Add(5 * time.Minute)) {
			delete(rl.counts, key)
		}
	}
}

// GetRateLimitStatus returns current status for a key (for monitoring)
func (rl *ConfigurableRateLimiter) GetRateLimitStatus(key string) map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	tracker, exists := rl.counts[key]
	if !exists {
		return map[string]interface{}{
			"requests_remaining": rl.cfg.RateLimitRequests,
			"reset_at":          time.Now().Add(rl.cfg.RateLimitWindow),
		}
	}

	remaining := rl.cfg.RateLimitRequests - tracker.count
	if remaining < 0 {
		remaining = 0
	}

	return map[string]interface{}{
		"current_count":        tracker.count,
		"requests_remaining":   remaining,
		"reset_at":            tracker.resetAt,
		"limit_window":        rl.cfg.RateLimitWindow.String(),
	}
}
