package consts

import "time"

// HTTP Status Codes (using constants instead of magic numbers) [LOW: Best practices]
const (
	StatusOK                   = 200
	StatusCreated              = 201
	StatusNoContent            = 204
	StatusBadRequest           = 400
	StatusUnauthorized         = 401
	StatusForbidden            = 403
	StatusNotFound             = 404
	StatusConflict             = 409
	StatusUnprocessableEntity  = 422
	StatusTooManyRequests      = 429
	StatusInternalServerError  = 500
	StatusServiceUnavailable   = 503
)

// Time Durations (extracted constants) [LOW: Best practices]
const (
	JWKSCacheDuration         = 24 * time.Hour
	PresignedURLExpiry        = 15 * time.Minute
	ShareLinkDefaultExpiry    = 30 * 24 * time.Hour
	RequestTimeout            = 30 * time.Second
	SlowRequestThreshold      = 5 * time.Second
	TokenExpirationWindow     = 5 * time.Minute
	DatabasePingTimeout       = 5 * time.Second
	RateLimitWindow           = 1 * time.Minute
	SessionTimeout            = 24 * time.Hour
	GDPRDataRetentionDays     = 7 * 365 // 7 years
	BackupRetentionDays       = 30
)

// File Size Limits (extracted constants) [LOW: Best practices]
const (
	MaxUploadSize            = 100 * 1024 * 1024       // 100MB
	MaxAudioFileSize         = 500 * 1024 * 1024       // 500MB
	MaxImageFileSize         = 50 * 1024 * 1024        // 50MB
	MaxRequestBodySize       = 10 * 1024 * 1024        // 10MB
	BufferSize               = 32 * 1024               // 32KB streaming buffer
	DownloadBufferSize       = 32 * 1024               // 32KB download buffer
)

// Database Constants [LOW: Code organization]
const (
	MaxDBConnections         = 25
	MinDBConnections         = 3
	MaxIdleConnections       = 5
	ConnMaxIdleTime          = 30 * time.Second
	ConnMaxLifetime          = 5 * time.Minute
	QueryTimeout             = 30 * time.Second
	BatchInsertSize          = 1000
)

// Pagination Defaults [LOW: Consistency]
const (
	DefaultPageLimit         = 20
	MaxPageLimit             = 100
	DefaultPageOffset        = 0
	MaxPageOffset            = 999999
)

// Storage Quota Limits [LOW: Best practices]
const (
	FreePlanStorageQuotaMB   = 1024        // 1GB
	ProPlanStorageQuotaMB    = 50 * 1024   // 50GB
	EnterprisePlanStorageQuotaMB = 500 * 1024 // 500GB
)

// Rate Limiting [LOW: Code organization]
const (
	RequestsPerMinute        = 100
	BurstRequestLimit        = 200
)

// String Constants [LOW: Consistency]
const (
	ContentTypeJSON          = "application/json"
	ContentTypeFormData      = "multipart/form-data"
	ContentTypeOctetStream   = "application/octet-stream"
	CharsetUTF8              = "charset=utf-8"
	MediaTypePrometheus      = "text/plain; charset=utf-8"
)

// Environment Constants [LOW: Code organization]
const (
	EnvProduction            = "production"
	EnvStaging               = "staging"
	EnvDevelopment           = "development"
)

// Action Constants for Audit Logging [LOW: Consistency]
const (
	ActionCreate             = "create"
	ActionUpdate             = "update"
	ActionDelete             = "delete"
	ActionRead               = "read"
	ActionShare              = "share"
	ActionRevokeShare        = "revoke_share"
	ActionExport             = "export"
	ActionAccessDenied       = "access_denied"
)

// Resource Types [LOW: Consistency]
const (
	ResourceProject          = "project"
	ResourceTrack            = "track"
	ResourceUser             = "user"
	ResourceComment          = "comment"
	ResourceLike             = "like"
	ResourceProjectShare     = "project_share"
	ResourceUserData         = "user_data"
)

// Status Constants [LOW: Consistency]
const (
	StatusSuccess            = "success"
	StatusFailure            = "failure"
	StatusPending            = "pending"
	StatusProcessing         = "processing"
	StatusCompleted          = "completed"
	StatusFailed             = "failed"
	StatusDenied             = "denied"
)
