// Package handlers provides HTTP request handlers for the PurpTape API [LOW: Documentation]
//
// Handler Organization:
// - AnalyticsHandlers: Project analytics and statistics
// - CollaborationHandlers: Project collaboration features
// - ComplianceHandlers: GDPR and privacy compliance
// - DownloadHandlers: File streaming downloads
// - HealthHandler: Application health checks
// - ImageHandlers: Image upload and management
// - OfflineHandlers: Offline mode functionality
// - PaymentHandlers: Stripe and RevenueCat webhooks
// - ProjectHandlers: Project CRUD operations
// - RollbackHandlers: Database rollback functionality
// - SearchHandlers: Full-text search
// - ShareHandlers: Project sharing and link generation
// - TrackHandlers: Track upload and management
//
// Handler Pattern:
//   type XxxHandlers struct {
//     db  *db.Database
//     log *slog.Logger
//   }
//
//   func NewXxxHandlers(database *db.Database, log *slog.Logger) *XxxHandlers {
//     return &XxxHandlers{db: database, log: log}
//   }
//
//   func (h *XxxHandlers) HandleRequest(w http.ResponseWriter, r *http.Request) {
//     // Implementation
//   }
//
// Error Handling:
// - Use helpers.WriteXxx() functions for standard HTTP responses
// - Use errors.APIError for structured error responses
// - Log errors with context using log.Error()
//
// Authentication:
// - All endpoints protected by auth middleware
// - User ID extracted from request context via helpers.GetUserID(r)
// - Check authorization after extracting user ID
//
// Input Validation:
// - Validate all request inputs before processing
// - Use validation.* functions from internal/validation package
// - Return 400 Bad Request for validation errors
//
// Database Access:
// - All database queries use context with timeout
// - Use transaction helpers for multi-step operations
// - Log database errors with context
//
// Logging:
// - Use structured logging with slog
// - Include relevant context: user_id, resource_id, method, status
// - Use log.Debug for detailed, log.Info for important, log.Warn for issues, log.Error for failures
package handlers
