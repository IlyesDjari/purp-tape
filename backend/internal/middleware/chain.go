package middleware

import (
	"compress/gzip"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// Chain applies multiple middleware in sequence [LOW: Code organization]
// Usage: Chain(handler, Auth, Logging, Recovery)
func Chain(handler http.Handler, middleware ...func(http.Handler) http.Handler) http.Handler {
	// Apply middleware in reverse order so first middleware is outermost
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	return handler
}

// DefaultMiddlewareStack returns the standard middleware stack [LOW: Consistency]
func DefaultMiddlewareStack(log *slog.Logger) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		RecoveryMiddleware(log),
		ContextMiddleware(log),
		RequestMetricsMiddleware(log),
		ChainSecurityHeadersMiddleware(),
	}
}

// APIMiddlewareStack returns the middleware stack for API endpoints [LOW: Consistency]
func APIMiddlewareStack(log *slog.Logger) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		RecoveryMiddleware(log),
		ContextMiddleware(log),
		RequestMetricsMiddleware(log),
		ChainSecurityHeadersMiddleware(),
		GzipMiddleware(),
	}
}

// ChainSecurityHeadersMiddleware adds security headers to all responses [LOW: Security]
func ChainSecurityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent clickjacking
			w.Header().Set("X-Frame-Options", "DENY")

			// Prevent MIME-type sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// Enable XSS protection
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			// Content Security Policy
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'")

			// Referrer policy
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Permissions policy
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			next.ServeHTTP(w, r)
		})
	}
}

// GzipMiddleware enables gzip compression for responses [LOW: Performance]
func GzipMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if client accepts gzip
			if !acceptsGzip(r.Header.Get("Accept-Encoding")) || r.Method == http.MethodHead || r.Header.Get("Upgrade") != "" {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Add("Vary", "Accept-Encoding")

			gzWriter := gzip.NewWriter(w)
			defer gzWriter.Close()

			gzResponseWriter := &gzipResponseWriter{
				ResponseWriter: w,
				writer:         gzWriter,
			}

			next.ServeHTTP(gzResponseWriter, r)
		})
	}
}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer      *gzip.Writer
	wroteHeader bool
}

func (g *gzipResponseWriter) WriteHeader(statusCode int) {
	if g.wroteHeader {
		return
	}
	g.wroteHeader = true

	if statusCode == http.StatusNoContent || statusCode == http.StatusNotModified {
		g.ResponseWriter.WriteHeader(statusCode)
		return
	}

	g.Header().Del("Content-Length")
	g.Header().Set("Content-Encoding", "gzip")
	g.ResponseWriter.WriteHeader(statusCode)
}

func (g *gzipResponseWriter) Write(data []byte) (int, error) {
	if !g.wroteHeader {
		g.WriteHeader(http.StatusOK)
	}
	return g.writer.Write(data)
}

func (g *gzipResponseWriter) Flush() {
	_ = g.writer.Flush()
	if flusher, ok := g.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (g *gzipResponseWriter) Unwrap() http.ResponseWriter {
	return g.ResponseWriter
}

func (g *gzipResponseWriter) ReadFrom(src io.Reader) (int64, error) {
	if !g.wroteHeader {
		g.WriteHeader(http.StatusOK)
	}
	return io.Copy(g.writer, src)
}

// LoggingDetailsMiddleware adds detailed logging to requests [LOW: Observability]
func LoggingDetailsMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Debug("request details",
				"headers", r.Header,
				"query_params", r.URL.Query(),
				"content_length", r.ContentLength,
			)
			next.ServeHTTP(w, r)
		})
	}
}

// ChainCORSMiddleware adds CORS headers [LOW: API compatibility]
func ChainCORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if isOriginAllowed(origin, allowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Max-Age", "3600")
			}

			// Handle preflight
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// contenString checks if string contains substring [LOW: Helper]
func acceptsGzip(acceptEncoding string) bool {
	if acceptEncoding == "" {
		return false
	}
	for _, encoding := range strings.Split(acceptEncoding, ",") {
		if strings.TrimSpace(strings.SplitN(encoding, ";", 2)[0]) == "gzip" {
			return true
		}
	}
	return false
}

// isOriginAllowed checks if origin is in allowed list [LOW: Helper]
func isOriginAllowed(origin string, allowed []string) bool {
	if len(allowed) == 0 {
		return false
	}
	for _, o := range allowed {
		if o == "*" || o == origin {
			return true
		}
	}
	return false
}
