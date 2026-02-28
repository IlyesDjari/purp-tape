package validation

import (
	"fmt"

	"github.com/google/uuid"
)

// ValidateUUID validates that a string is a valid UUID v4
func ValidateUUID(id string) error {
	if id == "" {
		return fmt.Errorf("id cannot be empty")
	}

	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("invalid UUID format: %w", err)
	}

	return nil
}

// ValidateUserID validates a user ID is a valid UUID
func ValidateUserID(userID string) error {
	return ValidateUUID(userID)
}

// ValidateProjectID validates a project ID is a valid UUID
func ValidateProjectID(projectID string) error {
	return ValidateUUID(projectID)
}

// ValidateTrackID validates a track ID is a valid UUID
func ValidateTrackID(trackID string) error {
	return ValidateUUID(trackID)
}

// ValidateTrackVersionID validates a track version ID is a valid UUID
func ValidateTrackVersionID(versionID string) error {
	return ValidateUUID(versionID)
}

// SafePathComponent validates a path component (UUID) to prevent path traversal
// Used for constructing R2 object keys
func SafePathComponent(component string) error {
	if err := ValidateUUID(component); err != nil {
		return fmt.Errorf("unsafe path component: must be valid UUID")
	}
	return nil
}
