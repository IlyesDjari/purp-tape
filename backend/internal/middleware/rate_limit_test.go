package middleware

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	limiter := NewRateLimiter(logger)

	if limiter == nil {
		t.Errorf("NewRateLimiter() returned nil")
	}

	if limiter.requests == nil {
		t.Errorf("requests map not initialized")
	}
}

func TestRateLimiter_AllowsRequests(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	limiter := NewRateLimiter(logger)

	// First 100 requests should be allowed
	for i := 0; i < 100; i++ {
		allowed := limiter.isAllowed("client-1")
		if !allowed {
			t.Errorf("request %d should be allowed, got denied", i+1)
		}
	}
}

func TestRateLimiter_BlocksExcessRequests(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	limiter := NewRateLimiter(logger)

	// Allow 100 requests
	for i := 0; i < 100; i++ {
		limiter.isAllowed("client-1")
	}

	// 101st request should be blocked
	allowed := limiter.isAllowed("client-1")
	if allowed {
		t.Errorf("request 101 should be blocked")
	}
}

func TestRateLimiter_DifferentClients(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	limiter := NewRateLimiter(logger)

	// Each client should have separate limit
	for i := 0; i < 50; i++ {
		limiter.isAllowed("client-1")
		limiter.isAllowed("client-2")
	}

	// Both should still be allowed (under 100 each)
	allow1 := limiter.isAllowed("client-1")
	allow2 := limiter.isAllowed("client-2")

	if !allow1 || !allow2 {
		t.Errorf("separate clients should have separate limits")
	}
}

func TestRateLimiter_WindowReset(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	limiter := NewRateLimiter(logger)

	identifier := "test-client"

	// Hit the limit
	for i := 0; i < 100; i++ {
		limiter.isAllowed(identifier)
	}

	// Should be blocked
	if limiter.isAllowed(identifier) {
		t.Errorf("should be blocked after 100 requests")
	}

	// Manually advance time in the entry (simulating window reset)
	limiter.mu.Lock()
	if entry, exists := limiter.requests[identifier]; exists {
		entry.windowStart = time.Now().Add(-2 * time.Minute)
	}
	limiter.mu.Unlock()

	// Now should be allowed again
	allowed := limiter.isAllowed(identifier)
	if !allowed {
		t.Errorf("should be allowed after window reset")
	}
}

func TestRateLimiter_IsAllowedWithLimit(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	limiter := NewRateLimiter(logger)

	identifier := "test-client"
	limit := uint32(50)

	// Allow up to limit
	for i := 0; i < int(limit); i++ {
		if !limiter.isAllowedWithLimit(identifier, limit) {
			t.Errorf("request %d should be allowed with limit %d", i+1, limit)
		}
	}

	// Over limit should be blocked
	if limiter.isAllowedWithLimit(identifier, limit) {
		t.Errorf("request over limit should be blocked")
	}
}
