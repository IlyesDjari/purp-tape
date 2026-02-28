package models

import (
	"time"
)

// User represents a PurpTape user
type User struct {
	ID        string     `db:"id"`
	Email     string     `db:"email"`
	Username  string     `db:"username"`
	AvatarURL string     `db:"avatar_url"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"` // Soft delete timestamp
}

// Project represents a "Vault" - a collection of tracks
type Project struct {
	ID                 string     `db:"id"`
	UserID             string     `db:"user_id"`
	Name               string     `db:"name"`
	Description        string     `db:"description"`
	DescriptionFull    string     `db:"description_full"`
	IsPrivate          bool       `db:"is_private"`
	IsCollaborative    bool       `db:"is_collaborative"`
	CoverImageID       *string    `db:"cover_image_id"`
	Genre              string     `db:"genre"`
	ReleaseDate        *time.Time `db:"release_date"`
	PlayCount          int64      `db:"play_count"`
	LikeCount          int        `db:"like_count"`
	CreatedAt          time.Time  `db:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at"`
	DeletedAt          *time.Time `db:"deleted_at"` // Soft delete timestamp [CRITICAL FIX]
}

// Track represents an individual audio track
type Track struct {
	ID        string     `db:"id"`
	ProjectID string     `db:"project_id"`
	UserID    string     `db:"user_id"`
	Name      string     `db:"name"`
	Duration  int        `db:"duration"` // in seconds
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"` // Soft delete timestamp [CRITICAL FIX]
}

// TrackVersion represents a version of a track (v1, v2, etc.)
type TrackVersion struct {
	ID            string     `db:"id"`
	TrackID       string     `db:"track_id"`
	VersionNumber int        `db:"version_number"`
	R2ObjectKey   string     `db:"r2_object_key"` // path in Cloudflare R2
	FileSize      int64      `db:"file_size"`
	Checksum      string     `db:"checksum"` // SHA256 for integrity verification
	CreatedAt     time.Time  `db:"created_at"`
	DeletedAt     *time.Time `db:"deleted_at"` // Soft delete timestamp [CRITICAL FIX]
}

// ProjectShare represents access control (who can listen to a project)
type ProjectShare struct {
	ID           string     `db:"id"`
	ProjectID    string     `db:"project_id"`
	SharedByID   string     `db:"shared_by_id"`
	SharedWithID string     `db:"shared_with_id"`
	ShareToken   string     `db:"share_token"` // unique token for share links
	ExpiresAt    *time.Time `db:"expires_at"`
	CreatedAt    time.Time  `db:"created_at"`
}

// AuditLog represents actions taken in the system (for compliance & debugging)
type AuditLog struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	Action    string    `db:"action"`
	Resource  string    `db:"resource"`
	Details   string    `db:"details"` // JSON
	CreatedAt time.Time `db:"created_at"`
}

// Image represents uploaded images (covers, artwork)
type Image struct {
	ID         string    `db:"id"`
	UserID     string    `db:"user_id"`
	R2ObjectKey string   `db:"r2_object_key"`
	MimeType   string    `db:"mime_type"`
	FileSize   int64     `db:"file_size"`
	AltText    string    `db:"alt_text"`
	CreatedAt  time.Time `db:"created_at"`
}

// PlayHistory represents when/who listened to what
type PlayHistory struct {
	ID              string     `db:"id"`
	TrackVersionID  string     `db:"track_version_id"`
	ListenerUserID  *string    `db:"listener_user_id"` // null = anonymous
	TrackID         string     `db:"track_id"`
	ProjectID       string     `db:"project_id"`
	DurationListened *int      `db:"duration_listened"` // seconds
	StartedAt       time.Time  `db:"started_at"`
	EndedAt         *time.Time `db:"ended_at"`
	Device          string     `db:"device"`
	Country         string     `db:"country"`
}

// Comment represents feedback on a track
type Comment struct {
	ID             string    `db:"id"`
	UserID         string    `db:"user_id"`
	TrackVersionID string    `db:"track_version_id"`
	Content        string    `db:"content"`
	TimestampMs    *int      `db:"timestamp_ms"` // where in track (milliseconds)
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// Like represents a user liking a track
type Like struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	TrackID   string    `db:"track_id"`
	CreatedAt time.Time `db:"created_at"`
}

