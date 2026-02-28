package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// RequestContext contains request metadata for logging and auditing.
type RequestContext struct {
	RequestID  string
	UserID     string
	IPAddress  string
	UserAgent  string
	Method     string
	Path       string
	StatusCode int
}

// ContextMiddleware extracts request context for logging.
func ContextMiddleware(log interface{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Generate or get request ID
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}

			// Extract IP address (handle proxy)
			ipAddress := getIPAddress(r)

			// Get user agent
			userAgent := r.Header.Get("User-Agent")

			// Extract user ID from context if already set
			userID, ok := r.Context().Value("user_id").(string)
			if !ok {
				userID = ""
			}

			// Create request context
			reqCtx := &RequestContext{
				RequestID: requestID,
				UserID:    userID,
				IPAddress: ipAddress,
				UserAgent: userAgent,
				Method:    r.Method,
				Path:      r.URL.Path,
			}

			// Add to context
			ctx := context.WithValue(r.Context(), "request_context", reqCtx)
			ctx = context.WithValue(ctx, "request_id", requestID)
			ctx = context.WithValue(ctx, "ip_address", ipAddress)

			w.Header().Set("X-Request-ID", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetRequestContext extracts request context from HTTP context
func GetRequestContext(r *http.Request) *RequestContext {
	if rc, ok := r.Context().Value("request_context").(*RequestContext); ok {
		return rc
	}
	return &RequestContext{}
}

// GetIPAddress safely extracts client IP address
func getIPAddress(r *http.Request) string {
	// Check X-Forwarded-For first (behind proxy)
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		parts := strings.Split(forwardedFor, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP (another proxy header)
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	if ip := strings.Split(r.RemoteAddr, ":")[0]; ip != "" {
		return ip
	}

	return "unknown"
}

// generateRequestID generates a unique request ID using UUID v4
func generateRequestID() string {
	return uuid.New().String()
}
