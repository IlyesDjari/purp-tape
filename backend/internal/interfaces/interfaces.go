package interfaces

import (
	"context"
	"net/http"
)

// Handler defines the interface all HTTP handlers must implement [LOW: Code organization]
type Handler interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// Service defines the interface for all business logic services [LOW: Testability]
type Service interface {
	// Health returns nil if service is healthy
	Health(ctx context.Context) error
}

// Repository defines the interface for data access [LOW: Code organization]
type Repository interface {
	// Close closes the repository connection
	Close(ctx context.Context) error
}

// Logger defines the interface for logging [LOW: Consistency]
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// StorageProvider defines the interface for file storage operations [LOW: Implementation flexibility]
type StorageProvider interface {
	// GetObject retrieves an object from storage
	GetObject(ctx context.Context, key string) ([]byte, error)

	// PutObject stores an object
	PutObject(ctx context.Context, key string, data []byte) error

	// DeleteObject removes an object
	DeleteObject(ctx context.Context, key string) error

	// GeneratePresignedURL generates a signed URL for temporary access
	GeneratePresignedURL(ctx context.Context, key string, expiry int64) (string, error)

	// ListObjects lists objects matching a prefix
	ListObjects(ctx context.Context, prefix string) ([]string, error)
}

// CacheProvider defines the interface for caching [LOW: Implementation flexibility]
type CacheProvider interface {
	// Get retrieves a value from cache
	Get(ctx context.Context, key string) (interface{}, error)

	// Set stores a value in cache
	Set(ctx context.Context, key string, value interface{}, ttl int64) error

	// Delete removes a key from cache
	Delete(ctx context.Context, key string) error

	// Clear removes all keys from cache
	Clear(ctx context.Context) error
}

// RequestValidator defines validation for HTTP requests [LOW: Code organization]
type RequestValidator interface {
	Validate() error
}

// ErrorWriter defines how errors should be written to HTTP response [LOW: Consistency]
type ErrorWriter interface {
	WriteError(w http.ResponseWriter, code string, message string, status int, details map[string]interface{})
}

// AuditLogger defines the interface for audit logging [LOW: Consistency]
type AuditLogger interface {
	LogAction(ctx context.Context, action string, resource string, resourceID string, userID string, details map[string]interface{}) error
	LogUnauthorizedAccess(ctx context.Context, userID string, resource string, resourceID string, reason string) error
	GetAuditLogs(ctx context.Context, userID string, limit int, offset int) (interface{}, error)
}

// Middleware defines the middleware pattern [LOW: Code organization]
type Middleware func(Handler) Handler

// HealthCheck defines the interface for health check endpoints [LOW: Consistency]
type HealthCheck interface {
	Check(ctx context.Context) (HealthStatus, error)
}

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}
