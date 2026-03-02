package storage

import (
	"context"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"sync/atomic"
	"time"

	cachepkg "github.com/IlyesDjari/purp-tape/backend/internal/cache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// HashingReader calculates hash while reading (memory-efficient streaming)
// ✅ EFFICIENT: Doesn't load entire file into memory
type HashingReader struct {
	reader    io.Reader
	hash      hash.Hash
	bytesRead int64
}

// Read implements io.Reader interface
func (hr *HashingReader) Read(p []byte) (int, error) {
	n, err := hr.reader.Read(p)
	if n > 0 {
		hr.hash.Write(p[:n])
		hr.bytesRead += int64(n)
	}
	return n, err
}

// R2Client handles file uploads and signed URLs for Cloudflare R2
type R2Client struct {
	client                 *s3.Client
	uploader               *manager.Uploader
	bucket                 string
	endpoint               string
	accountID              string
	log                    *slog.Logger
	presigned              *cachepkg.PresignedURLCache
	presignCacheHits       atomic.Uint64
	presignCacheMisses     atomic.Uint64
	uploadedBytes          atomic.Uint64
	deleteOps              atomic.Uint64
	batchDeleteOps         atomic.Uint64
	deletedObjects         atomic.Uint64
	lifecyclePolicyApplied atomic.Uint64
}

type FinOpsMetrics struct {
	PresignCacheHits       uint64
	PresignCacheMisses     uint64
	UploadedBytes          uint64
	DeleteOps              uint64
	BatchDeleteOps         uint64
	DeletedObjects         uint64
	LifecyclePolicyApplied uint64
}

// UploadResult contains metadata about an uploaded file
type UploadResult struct {
	Key      string
	FileSize int64
	Checksum string
	URL      string
}

// NewR2Client creates a new Cloudflare R2 client
func NewR2Client(accessKeyID, secretAccessKey, endpoint, bucket, accountID string, log *slog.Logger) (*R2Client, error) {
	// Create credentials provider
	creds := credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")

	// Create S3 client with custom endpoint resolver for R2
	cfg := aws.Config{
		Region:      "auto", // R2 uses 'auto' region
		Credentials: creds,
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
		o.UseAccelerate = false
	})

	uploader := manager.NewUploader(s3Client)

	return &R2Client{
		client:    s3Client,
		uploader:  uploader,
		bucket:    bucket,
		endpoint:  endpoint,
		accountID: accountID,
		log:       log,
		presigned: cachepkg.NewPresignedURLCache(4 * time.Minute),
	}, nil
}

// UploadFile uploads a file to R2 with streaming (memory-efficient)
// ✅ EFFICIENT: Uses HashingReader to stream upload without loading entire file into RAM
func (rc *R2Client) UploadFile(ctx context.Context, objectKey string, file io.Reader, contentType string) (*UploadResult, error) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Create hashing reader that calculates SHA256 while streaming
	hashingReader := &HashingReader{
		reader: file,
		hash:   sha256.New(),
	}

	// Upload to R2 using streaming
	// The S3 manager handles multipart uploads for large files automatically
	_, err := rc.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(rc.bucket),
		Key:         aws.String(objectKey),
		Body:        hashingReader,
		ContentType: aws.String(contentType),
		// Set metadata for easy identification
		Metadata: map[string]string{
			"checksum": fmt.Sprintf("%x", hashingReader.hash.Sum(nil)),
			"uploaded": time.Now().UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
		rc.log.Error("failed to upload to R2", "error", err, "key", objectKey)
		return nil, fmt.Errorf("failed to upload to R2: %w", err)
	}

	// Generate public URL
	url := fmt.Sprintf("%s/%s/%s", rc.endpoint, rc.bucket, objectKey)

	checksum := fmt.Sprintf("%x", hashingReader.hash.Sum(nil))
	rc.uploadedBytes.Add(uint64(hashingReader.bytesRead))
	rc.log.Info("file uploaded successfully", "key", objectKey, "size", hashingReader.bytesRead, "checksum", checksum)

	return &UploadResult{
		Key:      objectKey,
		FileSize: hashingReader.bytesRead,
		Checksum: checksum,
		URL:      url,
	}, nil
}

