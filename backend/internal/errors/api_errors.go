package errors

import (
	"fmt"
	"log/slog"
	"net/http"
)

// APIError represents a structured API error.
type APIError struct {
	Code       string      `json:"code"`
	Message    string      `json:"message"`
	StatusCode int         `json:"status_code"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

// Error implements error interface
func (e *APIError) Error() string {
	return e.Message
}

// Common error codes
const (
	ErrValidationFailed   = "VALIDATION_FAILED"
	ErrUnauthorized       = "UNAUTHORIZED"
	ErrForbidden          = "FORBIDDEN"
	ErrNotFound           = "NOT_FOUND"
	ErrConflict           = "CONFLICT"
	ErrInternalServer     = "INTERNAL_SERVER_ERROR"
	ErrQuotaExceeded      = "QUOTA_EXCEEDED"
	ErrTooManyRequests    = "TOO_MANY_REQUESTS"
	ErrInvalidInput       = "INVALID_INPUT"
	ErrStorageFull        = "STORAGE_FULL"
	ErrOperationFailed    = "OPERATION_FAILED"
)

// NewAPIError creates a new API error
func NewAPIError(code string, message string, statusCode int) *APIError {
	return &APIError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Details:    make(map[string]interface{}),
	}
}

// WithDetails adds details to error
func (e *APIError) WithDetails(key string, value interface{}) *APIError {
	e.Details[key] = value
	return e
}

// APIValidationError creates a validation error
func APIValidationError(field, reason string) *APIError {
	return &APIError{
		Code:       ErrValidationFailed,
		Message:    fmt.Sprintf("validation failed for field '%s': %s", field, reason),
		StatusCode: http.StatusBadRequest,
		Details: map[string]interface{}{
			"field":  field,
			"reason": reason,
		},
	}
}

// NotFoundError creates a 404 error
func NotFoundError(resource string) *APIError {
	return &APIError{
		Code:       ErrNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		StatusCode: http.StatusNotFound,
		Details: map[string]interface{}{
			"resource": resource,
		},
	}
}

// ConflictError creates a 409 error
func ConflictError(resource, reason string) *APIError {
	return &APIError{
		Code:       ErrConflict,
		Message:    fmt.Sprintf("conflict: %s", reason),
		StatusCode: http.StatusConflict,
		Details: map[string]interface{}{
			"resource": resource,
			"reason":   reason,
		},
	}
}

// QuotaExceededError creates a quota error
func QuotaExceededError(quotaType string, used, limit int64) *APIError {
	return &APIError{
		Code:       ErrQuotaExceeded,
		Message:    fmt.Sprintf("%s quota exceeded: %d/%d", quotaType, used, limit),
		StatusCode: http.StatusPaymentRequired,
		Details: map[string]interface{}{
			"quota_type": quotaType,
			"used":       used,
			"limit":      limit,
		},
	}
}

// LogAPIError logs an API error with context.
func LogAPIError(log *slog.Logger, err *APIError, context ...interface{}) {
	attrs := []interface{}{
		"error_code", err.Code,
		"status_code", err.StatusCode,
	}
	attrs = append(attrs, context...)

	if len(err.Details) > 0 {
		attrs = append(attrs, "details", err.Details)
	}

	log.Error(err.Message, attrs...)
}
