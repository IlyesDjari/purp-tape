package logging

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// RequestLogger provides structured logging for HTTP requests [LOW: Consistency]
type RequestLogger struct {
	log *slog.Logger
}

// NewRequestLogger creates a new request logger
func NewRequestLogger(log *slog.Logger) *RequestLogger {
	return &RequestLogger{log: log}
}

// LogRequest logs a request with details [LOW: Consistency]
func (rl *RequestLogger) LogRequest(r *http.Request, userID string) {
	rl.log.Info("incoming request",
		"method", r.Method,
		"path", r.URL.Path,
		"user_id", userID,
		"remote_addr", r.RemoteAddr,
		"user_agent", r.Header.Get("User-Agent"),
	)
}

// LogRequestError logs a request error with context [LOW: Consistency]
func (rl *RequestLogger) LogRequestError(r *http.Request, userID string, statusCode int, duration time.Duration, err error) {
	rl.log.Error("request failed",
		"method", r.Method,
		"path", r.URL.Path,
		"user_id", userID,
		"status", statusCode,
		"duration_ms", duration.Milliseconds(),
		"error", err,
	)
}

// LogRequestSuccess logs a successful request [LOW: Consistency]
func (rl *RequestLogger) LogRequestSuccess(r *http.Request, userID string, statusCode int, duration time.Duration) {
	if duration > 5*time.Second {
		rl.log.Warn("slow request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"user_id", userID,
			"status", statusCode,
			"duration_ms", duration.Milliseconds(),
		)
	} else {
		rl.log.Debug("request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"user_id", userID,
			"status", statusCode,
			"duration_ms", duration.Milliseconds(),
		)
	}
}

// DatabaseLogger provides structured logging for database operations [LOW: Consistency]
type DatabaseLogger struct {
	log *slog.Logger
}

// NewDatabaseLogger creates a new database logger
func NewDatabaseLogger(log *slog.Logger) *DatabaseLogger {
	return &DatabaseLogger{log: log}
}

// LogQuery logs a database query [LOW: Consistency]
func (dl *DatabaseLogger) LogQuery(ctx context.Context, query string, duration time.Duration, err error) {
	if err != nil {
		dl.log.Error("database query failed",
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)
		return
	}

	if duration > 1*time.Second {
		dl.log.Warn("slow database query",
			"duration_ms", duration.Milliseconds(),
		)
	}
}

// ServiceLogger provides structured logging for service operations [LOW: Consistency]
type ServiceLogger struct {
	log *slog.Logger
}

// NewServiceLogger creates a new service logger
func NewServiceLogger(log *slog.Logger) *ServiceLogger {
	return &ServiceLogger{log: log}
}

// LogServiceStart logs service startup [LOW: Consistency]
func (sl *ServiceLogger) LogServiceStart(serviceName string, version string) {
	sl.log.Info(fmt.Sprintf("%s starting", serviceName),
		"version", version,
	)
}

// LogServiceError logs a service error [LOW: Consistency]
func (sl *ServiceLogger) LogServiceError(serviceName string, operation string, err error) {
	sl.log.Error(fmt.Sprintf("%s operation failed", serviceName),
		"operation", operation,
		"error", err,
	)
}

// LogServiceEvent logs a service event [LOW: Consistency]
func (sl *ServiceLogger) LogServiceEvent(serviceName string, event string, details map[string]interface{}) {
	attrs := []interface{}{"event", event}
	for k, v := range details {
		attrs = append(attrs, k, v)
	}
	sl.log.Info(fmt.Sprintf("%s event", serviceName), attrs...)
}

// ContextLogger adds request context to all logs [LOW: Better context]
type ContextLogger struct {
	log       *slog.Logger
	requestID string
	userID    string
}

// NewContextLogger creates a logger with context
func NewContextLogger(log *slog.Logger, requestID, userID string) *ContextLogger {
	return &ContextLogger{
		log:       log,
		requestID: requestID,
		userID:    userID,
	}
}

// Debug logs at debug level with context
func (cl *ContextLogger) Debug(msg string, args ...interface{}) {
	cl.log.Debug(msg,
		append([]interface{}{"request_id", cl.requestID, "user_id", cl.userID}, args...)...,
	)
}

// Info logs at info level with context
func (cl *ContextLogger) Info(msg string, args ...interface{}) {
	cl.log.Info(msg,
		append([]interface{}{"request_id", cl.requestID, "user_id", cl.userID}, args...)...,
	)
}

// Warn logs at warn level with context
func (cl *ContextLogger) Warn(msg string, args ...interface{}) {
	cl.log.Warn(msg,
		append([]interface{}{"request_id", cl.requestID, "user_id", cl.userID}, args...)...,
	)
}

// Error logs at error level with context
func (cl *ContextLogger) Error(msg string, args ...interface{}) {
	cl.log.Error(msg,
		append([]interface{}{"request_id", cl.requestID, "user_id", cl.userID}, args...)...,
	)
}
