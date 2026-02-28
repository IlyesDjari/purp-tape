package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// ResponseWriter wrapper for tracking status codes [MEDIUM: Request metrics]
type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

// WriteHeader captures status code
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures written bytes
func (rw *ResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += int64(n)
	return n, err
}

// RequestMetricsMiddleware tracks request metrics [MEDIUM: Observability/monitoring]
func RequestMetricsMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to track status
			rw := &ResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Call next handler
			next.ServeHTTP(rw, r)

			// Log metrics
			duration := time.Since(start)
			log.Info("request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.statusCode,
				"duration_ms", duration.Milliseconds(),
				"bytes_written", rw.written,
				"client_ip", r.RemoteAddr,
			)

			// Warn on slow requests
			if duration > 5*time.Second {
				log.Warn("slow request detected",
					"method", r.Method,
					"path", r.URL.Path,
					"duration_ms", duration.Milliseconds(),
				)
			}
		})
	}
}

// RecoveryMiddleware recovers from panics [MEDIUM: Error handling, observability]
func RecoveryMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Error("panic recovered",
						"error", err,
						"method", r.Method,
						"path", r.URL.Path,
						"remote_addr", r.RemoteAddr,
					)

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error":"internal server error"}`))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