// GenerateSignedURL creates a time-limited download link
func (rc *R2Client) GenerateSignedURL(ctx context.Context, objectKey string, expiresIn time.Duration) (string, error) {
	cacheKey := fmt.Sprintf("%s|%d", objectKey, int(expiresIn.Seconds()))
	if cachedURL, ok := rc.presigned.Get(cacheKey); ok {
		rc.presignCacheHits.Add(1)
		return cachedURL, nil
	}
	rc.presignCacheMisses.Add(1)

	// Create a presigner for generating signed URLs
	presigner := s3.NewPresignClient(rc.client)

	getObjectInput := &s3.GetObjectInput{
		Bucket: aws.String(rc.bucket),
		Key:    aws.String(objectKey),
	}

	// Generate signed URL (valid for specified duration)
	presignResult, err := presigner.PresignGetObject(ctx, getObjectInput,
		func(opts *s3.PresignOptions) {
			opts.Expires = expiresIn
		})
	if err != nil {
		rc.log.Error("failed to generate signed URL", "error", err, "key", objectKey)
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	rc.presigned.Set(cacheKey, presignResult.URL)
	return presignResult.URL, nil
}

// DeleteFile removes a file from R2
func (rc *R2Client) DeleteFile(ctx context.Context, objectKey string) error {
	_, err := rc.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(rc.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		rc.log.Error("failed to delete file from R2", "error", err, "key", objectKey)
		return fmt.Errorf("failed to delete file from R2: %w", err)
	}
	rc.deleteOps.Add(1)
	rc.deletedObjects.Add(1)

	rc.log.Info("file deleted from R2", "key", objectKey)
	return nil
}

// DeleteFilesBatch removes multiple files from R2 in a single API call.
func (rc *R2Client) DeleteFilesBatch(ctx context.Context, objectKeys []string) error {
	if len(objectKeys) == 0 {
		return nil
	}

	objects := make([]types.ObjectIdentifier, 0, len(objectKeys))
	for _, key := range objectKeys {
		if key == "" {
			continue
		}
		objects = append(objects, types.ObjectIdentifier{Key: aws.String(key)})
	}

	if len(objects) == 0 {
		return nil
	}

	_, err := rc.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(rc.bucket),
		Delete: &types.Delete{Objects: objects, Quiet: aws.Bool(true)},
	})
	if err != nil {
		rc.log.Error("failed to batch delete files from R2", "error", err, "count", len(objects))
		return fmt.Errorf("failed to batch delete files from R2: %w", err)
	}
	rc.batchDeleteOps.Add(1)
	rc.deletedObjects.Add(uint64(len(objects)))

	rc.log.Info("batch deleted files from R2", "count", len(objects))
	return nil
}

func (rc *R2Client) GetFinOpsMetrics() FinOpsMetrics {
	return FinOpsMetrics{
		PresignCacheHits:       rc.presignCacheHits.Load(),
		PresignCacheMisses:     rc.presignCacheMisses.Load(),
		UploadedBytes:          rc.uploadedBytes.Load(),
		DeleteOps:              rc.deleteOps.Load(),
		BatchDeleteOps:         rc.batchDeleteOps.Load(),
		DeletedObjects:         rc.deletedObjects.Load(),
		LifecyclePolicyApplied: rc.lifecyclePolicyApplied.Load(),
	}
}