// ProjectLike represents a user liking a project
type ProjectLike struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	ProjectID string    `db:"project_id"`
	CreatedAt time.Time `db:"created_at"`
}

// UserFollow represents a user following another user
type UserFollow struct {
	ID          string    `db:"id"`
	FollowerID  string    `db:"follower_id"`
	FollowingID string    `db:"following_id"`
	CreatedAt   time.Time `db:"created_at"`
}

// Notification represents a notification for a user
type Notification struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	ActorUserID *string `db:"actor_user_id"`
	Type      string    `db:"type"` // "like", "comment", "follow", "share"
	TrackID   *string   `db:"track_id"`
	ProjectID *string   `db:"project_id"`
	CommentID *string   `db:"comment_id"`
	Content   string    `db:"content"`
	IsRead    bool      `db:"is_read"`
	CreatedAt time.Time `db:"created_at"`
}

// Collaborator represents project access
type Collaborator struct {
	ID        string    `db:"id"`
	ProjectID string    `db:"project_id"`
	UserID    string    `db:"user_id"`
	Role      string    `db:"role"` // "owner", "editor", "commenter", "viewer"
	InvitedAt time.Time `db:"invited_at"`
	JoinedAt  *time.Time `db:"joined_at"`
}

// Tag represents a category tag
type Tag struct {
	ID          string    `db:"id"`
	Name        string    `db:"name"`
	Slug        string    `db:"slug"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
}
// ShareLink represents a cryptographic share link for projects
type ShareLink struct {
	ID                  string     `db:"id"`
	ProjectID           string     `db:"project_id"`
	CreatorID           string     `db:"creator_id"`
	Hash                string     `db:"hash"`
	AccessLevel         string     `db:"access_level"` // viewer, commenter, collaborator
	IsPublic            bool       `db:"is_public"`
	IsPasswordProtected bool       `db:"is_password_protected"`
	PasswordHash        string     `db:"password_hash"`
	ExpiresAt           *time.Time `db:"expires_at"`
	RevokedAt           *time.Time `db:"revoked_at"`
	CreatedAt           time.Time  `db:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at"`
}

// Subscription represents user's subscription tier
type Subscription struct {
	ID                    string     `db:"id"`
	UserID                string     `db:"user_id"`
	IsPremium             bool       `db:"is_premium"`
	Tier                  string     `db:"tier"` // free, pro, pro_plus, unlimited
	StorageQuotaMB        int64      `db:"storage_quota_mb"`
	StorageUsedMB         int64      `db:"storage_used_mb"`
	StripeCustomerID      string     `db:"stripe_customer_id"`
	StripeSubscriptionID  string     `db:"stripe_subscription_id"`
	PurchaseDate          *time.Time `db:"purchase_date"`
	RenewalDate           *time.Time `db:"renewal_date"`
	CanceledAt            *time.Time `db:"canceled_at"`
	CreatedAt             time.Time  `db:"created_at"`
	UpdatedAt             time.Time  `db:"updated_at"`
}

// OfflineDownload represents a track downloaded for offline playback
type OfflineDownload struct {
	ID                 string     `db:"id"`
	UserID             string     `db:"user_id"`
	TrackVersionID     string     `db:"track_version_id"`
	TrackID            string     `db:"track_id"`
	ProjectID          string     `db:"project_id"`
	FileSizeBytes      int64      `db:"file_size_bytes"`
	R2ObjectKey        string     `db:"r2_object_key"`
	LocalFileHash      string     `db:"local_file_hash"` // SHA256 on device
	Status             string     `db:"status"`           // pending, downloading, completed, failed, removed
	DownloadProgressPercent int   `db:"download_progress_percent"`
	Title              string     `db:"title"`
	ArtistName         string     `db:"artist_name"`
	ProjectName        string     `db:"project_name"`
	CoverImageURL      string     `db:"cover_image_url"`
	DurationSeconds    int        `db:"duration_seconds"`
	LocalPathHash      string     `db:"local_path_hash"`
	StorageUsedBytes   int64      `db:"storage_used_bytes"`
	DownloadedAt       *time.Time `db:"downloaded_at"`
	LastPlayedAt       *time.Time `db:"last_played_at"`
	CreatedAt          time.Time  `db:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at"`
	ExpiresAt          *time.Time `db:"expires_at"`
}