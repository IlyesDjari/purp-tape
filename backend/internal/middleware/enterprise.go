package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"
)

// PanicRecoveryMiddleware recovers from panics and logs them
func PanicRecoveryMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Get stack trace
					buf := make([]byte, 4096)
					n := runtime.Stack(buf, false)
					stack := string(buf[:n])

					log.Error("panic recovered",
						"error", err,
						"path", r.URL.Path,
						"method", r.Method,
						"remote_addr", r.RemoteAddr,
						"stack_trace", stack)

					// Send error response
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, `{"error":"internal server error"}`)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeadersMiddleware adds security headers to all responses
func SecurityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent MIME sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// Prevent clickjacking
			w.Header().Set("X-Frame-Options", "DENY")

			// Enable XSS protection (legacy, for older browsers)
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			// HSTS - force HTTPS
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

			// CSP - restrict content sources
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' https:; font-src 'self'")

			// Disable referrer for privacy
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Feature policy / Permissions policy
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

			next.ServeHTTP(w, r)
		})
	}
}

// RequestContextMiddleware adds request context (ID, timestamps, etc)
func RequestContextMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Generate request ID
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = fmt.Sprintf("req_%d", time.Now().UnixNano())
			}

			// Add to context
			ctx := context.WithValue(r.Context(), "request_id", requestID)
			ctx = context.WithValue(ctx, "start_time", time.Now())

			// Log request
			log.InfoContext(ctx, "request started",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", getClientIP(r))

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ResponseLoggingMiddleware logs response details
func ResponseLoggingMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w}

			ctx := r.Context()
			startTime := ctx.Value("start_time").(time.Time)

			// Call handler
			next.ServeHTTP(wrapped, r)

			// Log response
			duration := time.Since(startTime)

			log.InfoContext(ctx, "request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.statusCode,
				"duration_ms", duration.Milliseconds(),
				"remote_addr", getClientIP(r))
		})
	}
}

// responseWriter captures status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}

// TimeoutMiddleware enforces request timeouts
func TimeoutMiddleware(timeout time.Duration, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Create channel for goroutine completion
			done := make(chan struct{})
			var once sync.Once

			go func() {
				next.ServeHTTP(w, r.WithContext(ctx))
				once.Do(func() { close(done) })
			}()

			// Wait for either completion or timeout
			select {
			case <-done:
				// Request completed
				return
			case <-ctx.Done():
				// Timeout occurred
				log.Warn("request timeout", "path", r.URL.Path, "timeout", timeout)
				http.Error(w, "request timeout", http.StatusRequestTimeout)
				return
			}
		})
	}
}

// CircuitBreakerMiddleware prevents cascading failures
type CircuitBreakerState string

const (
	StateClosed CircuitBreakerState = "closed" // Normal operation
	StateOpen   CircuitBreakerState = "open"   // Failing fast
	StateHalf   CircuitBreakerState = "half"   // Testing recovery
)

type CircuitBreaker struct {
	state             CircuitBreakerState
	failureCount      int
	successCount      int
	failureThreshold  int
	successThreshold  int
	timeout           time.Duration
	lastFailureTime   time.Time
	mu                sync.RWMutex
	log               *slog.Logger
}

func NewCircuitBreaker(failureThreshold, successThreshold int, timeout time.Duration, log *slog.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
		log:              log,
	}
}

func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check if should transition to half-open
	if cb.state == StateOpen {
		if time.Since(cb.lastFailureTime) > cb.timeout {
			cb.state = StateHalf
			cb.successCount = 0
			cb.log.Info("circuit breaker: transitioning to half-open")
		} else {
			return errors.New("circuit breaker open")
		}
	}

	// Execute function
	err := fn()

	if err == nil {
		// Success
		if cb.state == StateHalf {
			cb.successCount++
			if cb.successCount >= cb.successThreshold {
				cb.state = StateClosed
				cb.failureCount = 0
				cb.log.Info("circuit breaker: transitioning to closed")
			}
		} else if cb.state == StateClosed {
			cb.failureCount = 0
		}
	} else {
		// Failure
		cb.failureCount++
		cb.lastFailureTime = time.Now()

		if cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
			cb.log.Warn("circuit breaker: transitioning to open")
			return fmt.Errorf("circuit breaker open: %w", err)
		}
	}

	return err
}

// getClientIP extracts the real client IP (considering proxies)
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for proxied requests)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Fall back to remote address
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}

	return r.RemoteAddr
}
