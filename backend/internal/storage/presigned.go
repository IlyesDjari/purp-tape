package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// PresignedUploadResult contains info for client-side upload
type PresignedUploadResult struct {
	UploadURL  string            `json:"upload_url"`
	ExpiresIn  int               `json:"expires_in_seconds"` // 900 seconds = 15 minutes
	Headers    map[string]string `json:"headers"`             // headers client must send
	FileID     string            `json:"file_id"`             // identifier for client to reference
}

// QuotaChecker interface for checking storage quotas
type QuotaChecker interface {
	GetUserSubscription(ctx context.Context, userID string) (map[string]interface{}, error)
	GetUserStorageUsed(ctx context.Context, userID string) (int64, error)
}

// GeneratePresignedUploadURL creates a URL for direct R2 upload from client with validation.
func (rc *R2Client) GeneratePresignedUploadURL(
	ctx context.Context,
	userID string,
	objectKey string,
	contentType string,
	expectedFileSizeBytes int64,
	quotaChecker QuotaChecker,
) (*PresignedUploadResult, error) {
	// Validate object key with path traversal protection first
	if err := rc.validateObjectKey(userID, objectKey); err != nil {
		rc.log.Warn("invalid upload request - object key validation failed",
			"error", err,
			"user_id", userID,
			"key", objectKey)
		return nil, fmt.Errorf("invalid upload request: %w", err)
	}

	// 1. Validate object key and parameters
	if err := rc.ValidateUploadRequest(userID, objectKey, expectedFileSizeBytes, contentType); err != nil {
		rc.log.Warn("invalid upload request", "error", err, "user_id", userID, "key", objectKey)
		return nil, err
	}

	// Check user's storage quota before generating URL
	if quotaChecker != nil {
		subscription, err := quotaChecker.GetUserSubscription(ctx, userID)
		if err != nil {
			rc.log.Error("failed to get subscription", "error", err, "user_id", userID)
			return nil, fmt.Errorf("failed to check storage quota: %w", err)
		}

		// Extract quota info
		quotaMB := int64(0)
		usedMB := int64(0)

		if quota, ok := subscription["storage_quota_mb"].(int64); ok {
			quotaMB = quota
		}
		if used, ok := subscription["storage_used_mb"].(int64); ok {
			usedMB = used
		}

		availableMB := quotaMB - usedMB

		// Add overhead multiplier (client could upload slightly larger than claimed)
		// Client claims 100MB, but we reserve 110MB to account for metadata/compression variation
		const sizeOverheadMultiplier = 1.1
		effectiveFileSize := int64(float64(expectedFileSizeBytes) * sizeOverheadMultiplier)
		fileSizeMB := effectiveFileSize / (1024 * 1024)

		if fileSizeMB > availableMB {
			rc.log.Warn("storage quota exceeded",
				"user_id", userID,
				"available_mb", availableMB,
				"requested_mb", fileSizeMB,
				"claimed_bytes", expectedFileSizeBytes,
				"effective_bytes", effectiveFileSize)
			return nil, fmt.Errorf("insufficient storage quota: need %dMB but only %dMB available", fileSizeMB, availableMB)
		}
	}

	// 3. Generate presigned URL with validated parameters
	presigner := s3.NewPresignClient(rc.client)

	// Create PUT object request (upload)
	putObjectInput := &s3.PutObjectInput{
		Bucket:           aws.String(rc.bucket),
		Key:              aws.String(objectKey),
		ContentType:      aws.String(contentType),
		ContentLength:    aws.Int64(expectedFileSizeBytes), // Enforce file size
	}

	const presignExpiryDuration = 5 * time.Minute // Shorter expiry for security
	expiresAtTime := time.Now().Add(presignExpiryDuration)

	presignResult, err := presigner.PresignPutObject(ctx, putObjectInput, func(o *s3.PresignOptions) {
		o.Expires = presignExpiryDuration
	})
	if err != nil {
		rc.log.Error("failed to presign upload URL",
			"error", err,
			"user_id", userID,
			"key", objectKey)
		return nil, fmt.Errorf("failed to presign upload URL: %w", err)
	}

	// Validate presigned URL contains required components before returning
	if presignResult == nil || presignResult.URL == "" {
		return nil, fmt.Errorf("presigned URL is empty")
	}

	rc.log.Info("presigned upload URL generated",
		"user_id", userID,
		"key", objectKey,
		"expires_in", int(presignExpiryDuration.Seconds()),
		"expires_at", expiresAtTime.Format(time.RFC3339),
		"file_size_bytes", expectedFileSizeBytes)

	return &PresignedUploadResult{
		UploadURL: presignResult.URL,
		ExpiresIn: int(presignExpiryDuration.Seconds()),
		Headers:   map[string]string{},
		FileID:    objectKey,
	}, nil
}

// GenerateDownloadPresignedURL creates a time-limited download URL
func (rc *R2Client) GenerateDownloadPresignedURL(ctx context.Context, objectKey string, expiresIn time.Duration) (string, error) {
	presigner := s3.NewPresignClient(rc.client)

	getObjectInput := &s3.GetObjectInput{
		Bucket: aws.String(rc.bucket),
		Key:    aws.String(objectKey),
	}

	// Generate presigned URL with custom expiry
	presignResult, err := presigner.PresignGetObject(ctx, getObjectInput,
		func(opts *s3.PresignOptions) {
			opts.Expires = expiresIn
		})
	if err != nil {
		rc.log.Error("failed to generate download URL", "error", err, "key", objectKey)
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}

	rc.log.Info("presigned download URL generated", "key", objectKey, "expires_in", int(expiresIn.Seconds()))

	return presignResult.URL, nil
}

// QueryObjectMetadata gets file info without downloading
func (rc *R2Client) QueryObjectMetadata(ctx context.Context, objectKey string) (*ObjectMetadata, error) {
	headObjectOutput, err := rc.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(rc.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		rc.log.Error("failed to get object metadata", "error", err, "key", objectKey)
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	return &ObjectMetadata{
		Key:          objectKey,
		Size:         *headObjectOutput.ContentLength,
		ContentType:  aws.ToString(headObjectOutput.ContentType),
		LastModified: aws.ToTime(headObjectOutput.LastModified),
		ETag:         aws.ToString(headObjectOutput.ETag),
	}, nil
}

// ObjectMetadata represents file metadata from R2
type ObjectMetadata struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified time.Time
	ETag         string
}
