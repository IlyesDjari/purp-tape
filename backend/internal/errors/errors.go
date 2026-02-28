package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// AppError represents an application error with HTTP status
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
	Internal   error  `json:"-"` // Not exposed to clients
	Details    map[string]interface{} `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Internal
}

// Standard error codes
const (
	ErrCodeInvalidRequest   = "INVALID_REQUEST"
	ErrCodeNotFound         = "NOT_FOUND"
	ErrCodeUnauthorized     = "UNAUTHORIZED"
	ErrCodeForbidden        = "FORBIDDEN"
	ErrCodeConflict         = "CONFLICT"
	ErrCodeUnprocessable    = "UNPROCESSABLE_ENTITY"
	ErrCodeTooManyRequests  = "TOO_MANY_REQUESTS"
	ErrCodeInternalServer   = "INTERNAL_SERVER_ERROR"
	ErrCodeServiceUnavail   = "SERVICE_UNAVAILABLE"
	ErrCodeNotImplemented   = "NOT_IMPLEMENTED"
)

// Constructor functions for common errors
func NewInvalidRequestError(message string, details map[string]interface{}) *AppError {
	return &AppError{
		Code:       ErrCodeInvalidRequest,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
		Details:    details,
	}
}

func NewNotFoundError(resource string) *AppError {
	return &AppError{
		Code:       ErrCodeNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		HTTPStatus: http.StatusNotFound,
	}
}

func NewUnauthorizedError() *AppError {
	return &AppError{
		Code:       ErrCodeUnauthorized,
		Message:    "unauthorized",
		HTTPStatus: http.StatusUnauthorized,
	}
}

func NewForbiddenError(message string) *AppError {
	return &AppError{
		Code:       ErrCodeForbidden,
		Message:    message,
		HTTPStatus: http.StatusForbidden,
	}
}

func NewConflictError(message string) *AppError {
	return &AppError{
		Code:       ErrCodeConflict,
		Message:    message,
		HTTPStatus: http.StatusConflict,
	}
}

func NewInternalError(internal error) *AppError {
	return &AppError{
		Code:       ErrCodeInternalServer,
		Message:    "internal server error",
		HTTPStatus: http.StatusInternalServerError,
		Internal:   internal,
	}
}

func NewServiceUnavailableError() *AppError {
	return &AppError{
		Code:       ErrCodeServiceUnavail,
		Message:    "service unavailable",
		HTTPStatus: http.StatusServiceUnavailable,
	}
}

func NewQuotaExceededError(resource string) *AppError {
	return &AppError{
		Code:       "QUOTA_EXCEEDED",
		Message:    fmt.Sprintf("%s quota exceeded", resource),
		HTTPStatus: http.StatusPaymentRequired,
	}
}

// WriteErrorResponse writes error as JSON HTTP response
func WriteErrorResponse(w http.ResponseWriter, err error) {
	appErr := ToAppError(err)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.HTTPStatus)
	json.NewEncoder(w).Encode(appErr)
}

// ToAppError converts any error to AppError
func ToAppError(err error) *AppError {
	if err == nil {
		return nil
	}

	// Already an AppError
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	// Unknown error
	return NewInternalError(err)
}

// Validation errors
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func NewValidationError(field, message string) *AppError {
	return &AppError{
		Code:       ErrCodeUnprocessable,
		Message:    fmt.Sprintf("validation failed: %s", field),
		HTTPStatus: http.StatusUnprocessableEntity,
		Details: map[string]interface{}{
			"field":   field,
			"message": message,
		},
	}
}

// Multiple validation errors
func NewValidationErrors(errors []ValidationError) *AppError {
	details := make(map[string]interface{})
	details["errors"] = errors
	
	return &AppError{
		Code:       ErrCodeUnprocessable,
		Message:    "validation failed",
		HTTPStatus: http.StatusUnprocessableEntity,
		Details:    details,
	}
}

// Wrapping with context
func Wrap(err error, message string) *AppError {
	appErr := ToAppError(err)
	appErr.Message = fmt.Sprintf("%s: %s", message, appErr.Message)
	return appErr
}
