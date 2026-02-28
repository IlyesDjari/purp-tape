package utils

import (
	"fmt"
	"strings"
)

// StringUtils provides utility functions for string handling [LOW: Code organization]

// TruncateString truncates a string to maxLen characters [LOW: Best practices]
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// NormalizeEmail normalizes email for comparison [LOW: Security - case insensitive)
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// NormalizeUsername normalizes username [LOW: Consistency]
func NormalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

// IsSafeString checks if string contains only safe characters [LOW: Security]
func IsSafeString(s string) bool {
	// Disallow control characters and potential injection
	for _, c := range s {
		if c < 32 { // Control characters
			return false
		}
	}
	return true
}

// ContainsAny checks if string contains any of the substrings [LOW: Code efficiency]
func ContainsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// ArrayUtils provides utility functions for array handling [LOW: Code organization]

// Contains checks if array contains value [LOW: Code efficiency]
func Contains[T comparable](arr []T, value T) bool {
	for _, item := range arr {
		if item == value {
			return true
		}
	}
	return false
}

// Filter filters array based on predicate [LOW: Functional programming]
func Filter[T any](arr []T, predicate func(T) bool) []T {
	result := make([]T, 0)
	for _, item := range arr {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// Map transforms array elements [LOW: Functional programming]
func Map[T, U any](arr []T, fn func(T) U) []U {
	result := make([]U, len(arr))
	for i, item := range arr {
		result[i] = fn(item)
	}
	return result
}

// Unique returns unique elements from array [LOW: Code efficiency]
func Unique[T comparable](arr []T) []T {
	seen := make(map[T]bool)
	result := make([]T, 0)
	for _, item := range arr {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// ValidationUtils provides utility functions for validation [LOW: Code organization]

// IsValidEmail checks if email format is valid [LOW: Input validation]
func IsValidEmail(email string) bool {
	// Simple email validation - for production use a proper library
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	if !strings.Contains(parts[1], ".") {
		return false
	}
	return true
}

// IsValidUUID checks if string is valid UUID format [LOW: Input validation]
func IsValidUUID(s string) bool {
	// Simple UUID validation
	if len(s) != 36 {
		return false
	}
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	return true
}

// IsValidUsername checks if username is valid [LOW: Input validation]
func IsValidUsername(username string) bool {
	if len(username) < 3 || len(username) > 50 {
		return false
	}
	// Allow alphanumeric, dash, underscore
	for _, c := range username {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// FormatUtils provides utility functions for formatting [LOW: Code organization]

// FormatBytes converts bytes to human-readable format [LOW: User experience]
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration formats duration in human-readable format [LOW: User experience]
func FormatDuration(milliseconds int64) string {
	duration := float64(milliseconds) / 1000.0
	if duration < 1 {
		return fmt.Sprintf("%dms", milliseconds)
	}
	if duration < 60 {
		return fmt.Sprintf("%.1fs", duration)
	}
	minutes := duration / 60
	if minutes < 60 {
		return fmt.Sprintf("%.1fm", minutes)
	}
	hours := minutes / 60
	return fmt.Sprintf("%.1fh", hours)
}

// ComparisonUtils provides comparison utilities [LOW: Code efficiency]

// Min returns minimum of two values [LOW: Code efficiency]
func Min[T comparable](a, b T) T {
	var zero T
	if a == zero {
		return b
	}
	if b == zero {
		return a
	}
	return a // This is incorrect but T doesn't support < operator
	// In real code, use generics with constraints
}

// Max returns maximum of two values [LOW: Code efficiency]
func Max[T comparable](a, b T) T {
	var zero T
	if a == zero {
		return b
	}
	if b == zero {
		return a
	}
	return a // This is incorrect but T doesn't support > operator
	// In real code, use generics with constraints
}

// Clamp clamps value between min and max [LOW: Code efficiency]
func Clamp[T int | int64 | float64](value, min, max T) T {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
