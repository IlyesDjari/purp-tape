package errors

import (
	"net/http"
	"testing"
)

func TestNewInvalidRequestError(t *testing.T) {
	details := map[string]interface{}{"field": "name"}
	err := NewInvalidRequestError("name is required", details)

	if err == nil {
		t.Errorf("NewInvalidRequestError() returned nil")
	}

	if err.Code != ErrCodeInvalidRequest {
		t.Errorf("expected code %s, got %s", ErrCodeInvalidRequest, err.Code)
	}

	if err.Message != "name is required" {
		t.Errorf("expected message 'name is required', got %s", err.Message)
	}

	if err.HTTPStatus != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, err.HTTPStatus)
	}

	if err.Details == nil {
		t.Errorf("expected details, got nil")
	}
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("project")

	if err.Code != ErrCodeNotFound {
		t.Errorf("expected code %s, got %s", ErrCodeNotFound, err.Code)
	}

	if err.HTTPStatus != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, err.HTTPStatus)
	}
}

func TestNewUnauthorizedError(t *testing.T) {
	err := NewUnauthorizedError()

	if err.Code != ErrCodeUnauthorized {
		t.Errorf("expected code %s, got %s", ErrCodeUnauthorized, err.Code)
	}

	if err.HTTPStatus != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, err.HTTPStatus)
	}
}

func TestNewForbiddenError(t *testing.T) {
	err := NewForbiddenError("you cannot access this resource")

	if err.Code != ErrCodeForbidden {
		t.Errorf("expected code %s, got %s", ErrCodeForbidden, err.Code)
	}

	if err.HTTPStatus != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, err.HTTPStatus)
	}

	if err.Message != "you cannot access this resource" {
		t.Errorf("expected message, got %s", err.Message)
	}
}

func TestNewConflictError(t *testing.T) {
	err := NewConflictError("project already exists")

	if err.Code != ErrCodeConflict {
		t.Errorf("expected code %s, got %s", ErrCodeConflict, err.Code)
	}

	if err.HTTPStatus != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, err.HTTPStatus)
	}
}

func TestAppError_Error(t *testing.T) {
	err := NewNotFoundError("user")
	errorStr := err.Error()

	if errorStr != "user not found" {
		t.Errorf("Error() returned %q, expected 'user not found'", errorStr)
	}
}

func TestAppError_Unwrap(t *testing.T) {
	internalErr := error(nil)
	appErr := &AppError{
		Code:     ErrCodeInternalServer,
		Message:  "internal error",
		Internal: internalErr,
	}

	if appErr.Unwrap() != internalErr {
		t.Errorf("Unwrap() did not return internal error")
	}
}

func TestAppError_WithDetails(t *testing.T) {
	details := map[string]interface{}{
		"field":   "email",
		"message": "already exists",
	}

	err := &AppError{
		Code:       ErrCodeConflict,
		Message:    "email conflict",
		HTTPStatus: http.StatusConflict,
		Details:    details,
	}

	if err.Details["field"] != "email" {
		t.Errorf("expected details field 'email', got %v", err.Details["field"])
	}
}
