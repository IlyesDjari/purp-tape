package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
)

// ValidationError represents an input validation error [MEDIUM: Structured error handling]
type ValidationError struct {
	Field  string
	Reason string
}

// Error implements error interface
func (ve ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s: %s", ve.Field, ve.Reason)
}

// NewValidationError creates a new validation error
func NewValidationError(field, reason string) ValidationError {
	return ValidationError{Field: field, Reason: reason}
}

// GetUserID safely extracts user ID from request context
func GetUserID(r *http.Request) (string, error) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		return "", errors.New("user_id not in context")
	}
	return userID, nil
}

// ExtractPaginationParams extracts and validates limit and offset from query string [OPTIMIZED]
// Defaults: limit=20, offset=0
// Max: limit=100
func ExtractPaginationParams(r *http.Request) (limit, offset int) {
	limit = 20  // Default
	offset = 0  // Default

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	return limit, offset
}

// WriteJSON writes JSON response with status code
func WriteJSON(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

// WriteBadRequest writes 400 error response
func WriteBadRequest(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// WriteUnauthorized writes 401 error response
func WriteUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}

// WriteForbidden writes 403 error response
func WriteForbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// WriteNotFound writes 404 error response
func WriteNotFound(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// WriteInternalError writes 500 error response
func WriteInternalError(w http.ResponseWriter, log *slog.Logger, err error) {
	if log != nil && err != nil {
		log.Error("internal server error", "error", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
}

// WriteCreated writes 201 created response
func WriteCreated(w http.ResponseWriter, data interface{}) error {
	return WriteJSON(w, http.StatusCreated, data)
}

// WriteNoContent writes 204 no content response
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
// ✅ NIL-SAFETY HELPERS: Defensive programming for state management

// SafeString safely dereferences a string pointer
func SafeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// SafeInt64 safely dereferences an int64 pointer
func SafeInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

// SafeInt safely dereferences an int pointer
func SafeInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// SafeFloat64 safely dereferences a float64 pointer
func SafeFloat64(f *float64) float64 {
	if f == nil {
		return 0.0
	}
	return *f
}

// SafeBool safely dereferences a bool pointer
func SafeBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// CheckNilAfterError is a helper to enforce nil checks immediately after error checks
// Usage:
//   result, err := db.GetSomething()
//   if err != nil { handle error }
//   if helpers.CheckNilAfterError(result) { handle nil }
func CheckNilAfterError(value interface{}) bool {
	return value == nil
}