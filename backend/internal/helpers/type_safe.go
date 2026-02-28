package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// SafeTypeAssertion safely asserts a type and returns error if failed [LOW: Error handling]
func SafeTypeAssertion[T any](value interface{}) (T, error) {
	var zero T
	result, ok := value.(T)
	if !ok {
		return zero, fmt.Errorf("type assertion failed: expected %T, got %T", zero, value)
	}
	return result, nil
}

// GetUserIDSafe safely extracts user ID from context with better error handling [LOW: Error handling]
func GetUserIDSafe(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value("user_id").(string)
	return userID, ok && userID != ""
}

// GetContextValue safely gets a context value [LOW: Code safety]
func GetContextValue[T any](ctx context.Context, key interface{}) (T, bool) {
	value, ok := ctx.Value(key).(T)
	return value, ok
}

// SafeJSONDecode safely decodes JSON and returns structured error [LOW: Error handling]
func SafeJSONDecode(r io.Reader, v interface{}) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields() // Stricter decoding
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}
	return nil
}

// SafeJSONEncode safely encodes JSON and returns structured error [LOW: Error handling]
func SafeJSONEncode(w io.Writer, v interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(true) // Always escape HTML in JSON
	if err := encoder.Encode(v); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}

// GetPathValue safely gets a path parameter [LOW: Code safety]
func GetPathValue(r *http.Request, key string) (string, bool) {
	value := r.PathValue(key)
	return value, value != ""
}

// GetQueryParam safely gets a query parameter [LOW: Code safety]
func GetQueryParam(r *http.Request, key string) (string, bool) {
	value := r.URL.Query().Get(key)
	return value, value != ""
}

// GetQueryParams gets multiple query parameters at once [LOW: Code efficiency]
func GetQueryParams(r *http.Request, keys ...string) map[string]string {
	params := make(map[string]string)
	for _, key := range keys {
		params[key] = r.URL.Query().Get(key)
	}
	return params
}

// NilIfEmpty returns nil if string is empty, otherwise returns pointer [LOW: Code cleanliness]
func NilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// NilIfZero returns nil if int is zero, otherwise returns pointer [LOW: Code cleanliness]
func NilIfZero(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

// FirstNonEmpty returns the first non-empty string [LOW: Code efficiency]
func FirstNonEmpty(strs ...string) string {
	for _, s := range strs {
		if s != "" {
			return s
		}
	}
	return ""
}

// Must panics if error is not nil, otherwise returns value [LOW: Error handling - for startup]
// Use only in initialization, never in request handlers
func Must[T any](value T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("unexpected error: %v", err))
	}
	return value
}

// Ptr returns a pointer to the given value [LOW: Code cleanliness]
func Ptr[T any](t T) *T {
	return &t
}
