package helpers

import (
	"net/http"
	"strconv"
	"strings"
)

// ValidateAndParseInt safely parses and validates an integer from query params.
func ValidateAndParseInt(value string, min, max int) (int, error) {
	num, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if num < min || num > max {
		return 0, err
	}
	return num, nil
}

// ExtractPaginationParamsValidated extracts and validates pagination from query.
func ExtractPaginationParamsValidated(r *http.Request) (limit, offset int) {
	limit = 20
	offset = 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := ValidateAndParseInt(limitStr, 1, 100); err == nil {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := ValidateAndParseInt(offsetStr, 0, 999999); err == nil {
			offset = o
		}
	}

	return limit, offset
}

// ValidateSortField validates a sort field against allowed list.
func ValidateSortField(field string, allowed []string) string {
	field = strings.TrimSpace(field)
	for _, a := range allowed {
		if field == a {
			return field
		}
	}
	return allowed[0] // Default to first allowed field
}

// ExtractSortParams safely extracts sort parameters.
func ExtractSortParams(r *http.Request, defaultSort string, allowed []string) (field string, desc bool) {
	field = defaultSort
	desc = false

	if sortStr := r.URL.Query().Get("sort"); sortStr != "" {
		if strings.HasPrefix(sortStr, "-") {
			field = ValidateSortField(sortStr[1:], allowed)
			desc = true
		} else {
			field = ValidateSortField(sortStr, allowed)
		}
	}

	return field, desc
}

// SanitizeFilename removes unsafe characters from filenames.
func SanitizeFilename(filename string) string {
	// Remove path separators
	filename = strings.ReplaceAll(filename, "/", "")
	filename = strings.ReplaceAll(filename, "\\", "")
	filename = strings.ReplaceAll(filename, "..", "")

	// Limit length
	if len(filename) > 255 {
		filename = filename[:255]
	}

	return filename
}

// ValidateUserInput validates user input for safety.
type InputValidator struct {
	Field   string
	Value   string
	MinLen  int
	MaxLen  int
	Pattern string // Regex pattern for validation
}

// Validate validates a single input based on rules
func (iv InputValidator) Validate() error {
	value := strings.TrimSpace(iv.Value)

	if len(value) < iv.MinLen {
		return NewValidationError(iv.Field, "too short")
	}

	if len(value) > iv.MaxLen {
		return NewValidationError(iv.Field, "too long")
	}

	// Add pattern validation if needed
	return nil
}