func (rc *R2Client) EnsureLifecyclePolicies(ctx context.Context) error {
	input := &s3.PutBucketLifecycleConfigurationInput{
		Bucket: aws.String(rc.bucket),
		LifecycleConfiguration: &types.BucketLifecycleConfiguration{
			Rules: []types.LifecycleRule{
				{
					ID:                             aws.String("purptape-temp-expire"),
					Status:                         types.ExpirationStatusEnabled,
					Prefix:                         aws.String("temp/"),
					Expiration:                     &types.LifecycleExpiration{Days: aws.Int32(7)},
					AbortIncompleteMultipartUpload: &types.AbortIncompleteMultipartUpload{DaysAfterInitiation: aws.Int32(2)},
				},
				{
					ID:                             aws.String("purptape-covers-multipart-abort"),
					Status:                         types.ExpirationStatusEnabled,
					Prefix:                         aws.String("covers/"),
					AbortIncompleteMultipartUpload: &types.AbortIncompleteMultipartUpload{DaysAfterInitiation: aws.Int32(2)},
				},
				{
					ID:                             aws.String("purptape-tracks-multipart-abort"),
					Status:                         types.ExpirationStatusEnabled,
					Prefix:                         aws.String("tracks/"),
					AbortIncompleteMultipartUpload: &types.AbortIncompleteMultipartUpload{DaysAfterInitiation: aws.Int32(2)},
				},
				{
					ID:                             aws.String("purptape-processed-retention"),
					Status:                         types.ExpirationStatusEnabled,
					Prefix:                         aws.String("processed/"),
					Expiration:                     &types.LifecycleExpiration{Days: aws.Int32(365)},
					AbortIncompleteMultipartUpload: &types.AbortIncompleteMultipartUpload{DaysAfterInitiation: aws.Int32(2)},
				},
			},
		},
	}

	if _, err := rc.client.PutBucketLifecycleConfiguration(ctx, input); err != nil {
		return fmt.Errorf("failed to apply bucket lifecycle policy: %w", err)
	}

	rc.lifecyclePolicyApplied.Store(1)
	rc.log.Info("R2 lifecycle policy applied", "bucket", rc.bucket)
	return nil
}

// DownloadFileToWriter streams an object from R2 to the provided writer.
func (rc *R2Client) DownloadFileToWriter(ctx context.Context, objectKey string, writer io.Writer) (int64, error) {
	resp, err := rc.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(rc.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get file from R2: %w", err)
	}
	defer resp.Body.Close()

	bytesWritten, err := io.Copy(writer, resp.Body)
	if err != nil {
		return bytesWritten, fmt.Errorf("failed to copy file from R2 stream: %w", err)
	}

	return bytesWritten, nil
}

func (rc *R2Client) ListFiles(ctx context.Context, prefix string, maxKeys int32) ([]string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(rc.bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(maxKeys),
	}

	output, err := rc.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	keys := make([]string, 0, len(output.Contents))
	for _, object := range output.Contents {
		keys = append(keys, aws.ToString(object.Key))
	}

	return keys, nil
}

// ValidateAudioFile checks if the file is a valid audio format
func ValidateAudioFile(filename string, fileSize int64) error {
	// Check file size (max 500MB)
	if fileSize > 500*1024*1024 {
		return fmt.Errorf("file too large: max 500MB, got %d bytes", fileSize)
	}

	// Check file extension
	validExtensions := map[string]bool{
		".wav":  true,
		".mp3":  true,
		".aiff": true,
		".flac": true,
		".aac":  true,
		".m4a":  true,
		".ogg":  true,
	}

	// Extract extension from filename
	var ext string
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			ext = filename[i:]
			break
		}
		if filename[i] == '/' {
			break
		}
	}

	if ext == "" || !validExtensions[ext] {
		return fmt.Errorf("unsupported audio format: %s", ext)
	}

	return nil
}

// ValidateImageFile checks if the file is a valid image format
func ValidateImageFile(filename string, fileSize int64) error {
	// Check file size (max 5MB for images)
	if fileSize > 5*1024*1024 {
		return fmt.Errorf("image too large: max 5MB, got %d bytes", fileSize)
	}

	// Check file extension
	validExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
		".gif":  true,
	}

	// Extract extension from filename
	var ext string
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			ext = filename[i:]
			break
		}
		if filename[i] == '/' {
			break
		}
	}

	if ext == "" || !validExtensions[ext] {
		return fmt.Errorf("unsupported image format: %s", ext)
	}

	return nil
}
