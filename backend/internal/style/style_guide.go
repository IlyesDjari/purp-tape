// Code Style and Best Practices Guide [LOW: Code organization]
//
// This file serves as a reference for coding standards in the PurpTape backend.
//
// === IMPORTS ===
// Organize imports in three groups (separated by blank lines):
// 1. Standard library (fmt, io, net/http, time, etc.)
// 2. External packages (github.com/..., database/sql, etc.)
// 3. Internal packages (./internal/...)
//
// Example:
//   import (
//   	"fmt"
//   	"log/slog"
//   	"net/http"
//   	"time"
//
//   	"github.com/jackc/pgx/v5/pgxpool"
//   	"github.com/google/uuid"
//
//   	"github.com/IlyesDjari/purp-tape/backend/internal/db"
//   	"github.com/IlyesDjari/purp-tape/backend/internal/errors"
//   )
//
// === NAMING ===
// - PackageNames: lowercase, concise (auth, db, handlers, not authentication)
// - TypeNames: PascalCase (User, CreateUserRequest, interface Validator)
// - FunctionNames: PascalCase for exported, camelCase for unexported
// - VarNames: short and meaningful (user, err, not u, e, userData)
// - ConstantNames: CONSTANT_CASE for file-level, or PascalCase (PreferredName OK too)
// - Parameters: Concise single letters OK for common types (w, r, ctx)
//
// === COMMENTS ===
// - Package-level: Start with package name
//   // Package auth provides authentication services
//
// - Function-level: Start with function name
//   // GetUser retrieves a user by ID or nil if not found
//
// - Inline: Explain WHY, not WHAT
//   // BAD: i = i + 1  // increment i
//   // GOOD: i++  // skip deleted users
//
// - FIX tags: Use for known issues
//   // FIX: This should use prepared statements for performance
//
// - TODO tags: Use for future improvements
//   // TODO: Add caching layer for frequently accessed users
//
// === ERROR HANDLING ===
// - Always return (T, error) pairs, not just error
// - Wrap errors with context: fmt.Errorf("operation: %w", err)
// - Check sql.ErrNoRows separately from other errors
// - Don't use panic in handlers (only in init)
// - Use errors.Is() and errors.As() for checking error types
//
// Pattern:
//   if err != nil {
//     return nil, fmt.Errorf("failed to get user: %w", err)
//   }
//
// === LOGGING ===
// - Use log/slog for all logging
// - Include relevant context (user_id, resource_id, etc.)
// - Use appropriate levels: Debug, Info, Warn, Error
// - Don't log passwords or sensitive data
//
// Pattern:
//   log.Error("operation failed",
//     "user_id", userID,
//     "error", err,
//   )
//
// === VALIDATION ===
// - Validate all inputs at entry points
// - Return 400 Bad Request for validation errors
// - Use validation.* functions from internal/validation
// - Check length, format, and value bounds
//
// Pattern:
//   if user.Email == "" {
//     return fmt.Errorf("email is required")
//   }
//   if !validation.IsValidEmail(user.Email) {
//     return fmt.Errorf("email format is invalid")
//   }
//
// === DATABASE ===
// - Use context with timeout
// - Always defer rows.Close()
// - Check sql.ErrNoRows for not-found vs error
// - Use transactions for multi-step operations
// - Never hard-delete (use soft delete with deleted_at)
//
// === HANDLERS ===
// - Extract and validate user ID first
// - Validate all request inputs
// - Check authorization after auth
// - Use consistent response formats
// - Log requests with context
//
// Pattern:
//   func (h *Handler) HandleRequest(w http.ResponseWriter, r *http.Request) {
//     userID, ok := helpers.GetUserIDSafe(r)
//     if !ok {
//       helpers.WriteUnauthorized(w)
//       return
//     }
//
//     var req RequestType
//     if err := helpers.SafeJSONDecode(r.Body, &req); err != nil {
//       helpers.WriteBadRequest(w, "invalid request")
//       return
//     }
//
//     if err := req.Validate(); err != nil {
//       helpers.WriteBadRequest(w, err.Error())
//       return
//     }
//
//     // Process request
//     result, err := h.service.DoSomething(r.Context(), req)
//     if err != nil {
//       h.log.Error("operation failed", "error", err)
//       helpers.WriteInternalError(w, h.log, err)
//       return
//     }
//
//     helpers.WriteJSON(w, http.StatusOK, result)
//   }
//
// === MIDDLEWARE ===
// - Use Chain() to compose multiple middleware
// - Apply in order from outer to inner
// - Recovery should be outermost
// - Log request details for debugging
//
// === CONSTANTS ===
// - Use constants instead of magic numbers
// - Place in consts package or constants/ file
// - Group related constants
// - Document units (e.g., MaxUploadSize = 100 * 1024 * 1024 // 100MB)
//
// === TESTING ===
// - Use table-driven tests for multiple cases
// - Mock interfaces, not concrete types
// - Use testify/assert for clear assertions
// - Test error cases, not just happy path
// - Run tests: go test ./...
// - Coverage report: go test -coverprofile=coverage.out && go tool cover -html=coverage.out
//
// === GIT COMMITS ===
// - Use conventional commits: feat:, fix:, refactor:, docs:, style:, test:
// - Be specific about what changed and why
// - Reference issues: "Fixes #123"
//
// === PERFORMANCE ===
// - Avoid N+1 queries (use JOIN or batch)
// - Use indexes for frequently queried columns
// - Use Redis/cache for hot data
// - Use pagination for large result sets
// - Profile with pprof before optimizing
//
// === SECURITY ===
// - Always validate user input
// - Use parameterized queries (no string interpolation)
// - Check authorization after authentication
// - Hash passwords with bcrypt
// - Use HTTPS in production
// - Log security events for audit trail
// - Sanitize file uploads
// - Use context with timeout to prevent resource exhaustion
//
// === CONFIGURATION ===
// - Use environment variables for all config
// - Never commit secrets
// - Validate config on startup
// - Log config (without secrets) on startup
//
// === DOCUMENTATION ===
// - Write package-level documentation in doc.go files
// - Document public types and functions
// - Include usage examples for complex packages
// - Keep documentation in sync with code
//
// See also:
// - Go Code Review Comments: https://github.com/golang/go/wiki/CodeReviewComments
// - Effective Go: https://golang.org/doc/effective_go
// - uber-go/guide: https://github.com/uber-go/guide
package style
