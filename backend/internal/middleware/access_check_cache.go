package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/IlyesDjari/purp-tape/backend/internal/cache"
	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/helpers"
)

// AccessCheckCache provides high-performance access validation
// Uses Redis to cache user access lists, reducing database load by 10-100x
type AccessCheckCache struct {
	db    *db.Database
	cache *cache.AccessListCache
	log   *slog.Logger
}

// NewAccessCheckCache creates a new access check cache middleware
func NewAccessCheckCache(database *db.Database, cacheManager *cache.AccessListCache, log *slog.Logger) *AccessCheckCache {
	return &AccessCheckCache{
		db:    database,
		cache: cacheManager,
		log:   log,
	}
}

// CheckProjectAccessCached validates project access using Redis cache first, DB fallback
// Returns (hasAccess, errorIfAny)
func (acc *AccessCheckCache) CheckProjectAccessCached(ctx context.Context, userID, projectID string) (bool, error) {
	// 1. Try Redis cache (1M access checks/sec for hits)
	if hasAccess, cacheHit := acc.cache.GetProjectAccessibility(ctx, userID, projectID); cacheHit {
		return hasAccess, nil
	}

	// 2. Check database (O(1) with denormalized table)
	hasAccess, err := acc.db.CanUserAccessProject(ctx, userID, projectID)
	if err != nil {
		acc.log.Warn("access check failed", "error", err, "user_id", userID, "project_id", projectID)
		return false, err
	}

	// 3. Cache the result for next check
	go func() {
		if err := acc.cache.CacheProjectAccessibility(ctx, userID, projectID, hasAccess); err != nil {
			acc.log.Debug("failed to cache access result", "error", err)
		}
	}()

	return hasAccess, nil
}

// CheckTrackAccessCached validates track access with caching
func (acc *AccessCheckCache) CheckTrackAccessCached(ctx context.Context, userID, trackID string) (bool, error) {
	// Try cache first
	if hasAccess, cacheHit := acc.cache.GetProjectAccessibility(ctx, userID, trackID); cacheHit {
		return hasAccess, nil
	}

	// Check database
	hasAccess, err := acc.db.CanUserAccessTrack(ctx, userID, trackID)
	if err != nil {
		return false, err
	}

	// Cache result
	go func() {
		if err := acc.cache.CacheProjectAccessibility(ctx, userID, trackID, hasAccess); err != nil {
			acc.log.Debug("failed to cache track access", "error", err)
		}
	}()

	return hasAccess, nil
}

// ProjectAccessMiddleware wraps handler with project access validation
// Validates that user can access the project in path parameter
func (acc *AccessCheckCache) ProjectAccessMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := helpers.GetUserID(r)
		if err != nil {
			helpers.WriteUnauthorized(w)
			return
		}

		projectID := r.PathValue("project_id")
		if projectID == "" {
			helpers.WriteBadRequest(w, "missing project_id")
			return
		}

		// Check access with caching
		hasAccess, err := acc.CheckProjectAccessCached(r.Context(), userID, projectID)
		if err != nil {
			acc.log.Error("access check error", "error", err)
			helpers.WriteInternalError(w, acc.log, err)
			return
		}

		if !hasAccess {
			helpers.WriteForbidden(w, "you do not have access to this project")
			return
		}

		next(w, r)
	}
}

// TrackAccessMiddleware wraps handler with track access validation
func (acc *AccessCheckCache) TrackAccessMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := helpers.GetUserID(r)
		if err != nil {
			helpers.WriteUnauthorized(w)
			return
		}

		trackID := r.PathValue("track_id")
		if trackID == "" {
			helpers.WriteBadRequest(w, "missing track_id")
			return
		}

		hasAccess, err := acc.CheckTrackAccessCached(r.Context(), userID, trackID)
		if err != nil {
			acc.log.Error("track access check error", "error", err)
			helpers.WriteInternalError(w, acc.log, err)
			return
		}

		if !hasAccess {
			helpers.WriteForbidden(w, "you do not have access to this track")
			return
		}

		next(w, r)
	}
}

// InvalidateUserAccessOnPermissionChange clears user access cache after permission changes
// Call this whenever:
// - User gains/loses project access
// - Sharing is changed
// - Collaborators are modified
func (acc *AccessCheckCache) InvalidateUserAccessOnPermissionChange(ctx context.Context, userID string) error {
	return acc.cache.InvalidateUserAccessCache(ctx, userID)
}

// PreloadUserProjectAccessList loads and caches user's full project access list
// Useful for batch operations or reducing cache misses
// Returns cached list or DB list if not cached
func (acc *AccessCheckCache) PreloadUserProjectAccessList(ctx context.Context, userID string) ([]string, error) {
	// Try cache first
	if projectIDs, cacheHit := acc.cache.GetUserProjectAccessList(ctx, userID); cacheHit {
		return projectIDs, nil
	}

	// Load from database
	projectIDs, err := acc.db.GetUserProjectAccessList(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Store in cache
	go func() {
		if err := acc.cache.CacheUserProjectAccessList(ctx, userID, projectIDs); err != nil {
			acc.log.Debug("failed to cache project access list", "error", err)
		}
	}()

	return projectIDs, nil
}

// PreloadUserTrackAccessList loads and caches user's full track access list
func (acc *AccessCheckCache) PreloadUserTrackAccessList(ctx context.Context, userID string) ([]string, error) {
	// Try cache first
	if trackIDs, cacheHit := acc.cache.GetUserTrackAccessList(ctx, userID); cacheHit {
		return trackIDs, nil
	}

	// Load from database
	trackIDs, err := acc.db.GetUserTrackAccessList(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Store in cache
	go func() {
		if err := acc.cache.CacheUserTrackAccessList(ctx, userID, trackIDs); err != nil {
			acc.log.Debug("failed to cache track access list", "error", err)
		}
	}()

	return trackIDs, nil
}
