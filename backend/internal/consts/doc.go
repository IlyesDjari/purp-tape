// Package consts provides constants used across the application [LOW: Documentation]
//
// This package centralizes all magic numbers and string constants to:
// - Improve maintainability
// - Enable easy configuration changes
// - Provide clear documentation of limits and policies
// - Prevent errors from duplicated constants
//
// Organization:
// - HTTP Status Codes: Standard HTTP response codes
// - Time Durations: All time-based configuration
// - File Sizes: Storage and upload limits
// - Database: Connection pool settings
// - Pagination: Paging limits
// - Rate Limiting: Request limits
// - Content Types: MIME types
// - Environment: Deployment environment names
// - Audit: Audit logging constants
// - Status: Standard status strings
//
// Usage:
//
// DO:
//   import "github.com/IlyesDjari/purp-tape/backend/internal/consts"
//
//   if uploadSize > consts.MaxUploadSize {
//     return fmt.Errorf("file too large: max %d bytes", consts.MaxUploadSize)
//   }
//
//   w.Header().Set("Content-Type", consts.ContentTypeJSON)
//
// DON'T:
//   if uploadSize > 104857600 {  // What is 104857600?
//     return fmt.Errorf("file too large: max 100MB")
//   }
//
//   w.Header().Set("Content-Type", "application/json")
//
// Configuration:
// Some constants should be moved to environment variables in production:
// - MaxDBConnections -> DB_MAX_CONNS
// - MaxUploadSize -> MAX_UPLOAD_BYTES
// - RequestsPerMinute -> RATE_LIMIT_PER_MINUTE
//
// See internal/config/validate.go for environment variables
//
// Adding New Constants:
// 1. Determine the category (Time, File, Database, etc.)
// 2. Add to appropriate section with comment
// 3. Add descriptive comment with units
// 4. Update this documentation
// 5. Use throughout codebase instead of magic numbers
package consts

import _ "time" // Required for time constants
