package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/IlyesDjari/purp-tape/backend/internal/auth"
)

// AuthMiddleware validates JWT tokens from Supabase
func AuthMiddleware(validator *auth.Validator, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			// Extract token
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "invalid authorization header", http.StatusUnauthorized)
				return
			}

			token := parts[1]

			// Get user ID from token
			userID, err := validator.GetUserIDFromToken(authHeader)
			if err != nil {
				log.Error("token validation failed", "error", err)
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			// Add user ID and token to context
			ctx := context.WithValue(r.Context(), "user_id", userID)
			ctx = context.WithValue(ctx, "token", token)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := sanitizePathForLogs(r.URL.Path)
			log.Info("http_request",
				"method", r.Method,
				"path", path,
				"remote_addr", r.RemoteAddr,
			)
			next.ServeHTTP(w, r)
		})
	}
}

func sanitizePathForLogs(path string) string {
	if path == "" {
		return path
	}

	parts := strings.Split(path, "/")
	for index, part := range parts {
		if part == "" {
			continue
		}

		if part == "share" || part == "shares" || part == "token" {
			if index+1 < len(parts) && parts[index+1] != "" {
				parts[index+1] = "[REDACTED]"
			}
		}
	}

	return strings.Join(parts, "/")
}

// CORSMiddleware sets CORS headers with explicit origin whitelist
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// [CRITICAL FIX] Check if origin is in whitelist - NO wildcard matching
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				// Exact match only - never use wildcards with credentials
				if origin == allowedOrigin {
					allowed = true
					break
				}
			}

			if allowed {
				// Safe to set specific origin
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Max-Age", "3600")
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
