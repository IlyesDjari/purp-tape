package storage

import (
	"fmt"
	"path/filepath"
	"strings"
)

// validateObjectKey validates R2 object key to prevent directory traversal and enforce user scope.
func (rc *R2Client) validateObjectKey(userID, objectKey string) error {
	// 1. Prevent directory traversal attacks
	if strings.Contains(objectKey, "..") {
		return fmt.Errorf("invalid object key: directory traversal detected (..)")
	}

	if strings.Contains(objectKey, "//") {
		return fmt.Errorf("invalid object key: double slashes not allowed")
	}

	// 2. Enforce user-specific prefix for security
	expectedPrefix := fmt.Sprintf("tracks/%s/", userID)
	if !strings.HasPrefix(objectKey, expectedPrefix) {
		return fmt.Errorf("invalid object key: must start with %s", expectedPrefix)
	}

	// 3. Prevent absolute paths
	cleanPath := filepath.Clean(objectKey)
	if strings.HasPrefix(cleanPath, "/") {
		return fmt.Errorf("invalid object key: absolute paths not allowed")
	}

	// 4. Ensure key doesn't escape the user's directory
	rel, err := filepath.Rel(expectedPrefix, objectKey)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("invalid object key: path escapes user directory")
	}

	// 5. Check key length (R2 limit is 1024 chars)
	if len(objectKey) > 1024 {
		return fmt.Errorf("invalid object key: too long (max 1024 chars, got %d)", len(objectKey))
	}

	return nil
}

// ValidateUploadRequest validates presigned upload request parameters
func (rc *R2Client) ValidateUploadRequest(userID, objectKey string, expectedFileSizeBytes int64, contentType string) error {
	// Validate object key
	if err := rc.validateObjectKey(userID, objectKey); err != nil {
		return err
	}

	// Validate file size is reasonable
	const maxAudioFileSize = 500 * 1024 * 1024 // 500MB
	if expectedFileSizeBytes > maxAudioFileSize {
		return fmt.Errorf("file size exceeds maximum: expected %d bytes, max %d bytes", expectedFileSizeBytes, maxAudioFileSize)
	}

	if expectedFileSizeBytes <= 0 {
		return fmt.Errorf("invalid file size: must be greater than 0")
	}

	// Validate content type
	validContentTypes := map[string]bool{
		"audio/mpeg":               true,
		"audio/wav":                true,
		"audio/x-wav":              true,
		"audio/aiff":               true,
		"audio/x-aiff":             true,
		"audio/flac":               true,
		"audio/aac":                true,
		"audio/x-m4a":              true,
		"audio/mp4":                true,
		"application/octet-stream": true, // Fallback for ambiguous types
	}

	if !validContentTypes[contentType] {
		return fmt.Errorf("invalid content type: %s not allowed", contentType)
	}

	return nil
}
