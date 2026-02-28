package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
)

// Note: QueryUserProjectRole and three single-query functions (IsProjectOwner, GetCollaboratorRole, IsProjectSharedWith)
// have been REMOVED. Use GetUserRoleOptimized() instead for single-query role resolution.
// This saves 66% of database round-trips for access control checks.

// RBACMiddleware enforces role-based access control using database queries.
// NEVER trusts X-User-Role or X-Is-Owner headers - they can be forged
func RBACMiddleware(database *db.Database, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value("user_id").(string)
			if !ok || userID == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			projectID := r.PathValue("project_id")
			if projectID == "" {
				http.Error(w, "missing project_id", http.StatusBadRequest)
				return
			}

			// Query database for actual role (NOT from headers)
			// NEVER trust X-User-Role or X-Is-Owner headers - they can be forged
			// Log any attempt to forge role via headers for audit
			if claimedRole := r.Header.Get("X-User-Role"); claimedRole != "" {
				log.Warn("rejected forged role header",
					"user_id", userID,
					"claimed_role", claimedRole)
			}

			// Optimized single-query lookup for performance
			role, err := database.GetUserRoleOptimized(r.Context(), projectID, userID)
			if err != nil {
				log.Error("failed to query user role",
					"error", err,
					"user_id", userID,
					"project_id", projectID)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			if role == "denied" {
				log.Warn("access denied",
					"user_id", userID,
					"project_id", projectID,
					"reason", "no access to project")
				http.Error(w, "access denied", http.StatusForbidden)
				return
			}

			// Store role in context (from trusted database query)
			ctx := context.WithValue(r.Context(), "user_role", role)
			ctx = context.WithValue(ctx, "project_id", projectID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Role hierarchy: owner > editor > commenter > viewer
// Each role inherits permissions of lower roles
func checkRoleAccess(userRole, requiredRole string) bool {
	// Define role hierarchy (higher index = more permissions)
	hierarchy := map[string]int{
		"viewer":       0,
		"commenter":    1,
		"editor":       2,
		"collaborator": 2, // same as editor
		"owner":        3,
	}

	userLevel, userExists := hierarchy[userRole]
	requiredLevel, requiredExists := hierarchy[requiredRole]

	if !userExists || !requiredExists {
		return false
	}

	return userLevel >= requiredLevel
}

// EnforceRole checks if user has minimum required role.
// Uses role from context (already DB-verified by RBACMiddleware)
func EnforceRole(requiredRole string, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Role is already in context (from RBACMiddleware)
			userRole, ok := r.Context().Value("user_role").(string)
			if !ok || userRole == "" {
				http.Error(w, "permission denied", http.StatusForbidden)
				return
			}

			// Check role hierarchy
			if !checkRoleAccess(userRole, requiredRole) {
				userID, _ := r.Context().Value("user_id").(string)
				projectID, _ := r.Context().Value("project_id").(string)
				log.Warn("insufficient role",
					"user_id", userID,
					"project_id", projectID,
					"required", requiredRole,
					"actual", userRole)
				http.Error(w, "insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// EnforceOwnership checks if user is the project owner.
func EnforceOwnership(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get role from context (DB-verified)
			userRole, ok := r.Context().Value("user_role").(string)
			if !ok || userRole != "owner" {
				userID, _ := r.Context().Value("user_id").(string)
				projectID, _ := r.Context().Value("project_id").(string)
				log.Warn("ownership check failed",
					"user_id", userID,
					"project_id", projectID)
				http.Error(w, "only project owner can perform this action", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// EnforceModificationAccess checks if user can modify content
// Owners and Editors can modify; Commenters and Viewers cannot
func EnforceModificationAccess(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get role from context (DB-verified)
			role, ok := r.Context().Value("user_role").(string)
			if !ok {
				http.Error(w, "permission denied", http.StatusForbidden)
				return
			}

			// Check if role allows modifications
			canModify := checkRoleAccess(role, "editor")
			if !canModify {
				userID, _ := r.Context().Value("user_id").(string)
				projectID, _ := r.Context().Value("project_id").(string)
				log.Warn("modification denied",
					"user_id", userID,
					"project_id", projectID,
					"role", role)
				http.Error(w, "insufficient permissions to modify content", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
