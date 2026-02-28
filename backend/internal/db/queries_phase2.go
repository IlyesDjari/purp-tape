package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/IlyesDjari/purp-tape/backend/internal/models"
)

func (db *Database) UpdateProjectPrivacy(ctx context.Context, projectID string, isPrivate bool, genre string, releaseDate *time.Time) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE projects
		 SET is_private = $1,
		     genre = COALESCE(NULLIF($2, ''), genre),
		     release_date = $3,
		     updated_at = NOW()
		 WHERE id = $4`,
		isPrivate, genre, releaseDate, projectID,
	)
	return err
}

func (db *Database) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := db.pool.QueryRow(ctx,
		`SELECT id, email, username, avatar_url, created_at, updated_at, deleted_at
		 FROM users
		 WHERE email = $1 AND deleted_at IS NULL`,
		email,
	).Scan(&user.ID, &user.Email, &user.Username, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (db *Database) AddCollaborator(ctx context.Context, collab *models.Collaborator) error {
	_, err := db.pool.Exec(ctx,
		`INSERT INTO collaborators (id, project_id, user_id, role, invited_at, joined_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (project_id, user_id) DO UPDATE SET role = EXCLUDED.role, invited_at = EXCLUDED.invited_at`,
		collab.ID, collab.ProjectID, collab.UserID, collab.Role, collab.InvitedAt, collab.JoinedAt,
	)
	return err
}

func (db *Database) CreateLike(ctx context.Context, like *models.Like) error {
	_, err := db.pool.Exec(ctx,
		`INSERT INTO likes (id, user_id, track_id, created_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (track_id, user_id) DO NOTHING`,
		like.ID, like.UserID, like.TrackID, like.CreatedAt,
	)
	return err
}

func (db *Database) DeleteLike(ctx context.Context, userID, trackID string) error {
	_, err := db.pool.Exec(ctx,
		`DELETE FROM likes WHERE user_id = $1 AND track_id = $2`,
		userID, trackID,
	)
	return err
}

func (db *Database) CreateComment(ctx context.Context, comment *models.Comment) error {
	_, err := db.pool.Exec(ctx,
		`INSERT INTO comments (id, user_id, track_version_id, content, timestamp_ms, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		comment.ID, comment.UserID, comment.TrackVersionID, comment.Content, comment.TimestampMs, comment.CreatedAt, comment.UpdatedAt,
	)
	return err
}

func (db *Database) GetComments(ctx context.Context, trackVersionID string) ([]models.Comment, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, user_id, track_version_id, content, timestamp_ms, created_at, updated_at
		 FROM comments
		 WHERE track_version_id = $1
		 ORDER BY created_at ASC`,
		trackVersionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var c models.Comment
		if err := rows.Scan(&c.ID, &c.UserID, &c.TrackVersionID, &c.Content, &c.TimestampMs, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (db *Database) CreateFollow(ctx context.Context, follow *models.UserFollow) error {
	_, err := db.pool.Exec(ctx,
		`INSERT INTO user_follows (id, follower_id, following_id, created_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (follower_id, following_id) DO NOTHING`,
		follow.ID, follow.FollowerID, follow.FollowingID, follow.CreatedAt,
	)
	return err
}

func (db *Database) DeleteFollow(ctx context.Context, followerID, followingID string) error {
	_, err := db.pool.Exec(ctx,
		`DELETE FROM user_follows WHERE follower_id = $1 AND following_id = $2`,
		followerID, followingID,
	)
	return err
}

func (db *Database) GetFollowing(ctx context.Context, userID string) ([]models.User, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT u.id, u.email, u.username, u.avatar_url, u.created_at, u.updated_at, u.deleted_at
		 FROM user_follows uf
		 JOIN users u ON u.id = uf.following_id
		 WHERE uf.follower_id = $1
		 ORDER BY uf.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (db *Database) GetFollowers(ctx context.Context, userID string) ([]models.User, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT u.id, u.email, u.username, u.avatar_url, u.created_at, u.updated_at, u.deleted_at
		 FROM user_follows uf
		 JOIN users u ON u.id = uf.follower_id
		 WHERE uf.following_id = $1
		 ORDER BY uf.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (db *Database) GetNotifications(ctx context.Context, userID string) ([]models.Notification, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, user_id, actor_user_id, type, track_id, project_id, comment_id, content, is_read, created_at
		 FROM notifications
		 WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT 100`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.ActorUserID, &n.Type, &n.TrackID, &n.ProjectID, &n.CommentID, &n.Content, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

func (db *Database) MarkNotificationAsRead(ctx context.Context, notificationID string) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE notifications SET is_read = true WHERE id = $1`,
		notificationID,
	)
	return err
}

func (db *Database) GetSoftDeletedProjects(ctx context.Context, olderThan time.Duration) ([]models.Project, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, user_id, name, description, created_at, updated_at, deleted_at
		 FROM projects
		 WHERE deleted_at IS NOT NULL AND deleted_at < NOW() - $1::interval`,
		fmt.Sprintf("%d seconds", int64(olderThan.Seconds())),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (db *Database) GetAllProjectTrackVersions(ctx context.Context, projectID string) ([]models.TrackVersion, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT tv.id, tv.track_id, tv.version_number, tv.r2_object_key, tv.file_size, tv.checksum, tv.created_at, tv.deleted_at
		 FROM track_versions tv
		 JOIN tracks t ON t.id = tv.track_id
		 WHERE t.project_id = $1`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []models.TrackVersion
	for rows.Next() {
		var v models.TrackVersion
		if err := rows.Scan(&v.ID, &v.TrackID, &v.VersionNumber, &v.R2ObjectKey, &v.FileSize, &v.Checksum, &v.CreatedAt, &v.DeletedAt); err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}

func (db *Database) LogR2CleanupFailure(ctx context.Context, r2ObjectKey, errorMessage string) error {
	_, err := db.pool.Exec(ctx,
		`INSERT INTO audit_logs (action, resource, details, created_at)
		 VALUES ('r2_cleanup_failure', $1, jsonb_build_object('error', $2), NOW())`,
		r2ObjectKey,
		errorMessage,
	)
	return err
}

func (db *Database) HardDeleteProject(ctx context.Context, projectID string) error {
	_, err := db.pool.Exec(ctx,
		`DELETE FROM projects WHERE id = $1`,
		projectID,
	)
	return err
}

func (db *Database) CreateImage(ctx context.Context, image *models.Image) error {
	_, err := db.pool.Exec(ctx,
		`INSERT INTO images (id, user_id, r2_object_key, mime_type, file_size, alt_text, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		image.ID, image.UserID, image.R2ObjectKey, image.MimeType, image.FileSize, image.AltText, image.CreatedAt,
	)
	return err
}

func (db *Database) UpdateProjectCover(ctx context.Context, projectID, imageID string) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE projects SET cover_image_id = $1, updated_at = NOW() WHERE id = $2`,
		imageID, projectID,
	)
	return err
}

func (db *Database) GetUserByStripeCustomerID(ctx context.Context, customerID string) (*models.User, error) {
	var user models.User
	err := db.pool.QueryRow(ctx,
		`SELECT u.id, u.email, u.username, u.avatar_url, u.created_at, u.updated_at, u.deleted_at
		 FROM users u
		 JOIN subscriptions s ON s.user_id = u.id
		 WHERE s.stripe_customer_id = $1
		 LIMIT 1`,
		customerID,
	).Scan(&user.ID, &user.Email, &user.Username, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (db *Database) GetUserByStripeSubscriptionID(ctx context.Context, subscriptionID string) (*models.User, error) {
	var user models.User
	err := db.pool.QueryRow(ctx,
		`SELECT u.id, u.email, u.username, u.avatar_url, u.created_at, u.updated_at, u.deleted_at
		 FROM users u
		 JOIN subscriptions s ON s.user_id = u.id
		 WHERE s.stripe_subscription_id = $1
		 LIMIT 1`,
		subscriptionID,
	).Scan(&user.ID, &user.Email, &user.Username, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (db *Database) UpsertSubscriptionFromPayment(ctx context.Context, userID string, isPremium bool, tier, stripeCustomerID, stripeSubscriptionID string, purchaseDate, canceledAt *time.Time) error {
	_, err := db.pool.Exec(ctx,
		`INSERT INTO subscriptions (
			user_id,
			is_premium,
			tier,
			stripe_customer_id,
			stripe_subscription_id,
			purchase_date,
			canceled_at,
			created_at,
			updated_at
		 )
		 VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6, $7, NOW(), NOW())
		 ON CONFLICT (user_id)
		 DO UPDATE SET
			is_premium = EXCLUDED.is_premium,
			tier = EXCLUDED.tier,
			stripe_customer_id = COALESCE(EXCLUDED.stripe_customer_id, subscriptions.stripe_customer_id),
			stripe_subscription_id = COALESCE(EXCLUDED.stripe_subscription_id, subscriptions.stripe_subscription_id),
			purchase_date = COALESCE(EXCLUDED.purchase_date, subscriptions.purchase_date),
			canceled_at = EXCLUDED.canceled_at,
			updated_at = NOW()`,
		userID,
		isPremium,
		tier,
		stripeCustomerID,
		stripeSubscriptionID,
		purchaseDate,
		canceledAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert subscription from payment: %w", err)
	}
	return nil
}

func (db *Database) UpdateSubscription(ctx context.Context, userID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	_, err := db.pool.Exec(ctx,
		`UPDATE subscriptions
		 SET is_premium = COALESCE($1, is_premium),
		     tier = COALESCE($2, tier),
		     stripe_customer_id = COALESCE($3, stripe_customer_id),
		     stripe_subscription_id = COALESCE($4, stripe_subscription_id),
		     purchase_date = COALESCE($5, purchase_date),
		     canceled_at = COALESCE($6, canceled_at),
		     updated_at = NOW()
		 WHERE user_id = $7`,
		updates["is_premium"],
		updates["tier"],
		updates["stripe_customer_id"],
		updates["stripe_subscription_id"],
		updates["purchase_date"],
		updates["canceled_at"],
		userID,
	)
	return err
}

func (db *Database) CreateShareLink(ctx context.Context, shareLink *models.ShareLink) error {
	_, err := db.pool.Exec(ctx,
		`INSERT INTO share_links (id, project_id, creator_id, hash, access_level, is_public, is_password_protected, password_hash, expires_at, revoked_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		shareLink.ID,
		shareLink.ProjectID,
		shareLink.CreatorID,
		shareLink.Hash,
		shareLink.AccessLevel,
		shareLink.IsPublic,
		shareLink.IsPasswordProtected,
		shareLink.PasswordHash,
		shareLink.ExpiresAt,
		shareLink.RevokedAt,
		shareLink.CreatedAt,
		shareLink.UpdatedAt,
	)
	return err
}

func (db *Database) GetShareLinkByHash(ctx context.Context, hash string) (*models.ShareLink, error) {
	var shareLink models.ShareLink
	err := db.pool.QueryRow(ctx,
		`SELECT id, project_id, creator_id, hash, access_level, is_public, is_password_protected, password_hash, expires_at, revoked_at, created_at, updated_at
		 FROM share_links
		 WHERE hash = $1`,
		hash,
	).Scan(
		&shareLink.ID,
		&shareLink.ProjectID,
		&shareLink.CreatorID,
		&shareLink.Hash,
		&shareLink.AccessLevel,
		&shareLink.IsPublic,
		&shareLink.IsPasswordProtected,
		&shareLink.PasswordHash,
		&shareLink.ExpiresAt,
		&shareLink.RevokedAt,
		&shareLink.CreatedAt,
		&shareLink.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &shareLink, nil
}

func (db *Database) RevokeShareLink(ctx context.Context, hash string) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE share_links SET revoked_at = NOW(), updated_at = NOW() WHERE hash = $1`,
		hash,
	)
	return err
}
