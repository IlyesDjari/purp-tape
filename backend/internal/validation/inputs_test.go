package validation

import (
	"testing"
)

func TestValidateProjectName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{"valid name", "My Project", false},
		{"valid single char", "A", false},
		{"valid 255 chars", string(make([]byte, 255)) + "a", true}, // Exceeds
		{"valid exactly 255", string(make([]byte, 254)) + "a", false},
		{"empty", "", true},
		{"too long", string(make([]byte, 256, 256)), true},
		{"whitespace only", "   ", true},
		{"with spaces", "  Project Name  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProjectName(tt.input)
			if (err != nil) != tt.shouldErr {
				t.Errorf("ValidateProjectName(%q) err=%v, shouldErr=%v", tt.input, err, tt.shouldErr)
			}
		})
	}
}

func TestValidateTrackName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{"valid name", "Track 1", false},
		{"valid single char", "T", false},
		{"empty", "", true},
		{"too long", string(make([]byte, 256)), true},
		{"whitespace only", "   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTrackName(tt.input)
			if (err != nil) != tt.shouldErr {
				t.Errorf("ValidateTrackName(%q) err=%v, shouldErr=%v", tt.input, err, tt.shouldErr)
			}
		})
	}
}

func TestValidateDescription(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{"empty description", "", false},
		{"valid description", "This is a great project", false},
		{"max length", string(make([]byte, 2000)), false},
		{"exceeds max", string(make([]byte, 2001)), true},
		{"whitespace", "   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDescription(tt.input)
			if (err != nil) != tt.shouldErr {
				t.Errorf("ValidateDescription(%q) err=%v, shouldErr=%v", tt.input, err, tt.shouldErr)
			}
		})
	}
}

func TestValidateComment(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{"valid comment", "Great track!", false},
		{"valid long comment", string(make([]byte, 5000)), false},
		{"empty", "", true},
		{"too long", string(make([]byte, 5001)), true},
		{"whitespace only", "   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateComment(tt.input)
			if (err != nil) != tt.shouldErr {
				t.Errorf("ValidateComment(%q) err=%v, shouldErr=%v", tt.input, err, tt.shouldErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{"valid password", "TestPass123", false},
		{"valid long password", "VeryLongPassword123WithManyChars", false},
		{"too short", "Test12", true},
		{"missing uppercase", "testpass123", true},
		{"missing lowercase", "TESTPASS123", true},
		{"missing digit", "TestPassword", true},
		{"valid min length", "Test1234", false},
		{"too long", string(make([]byte, 256)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.input)
			if (err != nil) != tt.shouldErr {
				t.Errorf("ValidatePassword(%q) err=%v, shouldErr=%v", tt.input, err, tt.shouldErr)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{"valid email", "user@example.com", false},
		{"valid complex", "test.user+tag@example.co.uk", false},
		{"no @", "userexample.com", true},
		{"no domain", "user@", true},
		{"no extension", "user@example", true},
		{"too short", "a@b", true},
		{"too long", string(make([]byte, 256)) + "@example.com", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.input)
			if (err != nil) != tt.shouldErr {
				t.Errorf("ValidateEmail(%q) err=%v, shouldErr=%v", tt.input, err, tt.shouldErr)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no trim needed", "hello", "hello"},
		{"leading spaces", "  hello", "hello"},
		{"trailing spaces", "hello  ", "hello"},
		{"both", "  hello  ", "hello"},
		{"tabs and newlines", "\t\nhello\n\t", "hello"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeString(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
