package validation

import (
	"fmt"
	"strings"
	"unicode"
)

// ValidateProjectName validates project name
func ValidateProjectName(name string) error {
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 255 {
		return fmt.Errorf("project name must be 1-255 characters, got %d", len(name))
	}
	return nil
}

// ValidateDescription validates project/track description
func ValidateDescription(desc string) error {
	desc = strings.TrimSpace(desc)
	if len(desc) > 2000 {
		return fmt.Errorf("description must be less than 2000 characters, got %d", len(desc))
	}
	return nil
}

// ValidateTrackName validates track name
func ValidateTrackName(name string) error {
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 255 {
		return fmt.Errorf("track name must be 1-255 characters, got %d", len(name))
	}
	return nil
}

// ValidateComment validates comment content
func ValidateComment(content string) error {
	content = strings.TrimSpace(content)
	if len(content) < 1 || len(content) > 5000 {
		return fmt.Errorf("comment must be 1-5000 characters, got %d", len(content))
	}
	return nil
}

// ValidatePassword validates password strength
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	if len(password) > 255 {
		return fmt.Errorf("password must be less than 255 characters")
	}
	
	hasUpper := false
	hasLower := false
	hasDigit := false
	
	for _, ch := range password {
		if unicode.IsUpper(ch) {
			hasUpper = true
		}
		if unicode.IsLower(ch) {
			hasLower = true
		}
		if unicode.IsDigit(ch) {
			hasDigit = true
		}
	}
	
	if !hasUpper || !hasLower || !hasDigit {
		return fmt.Errorf("password must contain uppercase, lowercase, and digits")
	}
	
	return nil
}

// ValidateEmail validates email format (basic check)
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if len(email) < 5 || len(email) > 255 {
		return fmt.Errorf("invalid email format")
	}
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// SanitizeString trims and returns string
func SanitizeString(s string) string {
	return strings.TrimSpace(s)
}
